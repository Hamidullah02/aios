package models

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOpenAI_Complete(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Fatalf("auth = %q", r.Header.Get("Authorization"))
		}
		// Verify the request body has temperature=0 by default.
		var body chatRequest
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body.Temperature != 0 {
			t.Fatalf("temperature = %v, want 0", body.Temperature)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"model": "gpt-4o",
			"choices": [{"message": {"role": "assistant", "content": "hello"}}],
			"usage": {"prompt_tokens": 10, "completion_tokens": 4}
		}`))
	}))
	defer srv.Close()

	a := NewOpenAI(OpenAIConfig{APIKey: "test-key", Base: srv.URL})
	resp, err := a.Complete(context.Background(), Request{Model: "gpt-4o", Prompt: "hi"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Text != "hello" {
		t.Fatalf("text = %q", resp.Text)
	}
	if resp.Usage.InputTokens != 10 || resp.Usage.OutputTokens != 4 {
		t.Fatalf("usage = %+v", resp.Usage)
	}
}

func TestOpenAI_ProviderError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"bad key"}`))
	}))
	defer srv.Close()

	a := NewOpenAI(OpenAIConfig{APIKey: "test-key", Base: srv.URL})
	_, err := a.Complete(context.Background(), Request{Model: "gpt-4o", Prompt: "hi"})
	if err == nil {
		t.Fatal("expected error")
	}
	var pe *ProviderError
	if !asErr(err, &pe) {
		t.Fatalf("err type = %T, want *ProviderError", err)
	}
	if pe.Code != "http_401" {
		t.Fatalf("code = %q, want http_401", pe.Code)
	}
}

func TestOpenAI_NoAPIKey(t *testing.T) {
	// Use a server that should never be called.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server should not be called")
	}))
	defer srv.Close()
	// Unset the env var so the adapter sees no key.
	t.Setenv("OPENAI_API_KEY", "")
	a := NewOpenAI(OpenAIConfig{Base: srv.URL})
	_, err := a.Complete(context.Background(), Request{Model: "gpt-4o", Prompt: "hi"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestOpenAI_Stream(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)
		// 3 chunks: "hel", "lo", "" (final, with usage)
		w.Write([]byte("data: {\"choices\":[{\"message\":{\"content\":\"hel\"}}]}\n\n"))
		flusher.Flush()
		w.Write([]byte("data: {\"choices\":[{\"message\":{\"content\":\"lo\"}}]}\n\n"))
		flusher.Flush()
		w.Write([]byte("data: {\"choices\":[{\"message\":{\"content\":\"\"}}],\"usage\":{\"prompt_tokens\":5,\"completion_tokens\":2}}\n\n"))
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
	if u := stream.Usage(); u.InputTokens != 5 || u.OutputTokens != 2 {
		t.Fatalf("usage = %+v", u)
	}
}

// asErr is a tiny helper to dodge importing errors in every test.
func asErr(err error, target any) bool {
	type unwrapper interface{ Unwrap() error }
	for {
		if err == nil {
			return false
		}
		if pe, ok := err.(*ProviderError); ok {
			if pp, ok := target.(**ProviderError); ok {
				*pp = pe
				return true
			}
		}
		u, ok := err.(unwrapper)
		if !ok {
			return false
		}
		err = u.Unwrap()
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
