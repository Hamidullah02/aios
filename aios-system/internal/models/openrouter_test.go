package models

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestOpenRouter_Complete pins the happy path: the request
// carries the key, the model id, and the system prompt, and the
// response comes back as text plus usage.
func TestOpenRouter_Complete(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Errorf("path = %q, want /chat/completions", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Errorf("auth = %q, want Bearer test-key", got)
		}
		// The attribution headers are what OpenRouter asks for.
		if r.Header.Get("X-Title") == "" {
			t.Error("missing X-Title")
		}
		var body chatRequest
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body.Model != "nvidia/nemotron-3-ultra-550b-a55b" {
			t.Errorf("model = %q", body.Model)
		}
		if body.Temperature != 0 {
			t.Errorf("temperature = %v, want 0", body.Temperature)
		}
		if len(body.Messages) != 2 || body.Messages[0].Role != "system" {
			t.Errorf("messages = %+v, want a system message first", body.Messages)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"model": "nvidia/nemotron-3-ultra-550b-a55b",
			"choices": [{"message": {"role": "assistant", "content": "hello"}}],
			"usage": {"prompt_tokens": 12, "completion_tokens": 5}
		}`))
	}))
	defer srv.Close()

	a := NewOpenRouter(OpenRouterConfig{APIKey: "test-key", Base: srv.URL})
	resp, err := a.Complete(context.Background(), Request{
		Model:  "nvidia/nemotron-3-ultra-550b-a55b",
		Prompt: "hi",
		Options: Options{
			System: "you are aios-planner",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Text != "hello" {
		t.Errorf("text = %q, want hello", resp.Text)
	}
	if resp.Usage.InputTokens != 12 || resp.Usage.OutputTokens != 5 {
		t.Errorf("usage = %+v", resp.Usage)
	}
}

// TestOpenRouter_ProviderError pins that a rejected call comes
// back as a typed ProviderError naming the provider.
func TestOpenRouter_ProviderError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"no credits"}`))
	}))
	defer srv.Close()

	a := NewOpenRouter(OpenRouterConfig{APIKey: "bad", Base: srv.URL})
	_, err := a.Complete(context.Background(), Request{Model: "m", Prompt: "hi"})
	if err == nil {
		t.Fatal("expected error")
	}
	var pe *ProviderError
	if !asErr(err, &pe) {
		t.Fatalf("err type = %T, want *ProviderError", err)
	}
	if pe.Provider != "openrouter" {
		t.Errorf("provider = %q, want openrouter", pe.Provider)
	}
	if pe.Code != "http_401" {
		t.Errorf("code = %q, want http_401", pe.Code)
	}
}

// TestOpenRouter_NoAPIKey pins the missing-key error.
//
// The message names the variable the operator has to set. A
// generic "no key" would send them looking through a list of
// providers they do not use.
func TestOpenRouter_NoAPIKey(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("the server should not be called without a key")
	}))
	defer srv.Close()

	t.Setenv(EnvOpenRouterAPIKey, "")
	a := NewOpenRouter(OpenRouterConfig{Base: srv.URL})
	_, err := a.Complete(context.Background(), Request{Model: "m", Prompt: "hi"})
	if err == nil {
		t.Fatal("expected error")
	}
	var pe *ProviderError
	if !asErr(err, &pe) {
		t.Fatalf("err type = %T, want *ProviderError", err)
	}
	if pe.Code != "no_api_key" {
		t.Errorf("code = %q, want no_api_key", pe.Code)
	}
	if !contains(pe.Message, EnvOpenRouterAPIKey) {
		t.Errorf("message = %q, want it to name %s", pe.Message, EnvOpenRouterAPIKey)
	}
}

// TestOpenRouter_Stream pins that the stream carries the deltas
// and the final usage.
func TestOpenRouter_Stream(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)
		for _, tok := range []string{"hel", "lo"} {
			payload, _ := json.Marshal(map[string]any{
				"choices": []map[string]any{{"message": map[string]any{"content": tok}}},
			})
			_, _ = w.Write([]byte("data: " + string(payload) + "\n\n"))
			flusher.Flush()
		}
		final, _ := json.Marshal(map[string]any{
			"choices": []any{},
			"usage":   map[string]any{"prompt_tokens": 7, "completion_tokens": 3},
		})
		_, _ = w.Write([]byte("data: " + string(final) + "\n\n"))
		flusher.Flush()
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer srv.Close()

	a := NewOpenRouter(OpenRouterConfig{APIKey: "k", Base: srv.URL})
	stream, err := a.Stream(context.Background(), Request{Model: "m", Prompt: "hi"})
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
	if text != "hello" {
		t.Errorf("text = %q, want hello", text)
	}
	if u := stream.Usage(); u.InputTokens != 7 || u.OutputTokens != 3 {
		t.Errorf("usage = %+v", u)
	}
}

// TestOpenRouter_ProviderName pins the provider id. The router
// keys its adapters by it, so a wrong name here means the model
// cannot be routed.
func TestOpenRouter_ProviderName(t *testing.T) {
	a := NewOpenRouter(OpenRouterConfig{APIKey: "k"})
	if a.Provider() != "openrouter" {
		t.Errorf("provider = %q, want openrouter", a.Provider())
	}
}

// TestOpenRouter_AliasEnvKey pins that the adapter accepts the
// lower-case spelling operator files use.
//
// The operator's .env spells it "openrouterapi". A tool that
// accepts only one spelling sends them looking for a mechanism
// that does not exist. The alias is read only when the
// canonical name is unset.
func TestOpenRouter_AliasEnvKey(t *testing.T) {
	t.Setenv(EnvOpenRouterAPIKey, "")
	t.Setenv(EnvOpenRouterAPIKeyAlias, "sk-or-alias")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer sk-or-alias" {
			t.Errorf("auth = %q, want the alias key", got)
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"ok"}}]}`))
	}))
	defer srv.Close()

	a := NewOpenRouter(OpenRouterConfig{Base: srv.URL})
	if _, err := a.Complete(context.Background(), Request{Model: "m", Prompt: "hi"}); err != nil {
		t.Fatal(err)
	}
}

// TestOpenRouter_CanonicalEnvKeyWins pins that the canonical
// name is preferred over the alias when both are set.
//
// Two keys set at once is a mistake, and the canonical name is
// the one the documentation tells people to use.
func TestOpenRouter_CanonicalEnvKeyWins(t *testing.T) {
	t.Setenv(EnvOpenRouterAPIKey, "canonical")
	t.Setenv(EnvOpenRouterAPIKeyAlias, "alias")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer canonical" {
			t.Errorf("auth = %q, want the canonical key", got)
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"ok"}}]}`))
	}))
	defer srv.Close()

	a := NewOpenRouter(OpenRouterConfig{Base: srv.URL})
	if _, err := a.Complete(context.Background(), Request{Model: "m", Prompt: "hi"}); err != nil {
		t.Fatal(err)
	}
}
