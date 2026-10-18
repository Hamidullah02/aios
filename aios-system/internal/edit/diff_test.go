package edit

import (
	"strings"
	"testing"
)

func TestUnifiedDiff_NoChange(t *testing.T) {
	if d := unifiedDiff("a\nb\n", "a\nb\n"); d != "" {
		t.Fatalf("expected empty, got %q", d)
	}
}

func TestUnifiedDiff_SimpleReplace(t *testing.T) {
	d := unifiedDiff("a\nb\nc\n", "a\nB\nc\n")
	if !strings.Contains(d, "-b") || !strings.Contains(d, "+B") {
		t.Fatalf("missing -/+ lines: %q", d)
	}
	if !strings.Contains(d, "@@") {
		t.Fatalf("missing hunk header: %q", d)
	}
}

func TestUnifiedDiff_InsertAtEnd(t *testing.T) {
	d := unifiedDiff("a\n", "a\nb\n")
	if !strings.Contains(d, "+b") {
		t.Fatalf("missing +b: %q", d)
	}
}
