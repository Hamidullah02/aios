package egress

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFile_Valid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "egress.yaml")
	if err := os.WriteFile(path, []byte(`
allow:
  - api.openai.com
  - "*.example.com"
deny:
  - "*.evil.com"
headers:
  remove:
    - X-Internal-Token
  add:
    X-Aios-Request: "true"
`), 0o644); err != nil {
		t.Fatal(err)
	}
	a, h, err := LoadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !a.Allows("api.openai.com") {
		t.Fatal("allow list should contain api.openai.com")
	}
	if !a.Allows("api.example.com") {
		t.Fatal("allow list should contain glob *.example.com")
	}
	if a.Allows("api.evil.com") {
		t.Fatal("deny list should override allow list")
	}
	if h == nil {
		t.Fatal("header rules should not be nil")
	}
}

func TestLoadFile_MissingIsNotAnError(t *testing.T) {
	dir := t.TempDir()
	a, h, err := LoadFile(filepath.Join(dir, "does-not-exist.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if a == nil || a.Allows("anything.com") {
		t.Fatal("missing file should produce deny-by-default list")
	}
	if h == nil {
		t.Fatal("missing file should produce non-nil empty header rules")
	}
}

func TestLoadFile_EmptyPath(t *testing.T) {
	_, _, err := LoadFile("")
	if err == nil {
		t.Fatal("expected error for empty path")
	}
	var pe *ParseError
	if !errors.As(err, &pe) {
		t.Fatalf("expected *ParseError, got %T", err)
	}
}

func TestLoadFile_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "egress.yaml")
	if err := os.WriteFile(path, []byte("allow: [unclosed"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, _, err := LoadFile(path)
	if err == nil {
		t.Fatal("expected parse error")
	}
	var pe *ParseError
	if !errors.As(err, &pe) {
		t.Fatalf("expected *ParseError, got %T", err)
	}
}

func TestLoadFile_InvalidHeaderName(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "egress.yaml")
	if err := os.WriteFile(path, []byte(`
allow: [a.com]
headers:
  remove:
    - "  X-Bad"
`), 0o644); err != nil {
		t.Fatal(err)
	}
	_, _, err := LoadFile(path)
	if err == nil {
		t.Fatal("expected error for bad header name")
	}
}

func TestLoadFile_EmptyAdd(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "egress.yaml")
	if err := os.WriteFile(path, []byte(`
allow: [a.com]
`), 0o644); err != nil {
		t.Fatal(err)
	}
	_, h, err := LoadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(h.Add) != 0 || len(h.Remove) != 0 {
		t.Fatalf("no headers expected, got %+v", h)
	}
}

func TestParse_Empty(t *testing.T) {
	a, h, err := Parse([]byte(""), "")
	if err != nil {
		t.Fatal(err)
	}
	if a.Allows("x") {
		t.Fatal("empty should deny")
	}
	if h == nil {
		t.Fatal("rules should be non-nil")
	}
}

func TestParse_MultipleAddHeaders(t *testing.T) {
	src := []byte(`
allow: [a]
headers:
  add:
    X-A: "1"
    X-B: "2"
`)
	_, h, err := Parse(src, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(h.Add) != 2 {
		t.Fatalf("expected 2 add rules, got %d", len(h.Add))
	}
}
