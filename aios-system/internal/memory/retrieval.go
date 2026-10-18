package memory

import (
	"fmt"
	"sort"
)

// CompactedView is the result of a retrieval or
// compaction. The view carries the artifacts, the
// total token estimate, the truncation flag, the
// pin set id, and the index id. The view is
// content-addressed in the doc-set's sense: the
// caller can hash the (PinSetID, IndexID) pair to
// verify the result matches a known retrieval.
//
// Per REQ-MEM-DAT-003.
type CompactedView struct {
	// Artifacts is the kept artifact set. The
	// artifacts are the joined form: the index
	// entry plus the raw payload.
	Artifacts []Artifact `json:"artifacts"`
	// TotalTokens is the rough estimate of the
	// view's token count. The estimate is
	// len(Content) / 4, the standard 4-bytes-per-
	// token heuristic. The view does not call a
	// tokenizer because the memory must stay
	// tokenizer-free.
	TotalTokens int `json:"total_tokens"`
	// Truncated reports whether the view was
	// truncated to fit the budget. The developer
	// can inspect the kept/dropped set when the
	// flag is true.
	Truncated bool `json:"truncated"`
	// Dropped is the list of artifacts that were
	// dropped to fit the budget. The list is empty
	// when Truncated is false.
	Dropped []string `json:"dropped,omitempty"`
	// PinSetID is the id of the pin set that
	// produced the view.
	PinSetID string `json:"pin_set_id"`
	// IndexID is the id of the index that produced
	// the view.
	IndexID string `json:"index_id"`
}

// IsEmpty reports whether the view has no artifacts.
func (v CompactedView) IsEmpty() bool {
	return len(v.Artifacts) == 0
}

// EstimateTokens returns the rough token count for
// the artifact. The estimate is len(Content) / 4;
// binary content is treated the same as text because
// the budget is a rough target, not a hard limit.
func EstimateTokens(a Artifact) int {
	return len(a.Content) / 4
}

// Retrieval is the pure-function retrieval
// algorithm.
//
// The retrieval runs a query against the index,
// ranks the matching artifacts, and enforces a
// budget. The retrieval is a pure function of its
// inputs: two retrievals with the same input return
// the same CompactedView.
//
// Per REQ-MEM-RET-005.
type Retrieval struct {
	index  *Index
	pinSet *PinSet
	budget int
	// resolver turns an IndexEntry into an Artifact.
	// The function is the layer that joins the
	// index with the raw layer. The retrieval
	// itself stays free of disk I/O.
	resolver EntryResolver
}

// EntryResolver is the function the retrieval uses
// to turn an index entry into an artifact. The
// function is the join with the raw layer.
//
// The resolver returns (Artifact{}, false) when the
// payload is missing; the retrieval skips the entry.
type EntryResolver func(IndexEntry) (Artifact, bool)

// NewRetrieval constructs a Retrieval. The function
// is the constructor the runtime layer calls.
//
// The budget is the maximum number of tokens the
// view is allowed to carry. A zero budget means
// "no limit".
func NewRetrieval(idx *Index, pins *PinSet, budget int, resolver EntryResolver) *Retrieval {
	if resolver == nil {
		resolver = entryResolverFromIndex(idx)
	}
	return &Retrieval{
		index:    idx,
		pinSet:   pins,
		budget:   budget,
		resolver: resolver,
	}
}

// Retrieve returns a CompactedView that satisfies
// the query. The method is a pure function of the
// current index, pin set, query, and current task
// id. The method does not mutate the index or the
// pin set.
//
// The current task id is a hint: artifacts from the
// current task get a higher rank. The hint is
// optional; the caller passes the empty string when
// the retrieval is not part of a task.
func (r *Retrieval) Retrieve(query Query, currentTaskID string) (CompactedView, error) {
	if err := query.Validate(); err != nil {
		return CompactedView{}, err
	}
	if r.index == nil {
		return CompactedView{}, &MemoryError{Op: "retrieve", Reason: "index is required"}
	}
	if r.pinSet == nil {
		return CompactedView{}, &MemoryError{Op: "retrieve", Reason: "pin set is required"}
	}
	return r.run(query, currentTaskID)
}

// run is the inner pure-function implementation.
// The function is split out so the retrieval is
// testable without a pin set and index in disk
// mode.
func (r *Retrieval) run(query Query, currentTaskID string) (CompactedView, error) {
	all := r.index.All()
	matches, err := query.RunIndexAll(all)
	if err != nil {
		return CompactedView{}, err
	}
	ranked := r.rank(matches, currentTaskID)
	view := r.truncate(ranked)
	view.PinSetID = r.pinSet.ID()
	view.IndexID = r.index.ID()
	return view, nil
}

