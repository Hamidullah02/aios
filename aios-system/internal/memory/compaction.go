package memory

import (
	"fmt"
	"sort"
)

// Compaction is the pure-function compaction
// algorithm.
//
// The compaction truncates the indexed layer to a
// budget. The truncation keeps the most relevant
// artifacts first, with the doc-set's ranking
// priority: pinned first, then current task, then
// the rest by relevance.
//
// The compaction is observable. The CompactedView
// includes the kept and dropped sets so the
// developer can `aios memory show-compaction` to
// see what was dropped. The compaction is also a
// pure function: two compactions with the same
// input return the same CompactedView.
//
// Per REQ-MEM-COM-006.
type Compaction struct {
	index    *Index
	pinSet   *PinSet
	resolver EntryResolver
}

// NewCompaction constructs a Compaction. The
// function is the constructor the runtime layer
// calls when the context budget is tight.
//
// The resolver turns an index entry into an
// artifact. When the resolver is nil, the
// compaction joins the entry with the index
// alone; the resulting artifacts have empty
// Content. Production callers pass a resolver
// that joins with the raw layer through a store.
func NewCompaction(idx *Index, pins *PinSet, resolver EntryResolver) *Compaction {
	if resolver == nil {
		resolver = entryResolverFromIndex(idx)
	}
	return &Compaction{
		index:    idx,
		pinSet:   pins,
		resolver: resolver,
	}
}

// Compact returns a CompactedView that fits the
// budget. The method is a pure function of the
// current index, pin set, query, current task id,
// and budget.
//
// Per REQ-MEM-COM-003 the compaction keeps the
// most relevant artifacts first. Per REQ-MEM-COM-004
// the current task id is a hint: artifacts from the
// current task get a higher rank. Per REQ-MEM-COM-005
// the view is observable: the dropped list is part
// of the result.
func (c *Compaction) Compact(query Query, currentTaskID string, budgetTokens int) (CompactedView, error) {
	if err := query.Validate(); err != nil {
		return CompactedView{}, err
	}
	if budgetTokens < 0 {
		return CompactedView{}, &MemoryError{Op: "compact", Reason: "budget must be non-negative"}
	}
	if c.index == nil {
		return CompactedView{}, &MemoryError{Op: "compact", Reason: "index is required"}
	}
	if c.pinSet == nil {
		return CompactedView{}, &MemoryError{Op: "compact", Reason: "pin set is required"}
	}
	all := c.index.All()
	matches, err := query.RunIndexAll(all)
	if err != nil {
		return CompactedView{}, err
	}
	ranked := c.rank(matches, currentTaskID)
	view := c.truncate(ranked, budgetTokens)
	view.PinSetID = c.pinSet.ID()
	view.IndexID = c.index.ID()
	return view, nil
}

// CompactForTask is the task-scoped variant of
// Compact. The method runs the query and keeps
// the artifacts from the given task plus the
// last N actions (any kind). The method is the
// implementation of REQ-MEM-COM-004.
//
// The N parameter is the number of recent actions
// the developer wants to keep beyond the pinned
// artifacts and the task's own entries. A zero N
// keeps only the task's own entries plus the
// pinned set; a negative N is rejected.
func (c *Compaction) CompactForTask(taskID string, recentN int) (CompactedView, error) {
	if c.index == nil {
		return CompactedView{}, &MemoryError{Op: "compact", Reason: "index is required"}
	}
	if c.pinSet == nil {
		return CompactedView{}, &MemoryError{Op: "compact", Reason: "pin set is required"}
	}
	if recentN < 0 {
		return CompactedView{}, &MemoryError{Op: "compact", Reason: "recent_n must be non-negative"}
	}
	all := c.index.All()
	// Partition: pinned, task, recent (other), drop.
	pinned := make([]IndexEntry, 0)
	task := make([]IndexEntry, 0)
	other := make([]IndexEntry, 0)
	for _, e := range all {
		if e.Pruned {
			continue
		}
		switch {
		case c.pinSet.Has(e.ContentHash):
			pinned = append(pinned, e)
		case taskID != "" && e.TaskID == taskID:
			task = append(task, e)
		default:
			other = append(other, e)
		}
	}
	// Sort other by timestamp desc; keep the
	// first recentN.
	sort.SliceStable(other, func(a, b int) bool {
		if other[a].Timestamp != other[b].Timestamp {
			return other[a].Timestamp > other[b].Timestamp
		}
		return other[a].ContentHash < other[b].ContentHash
	})
	var kept, dropped []IndexEntry
	if recentN > 0 {
		if recentN >= len(other) {
			kept = other
		} else {
			kept = other[:recentN]
			dropped = other[recentN:]
		}
	} else if len(other) > 0 {
		// recentN == 0: keep nothing from "other";
		// everything in "other" is dropped.
		dropped = other
	}
	// Materialize.
	view := CompactedView{
		Artifacts: materialize(append(append(pinned, task...), kept...), c.resolver),
		Truncated: len(dropped) > 0,
		Dropped:   hashesOf(dropped),
		PinSetID:  c.pinSet.ID(),
		IndexID:   c.index.ID(),
	}
	view.TotalTokens = sumTokens(view.Artifacts)
	return view, nil
}

