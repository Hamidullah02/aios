package benchmark

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestCLI_Run_Happy(t *testing.T) {
	dir := t.TempDir()
	dsPath := filepath.Join(dir, "wedge.json")
	outPath := filepath.Join(dir, "out.json")
	ds := DefaultDataset()
	if err := ds.Save(dsPath); err != nil {
		t.Fatal(err)
	}
	cli := &CLI{DatasetPath: dsPath, OutputPath: outPath, Quiet: true}
	code := cli.Run(context.Background(), &fakeRunner{Default: Result{Success: true, DurationMS: 10}})
	if code != 0 {
		t.Fatalf("code = %d", code)
	}
	if _, err := os.Stat(outPath); err != nil {
		t.Fatal(err)
	}
}

func TestCLI_Run_TaskFailure_Returns1(t *testing.T) {
	dir := t.TempDir()
	dsPath := filepath.Join(dir, "wedge.json")
	outPath := filepath.Join(dir, "out.json")
	if err := DefaultDataset().Save(dsPath); err != nil {
		t.Fatal(err)
	}
	cli := &CLI{DatasetPath: dsPath, OutputPath: outPath, Quiet: true}
	code := cli.Run(context.Background(), &fakeRunner{Default: Result{Success: false}})
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
}

func TestCLI_Run_MissingDataset_Returns2(t *testing.T) {
	cli := &CLI{DatasetPath: filepath.Join(t.TempDir(), "missing.json"), OutputPath: filepath.Join(t.TempDir(), "out.json"), Quiet: true}
	code := cli.Run(context.Background(), &fakeRunner{})
	if code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
}

func TestCLI_RunTwoClass(t *testing.T) {
	dir := t.TempDir()
	dsPath := filepath.Join(dir, "wedge.json")
	outPath := filepath.Join(dir, "two.json")
	if err := DefaultDataset().Save(dsPath); err != nil {
		t.Fatal(err)
	}
	cli := &CLI{DatasetPath: dsPath, OutputPath: outPath, Quiet: true}
	code := cli.RunTwoClass(context.Background(),
		&fakeRunner{Default: Result{Success: true, DurationMS: 100, CostUSD: 0.01}},
		&fakeRunner{Default: Result{Success: true, DurationMS: 200, CostUSD: 0.02}},
	)
	if code != 0 {
		t.Fatalf("code = %d", code)
	}
	if _, err := os.Stat(outPath); err != nil {
		t.Fatal(err)
	}
}

func TestCLI_RunTwoClass_MissingDataset(t *testing.T) {
	cli := &CLI{DatasetPath: filepath.Join(t.TempDir(), "missing.json"), Quiet: true}
	if code := cli.RunTwoClass(context.Background(), &fakeRunner{}, &fakeRunner{}); code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
}

func TestCLI_Run_OutputWriteFails(t *testing.T) {
	dir := t.TempDir()
	dsPath := filepath.Join(dir, "wedge.json")
	if err := DefaultDataset().Save(dsPath); err != nil {
		t.Fatal(err)
	}
	// OutputPath points at an existing DIRECTORY so the
	// WriteFile fails.
	outPath := t.TempDir()
	cli := &CLI{DatasetPath: dsPath, OutputPath: outPath, Quiet: true}
	if code := cli.Run(context.Background(), &fakeRunner{Default: Result{Success: true}}); code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
}

func TestCLI_RunTwoClass_OutputWriteFails(t *testing.T) {
	dir := t.TempDir()
	dsPath := filepath.Join(dir, "wedge.json")
	if err := DefaultDataset().Save(dsPath); err != nil {
		t.Fatal(err)
	}
	outPath := t.TempDir()
	cli := &CLI{DatasetPath: dsPath, OutputPath: outPath, Quiet: true}
	if code := cli.RunTwoClass(context.Background(), &fakeRunner{Default: Result{Success: true}}, &fakeRunner{Default: Result{Success: true}}); code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
}

func TestCLI_PrintSummary_Quiet(t *testing.T) {
	dir := t.TempDir()
	dsPath := filepath.Join(dir, "wedge.json")
	outPath := filepath.Join(dir, "out.json")
	if err := DefaultDataset().Save(dsPath); err != nil {
		t.Fatal(err)
	}
	cli := &CLI{DatasetPath: dsPath, OutputPath: outPath, Quiet: false}
	// Quiet=false: printSummary runs but does not panic.
	cli.Run(context.Background(), &fakeRunner{Default: Result{Success: true, DurationMS: 10}})
}

func TestCLI_PrintTwoClass_Direct(t *testing.T) {
	// Drive printTwoClass directly via the public CLI
	// path with non-empty results, so the helper's
	// if-branches all run.
	dir := t.TempDir()
	dsPath := filepath.Join(dir, "wedge.json")
	outPath := filepath.Join(dir, "two.json")
	if err := DefaultDataset().Save(dsPath); err != nil {
		t.Fatal(err)
	}
	cli := &CLI{DatasetPath: dsPath, OutputPath: outPath, Quiet: false}
	cli.RunTwoClass(context.Background(),
		&fakeRunner{Default: Result{Success: true, DurationMS: 1000, CostUSD: 0.01}},
		&fakeRunner{Default: Result{Success: true, DurationMS: 2000, CostUSD: 0.02}},
	)
}
