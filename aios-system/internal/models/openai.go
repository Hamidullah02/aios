package models

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

// OpenAIAdapter calls the OpenAI Chat Completions API
// (REQ-MOD-ADP-003).
//
// The adapter reads OPENAI_API_KEY from the environment at
// construction time. Override with Config.APIKey for tests.
// Override the HTTP client with Config.Client for httptest.
// Override the base URL with Config.Base for staging / proxies.
type OpenAIAdapter struct {
	apiKey string
	client *http.Client
	base   string
}

// OpenAIConfig is the constructor input.
type OpenAIConfig struct {
	APIKey string       // empty -> use $OPENAI_API_KEY
	Client *http.Client // nil   -> http.DefaultClient
	Base   string       // empty -> https://api.openai.com/v1
}

// NewOpenAI returns an OpenAIAdapter. apiKey may be empty; in
// that case Complete/Stream return an error at call time.
func NewOpenAI(cfg OpenAIConfig) *OpenAIAdapter {
	key := cfg.APIKey
	if key == "" {
		key = os.Getenv("OPENAI_API_KEY")
	}
	client := cfg.Client
	if client == nil {
		client = http.DefaultClient
	}
	base := cfg.Base
	if base == "" {
		base = "https://api.openai.com/v1"
	}
	return &OpenAIAdapter{apiKey: key, client: client, base: base}
}

func (a *OpenAIAdapter) Provider() string { return "openai" }

// chatRequest is the wire shape for the /chat/completions
// endpoint.
type chatRequest struct {
	Model       string    `json:"model"`
	Messages    []message `json:"messages"`
	Temperature float64   `json:"temperature"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Stream      bool      `json:"stream"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Choices []struct {
		Message message `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
	Model string `json:"model"`
}

func (a *OpenAIAdapter) Complete(ctx context.Context, req Request) (Response, error) {
	body, err := json.Marshal(chatRequest{
		Model: req.Model,
		Messages: []message{
			{Role: "system", Content: req.Options.System},
			{Role: "user", Content: req.Prompt},
		},
		Temperature: req.Options.Temperature,
		MaxTokens:   req.Options.MaxTokens,
	})
	if err != nil {
		return Response{}, err
	}
	httpReq, err := a.newRequest(ctx, "POST", "/chat/completions", body)
	if err != nil {
		return Response{}, err
	}
	resp, err := a.client.Do(httpReq)
	if err != nil {
		return Response{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Response{}, readProviderError("openai", resp)
	}
	var cr chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&cr); err != nil {
		return Response{}, fmt.Errorf("openai: decode: %w", err)
	}
	if len(cr.Choices) == 0 {
		return Response{}, &ProviderError{Provider: "openai", Code: "no_choices", Message: "no choices in response"}
	}
	return Response{
		Text: cr.Choices[0].Message.Content,
		Usage: Usage{
			Model:        cr.Model,
			InputTokens:  cr.Usage.PromptTokens,
			OutputTokens: cr.Usage.CompletionTokens,
		},
	}, nil
}

// openaiStream implements Stream. It scans newline-delimited
// JSON; each line is one chat-completion-chunk. The final
// chunk carries the usage.
type openaiStream struct {
	body    io.ReadCloser
	scanner *bufio.Scanner
	usage   Usage
	done    bool
}

func (s *openaiStream) Close() error { return s.body.Close() }
func (s *openaiStream) Usage() Usage { return s.usage }

func (s *openaiStream) Next() (Chunk, error) {
	if s.done {
		return Chunk{}, io.EOF
	}
	for s.scanner.Scan() {
		line := s.scanner.Bytes()
		if len(line) == 0 || line[0] == ':' {
			continue
		}
		if len(line) > 6 && string(line[:6]) == "data: " {
			line = line[6:]
		}
		if string(line) == "[DONE]" {
			s.done = true
			return Chunk{Usage: &s.usage}, io.EOF
		}
		var cr chatResponse
		if err := json.Unmarshal(line, &cr); err != nil {
			return Chunk{}, fmt.Errorf("openai: stream decode: %w", err)
		}
		// The final chunk often has an empty choices array and
		// only the usage block. Capture the usage when we see
		// it, and end the stream.
		if cr.Usage.PromptTokens > 0 || cr.Usage.CompletionTokens > 0 {
			s.usage = Usage{
				Model:        cr.Model,
				InputTokens:  cr.Usage.PromptTokens,
				OutputTokens: cr.Usage.CompletionTokens,
			}
			if len(cr.Choices) == 0 || cr.Choices[0].Message.Content == "" {
				s.done = true
				return Chunk{Usage: &s.usage}, io.EOF
			}
		}
		if len(cr.Choices) > 0 {
			return Chunk{Delta: cr.Choices[0].Message.Content}, nil
		}
	}
	if err := s.scanner.Err(); err != nil {
		return Chunk{}, err
	}
	s.done = true
	return Chunk{Usage: &s.usage}, io.EOF
}

func (a *OpenAIAdapter) Stream(ctx context.Context, req Request) (Stream, error) {
	body, _ := json.Marshal(chatRequest{
		Model: req.Model, Stream: true,
		Messages:    []message{{Role: "user", Content: req.Prompt}},
		Temperature: req.Options.Temperature,
		MaxTokens:   req.Options.MaxTokens,
	})
	httpReq, err := a.newRequest(ctx, "POST", "/chat/completions", body)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Accept", "text/event-stream")
	resp, err := a.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		return nil, readProviderError("openai", resp)
	}
	return &openaiStream{
		body:    resp.Body,
		scanner: bufio.NewScanner(resp.Body),
	}, nil
}

func (a *OpenAIAdapter) newRequest(ctx context.Context, method, path string, body []byte) (*http.Request, error) {
	if a.apiKey == "" {
		return nil, &ProviderError{Provider: "openai", Code: "no_api_key", Message: "OPENAI_API_KEY is not set"}
	}
	req, err := http.NewRequestWithContext(ctx, method, a.base+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+a.apiKey)
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

// readProviderError reads the body and returns a ProviderError.
func readProviderError(provider string, resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)
	return &ProviderError{
		Provider: provider,
		Code:     fmt.Sprintf("http_%d", resp.StatusCode),
		Message:  string(body),
	}
}
