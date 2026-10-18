package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInit_FreshWorkspace(t *testing.T) {
	dir := t.TempDir()
	created, existed, err := Init(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(created) != 4 {
		t.Errorf("created: %d", len(created))
	}
	if len(existed) != 0 {
		t.Errorf("existed: %d", len(existed))
	}
	for _, name := range []string{"policy.yaml", "egress.yaml", "prices.yaml", "config.yaml"} {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("missing %s: %v", name, err)
		}
	}
}

func TestInit_ReRunIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	_, _, err := Init(dir)
	if err != nil {
		t.Fatal(err)
	}
	created, existed, err := Init(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(created) != 0 {
		t.Errorf("created: %d (should be 0 on re-run)", len(created))
	}
	if len(existed) != 4 {
		t.Errorf("existed: %d", len(existed))
	}
}

func TestInit_EmptyWorkspace(t *testing.T) {
	_, _, err := Init("")
	if err != nil {
		t.Fatal(err)
	}
	// Cleanup: the default workspace landed somewhere on
	// disk; we accept the side-effect for this test.
	_ = os.RemoveAll("./var")
}

func TestPrintInitReport(t *testing.T) {
	var buf bytes.Buffer
	PrintInitReport(&buf, []string{"/a/b/c"}, []string{"/a/b/d"})
	s := buf.String()
	if !strings.Contains(s, "Created:") {
		t.Errorf("missing Created: %s", s)
	}
	if !strings.Contains(s, "Already present:") {
		t.Errorf("missing Already present: %s", s)
	}
	if !strings.Contains(s, "/a/b/c") {
		t.Errorf("missing created path: %s", s)
	}
}

func TestPrintInitReport_BothEmpty(t *testing.T) {
	var buf bytes.Buffer
	PrintInitReport(&buf, nil, nil)
	if !strings.Contains(buf.String(), "Nothing to do") {
		t.Errorf("got %q", buf.String())
	}
}

func TestJoinLines(t *testing.T) {
	if got := joinLines("a", "b", "c"); got != "a\nb\nc" {
		t.Errorf("got %q", got)
	}
}
