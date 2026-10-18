package memory

import (
	"fmt"
	"sort"
)

// Query is the typed query language.
//
// Per REQ-MEM-QUE-001 the query accepts kinds, scopes,
// tags, a glob pattern, a since timestamp, a limit,
// and an AND/OR operator. The query is a value type;
// the retrieval and compaction algorithms use it as
// the input to their pure functions.
//
// An empty query matches every indexed artifact.
type Query struct {
	// Kinds is the list of artifact kinds. The query
	// matches artifacts of the listed kinds. Empty
	// matches every kind.
	Kinds []string `json:"kinds,omitempty"`
	// Scopes is the list of scopes. The query matches
	// artifacts in the listed scopes. Empty matches
	// every scope.
	Scopes []string `json:"scopes,omitempty"`
	// Tags is the list of tags. The query matches
	// artifacts that share at least one tag with the
	// list. Empty matches every artifact.
	Tags []string `json:"tags,omitempty"`
	// Pattern is the glob pattern. The query matches
	// artifacts whose content matches the pattern.
	// Empty matches every artifact.
	Pattern string `json:"pattern,omitempty"`
	// Since is the ISO 8601 timestamp. The query
	// matches artifacts seen at or after the
	// timestamp. Empty matches every artifact.
	Since string `json:"since,omitempty"`
	// Limit is the maximum number of artifacts the
	// query returns. Zero means no limit.
	Limit int `json:"limit,omitempty"`
	// Operator is the boolean operator that combines
	// the filters. Empty defaults to AND.
	Operator Operator `json:"operator,omitempty"`
}

// NewQuery constructs a query with the given kinds,
// scopes, and tags. The function is the typed
// constructor; callers that build the literal can
// skip it.
func NewQuery(kinds, scopes, tags []string) Query {
	return Query{
		Kinds:    append([]string(nil), kinds...),
		Scopes:   append([]string(nil), scopes...),
		Tags:     append([]string(nil), tags...),
		Operator: OpAND,
	}
}

// NewQueryWithPattern constructs a query with a
// glob pattern. The function is the constructor for
// the `aios memory grep` subcommand.
func NewQueryWithPattern(pattern string) Query {
	return Query{
		Pattern:  pattern,
		Operator: OpAND,
	}
}

// NewSessionQuery constructs a query that matches
// every artifact in the given session. The function
// is the constructor for the resume flow.
func NewSessionQuery(sessionID string) Query {
	return Query{
		// The session filter is applied at the index
		// level; the kind/scope filters stay empty.
		Operator: OpAND,
		// sessionID is stashed in Tags so a future
		// index filter can pick it up. The current
		// implementation matches on Tags through
		// IndexEntry.Match; the call site passes the
		// session id through.
		Tags: []string{"session:" + sessionID},
	}
}

// Validate checks the query is well-formed. The
// function is the gate the retrieval and compaction
// algorithms call before running.
//
// The validation rules are: the operator is empty or
// one of AND/OR, and the kinds/scopes are valid
// enums.
func (q Query) Validate() error {
	if q.Operator != "" && !q.Operator.IsValid() {
		return &MemoryError{Op: "query", Reason: fmt.Sprintf("operator must be AND or OR; got %q", q.Operator)}
	}
	for _, k := range q.Kinds {
		if !ArtifactKind(k).IsValid() {
			return &MemoryError{Op: "query", Reason: fmt.Sprintf("invalid kind %q", k)}
		}
	}
	for _, s := range q.Scopes {
		if !Scope(s).IsValid() {
			return &MemoryError{Op: "query", Reason: fmt.Sprintf("invalid scope %q", s)}
		}
	}
	return nil
}

// RunIndex executes the query against the index.
// The function returns the matching index entries,
// sorted by timestamp descending and content hash
// for stability.
//
// Per REQ-MEM-QUE-002 the result is sorted by
// timestamp in descending order. The function is the
// in-memory runner the retrieval algorithm uses.
//
// RunIndex applies the Limit after the sort, so a
// query with Limit=10 returns the 10 most recent
// matches.
func (q Query) RunIndex(entries []IndexEntry) ([]IndexEntry, error) {
	if err := q.Validate(); err != nil {
		return nil, err
	}
	matches := make([]IndexEntry, 0, len(entries))
	for _, e := range entries {
		if e.Match(q) {
			matches = append(matches, e)
		}
	}
	sort.Slice(matches, func(a, b int) bool {
		if matches[a].Timestamp != matches[b].Timestamp {
			return matches[a].Timestamp > matches[b].Timestamp
		}
		return matches[a].ContentHash < matches[b].ContentHash
	})
	if q.Limit > 0 && len(matches) > q.Limit {
		matches = matches[:q.Limit]
	}
	return matches, nil
}

// RunIndexAll executes the query against the index
// without a limit. The function is the variant the
// compaction algorithm uses when it needs the full
// match set.
func (q Query) RunIndexAll(entries []IndexEntry) ([]IndexEntry, error) {
	if err := q.Validate(); err != nil {
		return nil, err
	}
	matches := make([]IndexEntry, 0, len(entries))
	for _, e := range entries {
		if e.Match(q) {
			matches = append(matches, e)
		}
	}
	sort.Slice(matches, func(a, b int) bool {
		if matches[a].Timestamp != matches[b].Timestamp {
			return matches[a].Timestamp > matches[b].Timestamp
		}
		return matches[a].ContentHash < matches[b].ContentHash
	})
	return matches, nil
}

// IsEmpty reports whether the query has no filters.
// An empty query matches every indexed artifact. The
// function is the cheap check the retrieval uses to
// skip the filter pass.
func (q Query) IsEmpty() bool {
	return len(q.Kinds) == 0 && len(q.Scopes) == 0 && len(q.Tags) == 0 && q.Pattern == "" && q.Since == ""
}
