package benchmark

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMetrics_ContentHash_Stable(t *testing.T) {
	m := Metrics{TaskID: "a", DurationMS: 100, Success: true}
	h1 := m.ContentHash()
	m2 := m
	h2 := m2.ContentHash()
	if h1 != h2 {
		t.Fatalf("hash not stable: %q vs %q", h1, h2)
	}
}

func TestMetrics_ContentHash_DiffersOnChange(t *testing.T) {
	m := Metrics{TaskID: "a", DurationMS: 100}
	h1 := m.ContentHash()
	m.DurationMS = 200
	h2 := m.ContentHash()
	if h1 == h2 {
		t.Fatal("hash should differ on duration change")
	}
}

func TestSaveAndLoadReport_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "report.json")
	rows := []Metrics{
		{TaskID: "a", DurationMS: 100, Success: true},
		{TaskID: "b", DurationMS: 200, CostUSD: 0.01, ToolCalls: 3, Success: false},
	}
	if err := SaveReport(path, rows); err != nil {
		t.Fatal(err)
	}
	got, err := LoadReport(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d", len(got))
	}
	if got[1].TaskID != "b" || got[1].ToolCalls != 3 {
		t.Fatalf("got %+v", got[1])
	}
}

func TestSummarize_Empty(t *testing.T) {
	a := Summarize(nil)
	if a.Count != 0 {
		t.Fatalf("Count = %d", a.Count)
	}
}

func TestSummarize_Basic(t *testing.T) {
	rows := []Metrics{
		{Success: true, DurationMS: 100, CostUSD: 0.01, LoopCount: 0, ToolCalls: 1, Approvals: 0},
		{Success: true, DurationMS: 200, CostUSD: 0.02, LoopCount: 2, ToolCalls: 3, Approvals: 1},
		{Success: false, DurationMS: 50, CostUSD: 0.005, LoopCount: 0, ToolCalls: 1, Approvals: 0},
	}
	a := Summarize(rows)
	if a.Count != 3 || a.Pass != 2 || a.Fail != 1 {
		t.Fatalf("counts = %+v", a)
	}
	if a.TotalMS != 350 {
		t.Fatalf("TotalMS = %d", a.TotalMS)
	}
	if a.MaxLoops != 2 {
		t.Fatalf("MaxLoops = %d", a.MaxLoops)
	}
	if a.TotalApprovals != 1 {
		t.Fatalf("TotalApprovals = %d", a.TotalApprovals)
	}
}

func TestByWedge(t *testing.T) {
	tasks := []WedgeTask{
		{ID: "a", Wedge: "w1"},
		{ID: "b", Wedge: "w1"},
		{ID: "c", Wedge: "w2"},
	}
	rows := []Metrics{
		{TaskID: "a", Success: true, DurationMS: 100},
		{TaskID: "b", Success: false, DurationMS: 50},
		{TaskID: "c", Success: true, DurationMS: 200},
	}
	got := ByWedge(tasks, rows)
	if got["w1"].Count != 2 || got["w1"].Pass != 1 {
		t.Fatalf("w1 = %+v", got["w1"])
	}
	if got["w2"].Count != 1 || got["w2"].Pass != 1 {
		t.Fatalf("w2 = %+v", got["w2"])
	}
}

func TestResult_ToMetrics(t *testing.T) {
	r := Result{DurationMS: 100, CostUSD: 0.5, Success: true, LoopCount: 2, ToolCalls: 4, ModelCalls: 1, Approvals: 1}
	task := WedgeTask{ID: "a"}
	m := r.ToMetrics(task)
	if m.TaskID != "a" || m.CostUSD != 0.5 || m.LoopCount != 2 {
		t.Fatalf("m = %+v", m)
	}
}

func TestLoadReport_MissingFile(t *testing.T) {
	if _, err := LoadReport(filepath.Join(t.TempDir(), "missing.json")); err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoadReport_BadJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(path, []byte("not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadReport(path); err == nil {
		t.Fatal("expected parse error")
	}
}

func TestSaveReport_DirError(t *testing.T) {
	// OutputPath points at a directory; WriteFile fails.
	if err := SaveReport(t.TempDir(), nil); err == nil {
		t.Fatal("expected error for directory write")
	}
}
