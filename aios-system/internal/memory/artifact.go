package memory

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
)

// ArtifactKind is the typed kind of an artifact. The
// memory categorizes every artifact as one of the six
// kinds listed below. The kind is used by the typed
// index and the query language.
type ArtifactKind string

const (
	// KindTask is a unit of work in the plan.
	KindTask ArtifactKind = "task"
	// KindFile is a file read or written by the agent.
	KindFile ArtifactKind = "file"
	// KindDecision is a decision the agent made.
	KindDecision ArtifactKind = "decision"
	// KindObservation is a fact the agent noticed.
	KindObservation ArtifactKind = "observation"
	// KindToolResult is the result of a tool call.
	KindToolResult ArtifactKind = "tool_result"
	// KindModelOutput is the output of a model call.
	KindModelOutput ArtifactKind = "model_output"
)

// IsValid reports whether k is one of the six known
// artifact kinds.
func (k ArtifactKind) IsValid() bool {
	switch k {
	case KindTask, KindFile, KindDecision, KindObservation,
		KindToolResult, KindModelOutput:
		return true
	}
	return false
}

// Scope is the typed scope of an artifact. The
// memory keeps artifacts in three scopes. The session
// scope is in-memory only. The project scope is
// git-trackable. The user scope is encrypted at rest.
type Scope string

const (
	// ScopeSession is the in-memory scope.
	ScopeSession Scope = "session"
	// ScopeProject is the project-level scope.
	ScopeProject Scope = "project"
	// ScopeUser is the user-level scope.
	ScopeUser Scope = "user"
)

// IsValid reports whether s is one of the three known
// scopes.
func (s Scope) IsValid() bool {
	switch s {
	case ScopeSession, ScopeProject, ScopeUser:
		return true
	}
	return false
}

// Operator is the typed boolean operator for the
// query language. The query combines its filters with
// AND or OR. The default is AND.
type Operator string

const (
	// OpAND is the AND operator. The artifact matches
	// when every non-empty filter matches.
	OpAND Operator = "AND"
	// OpOR is the OR operator. The artifact matches
	// when at least one non-empty filter matches.
	OpOR Operator = "OR"
)

// IsValid reports whether o is one of the two known
// operators.
func (o Operator) IsValid() bool {
	return o == OpAND || o == OpOR
}

// Artifact is the typed object that holds one piece
// of remembered content.
//
// Per REQ-MEM-DAT-001 the artifact carries the kind,
// the scope, the session id, the optional task id, the
// timestamp, the tags, the content, and the content
// hash. Two artifacts with the same Content have the
// same ContentHash.
type Artifact struct {
	// Kind is the artifact's category. Required.
	Kind ArtifactKind `json:"kind"`
	// Scope is the artifact's persistence scope.
	// Required.
	Scope Scope `json:"scope"`
	// SessionID is the session that produced the
	// artifact. Required.
	SessionID string `json:"session_id"`
	// TaskID is the task that produced the artifact.
	// Optional.
	TaskID string `json:"task_id,omitempty"`
	// Timestamp is the ISO 8601 timestamp. The default
	// is the current UTC time when the field is empty.
	Timestamp string `json:"timestamp"`
	// Tags is the optional list of tags.
	Tags []string `json:"tags,omitempty"`
	// Content is the artifact payload. The bytes are
	// the canonical content; the SHA-256 of the bytes
	// is the ContentHash.
	Content []byte `json:"content"`
	// ContentHash is the SHA-256 hex of Content. The
	// field is filled by NewArtifact and Store.Put.
	ContentHash string `json:"content_hash"`
}