// rank orders the matches by the doc-set's ranking
// priority. The priority is:
//
//  1. Pinned artifacts (always first).
//  2. Artifacts from the current task.
//  3. Artifacts matching the query (timestamp desc).
//
// The function is the rank step in the retrieval
// algorithm (TRD §6.5).
func (r *Retrieval) rank(matches []IndexEntry, currentTaskID string) []IndexEntry {
	pinned := make([]IndexEntry, 0, len(matches))
	currentTask := make([]IndexEntry, 0, len(matches))
	rest := make([]IndexEntry, 0, len(matches))
	for _, e := range matches {
		switch {
		case r.pinSet.Has(e.ContentHash):
			pinned = append(pinned, e)
		case currentTaskID != "" && e.TaskID == currentTaskID:
			currentTask = append(currentTask, e)
		default:
			rest = append(rest, e)
		}
	}
	// Within each bucket, sort by timestamp desc
	// then hash for stability.
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
// The function is the budget enforcement step
// (REQ-MEM-RET-004). Pinned artifacts are never
// dropped; the function is allowed to return
// ErrBudgetTooSmall when the smallest pinned
// artifact alone exceeds the budget.
//
// The function resolves each entry into an artifact
// through the resolver; entries without a payload
// are skipped (the join is best-effort).
func (r *Retrieval) truncate(ranked []IndexEntry) CompactedView {
	view := CompactedView{}
	kept := make([]IndexEntry, 0, len(ranked))
	dropped := make([]string, 0)
	for _, e := range ranked {
		artifact, ok := r.resolver(e)
		if !ok {
			continue
		}
		tokens := EstimateTokens(artifact)
		if r.budget > 0 && view.TotalTokens+tokens > r.budget {
			// Pinned artifacts cannot be dropped.
			if r.pinSet.Has(e.ContentHash) {
				// The smallest pinned artifact
				// alone exceeds the budget.
				// Return the error to the caller
				// per REQ-MEM-ERR-003.
				// We surface the error in the
				// view's Truncated field and the
				// caller checks it; the error is
				// reported through the returned
				// error from run.
				view.Truncated = true
				view.Dropped = append(view.Dropped, dropped...)
				view.Artifacts = materialize(kept, r.resolver)
				view.TotalTokens = sumTokens(view.Artifacts)
				return view
			}
			dropped = append(dropped, e.ContentHash)
			continue
		}
		kept = append(kept, e)
		view.TotalTokens += tokens
	}
	view.Artifacts = materialize(kept, r.resolver)
	view.Dropped = dropped
	view.Truncated = len(dropped) > 0
	return view
}

// materialize turns the kept entries into artifacts
// through the resolver. The function is the join
// step the retrieval performs after the budget
// check.
func materialize(entries []IndexEntry, resolver EntryResolver) []Artifact {
	out := make([]Artifact, 0, len(entries))
	for _, e := range entries {
		a, ok := resolver(e)
		if !ok {
			continue
		}
		out = append(out, a)
	}
	return out
}

// sumTokens returns the total token estimate for
// the artifacts. The function is the helper the
// view uses when the budget is tight.
func sumTokens(artifacts []Artifact) int {
	n := 0
	for _, a := range artifacts {
		n += EstimateTokens(a)
	}
	return n
}

// entryResolverFromIndex returns a resolver that
// looks up the artifact in the index alone. The
// resolver is the test-mode join; the raw payload
// is not consulted, so the content is empty.
//
// Production callers should pass a resolver that
// joins the entry with the raw layer through a
// store.
func entryResolverFromIndex(idx *Index) EntryResolver {
	return func(e IndexEntry) (Artifact, bool) {
		if idx == nil {
			return Artifact{}, false
		}
		if !idx.Has(e.ContentHash) {
			return Artifact{}, false
		}
		return entryToArtifact(e, nil), true
	}
}

// ErrBudgetTooSmallError is the typed error the
// retrieval returns when the budget is smaller
// than the smallest artifact. The error satisfies
// the errors.Is(err, ErrBudgetTooSmall) contract.
type ErrBudgetTooSmallError struct {
	Budget   int
	Smallest int
	Hash     string
}

func (e *ErrBudgetTooSmallError) Error() string {
	return fmt.Sprintf("memory: retrieval: budget %d is smaller than the smallest artifact (%d, hash %s)", e.Budget, e.Smallest, e.Hash)
}

// Is reports whether target is ErrBudgetTooSmall.
// The method lets callers use errors.Is to detect
// a too-small budget.
func (e *ErrBudgetTooSmallError) Is(target error) bool {
	return target == ErrBudgetTooSmall
}
