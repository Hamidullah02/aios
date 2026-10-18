package fanout

import (
	"context"
	"errors"
	"testing"
)

// sampleResults returns a list of successful
// results for tests to merge.
func sampleResults() []FanOutResult {
	return []FanOutResult{
		{ID: "a", Index: 0, Output: "alpha", Status: StatusSucceeded, DurationMs: 10, CostUSD: 0.1},
		{ID: "b", Index: 1, Output: "beta", Status: StatusSucceeded, DurationMs: 20, CostUSD: 0.2},
		{ID: "c", Index: 2, Output: "gamma", Status: StatusSucceeded, DurationMs: 30, CostUSD: 0.3},
	}
}

// TestMergeConcat covers REQ-FAN-RMG-002.
func TestMergeConcat(t *testing.T) {
	m := NewResultMerger(MergeConcat)
	merged, err := m.Merge(context.Background(), sampleResults())
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}
	if merged.Strategy != MergeConcat {
		t.Fatalf("strategy = %q", merged.Strategy)
	}
	if !contains(merged.Output, "## a") || !contains(merged.Output, "## b") || !contains(merged.Output, "## c") {
		t.Fatalf("missing headers: %q", merged.Output)
	}
	if !contains(merged.Output, "alpha") || !contains(merged.Output, "beta") || !contains(merged.Output, "gamma") {
		t.Fatalf("missing bodies: %q", merged.Output)
	}
}

// TestMergeConcatEmpty covers the empty result
// path.
func TestMergeConcatEmpty(t *testing.T) {
	m := NewResultMerger(MergeConcat)
	merged, err := m.Merge(context.Background(), nil)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}
	if merged.Output != "" {
		t.Fatalf("output = %q, want empty", merged.Output)
	}
}

// TestMergeConcatCustomSeparator covers the
// separator and header customization.
func TestMergeConcatCustomSeparator(t *testing.T) {
	m := NewResultMerger(MergeConcat)
	m.Separator = " | "
	m.HeaderFmt = "[%s] "
	merged, err := m.Merge(context.Background(), sampleResults())
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}
	if !contains(merged.Output, "[a] alpha | [b] beta | [c] gamma") {
		t.Fatalf("got %q", merged.Output)
	}
}

// TestMergeSummary covers REQ-FAN-RMG-003.
func TestMergeSummary(t *testing.T) {
	m := NewResultMerger(MergeSummary)
	m.Summarizer = SummarizerFunc(func(ctx context.Context, outputs []string) (string, error) {
		return "summary of " + joinStrings(outputs, ","), nil
	})
	merged, err := m.Merge(context.Background(), sampleResults())
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}
	if merged.Strategy != MergeSummary {
		t.Fatalf("strategy = %q", merged.Strategy)
	}
	if !contains(merged.Output, "summary of") {
		t.Fatalf("output = %q", merged.Output)
	}
}

// TestMergeSummaryFallsBackToConcat covers the
// fallback when no Summarizer is wired.
func TestMergeSummaryFallsBackToConcat(t *testing.T) {
	m := NewResultMerger(MergeSummary)
	merged, err := m.Merge(context.Background(), sampleResults())
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}
	if merged.Strategy != MergeConcat {
		t.Fatalf("strategy = %q, want Concat fallback", merged.Strategy)
	}
}

// TestMergeSummaryError covers the Summarizer
// error path.
func TestMergeSummaryError(t *testing.T) {
	m := NewResultMerger(MergeSummary)
	m.Summarizer = SummarizerFunc(func(ctx context.Context, outputs []string) (string, error) {
		return "", errors.New("summary failed")
	})
	_, err := m.Merge(context.Background(), sampleResults())
	if err == nil {
		t.Fatal("expected error")
	}
	if !contains(err.Error(), "summary failed") {
		t.Fatalf("err = %v", err)
	}
}