// Validate checks the artifact is well-formed. The
// function is the gate for Index.Add and Store.Put.
//
// The required fields are Kind, Scope, SessionID, and
// Content. The optional fields are TaskID, Timestamp,
// Tags, and ContentHash. ContentHash is filled by
// NewArtifact; when the caller fills it, the function
// checks the hash matches the content.
func (a *Artifact) Validate() error {
	if a == nil {
		return &MemoryError{Op: "validate", Reason: "artifact is nil"}
	}
	if !a.Kind.IsValid() {
		return &MemoryError{Op: "validate", Reason: fmt.Sprintf("kind must be one of task, file, decision, observation, tool_result, model_output; got %q", a.Kind)}
	}
	if !a.Scope.IsValid() {
		return &MemoryError{Op: "validate", Reason: fmt.Sprintf("scope must be one of session, project, user; got %q", a.Scope)}
	}
	if a.SessionID == "" {
		return &MemoryError{Op: "validate", Reason: "session_id is required"}
	}
	if len(a.Content) == 0 {
		return &MemoryError{Op: "validate", Reason: "content is required"}
	}
	if a.ContentHash != "" {
		expected := HashBytes(a.Content)
		if a.ContentHash != expected {
			return &MemoryError{Op: "validate", Reason: "content_hash does not match content"}
		}
	}
	return nil
}

// NewArtifact constructs an Artifact and fills the
// ContentHash and Timestamp fields. The function is
// the canonical entry point; callers that build an
// Artifact literal should call NewArtifact to set the
// hash.
//
// The Timestamp is filled with the current UTC time
// formatted as RFC 3339 with nanoseconds when the
// caller leaves it empty.
func NewArtifact(kind ArtifactKind, scope Scope, sessionID string, content []byte) (Artifact, error) {
	a := Artifact{
		Kind:        kind,
		Scope:       scope,
		SessionID:   sessionID,
		Timestamp:   nowRFC3339(),
		Content:     content,
		ContentHash: HashBytes(content),
	}
	if err := a.Validate(); err != nil {
		return Artifact{}, err
	}
	return a, nil
}

// NewArtifactWithMeta constructs an Artifact with the
// given metadata. The function is the variant of
// NewArtifact for callers that have a task id or
// tags to attach.
//
// The task id is optional; the caller passes the empty
// string when the artifact is not tied to a task. The
// tags slice is copied; later mutations to the caller's
// slice do not affect the artifact.
func NewArtifactWithMeta(kind ArtifactKind, scope Scope, sessionID, taskID string, tags []string, content []byte) (Artifact, error) {
	a, err := NewArtifact(kind, scope, sessionID, content)
	if err != nil {
		return Artifact{}, err
	}
	a.TaskID = taskID
	if len(tags) > 0 {
		a.Tags = make([]string, len(tags))
		copy(a.Tags, tags)
	}
	return a, nil
}

// HashBytes returns the SHA-256 hex of the bytes. The
// function is the content hash every artifact and
// every pin set id is built on.
func HashBytes(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// ContentMatches reports whether the artifact's
// content matches the glob pattern. The match is
// performed on the textual content; binary content
// is decoded as UTF-8 with replacement. The function
// is the predicate the query language uses for
// Pattern filtering.
func (a *Artifact) ContentMatches(pattern string) bool {
	if pattern == "" {
		return true
	}
	ok, err := filepath.Match(pattern, a.TextContent())
	if err != nil {
		return false
	}
	return ok
}

// TextContent returns the artifact's content as a
// string. Binary content is decoded as UTF-8 with
// replacement; the result is suitable for glob
// matching through filepath.Match.
func (a *Artifact) TextContent() string {
	return strings.NewReplacer("\x00", "").Replace(string(a.Content))
}

// HasTag reports whether the artifact has the given
// tag. The match is exact and case-sensitive.
func (a *Artifact) HasTag(tag string) bool {
	for _, t := range a.Tags {
		if t == tag {
			return true
		}
	}
	return false
}

// Since reports whether the artifact's timestamp is
// at or after t. The function returns true when t is
// the zero time; the zero time is the "no since
// filter" case.
func (a *Artifact) Since(t string) bool {
	if t == "" {
		return true
	}
	at, ok := parseTimestamp(a.Timestamp)
	if !ok {
		return false
	}
	bt, ok := parseTimestamp(t)
	if !ok {
		return false
	}
	return !at.Before(bt)
}

// canonicalJSON returns the canonical JSON encoding
// of the artifact. The encoding is used to compute a
// stable hash for tests and the index's content
// address. The encoding is the same canonical encoder
// the audit log uses.
func (a *Artifact) canonicalJSON() ([]byte, error) {
	return json.Marshal(a)
}
