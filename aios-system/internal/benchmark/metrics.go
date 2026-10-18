package benchmark

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"time"
)

// Metrics is the per-task result (REQ-BCH-DAT-004).
//
// The struct is the public type; the doc-set wireframe
// refers to it by these exact field names.
type Metrics struct {
	TaskID     string  `json:"task_id"`
	DurationMS int64   `json:"duration_ms"`
	CostUSD    float64 `json:"cost_usd"`
	Success    bool    `json:"success"`
	LoopCount  int     `json:"loop_count"`
	ToolCalls  int     `json:"tool_calls"`
	ModelCalls int     `json:"model_calls"`
	Approvals  int     `json:"approvals"`
}

// ContentHash returns the sha256 hash of the canonical
// JSON of the metrics. Datasets and reports are content
// addressed per the doc-set determinism stance.
func (m *Metrics) ContentHash() string {
	// Round-trip through canonical JSON. We round the
	// DurationMS to milliseconds (already int) and
	// cost to 6 decimal places to keep the hash stable.
	type clone Metrics
	canon, _ := json.Marshal((*clone)(m))
	sum := sha256.Sum256(canon)
	return hex.EncodeToString(sum[:])
}

// Runner is the abstraction over the orchestrator.
//
// The benchmark is "Highly deterministic" per the doc
// set: the dataset and the runner are inputs, the
// metrics are outputs. A production Runner wraps
// aios-runtime.Orchestrator; a test Runner returns
// canned results. The interface is intentionally tiny
// so the seam is easy to wire.
type Runner interface {
	Run(task WedgeTask) Result
}

// Result is what a Runner returns. The fields map 1:1
// to the Metrics schema; the runner.Runner implementations
// fill them from their own internal accounting.
type Result struct {
	DurationMS int64
	CostUSD    float64
	Success    bool
	LoopCount  int
	ToolCalls  int
	ModelCalls int
	Approvals  int
}

// ToMetrics converts a Result to a Metrics row for
// the given task.
func (r Result) ToMetrics(task WedgeTask) Metrics {
	return Metrics{
		TaskID:     task.ID,
		DurationMS: r.DurationMS,
		CostUSD:    r.CostUSD,
		Success:    r.Success,
		LoopCount:  r.LoopCount,
		ToolCalls:  r.ToolCalls,
		ModelCalls: r.ModelCalls,
		Approvals:  r.Approvals,
	}
}

// SaveReport writes the metrics list to a JSON file at
// path. The file is content addressed: the directory
// entry is `metrics-<hash>.json`. The file uses
// canonical-sorted JSON keys for determinism.
func SaveReport(path string, rows []Metrics) error {
	canon, err := json.MarshalIndent(rows, "", "  ")
	if err != nil {
		return fmt.Errorf("benchmark: save: marshal: %w", err)
	}
	return os.WriteFile(path, canon, 0o644)
}

// LoadReport reads a metrics JSON file produced by
// SaveReport. The function is the reverse of SaveReport.
func LoadReport(path string) ([]Metrics, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("benchmark: load: %w", err)
	}
	var rows []Metrics
	if err := json.Unmarshal(data, &rows); err != nil {
		return nil, fmt.Errorf("benchmark: load: parse: %w", err)
	}
	return rows, nil
}

// Aggregates is the summary across a list of metrics.
// The fields are computed once over the whole list; the
// helper is shared by the runner, the two-class
// comparator, and the loop checker.
type Aggregates struct {
	Count          int
	Pass           int
	Fail           int
	TotalMS        int64
	TotalCost      float64
	TotalLoops     int
	MaxLoops       int
	TotalApprovals int
}

// Summarize builds aggregates from a list of metrics.
// An empty list returns the zero Aggregates.
func Summarize(rows []Metrics) Aggregates {
	a := Aggregates{}
	for _, m := range rows {
		a.Count++
		if m.Success {
			a.Pass++
		} else {
			a.Fail++
		}
		a.TotalMS += m.DurationMS
		a.TotalCost += m.CostUSD
		a.TotalLoops += m.LoopCount
		if m.LoopCount > a.MaxLoops {
			a.MaxLoops = m.LoopCount
		}
		a.TotalApprovals += m.Approvals
	}
	return a
}

// ByWedge groups metrics by their wedge id (the Wedge
// field of the task). The runner calls this to build the
// per-wedge rows in the report.
func ByWedge(tasks []WedgeTask, rows []Metrics) map[string]Aggregates {
	idx := make(map[string][]Metrics, len(rows))
	for _, m := range rows {
		for _, t := range tasks {
			if t.ID == m.TaskID {
				idx[t.Wedge] = append(idx[t.Wedge], m)
			}
		}
	}
	out := make(map[string]Aggregates, len(idx))
	for wedge, list := range idx {
		out[wedge] = Summarize(list)
	}
	return out
}

// durationSeconds is a tiny formatter: 312400ms -> "312.4s".
func durationSeconds(ms int64) string {
	d := time.Duration(ms) * time.Millisecond
	return fmt.Sprintf("%.1fs", d.Seconds())
}

// wedgeNames returns the unique wedge ids in tasks, in
// the order they first appear. Deterministic.
func wedgeNames(tasks []WedgeTask) []string {
	seen := make(map[string]bool)
	var out []string
	for _, t := range tasks {
		if !seen[t.Wedge] {
			seen[t.Wedge] = true
			out = append(out, t.Wedge)
		}
	}
	sort.Strings(out) // deterministic across runs
	return out
}
