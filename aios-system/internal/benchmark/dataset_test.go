package benchmark

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultDataset_HasAllWedges(t *testing.T) {
	ds := DefaultDataset()
	want := map[string]bool{
		WedgeFileRead: false, WedgeFileWrite: false, WedgeFileSearch: false,
		WedgeShellRun: false, WedgeApproval: false, WedgeAskUser: false,
		WedgeResume: false, WedgeCancel: false,
	}
	for _, task := range ds.Tasks {
		want[task.Wedge] = true
	}
	for w, seen := range want {
		if !seen {
			t.Errorf("wedge %q missing from default dataset", w)
		}
	}
}

func TestDataset_Validate_OK(t *testing.T) {
	ds := DefaultDataset()
	if err := ds.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestDataset_Validate_EmptyID(t *testing.T) {
	ds := WedgeDataset{Tasks: []WedgeTask{{Wedge: "x", Goal: "y"}}}
	if err := ds.Validate(); err == nil {
		t.Fatal("expected error for empty id")
	}
}

func TestDataset_Validate_DuplicateID(t *testing.T) {
	ds := WedgeDataset{Tasks: []WedgeTask{
		{ID: "a", Wedge: "w", Goal: "g"},
		{ID: "a", Wedge: "w", Goal: "g"},
	}}
	if err := ds.Validate(); err == nil {
		t.Fatal("expected error for duplicate id")
	}
}

func TestDataset_Validate_EmptyWedge(t *testing.T) {
	ds := WedgeDataset{Tasks: []WedgeTask{{ID: "a", Goal: "g"}}}
	if err := ds.Validate(); err == nil {
		t.Fatal("expected error for empty wedge")
	}
}

func TestDataset_Validate_EmptyGoal(t *testing.T) {
	ds := WedgeDataset{Tasks: []WedgeTask{{ID: "a", Wedge: "w"}}}
	if err := ds.Validate(); err == nil {
		t.Fatal("expected error for empty goal")
	}
}

func TestDataset_Validate_NegativeTimeout(t *testing.T) {
	ds := WedgeDataset{Tasks: []WedgeTask{{ID: "a", Wedge: "w", Goal: "g", TimeoutSeconds: -1}}}
	if err := ds.Validate(); err == nil {
		t.Fatal("expected error for negative timeout")
	}
}

func TestLoadAndSave_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "wedge.json")
	ds := DefaultDataset()
	if err := ds.Save(path); err != nil {
		t.Fatal(err)
	}
	got, err := LoadDataset(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Tasks) != len(ds.Tasks) {
		t.Fatalf("len = %d, want %d", len(got.Tasks), len(ds.Tasks))
	}
}

func TestLoadDataset_BadJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(path, []byte("not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadDataset(path); err == nil {
		t.Fatal("expected error for bad JSON")
	}
}

func TestLoadDataset_Missing(t *testing.T) {
	if _, err := LoadDataset(filepath.Join(t.TempDir(), "missing.json")); err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestFilterByWedge(t *testing.T) {
	ds := DefaultDataset()
	got := ds.FilterByWedge(WedgeFileRead, WedgeFileWrite)
	if len(got.Tasks) != 2 {
		t.Fatalf("len = %d, want 2", len(got.Tasks))
	}
	for _, task := range got.Tasks {
		if task.Wedge != WedgeFileRead && task.Wedge != WedgeFileWrite {
			t.Fatalf("unexpected wedge %q", task.Wedge)
		}
	}
}

func TestFilterByWedge_Empty_KeepsAll(t *testing.T) {
	ds := DefaultDataset()
	got := ds.FilterByWedge()
	if len(got.Tasks) != len(ds.Tasks) {
		t.Fatalf("len = %d, want %d", len(got.Tasks), len(ds.Tasks))
	}
}

func TestSave_RejectsInvalid(t *testing.T) {
	ds := WedgeDataset{Tasks: []WedgeTask{{Wedge: "x", Goal: "y"}}}
	dir := t.TempDir()
	if err := ds.Save(filepath.Join(dir, "x.json")); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestWedgeNames_Ordered(t *testing.T) {
	ds := WedgeDataset{Tasks: []WedgeTask{
		{ID: "1", Wedge: "b"},
		{ID: "2", Wedge: "a"},
		{ID: "3", Wedge: "a"},
		{ID: "4", Wedge: "c"},
	}}
	names := wedgeNames(ds.Tasks)
	want := []string{"a", "b", "c"}
	if !equalStrings(names, want) {
		t.Fatalf("names = %v, want %v", names, want)
	}
}

func TestDurationSeconds(t *testing.T) {
	if got := durationSeconds(312400); got != "312.4s" {
		t.Fatalf("got %q", got)
	}
}

func TestDSName(t *testing.T) {
	if got := dsName(WedgeDataset{}); got != "empty" {
		t.Fatalf("got %q", got)
	}
	if got := dsName(DefaultDataset()); !strings.Contains(got, "wedge") {
		t.Fatalf("got %q", got)
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