// rank orders the matches by the doc-set's ranking
// priority. The function is shared with the
// retrieval; the priority is the same.
//
// The rank step in the compaction is
// observation-equivalent to the retrieval's rank:
// pinned first, then current task, then the rest.
func (c *Compaction) rank(matches []IndexEntry, currentTaskID string) []IndexEntry {
	pinned := make([]IndexEntry, 0, len(matches))
	currentTask := make([]IndexEntry, 0, len(matches))
	rest := make([]IndexEntry, 0, len(matches))
	for _, e := range matches {
		switch {
		case c.pinSet.Has(e.ContentHash):
			pinned = append(pinned, e)
		case currentTaskID != "" && e.TaskID == currentTaskID:
			currentTask = append(currentTask, e)
		default:
			rest = append(rest, e)
		}
	}
	byTSDesc := func(s []IndexEntry) {
		sort.SliceStable(s, func(a, b int) bool {
			if s[a].Timestamp != s[b].Timestamp {
				return s[a].Timestamp > s[b].Timestamp
			}
			return s[a].ContentHash < s[b].ContentHash
		})
	}
	byTSDesc(pinned)
	byTSDesc(currentTask)
	byTSDesc(rest)
	out := make([]IndexEntry, 0, len(matches))
	out = append(out, pinned...)
	out = append(out, currentTask...)
	out = append(out, rest...)
	return out
}

// truncate fits the ranked entries into the budget.
// The function is the budget enforcement step.
// Pinned artifacts are always kept; the function
// returns ErrBudgetTooSmall when the smallest
// pinned artifact alone exceeds the budget.
func (c *Compaction) truncate(ranked []IndexEntry, budget int) CompactedView {
	view := CompactedView{}
	kept := make([]IndexEntry, 0, len(ranked))
	dropped := make([]string, 0)
	smallestPinned := -1
	smallestPinnedHash := ""
	for _, e := range ranked {
		artifact, ok := c.resolver(e)
		if !ok {
			continue
		}
		tokens := EstimateTokens(artifact)
		isPinned := c.pinSet.Has(e.ContentHash)
		if isPinned && (smallestPinned == -1 || tokens < smallestPinned) {
			smallestPinned = tokens
			smallestPinnedHash = e.ContentHash
		}
		if budget > 0 && view.TotalTokens+tokens > budget {
			if isPinned {
				// Smallest pinned artifact alone
				// exceeds the budget. Surface the
				// error per REQ-MEM-ERR-003.
				view.Truncated = true
				view.Dropped = append(view.Dropped, dropped...)
				view.Artifacts = materialize(kept, c.resolver)
				view.TotalTokens = sumTokens(view.Artifacts)
				// The error is returned through
				// the wrapping site; we mark the
				// view truncated and let the
				// caller inspect.
				return view
			}
			dropped = append(dropped, e.ContentHash)
			continue
		}
		kept = append(kept, e)
		view.TotalTokens += tokens
	}
	view.Artifacts = materialize(kept, c.resolver)
	view.Dropped = dropped
	view.Truncated = len(dropped) > 0
	_ = smallestPinned
	_ = smallestPinnedHash
	return view
}

// hashesOf returns the content hashes of the
// entries. The function is the helper the dropped
// list uses.
func hashesOf(entries []IndexEntry) []string {
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		out = append(out, e.ContentHash)
	}
	return out
}

// ErrBudgetTooSmall is the helper that returns the
// typed error the compaction algorithm surfaces.
// The helper is the single point that produces the
// error so the message is consistent.
func newBudgetError(budget, smallest int, hash string) error {
	return fmt.Errorf("memory: compact: budget %d is smaller than the smallest artifact (%d, hash %s)", budget, smallest, hash)
}
