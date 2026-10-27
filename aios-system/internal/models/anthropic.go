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

// AnthropicAdapter calls the Anthropic Messages API
// (REQ-MOD-ADP-004).
//
// The adapter reads ANTHROPIC_API_KEY from the environment.
// Override via Config.APIKey for tests; override the HTTP
// client via Config.Client for httptest.
type AnthropicAdapter struct {
	apiKey string
	client *http.Client
	base   string
}

// AnthropicConfig is the constructor input.
type AnthropicConfig struct {
	APIKey string
	Client *http.Client
	Base   string // empty -> https://api.anthropic.com/v1
}

// EnvAnthropicBase is the environment variable that overrides the
// Anthropic API root.
//
// It is the same escape hatch as EnvOpenRouterBase: an operator
// behind a gateway needs to name it, and a run that must not
// depend on a live provider needs a deterministic endpoint.
const EnvAnthropicBase = "ANTHROPIC_BASE_URL"

// NewAnthropic returns an AnthropicAdapter. apiKey may be
// empty; in that case Complete/Stream return an error at call
// time.
func NewAnthropic(cfg AnthropicConfig) *AnthropicAdapter {
	key := cfg.APIKey
	if key == "" {
		key = os.Getenv("ANTHROPIC_API_KEY")
	}
	client := cfg.Client
	if client == nil {
		client = http.DefaultClient
	}
	base := cfg.Base
	if base == "" {
		base = os.Getenv(EnvAnthropicBase)
	}
	if base == "" {
		base = "https://api.anthropic.com/v1"
	}
	return &AnthropicAdapter{apiKey: key, client: client, base: base}
}

func (a *AnthropicAdapter) Provider() string { return "anthropic" }

// anthropicRequest is the wire shape for the /messages endpoint.
type anthropicRequest struct {
	Model       string  `json:"model"`
	Messages    []anMsg `json:"messages"`
	MaxTokens   int     `json:"max_tokens"`
	Temperature float64 `json:"temperature"`
	System      string  `json:"system,omitempty"`
	Stream      bool    `json:"stream,omitempty"`
}

type anMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicResponse struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
	Model string `json:"model"`
}

func (a *AnthropicAdapter) Complete(ctx context.Context, req Request) (Response, error) {
	maxTokens := req.Options.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4096
	}
	body, err := json.Marshal(anthropicRequest{
		Model: req.Model,
		Messages: []anMsg{
			{Role: "user", Content: req.Prompt},
		},
		MaxTokens:   maxTokens,
		Temperature: req.Options.Temperature,
		System:      req.Options.System,
	})
	if err != nil {
		return Response{}, err
	}
	httpReq, err := a.newRequest(ctx, "POST", "/messages", body)
	if err != nil {
		return Response{}, err
	}
	resp, err := a.client.Do(httpReq)
	if err != nil {
		return Response{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Response{}, readProviderError("anthropic", resp)
	}
	var ar anthropicResponse
	if err := json.NewDecoder(resp.Body).Decode(&ar); err != nil {
		return Response{}, fmt.Errorf("anthropic: decode: %w", err)
	}
	if len(ar.Content) == 0 {
		return Response{}, &ProviderError{Provider: "anthropic", Code: "no_content", Message: "no content blocks in response"}
	}
	return Response{
		Text: ar.Content[0].Text,
		Usage: Usage{
			Model:        ar.Model,
			InputTokens:  ar.Usage.InputTokens,
			OutputTokens: ar.Usage.OutputTokens,
		},
	}, nil
}

type anthropicStream struct {
	body    io.ReadCloser
	scanner *bufio.Scanner
	usage   Usage
	done    bool
}

func (s *anthropicStream) Close() error { return s.body.Close() }
func (s *anthropicStream) Usage() Usage { return s.usage }

// Next parses one SSE event from the Anthropic stream. The
// Anthropic stream emits named events: message_start,
// content_block_start, content_block_delta, content_block_stop,
// message_delta (with usage), message_stop. We only care about
// content_block_delta (the text deltas) and message_delta (the
// final usage).
func (s *anthropicStream) Next() (Chunk, error) {
	if s.done {
		return Chunk{Usage: &s.usage}, io.EOF
	}
	var eventName string
	for s.scanner.Scan() {
		line := s.scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		switch {
		case bytes.HasPrefix(line, []byte("event: ")):
			eventName = string(line[7:])
		case bytes.HasPrefix(line, []byte("data: ")):
			payload := line[6:]
			switch eventName {
			case "content_block_delta":
				var ev struct {
					Delta struct {
						Text string `json:"text"`
					} `json:"delta"`
				}
				if err := json.Unmarshal(payload, &ev); err != nil {
					return Chunk{}, fmt.Errorf("anthropic: stream decode: %w", err)
				}
				return Chunk{Delta: ev.Delta.Text}, nil
			case "message_delta":
				var ev struct {
					Usage struct {
						OutputTokens int `json:"output_tokens"`
					} `json:"usage"`
				}
				if err := json.Unmarshal(payload, &ev); err != nil {
					return Chunk{}, fmt.Errorf("anthropic: stream usage: %w", err)
				}
				s.usage.OutputTokens = ev.Usage.OutputTokens
			case "message_start":
				var ev struct {
					Message struct {
						Usage struct {
							InputTokens int `json:"input_tokens"`
						} `json:"usage"`
						Model string `json:"model"`
					} `json:"message"`
				}
				if err := json.Unmarshal(payload, &ev); err != nil {
					return Chunk{}, fmt.Errorf("anthropic: stream start: %w", err)
				}
				s.usage.InputTokens = ev.Message.Usage.InputTokens
				s.usage.Model = ev.Message.Model
			case "message_stop":
				s.done = true
				u := s.usage
				return Chunk{Usage: &u}, io.EOF
			}
		}
	}
	if err := s.scanner.Err(); err != nil {
		return Chunk{}, err
	}
	s.done = true
	return Chunk{Usage: &s.usage}, io.EOF
}

func (a *AnthropicAdapter) Stream(ctx context.Context, req Request) (Stream, error) {
	maxTokens := req.Options.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4096
	}
	body, _ := json.Marshal(anthropicRequest{
		Model: req.Model, Stream: true,
		Messages:    []anMsg{{Role: "user", Content: req.Prompt}},
		MaxTokens:   maxTokens,
		Temperature: req.Options.Temperature,
		System:      req.Options.System,
	})
	httpReq, err := a.newRequest(ctx, "POST", "/messages", body)
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
		return nil, readProviderError("anthropic", resp)
	}
	return &anthropicStream{
		body:    resp.Body,
		scanner: bufio.NewScanner(resp.Body),
	}, nil
}

func (a *AnthropicAdapter) newRequest(ctx context.Context, method, path string, body []byte) (*http.Request, error) {
	if a.apiKey == "" {
		return nil, &ProviderError{Provider: "anthropic", Code: "no_api_key", Message: "ANTHROPIC_API_KEY is not set"}
	}
	req, err := http.NewRequestWithContext(ctx, method, a.base+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-api-key", a.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}
