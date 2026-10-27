package models

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// The base-URL overrides let an operator point the product at
// their own gateway, and let a user-flow run talk to a local
// stub instead of a live provider with a daily rate limit.
//
// The tests below pin the three adapters' behaviour. They are
// separate tests rather than one table because each adapter has
// its own constructor and its own default.

// TestOpenRouter_BaseURLFromEnvironment pins that the
// environment variable is honoured.
func TestOpenRouter_BaseURLFromEnvironment(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"ok"}}]}`))
	}))
	defer srv.Close()

	t.Setenv(EnvOpenRouterBase, srv.URL+"/api/v1")

	a := NewOpenRouter(OpenRouterConfig{APIKey: "k"})
	if a.base != srv.URL+"/api/v1" {
		t.Fatalf("base = %q, want the override", a.base)
	}
	if _, err := a.Complete(context.Background(), Request{Model: "m", Prompt: "p"}); err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if got != "/api/v1/chat/completions" {
		t.Errorf("path = %q, want the override's path", got)
	}
}

// TestOpenRouter_ExplicitBaseBeatsEnvironment pins the
// precedence: the constructor argument wins over the environment.
//
// A test that needs a specific httptest server must not be
// redirected by whatever the developer's environment says.
func TestOpenRouter_ExplicitBaseBeatsEnvironment(t *testing.T) {
	t.Setenv(EnvOpenRouterBase, "http://from-the-environment.invalid")

	a := NewOpenRouter(OpenRouterConfig{APIKey: "k", Base: "http://explicit.invalid"})
	if a.base != "http://explicit.invalid" {
		t.Fatalf("base = %q, want the explicit value", a.base)
	}
}

// TestOpenRouter_DefaultBaseWhenNothingIsSet pins the fallback:
// with no override, the adapter talks to the real provider.
func TestOpenRouter_DefaultBaseWhenNothingIsSet(t *testing.T) {
	t.Setenv(EnvOpenRouterBase, "")

	a := NewOpenRouter(OpenRouterConfig{APIKey: "k"})
	if a.base != DefaultOpenRouterBase {
		t.Fatalf("base = %q, want %q", a.base, DefaultOpenRouterBase)
	}
}

// TestOpenAI_BaseURLFromEnvironment pins the same override for the
// OpenAI adapter.
func TestOpenAI_BaseURLFromEnvironment(t *testing.T) {
	t.Setenv(EnvOpenAIBase, "http://gateway.invalid/v1")

	a := NewOpenAI(OpenAIConfig{APIKey: "k"})
	if a.base != "http://gateway.invalid/v1" {
		t.Fatalf("base = %q, want the override", a.base)
	}
}

// TestOpenAI_DefaultBaseWhenNothingIsSet pins the fallback.
func TestOpenAI_DefaultBaseWhenNothingIsSet(t *testing.T) {
	t.Setenv(EnvOpenAIBase, "")

	a := NewOpenAI(OpenAIConfig{APIKey: "k"})
	if a.base != "https://api.openai.com/v1" {
		t.Fatalf("base = %q, want the default", a.base)
	}
}

// TestAnthropic_BaseURLFromEnvironment pins the same override for
// the Anthropic adapter.
func TestAnthropic_BaseURLFromEnvironment(t *testing.T) {
	t.Setenv(EnvAnthropicBase, "http://gateway.invalid/v1")

	a := NewAnthropic(AnthropicConfig{APIKey: "k"})
	if a.base != "http://gateway.invalid/v1" {
		t.Fatalf("base = %q, want the override", a.base)
	}
}

// TestAnthropic_DefaultBaseWhenNothingIsSet pins the fallback.
func TestAnthropic_DefaultBaseWhenNothingIsSet(t *testing.T) {
	t.Setenv(EnvAnthropicBase, "")

	a := NewAnthropic(AnthropicConfig{APIKey: "k"})
	if a.base != "https://api.anthropic.com/v1" {
		t.Fatalf("base = %q, want the default", a.base)
	}
}
