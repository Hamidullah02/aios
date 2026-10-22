package webui

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
)

// flushRecorder is an httptest.ResponseRecorder that also
// satisfies http.Flusher.
//
// httptest.ResponseRecorder already has Flush, but it does
// not guard concurrent access: the SSE handler writes from
// its own goroutine while the test reads the buffer. These
// tests therefore use a mutex-protected variant so -race
// stays meaningful.
type flushRecorder struct {
	mu   sync.Mutex
	buf  strings.Builder
	hdr  map[string][]string
	code int
}

func newFlushRecorder() *flushRecorder {
	return &flushRecorder{hdr: map[string][]string{}}
}

func (f *flushRecorder) Header() http.Header {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.hdr == nil {
		f.hdr = map[string][]string{}
	}
	return http.Header(f.hdr)
}

func (f *flushRecorder) Write(p []byte) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.buf.Write(p)
}

func (f *flushRecorder) WriteHeader(code int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.code == 0 {
		f.code = code
	}
}

func (f *flushRecorder) Flush() {}

// String returns the buffered body.
func (f *flushRecorder) String() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.buf.String()
}

// contextWithCancel returns a cancellable context for the
// SSE request.
func contextWithCancel() (ctx context.Context, cancel context.CancelFunc) {
	return context.WithCancel(context.Background())
}

// waitFor polls the recorder until it contains want, or the
// test times out.
//
// Polling rather than a channel handshake because the SSE
// handler writes straight to the ResponseWriter: there is
// no callback to hang a channel off.
func waitFor(t *testing.T, w *flushRecorder, want string) {
	t.Helper()
	if !waitForQuiet(w, want, 2*time.Second) {
		t.Fatalf("timed out waiting for %q; stream so far: %q", want, w.String())
	}
}

// waitForQuiet is waitFor without a *testing.T, for use from
// a goroutine that is not the test goroutine. t.Fatalf from
// a non-test goroutine is a vet error and aborts the whole
// binary, so the goroutine callers get a bool instead.
func waitForQuiet(w *flushRecorder, want string, within time.Duration) bool {
	deadline := time.Now().Add(within)
	for time.Now().Before(deadline) {
		if strings.Contains(w.String(), want) {
			return true
		}
		time.Sleep(5 * time.Millisecond)
	}
	return false
}
