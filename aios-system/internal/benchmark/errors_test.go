package benchmark

import (
	"errors"
	"strings"
	"testing"
)

func TestBenchmarkTimeout_ErrorAndIs(t *testing.T) {
	e := &BenchmarkTimeout{TaskID: "a", TimeoutS: 30}
	if !strings.Contains(e.Error(), "a") {
		t.Fatal("error should include task id")
	}
	if !errors.Is(e, ErrBenchmarkTimeout) {
		t.Fatal("errors.Is should match")
	}
	if got := e.Unwrap(); got != ErrBenchmarkTimeout {
		t.Fatalf("unwrap = %v", got)
	}
}

func TestBenchmarkFailure_ErrorAndIs(t *testing.T) {
	e := &BenchmarkFailure{TaskID: "a", Cause: errors.New("boom")}
	if !strings.Contains(e.Error(), "a") {
		t.Fatal("error should include task id")
	}
	if !errors.Is(e, ErrBenchmarkFailure) {
		t.Fatal("errors.Is should match")
	}
	if got := e.Unwrap(); got != ErrBenchmarkFailure {
		t.Fatalf("unwrap = %v", got)
	}
}
