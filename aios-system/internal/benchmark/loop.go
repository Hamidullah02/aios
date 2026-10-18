package benchmark

import "context"

// LoopMetrics is the result of a loop check
// (REQ-BCH-LPC-003).
type LoopMetrics struct {
	Count        int      `json:"count"`
	AvgLoopCount float64  `json:"avg_loop_count"`
	MaxLoopCount int      `json:"max_loop_count"`
	LoopRate     float64  `json:"loop_rate"`
	LoopingTasks []string `json:"looping_tasks"`
}

// LoopChecker runs the dataset and counts loops
// (REQ-BCH-LPC-001).
type LoopChecker struct {
	runner  Runner
	dataset WedgeDataset
}

// NewLoopChecker returns a LoopChecker.
func NewLoopChecker(r Runner, d WedgeDataset) *LoopChecker {
	return &LoopChecker{runner: r, dataset: d}
}

// Check runs the dataset and returns the loop metrics
// (REQ-BCH-LPC-002 / 003). A task is considered to
// have "looped" when LoopCount > 0. The threshold can
// be raised by the caller; the default of 0 matches
// the doc-set wireframe.
func (l *LoopChecker) Check(ctx context.Context, threshold int) (LoopMetrics, error) {
	rows, err := New(l.runner, l.dataset).Run(ctx)
	if err != nil {
		return LoopMetrics{}, err
	}
	if threshold < 0 {
		threshold = 0
	}
	out := LoopMetrics{Count: len(rows), MaxLoopCount: 0}
	if out.Count == 0 {
		return out, nil
	}
	total := 0
	for _, m := range rows {
		total += m.LoopCount
		if m.LoopCount > out.MaxLoopCount {
			out.MaxLoopCount = m.LoopCount
		}
		if m.LoopCount > threshold {
			out.LoopingTasks = append(out.LoopingTasks, m.TaskID)
		}
	}
	out.AvgLoopCount = float64(total) / float64(out.Count)
	out.LoopRate = 100 * float64(len(out.LoopingTasks)) / float64(out.Count)
	return out, nil
}
