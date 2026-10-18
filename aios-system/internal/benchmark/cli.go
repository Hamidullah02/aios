package benchmark

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
)

// CLI is the entry point. The CLI is intentionally
// tiny: it loads a dataset, runs the bench, prints a
// short summary, and writes the metrics JSON to a
// file. The two-class and loop runners are exposed
// through RunTwoClass and RunLoop for the operator
// who wants them.
//
// The CLI returns an int suitable for os.Exit: 0 for
// success, 1 for any task failure, 2 for a usage or
// I/O error. The convention matches the doc-set
// wireframe.
type CLI struct {
	// DatasetPath is the JSON file to load. Default
	// is "./benchmark/wedge.json".
	DatasetPath string
	// OutputPath is the JSON report file. Default is
	// "./var/benchmark-report.json".
	OutputPath string
	// Quiet suppresses the human-readable summary.
	Quiet bool
}

// Run executes the bench and returns the exit code.
// The runner is supplied by the caller (typically a
// shim over aios-runtime). Errors that abort the run
// (file I/O, parse failures) are reported via stderr
// and return 2; per-task failures return 1.
func (c *CLI) Run(ctx context.Context, r Runner) int {
	if c.DatasetPath == "" {
		c.DatasetPath = "./benchmark/wedge.json"
	}
	if c.OutputPath == "" {
		c.OutputPath = "./var/benchmark-report.json"
	}
	ds, err := LoadDataset(c.DatasetPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	rows, err := New(r, ds).Run(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	if err := SaveReport(c.OutputPath, rows); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	if !c.Quiet {
		printSummary(ds, rows)
	}
	for _, m := range rows {
		if !m.Success {
			return 1
		}
	}
	return 0
}

// RunTwoClass runs the two-class benchmark with the
// same CLI conventions. The summary is printed to
// stdout; the JSON report is written to OutputPath.
func (c *CLI) RunTwoClass(ctx context.Context, strong, weak Runner) int {
	if c.DatasetPath == "" {
		c.DatasetPath = "./benchmark/wedge.json"
	}
	if c.OutputPath == "" {
		c.OutputPath = "./var/two-class-report.json"
	}
	ds, err := LoadDataset(c.DatasetPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	res, err := NewTwoClass(strong, weak, ds).Run(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	canon, _ := json.MarshalIndent(res, "", "  ")
	if err := os.WriteFile(c.OutputPath, canon, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	if !c.Quiet {
		printTwoClass(res)
	}
	return 0
}

// printSummary emits the per-wedge table from the
// wireframe (WIREFRAMES §6.1). The output is plain
// text, no external deps.
func printSummary(ds WedgeDataset, rows []Metrics) {
	byWedge := ByWedge(ds.Tasks, rows)
	agg := Summarize(rows)
	fmt.Println("================================================================")
	fmt.Println("  BENCHMARK REPORT")
	fmt.Println("================================================================")
	fmt.Printf("  Dataset:   %s (%d tasks)\n", dsName(ds), len(ds.Tasks))
	fmt.Printf("  Duration:  %s\n", durationSeconds(agg.TotalMS))
	fmt.Printf("  Cost:      $%.3f\n", agg.TotalCost)
	fmt.Println("================================================================")
	fmt.Println("  WEDGE             | TASKS | PASS | DURATION | COST   | LOOP")
	fmt.Println("  ------------------|-------|------|----------|--------|------")
	for _, w := range wedgeNames(ds.Tasks) {
		a := byWedge[w]
		fmt.Printf("  %-18s| %5d | %4d | %8s | $%.3f | %4d\n",
			w, a.Count, a.Pass, durationSeconds(a.TotalMS), a.TotalCost, a.TotalLoops)
	}
	fmt.Println("================================================================")
	fmt.Printf("  TOTAL             | %5d | %4d | %8s | $%.3f | %4d\n",
		agg.Count, agg.Pass, durationSeconds(agg.TotalMS), agg.TotalCost, agg.TotalLoops)
	fmt.Println("================================================================")
}

// printTwoClass emits the two-class report.
func printTwoClass(res TwoClassMetrics) {
	sa := Summarize(res.Strong)
	wa := Summarize(res.Weak)
	fmt.Println("================================================================")
	fmt.Println("  TWO-CLASS REPORT")
	fmt.Println("================================================================")
	fmt.Printf("  Metric        | Strong    | Weak      | Delta\n")
	fmt.Printf("  --------------|-----------|-----------|--------\n")
	sp := 100 * float64(sa.Pass) / float64(sa.Count)
	wp := 100 * float64(wa.Pass) / float64(wa.Count)
	fmt.Printf("  Pass rate     | %7.1f%% | %7.1f%% | %+6.1f%%\n", sp, wp, sp-wp)
	if sa.Count > 0 {
		fmt.Printf("  Avg duration  | %6.1fs  | %6.1fs  | %+6.1fs\n",
			float64(sa.TotalMS)/float64(sa.Count)/1000,
			float64(wa.TotalMS)/float64(wa.Count)/1000,
			(float64(sa.TotalMS)-float64(wa.TotalMS))/float64(sa.Count)/1000)
	}
	if sa.Count > 0 {
		fmt.Printf("  Avg cost      |  $%.3f  |  $%.3f  | $%+.3f\n",
			sa.TotalCost/float64(sa.Count),
			wa.TotalCost/float64(sa.Count),
			(sa.TotalCost-wa.TotalCost)/float64(sa.Count))
	}
	fmt.Println("================================================================")
}

// dsName returns a short name for the dataset based on
// the source file. Used only for the report header.
func dsName(ds WedgeDataset) string {
	if len(ds.Tasks) == 0 {
		return "empty"
	}
	return "wedge"
}
