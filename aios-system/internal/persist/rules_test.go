package persist

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// newTestStorage constructs a RuleStorage pointed at
// fresh temp files. The function returns the storage and
// a cleanup function the caller must defer.
func newTestStorage(t *testing.T) (*RuleStorage, func()) {
	t.Helper()
	dir := t.TempDir()
	userPath := filepath.Join(dir, "user.json")
	projectPath := filepath.Join(dir, "project.json")
	fixed := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	s, err := NewRuleStorage(RuleStorageConfig{
		UserPath:    userPath,
		ProjectPath: projectPath,
		Now:         func() time.Time { return fixed },
	})
	if err != nil {
		t.Fatalf("NewRuleStorage: %v", err)
	}
	return s, func() {}
}

// sampleRule returns a valid rule the tests can adapt.
func sampleRule(id string) Rule {
	return Rule{
		ID:     id,
		Match:  RuleMatch{Tool: "read_file"},
		Action: ActionAllow,
		Scope:  ScopeSession,
	}
}

// TestRuleActionIsValid covers the action enum.
func TestRuleActionIsValid(t *testing.T) {
	for _, a := range []RuleAction{ActionAllow, ActionDeny} {
		if !a.IsValid() {
			t.Fatalf("expected %q to be valid", a)
		}
	}
	for _, bad := range []RuleAction{"", "ask", "ALLOW", "Allow"} {
		if bad.IsValid() {
			t.Fatalf("expected %q to be invalid", bad)
		}
	}
}

// TestRuleScopeIsValid covers the scope enum.
func TestRuleScopeIsValid(t *testing.T) {
	for _, s := range []RuleScope{ScopeSession, ScopeProject, ScopeUser} {
		if !s.IsValid() {
			t.Fatalf("expected %q to be valid", s)
		}
	}
	for _, bad := range []RuleScope{"", "global", "USER"} {
		if bad.IsValid() {
			t.Fatalf("expected %q to be invalid", bad)
		}
	}
}

