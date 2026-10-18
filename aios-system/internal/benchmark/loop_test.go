package benchmark

import (
	"context"
	"errors"
	"testing"
)

func TestLoopChecker_Basic(t *testing.T) {
	ds := WedgeDataset{Tasks: []WedgeTask{
		{ID: "a", Wedge: "w", Goal: "g"},
		{ID: "b", Wedge: "w", Goal: "g"},
		{ID: "c", Wedge: "w", Goal: "g"},
		{ID: "d", Wedge: "w", Goal: "g"},
	}}
	r := &fakeRunner{Responses: map[string]Result{
		"a": {Success: true, LoopCount: 0},
		"b": {Success: true, LoopCount: 1},
		"c": {Success: true, LoopCount: 3},
		"d": {Success: true, LoopCount: 0},
	}}
	res, err := NewLoopChecker(r, ds).Check(context.Background(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if res.Count != 4 {
		t.Fatalf("Count = %d", res.Count)
	}
	if res.MaxLoopCount != 3 {
		t.Fatalf("MaxLoopCount = %d", res.MaxLoopCount)
	}
	if res.AvgLoopCount != 1.0 {
		t.Fatalf("AvgLoopCount = %f", res.AvgLoopCount)
	}
	if res.LoopRate != 50.0 {
		t.Fatalf("LoopRate = %f, want 50.0", res.LoopRate)
	}
	if len(res.LoopingTasks) != 2 {
		t.Fatalf("LoopingTasks = %v", res.LoopingTasks)
	}
}

func TestLoopChecker_Threshold(t *testing.T) {
	ds := WedgeDataset{Tasks: []WedgeTask{
		{ID: "a", Wedge: "w", Goal: "g"},
		{ID: "b", Wedge: "w", Goal: "g"},
	}}
	r := &fakeRunner{Default: Result{Success: true, LoopCount: 5}}
	res, _ := NewLoopChecker(r, ds).Check(context.Background(), 5)
	// threshold 5 means > 5 (strict). LoopCount 5 is NOT > 5.
	if res.LoopRate != 0 {
		t.Fatalf("LoopRate = %f, want 0", res.LoopRate)
	}
}

func TestLoopChecker_EmptyDataset(t *testing.T) {
	res, err := NewLoopChecker(&fakeRunner{}, WedgeDataset{}).Check(context.Background(), 0)
	if err == nil {
		t.Fatal("expected error")
	}
	if res.Count != 0 {
		t.Fatalf("Count = %d", res.Count)
	}
}

func TestLoopChecker_NegativeThreshold_ClampedToZero(t *testing.T) {
	ds := WedgeDataset{Tasks: []WedgeTask{{ID: "a", Wedge: "w", Goal: "g"}}}
	r := &fakeRunner{Default: Result{Success: true, LoopCount: 0}}
	res, _ := NewLoopChecker(r, ds).Check(context.Background(), -10)
	if res.LoopRate != 0 {
		t.Fatalf("LoopRate = %f", res.LoopRate)
	}
}

func TestLoopChecker_ContextError(t *testing.T) {
	ds := WedgeDataset{Tasks: []WedgeTask{
		{ID: "a", Wedge: "w", Goal: "g"},
		{ID: "b", Wedge: "w", Goal: "g"},
	}}
	r := &fakeRunner{Default: Result{Success: true}}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := NewLoopChecker(r, ds).Check(ctx, 0)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
}