// TestMergeVote covers REQ-FAN-RMG-004. Three
// similar results and one outlier; the vote
// should pick the most-similar one.
func TestMergeVote(t *testing.T) {
	results := []FanOutResult{
		{ID: "outlier", Index: 0, Output: "completely different content here", Status: StatusSucceeded, DurationMs: 10},
		{ID: "alpha1", Index: 1, Output: "the quick brown fox", Status: StatusSucceeded, DurationMs: 10},
		{ID: "alpha2", Index: 2, Output: "the quick brown fox jumps", Status: StatusSucceeded, DurationMs: 10},
		{ID: "alpha3", Index: 3, Output: "the quick brown fox runs", Status: StatusSucceeded, DurationMs: 10},
	}
	m := NewResultMerger(MergeVote)
	merged, err := m.Merge(context.Background(), results)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}
	if merged.Strategy != MergeVote {
		t.Fatalf("strategy = %q", merged.Strategy)
	}
	// The two alpha results share the longest
	// prefix "the quick brown fox" (19 chars);
	// the third alpha shares that too. So one
	// of the alphas wins. The winner's output
	// should be the one with the most overlap.
	if !contains(merged.Output, "the quick brown fox") {
		t.Fatalf("winner = %q", merged.Output)
	}
	if merged.Output == results[0].Output {
		t.Fatalf("vote picked the outlier")
	}
}

// TestMergeVoteSingleResult covers the
// single-result edge case.
func TestMergeVoteSingleResult(t *testing.T) {
	results := []FanOutResult{
		{ID: "a", Output: "only one", Status: StatusSucceeded, DurationMs: 5},
	}
	m := NewResultMerger(MergeVote)
	merged, err := m.Merge(context.Background(), results)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}
	if merged.Output != "only one" {
		t.Fatalf("output = %q", merged.Output)
	}
}

// TestMergeVoteEmpty covers the empty-result
// edge case.
func TestMergeVoteEmpty(t *testing.T) {
	m := NewResultMerger(MergeVote)
	merged, err := m.Merge(context.Background(), nil)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}
	if merged.Output != "" {
		t.Fatalf("output = %q, want empty", merged.Output)
	}
}

// TestMergeInvalidStrategy covers the strategy
// guard.
func TestMergeInvalidStrategy(t *testing.T) {
	m := NewResultMerger(MergeStrategy("nope"))
	_, err := m.Merge(context.Background(), sampleResults())
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestMergeInvalidResult covers the result
// validation.
func TestMergeInvalidResult(t *testing.T) {
	m := NewResultMerger(MergeConcat)
	results := []FanOutResult{{ID: "a", Status: "ok"}} // bad status
	_, err := m.Merge(context.Background(), results)
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestMergeNilMerger covers the nil receiver
// guard.
func TestMergeNilMerger(t *testing.T) {
	var m *ResultMerger
	_, err := m.Merge(context.Background(), sampleResults())
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestSimilarity covers the cheap heuristic
// the vote strategy uses.
func TestSimilarity(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"", "", 0},
		{"abc", "", 0},
		{"abc", "abc", 4}, // 3 shared prefix chars + 1 mid bonus
		{"abc", "abd", 3}, // 2 shared prefix chars + 1 mid bonus
		{"the quick brown fox", "the quick brown fox jumps", 20}, // 19 + 1 mid bonus
	}
	for _, c := range cases {
		if got := similarity(c.a, c.b); got != c.want {
			t.Fatalf("similarity(%q, %q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

// joinStrings joins a slice of strings with a
// separator. The function is the test helper
// the Summarizer test uses.
func joinStrings(s []string, sep string) string {
	if len(s) == 0 {
		return ""
	}
	out := s[0]
	for i := 1; i < len(s); i++ {
		out += sep + s[i]
	}
	return out
}

// SummarizerFunc is the test adapter the
// summarizer tests use.
type SummarizerFunc func(ctx context.Context, outputs []string) (string, error)

// Summarize invokes the underlying function.
func (f SummarizerFunc) Summarize(ctx context.Context, outputs []string) (string, error) {
	return f(ctx, outputs)
}
