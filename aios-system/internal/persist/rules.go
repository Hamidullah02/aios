package persist

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// RuleAction is the rule's verdict for a tool call. The
// action is one of allow or deny. The persist module does
// not have an "ask" action: the always-allow rules in this
// package are either allow or deny, not the three-state
// allow/deny/ask of aios-policy.
type RuleAction string

const (
	ActionAllow RuleAction = "allow"
	ActionDeny  RuleAction = "deny"
)

// IsValid reports whether a is one of the two known
// actions.
func (a RuleAction) IsValid() bool {
	switch a {
	case ActionAllow, ActionDeny:
		return true
	}
	return false
}

// RuleScope is the persistence scope of a rule. The
// scope controls where the rule is stored and when the
// rule expires.
type RuleScope string

const (
	// ScopeSession is the in-memory scope. Session rules
	// are stored in memory and discarded on exit.
	ScopeSession RuleScope = "session"
	// ScopeProject is the project-level scope. Project
	// rules are stored in $AIOS_DATA_DIR/rules.json and
	// shared across sessions.
	ScopeProject RuleScope = "project"
	// ScopeUser is the user-level scope. User rules are
	// stored in $AIOS_HOME/rules.json and shared across
	// projects.
	ScopeUser RuleScope = "user"
)

// IsValid reports whether s is one of the three known
// scopes.
func (s RuleScope) IsValid() bool {
	switch s {
	case ScopeSession, ScopeProject, ScopeUser:
		return true
	}
	return false
}

// RuleMatch is the typed match conditions on a rule.
//
// Per REQ-PRS-RSC-002 the match carries a tool (exact), a
// path_glob (glob), a host_glob (glob), and an actor
// (exact). An empty condition is "match anything" for
// that condition; an entirely empty match matches every
// call.
type RuleMatch struct {
	Tool     string `json:"tool,omitempty"`
	PathGlob string `json:"path_glob,omitempty"`
	HostGlob string `json:"host_glob,omitempty"`
	Actor    string `json:"actor,omitempty"`
}

// MatchesCall reports whether the rule match is satisfied
// by the call. The function takes a Call-shaped struct
// with the four fields the rule cares about: Tool, Path,
// Host, Actor.
//
// An empty match matches every call.
func (m RuleMatch) MatchesCall(c CallShape) bool {
	if m.Tool != "" && m.Tool != c.Tool {
		return false
	}
	if m.PathGlob != "" {
		ok, err := filepath.Match(m.PathGlob, c.Path)
		if err != nil || !ok {
			return false
		}
	}
	if m.HostGlob != "" {
		ok, err := filepath.Match(m.HostGlob, c.Host)
		if err != nil || !ok {
			return false
		}
	}
	if m.Actor != "" && m.Actor != c.Actor {
		return false
	}
	return true
}

// HasAny reports whether at least one condition is set.
// The function is used by Validate to require at least
// one non-empty condition; an empty match would silently
// allow every call.
func (m RuleMatch) HasAny() bool {
	return m.Tool != "" || m.PathGlob != "" || m.HostGlob != "" || m.Actor != ""
}

// CallShape is the minimal call context the rule match
// needs. The persist package defines its own type so the
// match does not depend on aios-policy.
//
// Fields:
//
//   - Tool:  the tool name (e.g. "read_file").
//   - Path:  the resource path (e.g. "internal/core/foo.go").
//   - Host:  the URL host (e.g. "api.github.com").
//   - Actor: the entity that issued the call.
type CallShape struct {
	Tool  string
	Path  string
	Host  string
	Actor string
}

// Rule is the typed object that holds the rule definition.
//
// Per REQ-PRS-RSC-001 a rule has an id, a match, an
// action, a scope, and a created_at timestamp. The id is
// unique within the storage. The action is allow or deny
// (REQ-PRS-RSC-003). The scope is session, project, or
// user (REQ-PRS-RSC-004).
type Rule struct {
	ID        string     `json:"id"`
	Match     RuleMatch  `json:"match"`
	Action    RuleAction `json:"action"`
	Scope     RuleScope  `json:"scope"`
	CreatedAt string     `json:"created_at"`
}