// TestRuleMatchMatchesCall covers the four match
// conditions.
func TestRuleMatchMatchesCall(t *testing.T) {
	m := RuleMatch{
		Tool:     "read_file",
		PathGlob: "*.go",
		HostGlob: "*.github.com",
		Actor:    "alice",
	}
	cases := []struct {
		name string
		call CallShape
		want bool
	}{
		{
			name: "all match",
			call: CallShape{Tool: "read_file", Path: "main.go", Host: "api.github.com", Actor: "alice"},
			want: true,
		},
		{
			name: "tool miss",
			call: CallShape{Tool: "write_file", Path: "main.go", Host: "api.github.com", Actor: "alice"},
			want: false,
		},
		{
			name: "path miss",
			call: CallShape{Tool: "read_file", Path: "main.txt", Host: "api.github.com", Actor: "alice"},
			want: false,
		},
		{
			name: "host miss",
			call: CallShape{Tool: "read_file", Path: "main.go", Host: "api.example.com", Actor: "alice"},
			want: false,
		},
		{
			name: "actor miss",
			call: CallShape{Tool: "read_file", Path: "main.go", Host: "api.github.com", Actor: "bob"},
			want: false,
		},
		{
			name: "glob syntax error on path",
			call: CallShape{Tool: "read_file", Path: "main.go", Host: "api.github.com", Actor: "alice"},
			want: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := m.MatchesCall(tc.call); got != tc.want {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
}

// TestRuleMatchEmpty covers the empty-match branch. An
// empty match is "match everything", but Validate rejects
// it because the storage requires at least one
// condition.
func TestRuleMatchEmpty(t *testing.T) {
	var m RuleMatch
	if !m.MatchesCall(CallShape{Tool: "anything"}) {
		t.Fatalf("empty match should match everything")
	}
	if m.HasAny() {
		t.Fatalf("empty match should not have any condition")
	}
}

// TestRuleValidate covers the validation rules.
func TestRuleValidate(t *testing.T) {
	cases := []struct {
		name    string
		rule    Rule
		wantErr bool
	}{
		{
			name:    "valid",
			rule:    sampleRule("r1"),
			wantErr: false,
		},
		{
			name:    "missing id",
			rule:    Rule{Match: RuleMatch{Tool: "x"}, Action: ActionAllow, Scope: ScopeSession},
			wantErr: true,
		},
		{
			name:    "bad action",
			rule:    Rule{ID: "r1", Match: RuleMatch{Tool: "x"}, Action: RuleAction("ask"), Scope: ScopeSession},
			wantErr: true,
		},
		{
			name:    "bad scope",
			rule:    Rule{ID: "r1", Match: RuleMatch{Tool: "x"}, Action: ActionAllow, Scope: RuleScope("global")},
			wantErr: true,
		},
		{
			name:    "empty match",
			rule:    Rule{ID: "r1", Action: ActionAllow, Scope: ScopeSession},
			wantErr: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.rule.Validate()
			if tc.wantErr && err == nil {
				t.Fatalf("expected error")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

// TestRuleContentHash covers the content-address hash.
// The hash is deterministic for the same JSON.
func TestRuleContentHash(t *testing.T) {
	r1 := sampleRule("r1")
	r2 := sampleRule("r1")
	if r1.ContentHash() != r2.ContentHash() {
		t.Fatalf("same rule should hash the same")
	}
	r3 := sampleRule("r2")
	if r1.ContentHash() == r3.ContentHash() {
		t.Fatalf("different rule should hash differently")
	}
}

// TestRuleStorageAddSession covers the session-scope add.
// The rule is kept in memory and not written to disk.
func TestRuleStorageAddSession(t *testing.T) {
	s, cleanup := newTestStorage(t)
	defer cleanup()

	r := sampleRule("r1")
	r.Scope = ScopeSession
	if err := s.Add(r); err != nil {
		t.Fatalf("Add: %v", err)
	}
	list := s.List()
	if len(list) != 1 || list[0].ID != "r1" {
		t.Fatalf("expected one rule, got %+v", list)
	}
	if list[0].CreatedAt == "" {
		t.Fatalf("CreatedAt should be filled")
	}
	// The on-disk files should still be empty.
	if _, err := os.Stat(s.projectPath); err == nil {
		t.Fatalf("session rule should not write project file")
	}
}

// TestRuleStorageAddProject covers the project-scope
// add. The rule is appended and persisted to the project
// file.
func TestRuleStorageAddProject(t *testing.T) {
	s, cleanup := newTestStorage(t)
	defer cleanup()

	r := sampleRule("r1")
	r.Scope = ScopeProject
	if err := s.Add(r); err != nil {
		t.Fatalf("Add: %v", err)
	}
	raw, err := os.ReadFile(s.projectPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !strings.Contains(string(raw), "r1") {
		t.Fatalf("expected project file to contain r1, got %q", raw)
	}
}

// TestRuleStorageAddUser covers the user-scope add. The
// rule is appended and persisted to the user file.
func TestRuleStorageAddUser(t *testing.T) {
	s, cleanup := newTestStorage(t)
	defer cleanup()

	r := sampleRule("r1")
	r.Scope = ScopeUser
	if err := s.Add(r); err != nil {
		t.Fatalf("Add: %v", err)
	}
	raw, err := os.ReadFile(s.userPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !strings.Contains(string(raw), "r1") {
		t.Fatalf("expected user file to contain r1, got %q", raw)
	}
}

// TestRuleStorageAddDuplicate covers the duplicate-id
// branch. The storage rejects a second rule with the same
// id in the same scope.
func TestRuleStorageAddDuplicate(t *testing.T) {
	s, cleanup := newTestStorage(t)
	defer cleanup()

	r := sampleRule("r1")
	if err := s.Add(r); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := s.Add(r); err == nil {
		t.Fatalf("expected duplicate error")
	}
}

// TestRuleStorageRemoveSession covers the remove path
// for a session rule. The rule is dropped from memory
// and no file is touched.
func TestRuleStorageRemoveSession(t *testing.T) {
	s, cleanup := newTestStorage(t)
	defer cleanup()

	r := sampleRule("r1")
	if err := s.Add(r); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := s.Remove("r1"); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if got := s.List(); len(got) != 0 {
		t.Fatalf("expected empty, got %+v", got)
	}
}

// TestRuleStorageRemoveProject covers the remove path
// for a project rule. The project file is rewritten.
func TestRuleStorageRemoveProject(t *testing.T) {
	s, cleanup := newTestStorage(t)
	defer cleanup()

	if err := s.Add(Rule{ID: "r1", Match: RuleMatch{Tool: "x"}, Action: ActionAllow, Scope: ScopeProject}); err != nil {
		t.Fatalf("Add r1: %v", err)
	}
	if err := s.Add(Rule{ID: "r2", Match: RuleMatch{Tool: "y"}, Action: ActionDeny, Scope: ScopeProject}); err != nil {
		t.Fatalf("Add r2: %v", err)
	}
	if err := s.Remove("r1"); err != nil {
		t.Fatalf("Remove r1: %v", err)
	}
	raw, err := os.ReadFile(s.projectPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if strings.Contains(string(raw), "r1") {
		t.Fatalf("expected project file to not contain r1")
	}
	if !strings.Contains(string(raw), "r2") {
		t.Fatalf("expected project file to still contain r2")
	}
}

// TestRuleStorageRemoveNotFound covers the not-found
// branch. The function returns a *RuleNotFoundError
// that errors.Is matches.
func TestRuleStorageRemoveNotFound(t *testing.T) {
	s, cleanup := newTestStorage(t)
	defer cleanup()

	err := s.Remove("missing")
	if err == nil {
		t.Fatalf("expected error")
	}
	if !errors.Is(err, &RuleNotFoundError{}) {
		t.Fatalf("expected *RuleNotFoundError, got %T", err)
	}
}

// TestRuleStorageList covers the list path. The function
// returns a copy, so mutating the returned slice does not
// affect the storage.
func TestRuleStorageList(t *testing.T) {
	s, cleanup := newTestStorage(t)
	defer cleanup()

	for _, id := range []string{"r2", "r1", "r3"} {
		if err := s.Add(sampleRule(id)); err != nil {
			t.Fatalf("Add %s: %v", id, err)
		}
	}
	list := s.List()
	ids := make([]string, len(list))
	for i, r := range list {
		ids[i] = r.ID
	}
	want := []string{"r1", "r2", "r3"}
	if !equalStrings(ids, want) {
		t.Fatalf("expected sorted ids, got %v", ids)
	}
	// Mutate the returned slice; the storage must not see it.
	list[0].ID = "mutated"
	if s.List()[0].ID != "r1" {
		t.Fatalf("storage leaked through List")
	}
}

// TestRuleStorageMatch covers the match path. The first
// matching rule wins.
func TestRuleStorageMatch(t *testing.T) {
	s, cleanup := newTestStorage(t)
	defer cleanup()

	if err := s.Add(Rule{ID: "r1", Match: RuleMatch{Tool: "read_file"}, Action: ActionAllow, Scope: ScopeSession}); err != nil {
		t.Fatalf("Add r1: %v", err)
	}
	if err := s.Add(Rule{ID: "r2", Match: RuleMatch{Tool: "write_file"}, Action: ActionDeny, Scope: ScopeSession}); err != nil {
		t.Fatalf("Add r2: %v", err)
	}
	if got := s.Match(CallShape{Tool: "read_file"}); got == nil || got.ID != "r1" {
		t.Fatalf("expected r1, got %+v", got)
	}
	if got := s.Match(CallShape{Tool: "write_file"}); got == nil || got.ID != "r2" {
		t.Fatalf("expected r2, got %+v", got)
	}
	if got := s.Match(CallShape{Tool: "missing"}); got != nil {
		t.Fatalf("expected nil, got %+v", got)
	}
}

// TestRuleStorageProjectOverridesUser covers the
// project-overrides-user merge. The project rule with the
// same id replaces the user rule.
func TestRuleStorageProjectOverridesUser(t *testing.T) {
	s, cleanup := newTestStorage(t)
	defer cleanup()

	// Seed the user file with a rule.
	userRule := Rule{
		ID:     "r1",
		Match:  RuleMatch{Tool: "read_file"},
		Action: ActionDeny,
		Scope:  ScopeUser,
	}
	raw, err := json.MarshalIndent([]Rule{userRule}, "", "  ")
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if err := os.WriteFile(s.userPath, raw, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := s.Reload(); err != nil {
		t.Fatalf("Reload: %v", err)
	}
	if got := s.Match(CallShape{Tool: "read_file"}); got == nil || got.Action != ActionDeny {
		t.Fatalf("expected user rule, got %+v", got)
	}
	// Add a project rule with the same id but different action.
	projectRule := Rule{
		ID:     "r1",
		Match:  RuleMatch{Tool: "read_file"},
		Action: ActionAllow,
		Scope:  ScopeProject,
	}
	if err := s.Add(projectRule); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if got := s.Match(CallShape{Tool: "read_file"}); got == nil || got.Action != ActionAllow {
		t.Fatalf("expected project override, got %+v", got)
	}
}

// TestRuleStorageSessionOverridesProject covers the
// session-overrides-project merge.
func TestRuleStorageSessionOverridesProject(t *testing.T) {
	s, cleanup := newTestStorage(t)
	defer cleanup()

	// Seed the project file with a deny rule.
	projectRule := Rule{
		ID:     "r1",
		Match:  RuleMatch{Tool: "read_file"},
		Action: ActionDeny,
		Scope:  ScopeProject,
	}
	if err := s.Add(projectRule); err != nil {
		t.Fatalf("Add project: %v", err)
	}
	if got := s.Match(CallShape{Tool: "read_file"}); got == nil || got.Action != ActionDeny {
		t.Fatalf("expected deny, got %+v", got)
	}
	// Add a session rule with the same id but allow.
	sessionRule := Rule{
		ID:     "r1",
		Match:  RuleMatch{Tool: "read_file"},
		Action: ActionAllow,
		Scope:  ScopeSession,
	}
	if err := s.Add(sessionRule); err != nil {
		t.Fatalf("Add session: %v", err)
	}
	if got := s.Match(CallShape{Tool: "read_file"}); got == nil || got.Action != ActionAllow {
		t.Fatalf("expected session override, got %+v", got)
	}
}

// TestRuleStorageLoadInvalid covers the load branch
// where the file is corrupt. The constructor returns a
// RuleError.
func TestRuleStorageLoadInvalid(t *testing.T) {
	dir := t.TempDir()
	userPath := filepath.Join(dir, "user.json")
	projectPath := filepath.Join(dir, "project.json")
	if err := os.WriteFile(userPath, []byte("not json"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	_, err := NewRuleStorage(RuleStorageConfig{
		UserPath:    userPath,
		ProjectPath: projectPath,
	})
	if err == nil {
		t.Fatalf("expected error for invalid JSON")
	}
	var re *RuleError
	if !errors.As(err, &re) {
		t.Fatalf("expected *RuleError, got %T", err)
	}
}

// TestRuleStorageLoadInvalidRule covers the load branch
// where the file is valid JSON but a rule fails
// validation.
func TestRuleStorageLoadInvalidRule(t *testing.T) {
	dir := t.TempDir()
	userPath := filepath.Join(dir, "user.json")
	projectPath := filepath.Join(dir, "project.json")
	raw := `[{"id":"","match":{"tool":"x"},"action":"allow","scope":"user"}]`
	if err := os.WriteFile(userPath, []byte(raw), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	_, err := NewRuleStorage(RuleStorageConfig{
		UserPath:    userPath,
		ProjectPath: projectPath,
	})
	if err == nil {
		t.Fatalf("expected error for invalid rule")
	}
}

// TestRuleStorageReload covers the reload path.
func TestRuleStorageReload(t *testing.T) {
	s, cleanup := newTestStorage(t)
	defer cleanup()

	// Write a rule directly to the user file, then reload.
	userRule := Rule{
		ID:     "r1",
		Match:  RuleMatch{Tool: "read_file"},
		Action: ActionAllow,
		Scope:  ScopeUser,
	}
	raw, err := json.MarshalIndent([]Rule{userRule}, "", "  ")
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if err := os.WriteFile(s.userPath, raw, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := s.Reload(); err != nil {
		t.Fatalf("Reload: %v", err)
	}
	if got := s.Match(CallShape{Tool: "read_file"}); got == nil || got.ID != "r1" {
		t.Fatalf("expected r1 after reload, got %+v", got)
	}
}

// TestRuleStorageAccessors covers the path accessors.
func TestRuleStorageAccessors(t *testing.T) {
	s, cleanup := newTestStorage(t)
	defer cleanup()
	if s.UserPath() == "" {
		t.Fatalf("expected non-empty user path")
	}
	if s.ProjectPath() == "" {
		t.Fatalf("expected non-empty project path")
	}
}

// TestRuleNotFoundErrorIs covers the errors.Is match for
// the typed error.
func TestRuleNotFoundErrorIs(t *testing.T) {
	var err error = &RuleNotFoundError{ID: "x"}
	if !errors.Is(err, &RuleNotFoundError{}) {
		t.Fatalf("expected errors.Is to match")
	}
	if msg := err.Error(); !strings.Contains(msg, "x") {
		t.Fatalf("expected id in message: %q", msg)
	}
}

// TestRuleErrorMessage covers the error format.
func TestRuleErrorMessage(t *testing.T) {
	err := (&RuleError{Op: "add", Reason: "boom"}).Error()
	if !strings.HasPrefix(err, "persist: rules:") {
		t.Fatalf("unexpected prefix: %q", err)
	}
	if !strings.HasSuffix(err, ".") {
		t.Fatalf("missing trailing period: %q", err)
	}
}

// TestDefaultUserRulesPath covers the default-path
// resolver.
func TestDefaultUserRulesPath(t *testing.T) {
	t.Setenv("AIOS_HOME", "")
	t.Setenv("HOME", "")
	t.Setenv("USERPROFILE", "")
	got := defaultUserRulesPath()
	if got == "" {
		t.Fatalf("expected non-empty path")
	}
	t.Setenv("AIOS_HOME", "/custom/home")
	if got := defaultUserRulesPath(); got != filepath.Join("/custom/home", "rules.json") {
		t.Fatalf("AIOS_HOME not honored: %q", got)
	}
}

// TestDefaultProjectRulesPath covers the default-path
// resolver.
func TestDefaultProjectRulesPath(t *testing.T) {
	t.Setenv("AIOS_DATA_DIR", "")
	got := defaultProjectRulesPath()
	if got == "" {
		t.Fatalf("expected non-empty path")
	}
	t.Setenv("AIOS_DATA_DIR", "/custom/data")
	if got := defaultProjectRulesPath(); got != filepath.Join("/custom/data", "rules.json") {
		t.Fatalf("AIOS_DATA_DIR not honored: %q", got)
	}
}

// equalStrings reports whether two slices contain the
// same elements in the same order.
func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// TestRuleStorageAtomicWrite covers the atomic-write
// path. After the write, the file must be valid JSON
// even if the parent dir was missing.
func TestRuleStorageAtomicWrite(t *testing.T) {
	dir := t.TempDir()
	userPath := filepath.Join(dir, "nested", "user.json")
	projectPath := filepath.Join(dir, "nested", "project.json")
	s, err := NewRuleStorage(RuleStorageConfig{
		UserPath:    userPath,
		ProjectPath: projectPath,
	})
	if err != nil {
		t.Fatalf("NewRuleStorage: %v", err)
	}
	r := sampleRule("r1")
	r.Scope = ScopeProject
	if err := s.Add(r); err != nil {
		t.Fatalf("Add: %v", err)
	}
	raw, err := os.ReadFile(projectPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var got []Rule
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if len(got) != 1 || got[0].ID != "r1" {
		t.Fatalf("unexpected file content: %+v", got)
	}
}
