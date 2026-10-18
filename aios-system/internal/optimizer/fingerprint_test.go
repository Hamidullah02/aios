package optimizer

import (
	"strings"
	"testing"
)

func TestNewFingerprint_Pure(t *testing.T) {
	a := NewFingerprint("hello", []string{"read", "write"}, "coder", 1000)
	b := NewFingerprint("hello", []string{"read", "write"}, "coder", 1000)
	if a != b {
		t.Fatalf("not pure: %s vs %s", a, b)
	}
}

func TestNewFingerprint_ToolOrderInsensitive(t *testing.T) {
	a := NewFingerprint("p", []string{"a", "b"}, "r", 100)
	b := NewFingerprint("p", []string{"b", "a"}, "r", 100)
	if a != b {
		t.Fatalf("order matters: %s vs %s", a, b)
	}
}

func TestNewFingerprint_DiffPrompt(t *testing.T) {
	a := NewFingerprint("p1", []string{"a"}, "r", 100)
	b := NewFingerprint("p2", []string{"a"}, "r", 100)
	if a == b {
		t.Fatal("prompts should differ")
	}
}

func TestNewFingerprint_DiffTools(t *testing.T) {
	a := NewFingerprint("p", []string{"a"}, "r", 100)
	b := NewFingerprint("p", []string{"b"}, "r", 100)
	if a == b {
		t.Fatal("tools should differ")
	}
}

func TestNewFingerprint_DiffRole(t *testing.T) {
	a := NewFingerprint("p", []string{"a"}, "r1", 100)
	b := NewFingerprint("p", []string{"a"}, "r2", 100)
	if a == b {
		t.Fatal("roles should differ")
	}
}

func TestNewFingerprint_DiffBucket(t *testing.T) {
	a := NewFingerprint("p", []string{"a"}, "r", 100)
	b := NewFingerprint("p", []string{"a"}, "r", 2000)
	if a == b {
		t.Fatal("buckets should differ")
	}
}

func TestNewFingerprint_ShapeInsensitive(t *testing.T) {
	a := NewFingerprint(ShapePrompt("hello \"world\""), nil, "r", 100)
	b := NewFingerprint(ShapePrompt("hello \"there\""), nil, "r", 100)
	if a != b {
		t.Fatal("shape should ignore string content")
	}
}

func TestFingerprint_Length(t *testing.T) {
	fp := NewFingerprint("p", nil, "r", 100)
	if len(fp) != 64 {
		t.Fatalf("len = %d, want 64", len(fp))
	}
	for _, c := range fp {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Fatalf("non-hex char %q", c)
		}
	}
}

func TestContextBucket(t *testing.T) {
	cases := []struct {
		in   int
		want int
	}{
		{0, 256},
		{1, 256},
		{256, 256},
		{257, 256},
		{500, 256},
		{512, 512},
		{1000, 512},
		{1024, 1024},
		{1500, 1024},
		{200000, 131072},
	}
	for _, c := range cases {
		if got := ContextBucket(c.in); got != c.want {
			t.Errorf("ContextBucket(%d) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestShapePrompt(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{`hello "world"`, "hello str"},
		{"count is 42", "count is num"},
		{"path is /etc/hosts end", "path is path end"},
		{"```\ncode\n```", "code"},
		{"plain", "plain"},
		{"a'quote'b", "astrb"},
		{"x 1.5 y", "x num y"},
		{"./rel/path", "path"},
		{"\"abc\"def\"ghi\"", "strdefstr"},
	}
	for _, c := range cases {
		got := ShapePrompt(c.in)
		if !strings.Contains(got, c.want) {
			t.Errorf("ShapePrompt(%q) = %q, expected to contain %q", c.in, got, c.want)
		}
	}
}

func TestItoa(t *testing.T) {
	cases := []struct {
		in   int
		want string
	}{
		{0, "0"},
		{1, "1"},
		{123, "123"},
		{-5, "-5"},
		{100000, "100000"},
	}
	for _, c := range cases {
		if got := itoa(c.in); got != c.want {
			t.Errorf("itoa(%d) = %q, want %q", c.in, got, c.want)
		}
	}
}
