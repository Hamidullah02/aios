package fanout

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

// Summarizer is the model-router surface the
// Summary strategy depends on (REQ-FAN-RMG-003).
//
// The interface lets the caller wire the merger
// to any model router. The default aios-models
// package exposes a Router type that satisfies
// this interface; the merger itself does not
// import aios-models, so the package stays
// decoupled from the model implementation.
type Summarizer interface {
	// Summarize turns a list of result outputs
	// into a single output. The function is the
	// model-router call the merger invokes.
	Summarize(ctx context.Context, outputs []string) (string, error)
}

// ResultMerger is the typed object that merges
// the per-task results into a single result set
// (REQ-FAN-RMG-001).
//
// The merger accepts a strategy and a list of
// results. The merger applies the strategy and
// returns the merged result. The merger is a
// pure function of its inputs: two merges with
// the same input return the same output (modulo
// the Summarizer, which is a side-effecting
// model call).
type ResultMerger struct {
	Strategy   MergeStrategy
	Summarizer Summarizer
	// Separator is the string the Concat
	// strategy inserts between the per-task
	// outputs. The default is a double newline.
	Separator string
	// HeaderFmt is the per-task header format
	// the Concat strategy uses. The format is
	// applied to each result; the default is
	// "## %s\n\n" where %s is the result id.
	HeaderFmt string
}

// NewResultMerger constructs a ResultMerger with
// the given strategy. The summarizer is required
// when the strategy is Summary.
func NewResultMerger(strategy MergeStrategy) *ResultMerger {
	return &ResultMerger{
		Strategy:  strategy,
		Separator: "\n\n",
		HeaderFmt: "## %s\n\n",
	}
}

// Merged is the result of a merge. The result
// carries the per-task outputs (in input order)
// and the merged output. The merged output is
// what the FanOutTool returns to the caller.
type Merged struct {
	// Results is the per-task result set, in
	// input order.
	Results []FanOutResult
	// Output is the merged output the caller
	// sees.
	Output string
	// Strategy is the strategy that produced
	// the merge.
	Strategy MergeStrategy
}

// Merge runs the merger over results. The
// function is the public entry point; the
// function dispatches to the strategy-specific
// implementation.
func (m *ResultMerger) Merge(ctx context.Context, results []FanOutResult) (Merged, error) {
	if m == nil {
		return Merged{}, &FanOutError{Op: "merge", Reason: "merger is nil"}
	}
	if !m.Strategy.IsValid() {
		return Merged{}, &FanOutError{Op: "merge", Reason: fmt.Sprintf("unknown strategy %q", m.Strategy)}
	}
	// Validate every result.
	for i := range results {
		if err := results[i].Validate(); err != nil {
			return Merged{}, err
		}
	}
	switch m.Strategy {
	case MergeConcat:
		return m.mergeConcat(results), nil
	case MergeSummary:
		return m.mergeSummary(ctx, results)
	case MergeVote:
		return m.mergeVote(results), nil
	}
	return Merged{}, &FanOutError{Op: "merge", Reason: fmt.Sprintf("unhandled strategy %q", m.Strategy)}
}

// mergeConcat concatenates the per-task outputs in
// order (REQ-FAN-RMG-002). The strategy is the
// safe default: no information loss.
func (m *ResultMerger) mergeConcat(results []FanOutResult) Merged {
	sep := m.Separator
	if sep == "" {
		sep = "\n\n"
	}
	hdr := m.HeaderFmt
	if hdr == "" {
		hdr = "## %s\n\n"
	}
	var b strings.Builder
	for i, r := range results {
		if i > 0 {
			b.WriteString(sep)
		}
		fmt.Fprintf(&b, hdr, r.ID)
		b.WriteString(r.Output)
	}
	return Merged{
		Results:  results,
		Output:   b.String(),
		Strategy: MergeConcat,
	}
}

// mergeSummary summarizes the per-task outputs
// through the caller's Summarizer
// (REQ-FAN-RMG-003). When the summarizer is nil,
// the merger falls back to Concat and records
// the fallback in the result.
func (m *ResultMerger) mergeSummary(ctx context.Context, results []FanOutResult) (Merged, error) {
	if m.Summarizer == nil {
		// Fallback: concat. The caller did not
		// supply a Summarizer, so the merger
		// cannot summarize. The fallback is the
		// safe default.
		merged := m.mergeConcat(results)
		merged.Strategy = MergeConcat
		return merged, nil
	}
	outputs := make([]string, len(results))
	for i, r := range results {
		outputs[i] = r.Output
	}
	summary, err := m.Summarizer.Summarize(ctx, outputs)
	if err != nil {
		return Merged{}, &FanOutError{Op: "merge", Reason: "summary failed", Err: err}
	}
	return Merged{
		Results:  results,
		Output:   summary,
		Strategy: MergeSummary,
	}, nil
}

// mergeVote picks the result that matches the
// majority of the other results (REQ-FAN-RMG-004).
//
// The vote algorithm is content-based: each
// result is scored by character-level similarity
// to every other result. The result with the
// highest total score wins. The vote returns the
// winning result; the per-task outputs are still
// available on the Merged.Results field.
func (m *ResultMerger) mergeVote(results []FanOutResult) Merged {
	if len(results) == 0 {
		return Merged{Results: results, Strategy: MergeVote}
	}
	if len(results) == 1 {
		return Merged{
			Results:  results,
			Output:   results[0].Output,
			Strategy: MergeVote,
		}
	}
	// Score each result.
	type scored struct {
		index int
		score int
	}
	scores := make([]scored, len(results))
	for i := range results {
		scores[i] = scored{index: i, score: 0}
	}
	for i := range results {
		for j := i + 1; j < len(results); j++ {
			s := similarity(results[i].Output, results[j].Output)
			scores[i].score += s
			scores[j].score += s
		}
	}
	// Pick the highest-scoring result. Tie-break
	// by the earliest input index.
	sort.SliceStable(scores, func(a, b int) bool {
		if scores[a].score != scores[b].score {
			return scores[a].score > scores[b].score
		}
		return scores[a].index < scores[b].index
	})
	winner := results[scores[0].index]
	return Merged{
		Results:  results,
		Output:   winner.Output,
		Strategy: MergeVote,
	}
}

// similarity returns a non-negative similarity
// score for the two strings. The score is the
// length of the longest common prefix plus the
// character-level overlap of the first 64
// characters. The function is the cheap
// heuristic the vote strategy uses; the doc set
// does not pin the algorithm and a real
// implementation can swap in a stronger metric.
func similarity(a, b string) int {
	if a == "" || b == "" {
		return 0
	}
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	if n > 64 {
		n = 64
	}
	score := 0
	for i := 0; i < n; i++ {
		if a[i] == b[i] {
			score++
		}
	}
	// Add a small bonus for shared runes across
	// the rest of the strings. The bonus is
	// bounded so the function is O(n).
	mid := len(a) / 2
	if mid > 0 && mid < len(b) && a[mid] == b[mid] {
		score++
	}
	return score
}
