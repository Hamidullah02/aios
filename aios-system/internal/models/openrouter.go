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

// EnvOpenRouterAPIKey is the environment variable that holds the
// OpenRouter key.
//
// OpenRouter fronts many providers behind one OpenAI-compatible
// API, so it is a third provider rather than a variant of the
// first: it has its own key, its own base URL, and its own model
// ids (which are namespaced, e.g. "nvidia/nemotron-3-ultra").
const EnvOpenRouterAPIKey = "OPENROUTER_API_KEY"

// EnvOpenRouterAPIKeyAlias is an alternative name for the key.
//
// It exists because operator `.env` files in the wild spell it
// this way, and a tool that accepts only one spelling sends them
// looking for a mechanism that does not exist. The canonical
// name is OPENROUTER_API_KEY; this one is accepted too.
const EnvOpenRouterAPIKeyAlias = "openrouterapi"

// DefaultOpenRouterBase is the OpenRouter API root.
//
// The path ends in /v1 because the adapter appends the endpoint
// itself, the same way the other two adapters do.
const DefaultOpenRouterBase = "https://openrouter.ai/api/v1"

// EnvOpenRouterBase is the environment variable that overrides the
// OpenRouter API root.
//
// It exists for two reasons. An operator who routes OpenRouter
// through their own gateway needs a way to say where that gateway
// is. And a test -- or a user-flow run -- that must not depend on
// a live provider, a shared rate limit, or the weather needs a
// deterministic endpoint to talk to.
//
// The value is the API root, with the /v1 suffix the adapter
// expects, e.g. "http://127.0.0.1:8080/api/v1".
const EnvOpenRouterBase = "OPENROUTER_BASE_URL"

// OpenRouterAdapter calls the OpenRouter chat-completions API.
//
// OpenRouter speaks the OpenAI wire format, so the request and
// response shapes are the OpenAI ones. What differs is the
// endpoint, the key, and the model ids.
//
// The adapter reads OPENROUTER_API_KEY from the environment at
// construction time. Override with Config.APIKey for tests.
// Override the HTTP client with Config.Client for httptest.
// Override the base URL with Config.Base for a self-hosted
// gateway.
type OpenRouterAdapter struct {
	apiKey string
	client *http.Client
	base   string
}

// OpenRouterConfig is the constructor input.
type OpenRouterConfig struct {
	// APIKey is empty -> use $OPENROUTER_API_KEY.
	APIKey string
	// Client is nil -> http.DefaultClient.
	Client *http.Client
	// Base is empty -> https://openrouter.ai/api/v1.
	Base string
}

// NewOpenRouter returns an OpenRouterAdapter. apiKey may be
// empty; in that case Complete/Stream return an error at call
// time.
func NewOpenRouter(cfg OpenRouterConfig) *OpenRouterAdapter {
	key := cfg.APIKey
	if key == "" {
		key = os.Getenv(EnvOpenRouterAPIKey)
	}
	if key == "" {
		key = os.Getenv(EnvOpenRouterAPIKeyAlias)
	}
	client := cfg.Client
	if client == nil {
		client = http.DefaultClient
	}
	base := cfg.Base
	if base == "" {
		base = os.Getenv(EnvOpenRouterBase)
	}
	if base == "" {
		base = DefaultOpenRouterBase
	}
	return &OpenRouterAdapter{apiKey: key, client: client, base: base}
}

// Provider implements Adapter.
func (a *OpenRouterAdapter) Provider() string { return "openrouter" }

// Complete runs one chat completion.
//
// The request body is the OpenAI shape with two additions
// OpenRouter understands: HTTP-Referer and X-Title, which it
// uses to attribute the call to an app. They are optional on
// their side, but sending them is what OpenRouter asks for, and
// they cost nothing.
func (a *OpenRouterAdapter) Complete(ctx context.Context, req Request) (Response, error) {
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
		return Response{}, readProviderError("openrouter", resp)
	}
	var cr chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&cr); err != nil {
		return Response{}, fmt.Errorf("openrouter: decode: %w", err)
	}
	if len(cr.Choices) == 0 {
		return Response{}, &ProviderError{Provider: "openrouter", Code: "no_choices", Message: "no choices in response"}
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

// openrouterStream implements Stream over OpenRouter's
// server-sent events.
//
// The wire format is OpenAI's: each event is a `data:` line
// carrying one chat-completion chunk, and the stream ends with
// `data: [DONE]`.
type openrouterStream struct {
	body    io.ReadCloser
	scanner *bufio.Scanner
	usage   Usage
	done    bool
}

func (s *openrouterStream) Close() error { return s.body.Close() }
func (s *openrouterStream) Usage() Usage { return s.usage }

// Next returns the next chunk.
func (s *openrouterStream) Next() (Chunk, error) {
	if s.done {
		return Chunk{Usage: &s.usage}, io.EOF
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
			return Chunk{}, fmt.Errorf("openrouter: stream decode: %w", err)
		}
		// The final chunk often has an empty choices array and
		// only the usage block.
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

// Stream opens a streaming completion.
func (a *OpenRouterAdapter) Stream(ctx context.Context, req Request) (Stream, error) {
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
		return nil, readProviderError("openrouter", resp)
	}
	return &openrouterStream{
		body:    resp.Body,
		scanner: bufio.NewScanner(resp.Body),
	}, nil
}

// newRequest builds one request to the provider.
//
// The missing-key error names OPENROUTER_API_KEY rather than a
// generic "no key", because the operator has to know which
// variable to set. The other two adapters read their own
// variables; this one reads this one.
func (a *OpenRouterAdapter) newRequest(ctx context.Context, method, path string, body []byte) (*http.Request, error) {
	if a.apiKey == "" {
		return nil, &ProviderError{
			Provider: "openrouter",
			Code:     "no_api_key",
			Message:  "OPENROUTER_API_KEY is not set",
		}
	}
	req, err := http.NewRequestWithContext(ctx, method, a.base+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+a.apiKey)
	req.Header.Set("Content-Type", "application/json")
	// OpenRouter attributes calls to an app using these two
	// headers. They are optional, but sending them is what
	// OpenRouter asks for and they cost nothing.
	req.Header.Set("HTTP-Referer", "https://github.com/Hamidullah02/aios")
	req.Header.Set("X-Title", "aios")
	return req, nil
}
