package watch

import (
	"context"
	"io"
	"sync"

	"github.com/Hamidullah02/aios/aios-system/internal/models"
)

// Subscriber is a callback that runs on every chunk. Errors are
// not propagated back to the stream; the subscriber must handle
// them itself (the typical use is to log them).
type Subscriber func(models.Chunk)

// StreamChannel wraps the model router and exposes the chunk
// stream to the operator UI and to Go callers
// (REQ-WTC-STR-001 to 005).
//
// Usage:
//
//	ch := NewStreamChannel(router)
//	ch.Subscribe(func(c models.Chunk) { fmt.Print(c.Delta) })
//	usage, err := ch.Run(ctx, "hello", "gpt-4o")
//
// The channel closes when the model is done or on error. The
// final Usage is reported via the chunk with Usage != nil (the
// same contract as models.Stream).
type StreamChannel struct {
	router *models.Router
	mu     sync.Mutex
	subs   []Subscriber
}

// NewStreamChannel returns a channel wired to router.
func NewStreamChannel(router *models.Router) *StreamChannel {
	return &StreamChannel{router: router}
}

// Subscribe registers a callback that runs on every chunk.
// Subscribers run synchronously inside the Run loop, so a slow
// subscriber slows the whole channel (REQ-WTC-NFR-003). Callers
// that need to do expensive work should buffer through a
// channel of their own.
//
// Subscribe is safe to call before, after, or during Run. Calls
// made during Run take effect from the next chunk.
func (s *StreamChannel) Subscribe(cb Subscriber) {
	s.mu.Lock()
	s.subs = append(s.subs, cb)
	s.mu.Unlock()
}

// Run opens a streaming call to the model and pumps the chunks
// through every subscriber. The function returns the final
// Usage and the first non-EOF error, if any. On a clean
// stream end the returned error is nil (io.EOF is swallowed;
// the caller's "model done" signal is Usage != nil).
func (s *StreamChannel) Run(ctx context.Context, prompt, model string) (models.Usage, error) {
	if s.router == nil {
		return models.Usage{}, &SubAgentError{Reason: "router is nil"}
	}
	stream, err := s.router.Stream(ctx, models.Request{
		Model:  model,
		Prompt: prompt,
	})
	if err != nil {
		return models.Usage{}, err
	}
	defer stream.Close()

	var last models.Usage
	for {
		chunk, err := stream.Next()
		s.fanout(chunk)
		if chunk.Usage != nil {
			last = *chunk.Usage
		}
		if err != nil {
			// io.EOF is the normal end of stream. Anything else
			// is a real failure and we surface it.
			if err == io.EOF {
				return last, nil
			}
			return last, err
		}
	}
}

// fanout copies chunk to every subscriber under a single read
// lock. The chunk struct is small; copying is cheaper than
// sharing.
func (s *StreamChannel) fanout(chunk models.Chunk) {
	s.mu.Lock()
	subs := append([]Subscriber(nil), s.subs...)
	s.mu.Unlock()
	for _, sub := range subs {
		sub(chunk)
	}
}
