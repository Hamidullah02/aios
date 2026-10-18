package benchmark

import (
	"context"
	"time"
)

// BenchmarkRunner runs the dataset (REQ-BCH-RUN-001).
//
// The runner is a thin loop: for each task, call
// runner.Run, capture the Result, convert to Metrics.
// The runner also enforces a per-task timeout when the
// task's TimeoutSeconds is > 0.
//
// The runner does NOT make the runner.Run call
// cancellable: a slow orchestrator blocks the runner
// for the duration of the task. Production code wires
// runner.Run to the orchestrator's own cancel path
// (REQ-RT-ORC-009 in aios-runtime).
type BenchmarkRunner struct {
	runner  Runner
	dataset WedgeDataset
}

// New returns a BenchmarkRunner. The runner and dataset
// are required; an empty dataset returns an error at
// Run time, not at construction.
func New(r Runner, d WedgeDataset) *BenchmarkRunner {
	return &BenchmarkRunner{runner: r, dataset: d}
}

// Run executes the dataset and returns the metrics
// (REQ-BCH-RUN-002 / 003 / 004). Per-task failures do
// not stop the run; the metrics row carries Success=false
// and the failure is reported in the optional error
// channel.
//
// The function blocks until every task has finished.
// The caller can pass a context to cancel the run; the
// context is checked between tasks, not during a
// single runner.Run call.
func (b *BenchmarkRunner) Run(ctx context.Context) ([]Metrics, error) {
	if len(b.dataset.Tasks) == 0 {
		return nil, ErrEmptyDataset
	}
	rows := make([]Metrics, 0, len(b.dataset.Tasks))
	for _, task := range b.dataset.Tasks {
		if err := ctx.Err(); err != nil {
			return rows, err
		}
		row, err := b.runOne(task)
		if err != nil {
			// Failures don't abort the loop; we still
			// emit a metrics row for the task.
			row = task.failureRow(err)
		}
		rows = append(rows, row)
	}
	return rows, nil
}

// runOne runs a single task. The task's TimeoutSeconds
// is enforced via a goroutine + select. A timeout
// produces a *BenchmarkTimeout.
func (b *BenchmarkRunner) runOne(task WedgeTask) (Metrics, error) {
	if task.TimeoutSeconds <= 0 {
		return b.runner.Run(task).ToMetrics(task), nil
	}
	type out struct {
		r Result
	}
	ch := make(chan out, 1)
	go func() {
		ch <- out{r: b.runner.Run(task)}
	}()
	timer := time.NewTimer(time.Duration(task.TimeoutSeconds) * time.Second)
	defer timer.Stop()
	select {
	case o := <-ch:
		return o.r.ToMetrics(task), nil
	case <-timer.C:
		return task.failureRow(&BenchmarkTimeout{TaskID: task.ID, TimeoutS: task.TimeoutSeconds}),
			&BenchmarkTimeout{TaskID: task.ID, TimeoutS: task.TimeoutSeconds}
	}
}

// failureRow builds a metrics row for a task that
// failed before producing a Result.
func (t WedgeTask) failureRow(err error) Metrics {
	return Metrics{
		TaskID:  t.ID,
		Success: false,
	}
}
