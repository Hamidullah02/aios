package runtime

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestResumeHandler_SaveLoad(t *testing.T) {
	dir := t.TempDir()
	h := NewResumeHandler(dir)

	cp := &Checkpoint{
		TaskID:    "t1",
		TaskIndex: 2,
		Plan:      json.RawMessage(`{"id":"p1","goal":"g","tasks":[]}`),
		Context:   json.RawMessage(`{"vars":{"a":1}}`),
		Budget:    Budget{MaxCostUSD: 1.0},
	}
	if err := h.Save(cp); err != nil {
		t.Fatal(err)
	}
	got, err := h.Load("t1")
	if err != nil {
		t.Fatal(err)
	}
	if got.TaskIndex != 2 {
		t.Fatalf("TaskIndex = %d", got.TaskIndex)
	}
	if got.Budget.MaxCostUSD != 1.0 {
		t.Fatalf("MaxCostUSD = %v", got.Budget.MaxCostUSD)
	}
	if got.SavedAt == "" {
		t.Fatal("SavedAt should be set by Save")
	}
}

func TestResumeHandler_LoadMissing(t *testing.T) {
	h := NewResumeHandler(t.TempDir())
	_, err := h.Load("nonexistent")
	if !errors.Is(err, ErrCheckpointNotFound) {
		t.Fatalf("err = %v, want ErrCheckpointNotFound", err)
	}
}

func TestResumeHandler_Delete(t *testing.T) {
	dir := t.TempDir()
	h := NewResumeHandler(dir)
	cp := &Checkpoint{TaskID: "t1", Plan: json.RawMessage(`{}`)}
	if err := h.Save(cp); err != nil {
		t.Fatal(err)
	}
	if err := h.Delete("t1"); err != nil {
		t.Fatal(err)
	}
	// Deleting again is a no-op.
	if err := h.Delete("t1"); err != nil {
		t.Fatalf("second delete: %v", err)
	}
}

func TestResumeHandler_DefaultDir(t *testing.T) {
	// When the constructor gets an empty string, the default
	// is "var/checkpoints". The Save creates the directory.
	h := NewResumeHandler("")
	defer h.Delete("t1")
	defer os.RemoveAll("var/checkpoints")

	cp := &Checkpoint{TaskID: "t1", Plan: json.RawMessage(`{}`)}
	if err := h.Save(cp); err != nil {
		t.Fatal(err)
	}
	// File should exist at the default path.
	path := filepath.Join("var", "checkpoints", "t1.json")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected file at %q: %v", path, err)
	}
}
