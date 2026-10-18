// Package benchmark is the benchmark harness for aios.
//
// The package loads a wedge dataset, runs the tasks
// against a Runner (a small interface that abstracts
// the orchestrator), and produces Metrics for every
// task. The TwoClassRunner and LoopChecker are thin
// wrappers over the BenchmarkRunner.
//
// The package is dependency-free: it imports the Go
// standard library only. Production wiring plugs in
// a Runner that adapts the aios-runtime orchestrator;
// tests plug in a deterministic fake.
package benchmark

import (
	"errors"
	"fmt"
)

// Sentinel errors per aios-benchmark SRS §9.
var (
	// ErrBenchmarkTimeout is raised when a task exceeds
	// the per-task timeout (REQ-BCH-ERR-001).
	ErrBenchmarkTimeout = errors.New("benchmark: task timeout")

	// ErrBenchmarkFailure is raised when a task fails
	// (REQ-BCH-ERR-002).
	ErrBenchmarkFailure = errors.New("benchmark: task failure")
)

// BenchmarkTimeout is the typed error for per-task
// timeouts. The TaskID identifies the timed-out task.
type BenchmarkTimeout struct {
	TaskID   string
	TimeoutS int
}

func (e *BenchmarkTimeout) Error() string {
	return fmt.Sprintf("benchmark: task %q timed out after %ds", e.TaskID, e.TimeoutS)
}

func (e *BenchmarkTimeout) Is(target error) bool { return target == ErrBenchmarkTimeout }
func (e *BenchmarkTimeout) Unwrap() error        { return ErrBenchmarkTimeout }

// BenchmarkFailure is the typed error for task failures.
type BenchmarkFailure struct {
	TaskID string
	Cause  error
}

func (e *BenchmarkFailure) Error() string {
	return fmt.Sprintf("benchmark: task %q failed: %v", e.TaskID, e.Cause)
}

func (e *BenchmarkFailure) Is(target error) bool { return target == ErrBenchmarkFailure }
func (e *BenchmarkFailure) Unwrap() error        { return ErrBenchmarkFailure }