// Validate checks the rule is well-formed.
func (r *Rule) Validate() error {
	if r == nil {
		return &RuleError{Op: "validate", Reason: "rule is nil"}
	}
	if r.ID == "" {
		return &RuleError{Op: "validate", Reason: "id is required"}
	}
	if !r.Action.IsValid() {
		return &RuleError{Op: "validate", Reason: fmt.Sprintf("action must be allow or deny; got %q", r.Action)}
	}
	if !r.Scope.IsValid() {
		return &RuleError{Op: "validate", Reason: fmt.Sprintf("scope must be session, project, or user; got %q", r.Scope)}
	}
	if !r.Match.HasAny() {
		return &RuleError{Op: "validate", Reason: "match must have at least one condition"}
	}
	return nil
}

// ContentHash returns the SHA-256 of the rule's
// canonical-JSON form. The hash is the rule's content
// address; the storage uses it for dedup and for the
// stable listing order.
func (r *Rule) ContentHash() string {
	raw, err := json.Marshal(r)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

// RuleError is the typed error for rule storage. The
// error message includes the failing operation and the
// reason.
type RuleError struct {
	Op     string
	Reason string
}

func (e *RuleError) Error() string {
	return fmt.Sprintf("persist: rules: %s: %s.", e.Op, e.Reason)
}

// RuleNotFoundError is raised by Remove when the rule id
// is not in the storage (REQ-PRS-ERR-001).
type RuleNotFoundError struct {
	ID string
}

func (e *RuleNotFoundError) Error() string {
	return fmt.Sprintf("persist: rules: rule %q not found.", e.ID)
}

// Is reports whether target is also a RuleNotFoundError.
// The method lets callers use errors.Is to detect a
// missing rule regardless of the concrete value.
func (e *RuleNotFoundError) Is(target error) bool {
	_, ok := target.(*RuleNotFoundError)
	return ok
}

// RuleStorage is the always-allow rule storage.
//
// The storage loads the user-level rules from
// $AIOS_HOME/rules.json and the project-level rules from
// $AIOS_DATA_DIR/rules.json. The project rules override
// the user rules (REQ-PRS-RUL-006). The session rules
// are kept in memory and discarded on exit (TRD §6.1).
//
// The storage is safe for concurrent use by multiple
// goroutines because the rule table is guarded by a
// mutex.
type RuleStorage struct {
	mu          sync.RWMutex
	userPath    string
	projectPath string
	user        []Rule
	project     []Rule
	session     []Rule
	now         func() time.Time
}

// RuleStorageConfig controls RuleStorage construction.
type RuleStorageConfig struct {
	// UserPath is the path to the user-level rules file.
	// Defaults to $AIOS_HOME/rules.json when empty.
	UserPath string
	// ProjectPath is the path to the project-level rules
	// file. Defaults to $AIOS_DATA_DIR/rules.json when
	// empty.
	ProjectPath string
	// Now is the decision clock. The clock is used for the
	// CreatedAt field when the caller leaves it zero on
	// Add. Defaults to time.Now when nil.
	Now func() time.Time
}

// NewRuleStorage constructs a RuleStorage and loads the
// on-disk rule files. The session rule table starts
// empty. A missing on-disk file is treated as "no rules",
// not as an error.
//
// The function honors the AIOS_HOME and AIOS_DATA_DIR
// environment variables for the default paths. When the
// caller provides explicit paths, those take precedence.
func NewRuleStorage(cfg RuleStorageConfig) (*RuleStorage, error) {
	userPath := cfg.UserPath
	if userPath == "" {
		userPath = defaultUserRulesPath()
	}
	projectPath := cfg.ProjectPath
	if projectPath == "" {
		projectPath = defaultProjectRulesPath()
	}
	now := cfg.Now
	if now == nil {
		now = time.Now
	}
	s := &RuleStorage{
		userPath:    userPath,
		projectPath: projectPath,
		now:         now,
	}
	if err := s.loadFile(userPath, &s.user); err != nil {
		return nil, &RuleError{Op: "load user rules", Reason: err.Error()}
	}
	if err := s.loadFile(projectPath, &s.project); err != nil {
		return nil, &RuleError{Op: "load project rules", Reason: err.Error()}
	}
	return s, nil
}

// Add appends a rule to the storage. The rule's scope
// determines where the rule is written: a user or
// project rule is persisted to the corresponding file; a
// session rule is kept in memory.
//
// Add validates the rule. The rule's ID must be unique
// within the storage. The rule's CreatedAt is filled
// from the injected clock when zero.
//
// REQ-PRS-RUL-002.
func (s *RuleStorage) Add(r Rule) error {
	if err := r.Validate(); err != nil {
		return err
	}
	if r.CreatedAt == "" {
		r.CreatedAt = s.now().UTC().Format(time.RFC3339Nano)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	switch r.Scope {
	case ScopeSession:
		if s.indexLocked(s.session, r.ID) >= 0 {
			return &RuleError{Op: "add", Reason: fmt.Sprintf("session rule %q already exists", r.ID)}
		}
		s.session = append(s.session, r)
		return nil
	case ScopeProject:
		if s.indexLocked(s.project, r.ID) >= 0 {
			return &RuleError{Op: "add", Reason: fmt.Sprintf("project rule %q already exists", r.ID)}
		}
		s.project = append(s.project, r)
		return s.writeFile(s.projectPath, s.project)
	case ScopeUser:
		if s.indexLocked(s.user, r.ID) >= 0 {
			return &RuleError{Op: "add", Reason: fmt.Sprintf("user rule %q already exists", r.ID)}
		}
		s.user = append(s.user, r)
		return s.writeFile(s.userPath, s.user)
	}
	return &RuleError{Op: "add", Reason: fmt.Sprintf("unknown scope %q", r.Scope)}
}

// Remove deletes a rule from the storage by id. The
// operation looks in all three scopes. When the rule is
// in the user or project scope, the corresponding file
// is rewritten.
//
// REQ-PRS-RUL-003.
func (s *RuleStorage) Remove(id string) error {
	if id == "" {
		return &RuleError{Op: "remove", Reason: "id is required"}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if i := s.indexLocked(s.session, id); i >= 0 {
		s.session = append(s.session[:i], s.session[i+1:]...)
		return nil
	}
	if i := s.indexLocked(s.project, id); i >= 0 {
		s.project = append(s.project[:i], s.project[i+1:]...)
		return s.writeFile(s.projectPath, s.project)
	}
	if i := s.indexLocked(s.user, id); i >= 0 {
		s.user = append(s.user[:i], s.user[i+1:]...)
		return s.writeFile(s.userPath, s.user)
	}
	return &RuleNotFoundError{ID: id}
}

// List returns a copy of the merged rule table. The
// merged table is the concatenation of user, project,
// and session rules in that order, with the project's
// duplicate ids overriding the user's and the session's
// overriding both.
//
// REQ-PRS-RUL-004.
func (s *RuleStorage) List() []Rule {
	s.mu.RLock()
	defer s.mu.RUnlock()
	merged := s.mergedLocked()
	out := make([]Rule, len(merged))
	copy(out, merged)
	return out
}

// Match returns the first rule that matches the call.
// The function iterates the merged table in priority
// order: user, then project, then session. The first
// matching rule wins.
//
// When no rule matches, the function returns nil.
//
// REQ-PRS-RUL-005.
func (s *RuleStorage) Match(c CallShape) *Rule {
	s.mu.RLock()
	defer s.mu.RUnlock()
	merged := s.mergedLocked()
	for i := range merged {
		if merged[i].Match.MatchesCall(c) {
			r := merged[i]
			return &r
		}
	}
	return nil
}

// Reload re-reads the on-disk rule files. The operation
// is used by callers that want to pick up external edits
// without restarting the process. The in-memory session
// rules are preserved.
func (s *RuleStorage) Reload() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.loadFile(s.userPath, &s.user); err != nil {
		return &RuleError{Op: "reload user rules", Reason: err.Error()}
	}
	if err := s.loadFile(s.projectPath, &s.project); err != nil {
		return &RuleError{Op: "reload project rules", Reason: err.Error()}
	}
	return nil
}

// UserPath returns the user-level rules file path. The
// path is set at construction.
func (s *RuleStorage) UserPath() string { return s.userPath }

// ProjectPath returns the project-level rules file path.
// The path is set at construction.
func (s *RuleStorage) ProjectPath() string { return s.projectPath }

// mergedLocked returns the merged rule table. The merge
// is layered: session overrides project, project
// overrides user. The function is called with the
// storage's mutex held.
func (s *RuleStorage) mergedLocked() []Rule {
	byID := make(map[string]Rule, len(s.user)+len(s.project)+len(s.session))
	order := make([]string, 0, len(s.user)+len(s.project)+len(s.session))
	addAll := func(rules []Rule) {
		for _, r := range rules {
			if _, ok := byID[r.ID]; !ok {
				order = append(order, r.ID)
			}
			byID[r.ID] = r
		}
	}
	addAll(s.user)
	addAll(s.project)
	addAll(s.session)
	out := make([]Rule, 0, len(order))
	for _, id := range order {
		out = append(out, byID[id])
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// indexLocked returns the index of id in rules, or -1.
// The function is called with the storage's mutex held.
func (s *RuleStorage) indexLocked(rules []Rule, id string) int {
	for i, r := range rules {
		if r.ID == id {
			return i
		}
	}
	return -1
}

// loadFile reads a rules file into *rules. A missing
// file is treated as "no rules". A file with invalid
// JSON is returned as a RuleError.
func (s *RuleStorage) loadFile(path string, rules *[]Rule) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			*rules = nil
			return nil
		}
		return err
	}
	if len(raw) == 0 {
		*rules = nil
		return nil
	}
	var list []Rule
	if err := json.Unmarshal(raw, &list); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	for i := range list {
		if err := list[i].Validate(); err != nil {
			return fmt.Errorf("%s: rule %d: %w", path, i, err)
		}
	}
	*rules = list
	return nil
}

// writeFile writes the rules to path as a JSON array.
// The function writes to a temporary file in the same
// directory and renames atomically. The parent directory
// is created if missing.
func (s *RuleStorage) writeFile(path string, rules []Rule) error {
	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	raw, err := json.MarshalIndent(rules, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op if rename succeeded
	if _, err := tmp.Write(raw); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		return err
	}
	return nil
}

// defaultUserRulesPath returns the user-level rules file
// path. The path is $AIOS_HOME/rules.json when set, or
// $HOME/.aios/rules.json otherwise.
func defaultUserRulesPath() string {
	if home := os.Getenv("AIOS_HOME"); home != "" {
		return filepath.Join(home, "rules.json")
	}
	if home := os.Getenv("HOME"); home != "" {
		return filepath.Join(home, ".aios", "rules.json")
	}
	if home := os.Getenv("USERPROFILE"); home != "" {
		return filepath.Join(home, ".aios", "rules.json")
	}
	return filepath.Join(".aios", "rules.json")
}

// defaultProjectRulesPath returns the project-level rules
// file path. The path is $AIOS_DATA_DIR/rules.json when
// set, or ./var/rules.json otherwise.
func defaultProjectRulesPath() string {
	if dir := os.Getenv("AIOS_DATA_DIR"); dir != "" {
		return filepath.Join(dir, "rules.json")
	}
	return filepath.Join("var", "rules.json")
}
