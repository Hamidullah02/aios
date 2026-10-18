package models

import (
	"context"
	"errors"
	"testing"
)

type fakeAdapter struct {
	name string
	resp Response
	err  error
}

func (f *fakeAdapter) Provider() string { return f.name }
func (f *fakeAdapter) Complete(ctx context.Context, req Request) (Response, error) {
	return f.resp, f.err
}
func (f *fakeAdapter) Stream(ctx context.Context, req Request) (Stream, error) {
	return nil, errors.New("not implemented in fake")
}

func newTestRouter(t *testing.T) *Router {
	t.Helper()
	reg, _ := NewRegistry([]Capability{
		{Model: "gpt-4o", Provider: "openai", ContextWindow: 128000},
		{Model: "claude-sonnet-4-5", Provider: "anthropic", ContextWindow: 200000},
	})
	prices := NewPriceTable([]Price{
		{Model: "gpt-4o", InputPer1K: 0.005, OutputPer1K: 0.015},
		{Model: "claude-sonnet-4-5", InputPer1K: 0.003, OutputPer1K: 0.015},
	}, nil)
	return NewRouter(map[string]Adapter{
		"openai":    &fakeAdapter{name: "openai", resp: Response{Text: "hi from openai"}},
		"anthropic": &fakeAdapter{name: "anthropic", resp: Response{Text: "hi from anthropic"}},
	}, reg, prices)
}

func TestRouter_Complete_AttachesCost(t *testing.T) {
	r := newTestRouter(t)
	r.adapters["openai"] = &fakeAdapter{
		name: "openai",
		resp: Response{Text: "hi", Usage: Usage{InputTokens: 1000, OutputTokens: 500, Model: "gpt-4o"}},
	}
	resp, err := r.Complete(context.Background(), Request{Model: "gpt-4o", Prompt: "hi"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Usage.CostUSD != 0.005+0.5*0.015 {
		t.Fatalf("cost = %v, want 0.0125", resp.Usage.CostUSD)
	}
}

func TestRouter_UnknownModel(t *testing.T) {
	r := newTestRouter(t)
	_, err := r.Complete(context.Background(), Request{Model: "nope", Prompt: "hi"})
	if !errors.Is(err, ErrUnknownModel) {
		t.Fatalf("err = %v, want ErrUnknownModel", err)
	}
}

func TestRouter_PicksAdapterByProvider(t *testing.T) {
	r := newTestRouter(t)
	resp, err := r.Complete(context.Background(), Request{Model: "claude-sonnet-4-5", Prompt: "hi"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Text != "hi from anthropic" {
		t.Fatalf("text = %q, want anthropic", resp.Text)
	}
}

func TestRouter_NoAdapterForProvider(t *testing.T) {
	r := newTestRouter(t)
	delete(r.adapters, "anthropic")
	_, err := r.Complete(context.Background(), Request{Model: "claude-sonnet-4-5", Prompt: "hi"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRouter_AdapterCaching(t *testing.T) {
	r := newTestRouter(t)
	// Two calls for the same model should hit the same adapter
	// instance; we can verify by capturing the pointer.
	a1, _ := r.AdapterFor("gpt-4o")
	a2, _ := r.AdapterFor("gpt-4o")
	if a1 != a2 {
		t.Fatal("adapter cache miss: same model gave different adapters")
	}
}
