package benchmark

import (
	"context"
	"testing"
)

func TestTwoClass_BothPass(t *testing.T) {
	ds := WedgeDataset{Tasks: []WedgeTask{
		{ID: "a", Wedge: "w", Goal: "g"},
		{ID: "b", Wedge: "w", Goal: "g"},
	}}
	strong := &fakeRunner{Default: Result{Success: true, DurationMS: 100, CostUSD: 0.01}}
	weak := &fakeRunner{Default: Result{Success: true, DurationMS: 200, CostUSD: 0.02}}
	res, err := NewTwoClass(strong, weak, ds).Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Strong) != 2 || len(res.Weak) != 2 {
		t.Fatalf("len = %d/%d", len(res.Strong), len(res.Weak))
	}
	if res.Delta.AvgDurationMS >= 0 {
		t.Fatalf("expected negative duration delta, got %f", res.Delta.AvgDurationMS)
	}
}

func TestTwoClass_StrongWinsPassRate(t *testing.T) {
	ds := WedgeDataset{Tasks: []WedgeTask{
		{ID: "a", Wedge: "w", Goal: "g"},
		{ID: "b", Wedge: "w", Goal: "g"},
	}}
	strong := &fakeRunner{Default: Result{Success: true}}
	weak := &fakeRunner{Responses: map[string]Result{
		"a": {Success: true},
		"b": {Success: false},
	}}
	res, err := NewTwoClass(strong, weak, ds).Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if res.Delta.PassRate != 50.0 {
		t.Fatalf("PassRate = %f, want 50.0", res.Delta.PassRate)
	}
}

func TestTwoClass_LoopRate(t *testing.T) {
	ds := WedgeDataset{Tasks: []WedgeTask{
		{ID: "a", Wedge: "w", Goal: "g"},
		{ID: "b", Wedge: "w", Goal: "g"},
	}}
	strong := &fakeRunner{Default: Result{Success: true, LoopCount: 1}}
	weak := &fakeRunner{Default: Result{Success: true, LoopCount: 0}}
	res, _ := NewTwoClass(strong, weak, ds).Run(context.Background())
	// Strong has 1 loop per task on average, weak has 0.
	// The delta (strong - weak) is +100 percentage points
	// of average loop count.
	if res.Delta.LoopRate != 100.0 {
		t.Fatalf("LoopRate = %f, want 100.0", res.Delta.LoopRate)
	}
}

func TestTwoClass_EmptyDataset(t *testing.T) {
	res, err := NewTwoClass(&fakeRunner{}, &fakeRunner{}, WedgeDataset{}).Run(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	_ = res
}

func TestDelta_Empty(t *testing.T) {
	d := delta(Aggregates{}, Aggregates{})
	if d.PassRate != 0 || d.AvgCostUSD != 0 {
		t.Fatalf("d = %+v", d)
	}
}

func TestTwoClass_StrongRunnerFails(t *testing.T) {
	ds := WedgeDataset{Tasks: []WedgeTask{{ID: "a", Wedge: "w", Goal: "g"}}}
	// Pre-cancelled context surfaces as context.Canceled.
	strong := &fakeRunner{Default: Result{Success: true}}
	weak := &fakeRunner{Default: Result{Success: true}}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	res, err := NewTwoClass(strong, weak, ds).Run(ctx)
	if err == nil {
		t.Fatal("expected error from pre-cancelled context")
	}
	_ = res
}
