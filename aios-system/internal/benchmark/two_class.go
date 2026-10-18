package benchmark

import "context"

// TwoClassMetrics is the result of a two-class run
// (REQ-BCH-2TR-003).
//
// The struct holds the strong and the weak metrics plus
// the per-metric deltas (strong - weak). The deltas use
// a signed convention: positive numbers mean the
// strong class is better, negative means the weak
// class is better.
type TwoClassMetrics struct {
	Strong  []Metrics    `json:"strong"`
	Weak    []Metrics    `json:"weak"`
	Delta   MetricsDelta `json:"delta"`
	Summary Aggregates   `json:"summary"`
}

// MetricsDelta is the per-axis delta. The wireframe
// shows pass rate, average duration, average cost, and
// loop rate. The struct is flat so JSON output is
// stable.
type MetricsDelta struct {
	PassRate      float64 `json:"pass_rate"`       // percentage points (strong - weak)
	AvgDurationMS float64 `json:"avg_duration_ms"` // ms per task
	AvgCostUSD    float64 `json:"avg_cost_usd"`    // $ per task
	LoopRate      float64 `json:"loop_rate"`       // percentage points
	TotalCostUSD  float64 `json:"total_cost_usd"`  // $ across the dataset
}

// TwoClassRunner runs the dataset with two runners
// (REQ-BCH-2TR-001).
type TwoClassRunner struct {
	strong  Runner
	weak    Runner
	dataset WedgeDataset
}

// NewTwoClass returns a TwoClassRunner. The strong and
// weak runners share the dataset.
func NewTwoClass(strong, weak Runner, d WedgeDataset) *TwoClassRunner {
	return &TwoClassRunner{strong: strong, weak: weak, dataset: d}
}

// Run executes the dataset on both runners and returns
// the deltas (REQ-BCH-2TR-002 / 003).
func (t *TwoClassRunner) Run(ctx context.Context) (TwoClassMetrics, error) {
	strong, err := New(t.strong, t.dataset).Run(ctx)
	if err != nil {
		return TwoClassMetrics{}, err
	}
	weak, err := New(t.weak, t.dataset).Run(ctx)
	if err != nil {
		return TwoClassMetrics{}, err
	}
	sa := Summarize(strong)
	wa := Summarize(weak)
	return TwoClassMetrics{
		Strong: strong,
		Weak:   weak,
		Delta:  delta(sa, wa),
		Summary: Aggregates{
			Count: sa.Count + wa.Count,
			Pass:  sa.Pass + wa.Pass,
		},
	}, nil
}

// delta computes the per-axis delta. Pass rate and loop
// rate are in percentage points; the rest are per-task
// averages. The division-by-zero is guarded.
func delta(s, w Aggregates) MetricsDelta {
	if s.Count == 0 || w.Count == 0 {
		return MetricsDelta{}
	}
	sp := 100 * float64(s.Pass) / float64(s.Count)
	wp := 100 * float64(w.Pass) / float64(w.Count)
	// Loop rate is the percentage of tasks with at
	// least one loop (LoopCount > 0).
	// Aggregates does not carry that, so we approximate
	// with the average loop count: low average = no
	// looping. This is the doc-set's "loop rate" proxy.
	sl := 100 * float64(s.TotalLoops) / float64(s.Count)
	wl := 100 * float64(w.TotalLoops) / float64(w.Count)
	return MetricsDelta{
		PassRate:      sp - wp,
		AvgDurationMS: float64(s.TotalMS)/float64(s.Count) - float64(w.TotalMS)/float64(w.Count),
		AvgCostUSD:    s.TotalCost/float64(s.Count) - w.TotalCost/float64(w.Count),
		LoopRate:      sl - wl,
		TotalCostUSD:  s.TotalCost - w.TotalCost,
	}
}
