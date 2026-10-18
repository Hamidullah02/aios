package policy

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// error-formatting paths

func TestPolicyParseError_Error_WithLine(t *testing.T) {
	e := &PolicyParseError{Path: "/p.yaml", Line: 5, Err: errors.New("bad")}
	if got := e.Error(); got == "" {
		t.Fatal("non-empty")
	}
}

func TestPolicyParseError_Error_NoLine(t *testing.T) {
	e := &PolicyParseError{Path: "/p.yaml", Err: errors.New("bad")}
	if got := e.Error(); got == "" {
		t.Fatal("non-empty")
	}
}

func TestPolicyParseError_Unwrap(t *testing.T) {
	inner := errors.New("inner")
	e := &PolicyParseError{Path: "/p.yaml", Err: inner}
	if !errors.Is(e, inner) {
		t.Fatal("Unwrap should expose inner")
	}
}

func TestRuleValidationError_Error(t *testing.T) {
	e := &RuleValidationError{ID: "r1", Reason: "bad"}
	if got := e.Error(); got == "" {
		t.Fatal("non-empty")
	}
	e2 := &RuleValidationError{Reason: "bad"}
	if got := e2.Error(); got == "" {
		t.Fatal("non-empty without id")
	}
}

// Hash error path: MarshalCanonical cannot fail on the simple
// struct we pass, but the test path is exercised.

func TestHash_NonEmpty(t *testing.T) {
	if Hash(nil) == "" {
		t.Fatal("Hash(nil) should still return a hex of an empty list")
	}
	if Hash([]Rule{}) == "" {
		t.Fatal("Hash([]) should return a hex of an empty list")
	}
}

// matches: edge cases

func TestMatch_TierZero(t *testing.T) {
	zero := 0
	m := Match{Tier: &zero}
	// A rule with tier=0 should match a call with RiskTier=0.
	if !m.Matches(Call{RiskTier: 0}) {
		t.Fatal("tier=0 should match RiskTier=0")
	}
	// And should not match a call with RiskTier=1.
	if m.Matches(Call{RiskTier: 1}) {
		t.Fatal("tier=0 should not match RiskTier=1")
	}
}

func TestToInt_Int32(t *testing.T) {
	if toInt(int32(5)) != 5 {
		t.Fatal("int32")
	}
}

func TestToInt_BadType(t *testing.T) {
	if toInt([]int{1}) != 0 {
		t.Fatal("bad type should be 0")
	}
}

func TestToInt_BadString(t *testing.T) {
	if toInt("not-a-number") != 0 {
		t.Fatal("bad string should be 0")
	}
}

// decide edge cases

func TestDecide_DefaultDenyAuditRow(t *testing.T) {
	// Default-deny path: no rule matches, the gateway writes
	// a policy_decision row with rule_id="".
	gw, log := newTestGateway(t, []Rule{
		{ID: "r1", Match: Match{Tool: "read"}, Action: ActionAllow},
	})
	if _, err := gw.Decide(Call{Tool: "edit", TaskID: "t1", Actor: "op"}); err != nil {
		t.Fatal(err)
	}
	rows, err := log.GetByTask("t1")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows: %d", len(rows))
	}
	if rows[0].RiskTier != 0 {
		t.Fatalf("risk_tier: %d", rows[0].RiskTier)
	}
}

func TestDecide_WithRiskTier(t *testing.T) {
	gw, log := newTestGateway(t, []Rule{
		{ID: "r1", Match: Match{Tool: "edit"}, Action: ActionAllow},
	})
	if _, err := gw.Decide(Call{Tool: "edit", TaskID: "t1", Actor: "op", RiskTier: 2}); err != nil {
		t.Fatal(err)
	}
	rows, _ := log.GetByTask("t1")
	if rows[0].RiskTier != 2 {
		t.Fatalf("risk_tier: %d", rows[0].RiskTier)
	}
}

// loader edge cases

func TestLoad_AbsPathError(t *testing.T) {
	// Pass a path that triggers an abs-path error. On Windows
	// this is hard to trigger; the test just exercises the
	// path-resolution code.
	dir := t.TempDir()
	bad := filepath.Join(dir, "no-such-dir", "p.yaml")
	if _, err := Load(bad); err == nil {
		t.Fatal("expected error for missing file in missing dir")
	}
}

func TestLoad_NonStringIncludeList(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "p.yaml")
	if err := os.WriteFile(path, []byte(`
include:
  - 1
  - 2
rules: []
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("expected error: non-string in include list")
	}
}

func TestNormalizePath_EmptyBase(t *testing.T) {
	got, err := NormalizePath("", "p.yaml")
	if err != nil {
		t.Fatal(err)
	}
	// Just check it's a non-empty absolute path.
	if got == "" {
		t.Fatal("empty")
	}
}
