package runtime

import (
	"errors"
	"testing"
)

func TestNoProgress_Trips(t *testing.T) {
	d := NewNoProgressDetector(3)
	for i := 0; i < 3; i++ {
		if err := d.Observe("same"); err != nil {
			t.Fatalf("observe %d: %v", i, err)
		}
	}
	err := d.Observe("same")
	var npr *NoProgressDetected
	if !errors.As(err, &npr) {
		t.Fatalf("err type = %T, want *NoProgressDetected", err)
	}
	if npr.Count < 3 {
		t.Fatalf("Count = %d", npr.Count)
	}
}

func TestNoProgress_DifferentResets(t *testing.T) {
	d := NewNoProgressDetector(2)
	_ = d.Observe("a")
	_ = d.Observe("a")
	// Now b -> resets
	if err := d.Observe("b"); err != nil {
		t.Fatalf("different content: %v", err)
	}
	// a -> first occurrence of new run
	if err := d.Observe("a"); err != nil {
		t.Fatalf("after reset, first: %v", err)
	}
}

func TestNoProgress_Disabled(t *testing.T) {
	d := NewNoProgressDetector(0)
	for i := 0; i < 100; i++ {
		if err := d.Observe("same"); err != nil {
			t.Fatalf("disabled should never trip: %v", err)
		}
	}
}

func TestNoProgress_Reset(t *testing.T) {
	d := NewNoProgressDetector(2)
	_ = d.Observe("a")
	_ = d.Observe("a")
	d.Reset()
	// After reset, "a" is the first occurrence.
	if err := d.Observe("a"); err != nil {
		t.Fatalf("after reset: %v", err)
	}
}
