package policy

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLoad_Valid(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "policy.yaml", `
rules:
  - id: r1
    match: {tool: read}
    action: allow
  - id: r2
    match: {tool: write}
    action: deny
`)
	rules, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) != 2 {
		t.Fatalf("rules: %d", len(rules))
	}
}

func TestLoad_Empty(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "policy.yaml", `
rules: []
`)
	rules, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) != 0 {
		t.Fatalf("rules: %d", len(rules))
	}
}

func TestLoad_EmptyFile(t *testing.T) {
	// An empty YAML file is a valid policy with no rules; every
	// tool call falls through to the default deny at the gateway.
	dir := t.TempDir()
	path := writeFile(t, dir, "policy.yaml", "")
	rules, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) != 0 {
		t.Fatalf("rules: %d", len(rules))
	}
}

func TestLoad_InvalidAction(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "policy.yaml", `
rules:
  - id: r1
    match: {tool: x}
    action: maybe
`)
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected validation error")
	}
	var rve *RuleValidationError
	if !errors.As(err, &rve) {
		t.Fatalf("expected *RuleValidationError, got %T", err)
	}
}

func TestLoad_DuplicateID(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "policy.yaml", `
rules:
  - id: r1
    match: {tool: a}
    action: allow
  - id: r1
    match: {tool: b}
    action: deny
`)
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected duplicate id error")
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := Load("/nope/does/not/exist.yaml")
	if err == nil {
		t.Fatal("expected error")
	}
	var ppe *PolicyParseError
	if !errors.As(err, &ppe) {
		t.Fatalf("expected *PolicyParseError, got %T", err)
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "policy.yaml", `
rules:
  - id: r1
    match: {tool:
    action: oops
`)
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error")
	}
	var ppe *PolicyParseError
	if !errors.As(err, &ppe) {
		t.Fatalf("expected *PolicyParseError, got %T", err)
	}
}

func TestLoad_Include(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "base.yaml", `
rules:
  - id: base-1
    match: {tool: read}
    action: allow
`)
	path := writeFile(t, dir, "policy.yaml", `
include: ./base.yaml
rules:
  - id: top-1
    match: {tool: write}
    action: deny
`)
	rules, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) != 2 {
		t.Fatalf("rules: %d", len(rules))
	}
	if rules[0].ID != "base-1" || rules[1].ID != "top-1" {
		t.Fatalf("order: %v", []string{rules[0].ID, rules[1].ID})
	}
}

func TestLoad_IncludeList(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "a.yaml", `
rules:
  - id: a-1
    match: {tool: a}
    action: allow
`)
	writeFile(t, dir, "b.yaml", `
rules:
  - id: b-1
    match: {tool: b}
    action: allow
`)
	path := writeFile(t, dir, "policy.yaml", `
include:
  - ./a.yaml
  - ./b.yaml
rules:
  - id: top-1
    match: {tool: top}
    action: allow
`)
	rules, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) != 3 {
		t.Fatalf("rules: %d", len(rules))
	}
}

func TestLoad_IncludeCycle(t *testing.T) {
	dir := t.TempDir()
	a := writeFile(t, dir, "a.yaml", `
include: ./b.yaml
rules:
  - id: a-1
    match: {tool: a}
    action: allow
`)
	b := writeFile(t, dir, "b.yaml", `
include: ./a.yaml
rules:
  - id: b-1
    match: {tool: b}
    action: allow
`)
	_, err := Load(a)
	if err == nil {
		// also try b
		_, err = Load(b)
	}
	if err == nil {
		t.Fatal("expected cycle error")
	}
	var ppe *PolicyParseError
	if !errors.As(err, &ppe) {
		t.Fatalf("expected *PolicyParseError, got %T", err)
	}
}

func TestLoad_IncludeNotFound(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "policy.yaml", `
include: ./nope.yaml
rules: []
`)
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLoad_IncludeWrongType(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "policy.yaml", `
include:
  foo: bar
rules: []
`)
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestNormalizePath(t *testing.T) {
	base, _ := filepath.Abs("a/b")
	got, err := NormalizePath(base, "c.yaml")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Clean(filepath.Join(base, "c.yaml"))
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	// Absolute input: should be cleaned and returned as-is.
	abs := filepath.Clean(filepath.Join(base, "..", "abs", "c.yaml"))
	got, err = NormalizePath(base, abs)
	if err != nil {
		t.Fatal(err)
	}
	if got != abs {
		t.Fatalf("got %q want %q", got, abs)
	}
}
