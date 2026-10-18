package models

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestRouter_EndToEnd_OpenAI walks the full router -> adapter ->
// price-table pipeline using a recorded OpenAI response.
func TestRouter_EndToEnd_OpenAI(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"model": "gpt-4o",
			"choices": [{"message": {"role": "assistant", "content": "hi"}}],
			"usage": {"prompt_tokens": 2000, "completion_tokens": 500}
		}`))
	}))
	defer srv.Close()

	reg, _ := NewRegistry([]Capability{
		{Model: "gpt-4o", Provider: "openai", ContextWindow: 128000},
	})
	prices := NewPriceTable([]Price{
		{Model: "gpt-4o", InputPer1K: 0.005, OutputPer1K: 0.015},
	}, nil)
	r := NewRouter(map[string]Adapter{
		"openai": NewOpenAI(OpenAIConfig{APIKey: "k", Base: srv.URL}),
	}, reg, prices)

	resp, err := r.Complete(context.Background(), Request{Model: "gpt-4o", Prompt: "hi"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Text != "hi" {
		t.Fatalf("text = %q", resp.Text)
	}
	// 2000 * 0.005/1000 + 500 * 0.015/1000 = 0.01 + 0.0075 = 0.0175
	if resp.Usage.CostUSD != 0.0175 {
		t.Fatalf("cost = %v, want 0.0175", resp.Usage.CostUSD)
	}
	if resp.Usage.Model != "gpt-4o" {
		t.Fatalf("model = %q, want gpt-4o", resp.Usage.Model)
	}
}

// TestRouter_EndToEnd_Anthropic walks the same pipeline for the
// Anthropic adapter. Confirms the messages endpoint is called
// and the cost is attached.
func TestRouter_EndToEnd_Anthropic(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the path.
		if r.URL.Path != "/messages" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		// Verify the Anthropic headers.
		if r.Header.Get("anthropic-version") == "" {
			t.Fatal("missing anthropic-version")
		}
		_, _ = w.Write([]byte(`{
			"model": "claude-sonnet-4-5",
			"content": [{"text": "bonjour"}],
			"usage": {"input_tokens": 1000, "output_tokens": 200}
		}`))
	}))
	defer srv.Close()

	reg, _ := NewRegistry([]Capability{
		{Model: "claude-sonnet-4-5", Provider: "anthropic", ContextWindow: 200000},
	})
	prices := NewPriceTable([]Price{
		{Model: "claude-sonnet-4-5", InputPer1K: 0.003, OutputPer1K: 0.015},
	}, nil)
	r := NewRouter(map[string]Adapter{
		"anthropic": NewAnthropic(AnthropicConfig{APIKey: "k", Base: srv.URL}),
	}, reg, prices)

	resp, err := r.Complete(context.Background(), Request{Model: "claude-sonnet-4-5", Prompt: "hi"})
	if err != nil {
		t.Fatal(err)
	}
	// 1000 * 0.003/1000 + 200 * 0.015/1000 = 0.003 + 0.003 = 0.006
	if resp.Usage.CostUSD != 0.006 {
		t.Fatalf("cost = %v, want 0.006", resp.Usage.CostUSD)
	}
}

// TestOpenAI_Stream_EndToEnd walks the streaming pipeline.
// Asserts the delta accumulates to the expected text and the
// final usage is attached.
func TestOpenAI_Stream_EndToEnd(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)
		for _, tok := range []string{"hel", "lo"} {
			payload, _ := json.Marshal(map[string]any{
				"choices": []map[string]any{{"message": map[string]any{"content": tok}}},
			})
			w.Write([]byte("data: " + string(payload) + "\n\n"))
			flusher.Flush()
		}
		// Final chunk with usage.
		final, _ := json.Marshal(map[string]any{
			"choices": []any{},
			"usage":   map[string]any{"prompt_tokens": 3, "completion_tokens": 2},
		})
		w.Write([]byte("data: " + string(final) + "\n\n"))
		flusher.Flush()
		w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer srv.Close()

	a := NewOpenAI(OpenAIConfig{APIKey: "k", Base: srv.URL})
	stream, err := a.Stream(context.Background(), Request{Model: "gpt-4o", Prompt: "hi"})
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()

	var text string
	for {
		chunk, err := stream.Next()
		text += chunk.Delta
		if err != nil {
			break
		}
	}
	if text != "hello" {
		t.Fatalf("text = %q, want hello", text)
	}
	if u := stream.Usage(); u.InputTokens != 3 || u.OutputTokens != 2 {
		t.Fatalf("usage = %+v", u)
	}
}
