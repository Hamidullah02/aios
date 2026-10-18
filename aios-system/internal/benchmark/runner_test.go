package benchmark

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestBenchmarkRunner_Happy(t *testing.T) {
	ds := WedgeDataset{Tasks: []WedgeTask{
		{ID: "a", Wedge: "w", Goal: "g", TimeoutSeconds: 5},
		{ID: "b", Wedge: "w", Goal: "g", TimeoutSeconds: 5},
	}}
	r := &fakeRunner{Default: Result{Success: true, DurationMS: 10, CostUSD: 0.001}}
	br := New(r, ds)
	rows, err := br.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("rows = %d", len(rows))
	}
	if !rows[0].Success || !rows[1].Success {
		t.Fatal("expected success")
	}
	if r.Calls != 2 {
		t.Fatalf("Calls = %d", r.Calls)
	}
}

func TestBenchmarkRunner_EmptyDataset(t *testing.T) {
	br := New(&fakeRunner{}, WedgeDataset{})
	_, err := br.Run(context.Background())
	if !errors.Is(err, ErrEmptyDataset) {
		t.Fatalf("err = %v, want ErrEmptyDataset", err)
	}
}

func TestBenchmarkRunner_Timeout(t *testing.T) {
	ds := WedgeDataset{Tasks: []WedgeTask{
		{ID: "slow", Wedge: "w", Goal: "g", TimeoutSeconds: 1},
	}}
	r := &fakeRunner{Default: Result{Success: true}, Sleep: 3 * time.Second}
	br := New(r, ds)
	rows, err := br.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if rows[0].Success {
		t.Fatal("expected failure on timeout")
	}
}

func TestBenchmarkRunner_ContextCancel(t *testing.T) {
	ds := WedgeDataset{Tasks: []WedgeTask{
		{ID: "a", Wedge: "w", Goal: "g"},
		{ID: "b", Wedge: "w", Goal: "g"},
		{ID: "c", Wedge: "w", Goal: "g"},
	}}
	r := &fakeRunner{Default: Result{Success: true}}
	br := New(r, ds)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // pre-cancel
	_, err := br.Run(ctx)
	if err == nil {
		t.Fatal("expected context error")
	}
}

func TestBenchmarkRunner_NoTimeout(t *testing.T) {
	// Tasks with TimeoutSeconds=0 skip the timer branch.
	ds := WedgeDataset{Tasks: []WedgeTask{{ID: "a", Wedge: "w", Goal: "g"}}}
	r := &fakeRunner{Default: Result{Success: true}}
	br := New(r, ds)
	rows, err := br.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !rows[0].Success {
		t.Fatal("expected success")
	}
}

func TestBenchmarkRunner_FailureRow(t *testing.T) {
	task := WedgeTask{ID: "a"}
	row := task.failureRow(errors.New("boom"))
	if row.TaskID != "a" || row.Success {
		t.Fatalf("row = %+v", row)
	}
}
