package egress

import (
	"errors"
	"testing"
)

func TestHostNotAllowedError_Error(t *testing.T) {
	e := &HostNotAllowedError{Host: "evil.com"}
	if e.Error() == "" {
		t.Fatal("non-empty")
	}
}

func TestUpstreamError_Error(t *testing.T) {
	inner := errors.New("connection refused")
	e := &UpstreamError{Host: "api", Method: "GET", Path: "/x", Err: inner}
	if e.Error() == "" {
		t.Fatal("non-empty")
	}
	if !errors.Is(e, inner) {
		t.Fatal("Unwrap should expose inner")
	}
	// nil inner should still produce a message
	e2 := &UpstreamError{Host: "api", Method: "GET", Path: "/x"}
	if e2.Error() == "" {
		t.Fatal("non-empty with nil inner")
	}
}

func TestParseError_Error(t *testing.T) {
	e := &ParseError{Path: "/p.yaml", Reason: "bad"}
	if e.Error() == "" {
		t.Fatal("non-empty")
	}
}

func TestEmptyURLError(t *testing.T) {
	if errEmptyURL.Error() == "" {
		t.Fatal("non-empty")
	}
}
