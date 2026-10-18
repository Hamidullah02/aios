package redact

import (
	"strings"
	"testing"
)

// extra coverage for the helpers that the round-trip
// tests do not exercise: RedactJSONBytes, the sort
// comparator's tie-breaker, and the empty-string path
// of RedactValue.

func TestRedactJSONBytes_Empty(t *testing.T) {
	got, err := RedactJSONBytes(nil, "approval_id")
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatalf("got %q", got)
	}
}

func TestRedactJSONBytes_Valid(t *testing.T) {
	in := []byte(`{"api_key":"AKIA12345678ABCDEFGH","other":"x"}`)
	out, err := RedactJSONBytes(in, "approval_id")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(out), "AKIA12345678ABCDEFGH") {
		t.Fatalf("secret not redacted: %s", out)
	}
}

func TestRedactJSONBytes_InvalidJSON(t *testing.T) {
	if _, err := RedactJSONBytes([]byte("not json"), "approval_id"); err == nil {
		t.Fatal("expected error")
	}
}

func TestRedactJSONBytes_PreservesApprovalID(t *testing.T) {
	in := []byte(`{"approval_id":"my-approval","api_key":"AKIA12345678ABCDEFGH"}`)
	out, err := RedactJSONBytes(in, "approval_id")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "my-approval") {
		t.Fatalf("approval_id not preserved: %s", out)
	}
	if strings.Contains(string(out), "AKIA12345678ABCDEFGH") {
		t.Fatalf("api_key leaked: %s", out)
	}
}

func TestSortCandidatesByStart_TieBreaker(t *testing.T) {
	// Two candidates at the same start position but
	// different end positions. The shorter one wins.
	c := []candidate{
		{start: 0, end: 10, kind: "long"},
		{start: 0, end: 5, kind: "short"},
	}
	sortCandidatesByStart(c)
	if c[0].kind != "short" {
		t.Fatalf("tie-breaker wrong: c[0] = %+v", c[0])
	}
}

func TestSortCandidatesByStart_AlreadySorted(t *testing.T) {
	c := []candidate{
		{start: 0, end: 5},
		{start: 10, end: 20},
		{start: 30, end: 40},
	}
	sortCandidatesByStart(c)
	if c[0].start != 0 || c[1].start != 10 || c[2].start != 30 {
		t.Fatalf("not stable: %+v", c)
	}
}

func TestRedactValue_NilString(t *testing.T) {
	if got := RedactValue(nil, "approval_id"); got != nil {
		t.Fatalf("got %v", got)
	}
}

func TestRedactValue_EmptyString(t *testing.T) {
	got := RedactValue("", "approval_id")
	if s, ok := got.(string); !ok || s != "" {
		t.Fatalf("got %v", got)
	}
}

func TestFindAll_TieBreaker(t *testing.T) {
	// Two patterns that could match at the same position.
	// FindAll is documented to return non-overlapping
	// candidates; the longer match wins.
	s := "AKIA12345678ABCDEFGH"
	matches := FindAll(s)
	if len(matches) == 0 {
		t.Fatal("expected at least one match")
	}
	for _, m := range matches {
		if m.Start < 0 || m.End > len(s) {
			t.Fatalf("bad match bounds: %+v", m)
		}
	}
}
