package models

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAnthropic_Complete(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/messages" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if r.Header.Get("x-api-key") != "test-key" {
			t.Fatalf("x-api-key = %q", r.Header.Get("x-api-key"))
		}
		if r.Header.Get("anthropic-version") == "" {
			t.Fatal("missing anthropic-version")
		}
		var body anthropicRequest
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body.Model != "claude-sonnet-4-5" {
			t.Fatalf("model = %q", body.Model)
		}
		_, _ = w.Write([]byte(`{
			"model": "claude-sonnet-4-5",
			"content": [{"text": "bonjour"}],
			"usage": {"input_tokens": 8, "output_tokens": 3}
		}`))
	}))
	defer srv.Close()

	a := NewAnthropic(AnthropicConfig{APIKey: "test-key", Base: srv.URL})
	resp, err := a.Complete(context.Background(), Request{Model: "claude-sonnet-4-5", Prompt: "hi"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Text != "bonjour" {
		t.Fatalf("text = %q", resp.Text)
	}
	if resp.Usage.InputTokens != 8 || resp.Usage.OutputTokens != 3 {
		t.Fatalf("usage = %+v", resp.Usage)
	}
}

func TestAnthropic_ProviderError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":"rate limit"}`))
	}))
	defer srv.Close()

	a := NewAnthropic(AnthropicConfig{APIKey: "test-key", Base: srv.URL})
	_, err := a.Complete(context.Background(), Request{Model: "claude-sonnet-4-5", Prompt: "hi"})
	if err == nil {
		t.Fatal("expected error")
	}
	var pe *ProviderError
	if !asErr(err, &pe) {
		t.Fatalf("err type = %T, want *ProviderError", err)
	}
	if pe.Code != "http_429" {
		t.Fatalf("code = %q", pe.Code)
	}
}

func TestAnthropic_NoAPIKey(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server should not be called")
	}))
	defer srv.Close()
	t.Setenv("ANTHROPIC_API_KEY", "")
	a := NewAnthropic(AnthropicConfig{Base: srv.URL})
	_, err := a.Complete(context.Background(), Request{Model: "claude-sonnet-4-5", Prompt: "hi"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestAnthropic_Stream(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)
		w.Write([]byte("event: message_start\ndata: {\"message\":{\"model\":\"claude-sonnet-4-5\",\"usage\":{\"input_tokens\":7}}}\n\n"))
		flusher.Flush()
		w.Write([]byte("event: content_block_delta\ndata: {\"delta\":{\"text\":\"hel\"}}\n\n"))
		flusher.Flush()
		w.Write([]byte("event: content_block_delta\ndata: {\"delta\":{\"text\":\"lo\"}}\n\n"))
		flusher.Flush()
		w.Write([]byte("event: message_delta\ndata: {\"usage\":{\"output_tokens\":2}}\n\n"))
		flusher.Flush()
		w.Write([]byte("event: message_stop\ndata: {}\n\n"))
	}))
	defer srv.Close()

	a := NewAnthropic(AnthropicConfig{APIKey: "k", Base: srv.URL})
	stream, err := a.Stream(context.Background(), Request{Model: "claude-sonnet-4-5", Prompt: "hi"})
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()

	var text string
	for {
		chunk, err := stream.Next()
		text += chunk.Delta
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	if !contains(text, "hel") || !contains(text, "lo") {
		t.Fatalf("stream text = %q", text)
	}
	if u := stream.Usage(); u.InputTokens != 7 || u.OutputTokens != 2 {
		t.Fatalf("usage = %+v", u)
	}
}
