package watch

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/sadi/aios-system/internal/models"
)

// fakeStream is a Stream backed by a sequence of chunks. Used
// when the router itself is the wrong layer to mock.
type fakeStream struct {
	chunks []models.Chunk
	err    error
	closed bool
	mu     sync.Mutex
}

func (f *fakeStream) Next() (models.Chunk, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.chunks) == 0 {
		return models.Chunk{}, io.EOF
	}
	c := f.chunks[0]
	f.chunks = f.chunks[1:]
	if len(f.chunks) == 0 && f.err == nil {
		f.err = io.EOF
	}
	return c, f.err
}
func (f *fakeStream) Usage() models.Usage {
	return models.Usage{InputTokens: 1, OutputTokens: 2, Model: "test"}
}
func (f *fakeStream) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.closed = true
	return nil
}

// fakeAdapter routes everything to fakeStream.
type fakeStreamAdapter struct{ s *fakeStream }

func (a *fakeStreamAdapter) Provider() string { return "fake" }
func (a *fakeStreamAdapter) Complete(ctx context.Context, req models.Request) (models.Response, error) {
	return models.Response{}, nil
}
func (a *fakeStreamAdapter) Stream(ctx context.Context, req models.Request) (models.Stream, error) {
	return a.s, nil
}

func newStreamChannel(t *testing.T, s *fakeStream) *StreamChannel {
	t.Helper()
	r, err := newRouter(s)
	if err != nil {
		t.Fatal(err)
	}
	return NewStreamChannel(r)
}

func newRouter(s *fakeStream) (*models.Router, error) {
	reg, err := models.NewRegistry([]models.Capability{
		{Model: "x", Provider: "fake", ContextWindow: 1000},
	})
	if err != nil {
		return nil, err
	}
	prices := models.NewPriceTable([]models.Price{
		{Model: "x", InputPer1K: 0, OutputPer1K: 0},
	}, nil)
	r := models.NewRouter(map[string]models.Adapter{"fake": &fakeStreamAdapter{s: s}}, reg, prices)
	return r, nil
}

// TestStreamChannel_YieldsChunks covers REQ-WTC-STR-002.
func TestStreamChannel_YieldsChunks(t *testing.T) {
	s := &fakeStream{chunks: []models.Chunk{
		{Delta: "hel"},
		{Delta: "lo"},
		{Usage: &models.Usage{Model: "x"}},
	}}
	ch := newStreamChannel(t, s)
	var got string
	ch.Subscribe(func(c models.Chunk) { got += c.Delta })
	usage, err := ch.Run(context.Background(), "hi", "x")
	if err != nil {
		t.Fatal(err)
	}
	if got != "hello" {
		t.Fatalf("got = %q, want hello", got)
	}
	if usage.Model != "x" {
		t.Fatalf("usage = %+v", usage)
	}
}

// TestStreamChannel_MultipleSubscribers covers REQ-WTC-STR-004.
func TestStreamChannel_MultipleSubscribers(t *testing.T) {
	s := &fakeStream{chunks: []models.Chunk{
		{Delta: "a"},
		{Delta: "b"},
		{Usage: &models.Usage{Model: "x"}},
	}}
	ch := newStreamChannel(t, s)
	var a, b, c string
	ch.Subscribe(func(chunk models.Chunk) { a += chunk.Delta })
	ch.Subscribe(func(chunk models.Chunk) { b += chunk.Delta })
	ch.Subscribe(func(chunk models.Chunk) { c += chunk.Delta })
	if _, err := ch.Run(context.Background(), "hi", "x"); err != nil {
		t.Fatal(err)
	}
	if a != "ab" || b != "ab" || c != "ab" {
		t.Fatalf("a=%q b=%q c=%q want all ab", a, b, c)
	}
}

// TestStreamChannel_PropagatesError covers REQ-WTC-STR-005.
func TestStreamChannel_PropagatesError(t *testing.T) {
	want := errors.New("boom")
	s := &fakeStream{chunks: []models.Chunk{{Delta: "x"}}, err: want}
	ch := newStreamChannel(t, s)
	var seen int32
	ch.Subscribe(func(c models.Chunk) { atomic.AddInt32(&seen, 1) })
	_, err := ch.Run(context.Background(), "hi", "x")
	if !errors.Is(err, want) {
		t.Fatalf("err = %v, want %v", err, want)
	}
	if atomic.LoadInt32(&seen) != 1 {
		t.Fatalf("subscriber called %d times, want 1", seen)
	}
}

// TestStreamChannel_NilRouter covers the no-router safety
// path.
func TestStreamChannel_NilRouter(t *testing.T) {
	ch := &StreamChannel{}
	_, err := ch.Run(context.Background(), "hi", "x")
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestStreamChannel_RealAdapter is an end-to-end smoke test
// against a recorded OpenAI-compatible stream.
func TestStreamChannel_RealAdapter(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)
		for _, tok := range []string{"foo", "bar"} {
			payload := `{"choices":[{"message":{"content":"` + tok + `"}}],"usage":{}}`
			_, _ = w.Write([]byte("data: " + payload + "\n\n"))
			flusher.Flush()
		}
		_, _ = w.Write([]byte(`data: {"choices":[],"usage":{"prompt_tokens":3,"completion_tokens":6,"model":"x"}}` + "\n\n"))
		flusher.Flush()
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer srv.Close()

	reg, _ := models.NewRegistry([]models.Capability{{Model: "x", Provider: "openai", ContextWindow: 1000}})
	a := models.NewOpenAI(models.OpenAIConfig{APIKey: "k", Base: srv.URL, Client: srv.Client()})
	r := models.NewRouter(map[string]models.Adapter{"openai": a}, reg,
		models.NewPriceTable([]models.Price{{Model: "x", InputPer1K: 0, OutputPer1K: 0}}, nil))
	ch := NewStreamChannel(r)
	var got strings.Builder
	ch.Subscribe(func(c models.Chunk) { got.WriteString(c.Delta) })
	u, err := ch.Run(context.Background(), "hi", "x")
	if err != nil {
		t.Fatal(err)
	}
	if got.String() != "foobar" {
		t.Fatalf("got %q", got.String())
	}
	if u.InputTokens != 3 || u.OutputTokens != 6 {
		t.Fatalf("usage = %+v", u)
	}
}
