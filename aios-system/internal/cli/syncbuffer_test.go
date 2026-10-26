package cli

import (
	"bytes"
	"sync"
)

// syncBuffer is a writer a test can read while another goroutine
// is still writing.
//
// bytes.Buffer is not safe for concurrent use, and the approval
// handler writes from the run's goroutine while the test reads
// from its own. The handler now serialises its own writes, but a
// test that reads the buffer directly still races with a write
// that is in flight: the lock inside the handler does not
// protect the test's read.
//
// This is the same shape as the handler's own lock: one mutex,
// every access through it.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

// Write appends to the buffer.
func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

// String returns the contents so far.
func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}
