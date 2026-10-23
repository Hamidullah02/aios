package cli

import (
	"bytes"
	"flag"
	"testing"
)

// --- flag hoisting (v0.41.2) ---
//
// TestHoistFlags_SeparatedValueStaysWithItsFlag is the
// regression gate for the second defect the first fix
// introduced.
//
// Hoisting naively split "--plan p.yaml" into the flag group
// and the positional group, which left PlanPath set to the
// NEXT flag's name:
//
//	aios resume --plan p.yaml --task t1
//	              PlanPath="--task"  TaskID="p.yaml"
//
// The value has to travel with its flag. This also covers the
// boolean case: a boolean flag must never swallow the
// argument after it.
func TestHoistFlags_SeparatedValueStaysWithItsFlag(t *testing.T) {
	a, err := Parse([]string{"aios", "resume", "--plan", "p.yaml", "--task", "t1"}, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if a.PlanPath != "p.yaml" {
		t.Errorf("PlanPath = %q, want p.yaml", a.PlanPath)
	}
	if a.TaskID != "t1" {
		t.Errorf("TaskID = %q, want t1", a.TaskID)
	}
}

// TestHoistFlags_BooleanFlagDoesNotSwallowNextArg guards the
// other half of the arity rule: --web is a bool, so the
// argument after it is a positional, not its value.
func TestHoistFlags_BooleanFlagDoesNotSwallowNextArg(t *testing.T) {
	a, err := Parse([]string{"aios", "run", "--web", "plan.yaml"}, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if !a.WebLive {
		t.Error("WebLive = false, want true")
	}
	if a.PlanPath != "plan.yaml" {
		t.Errorf("PlanPath = %q, want plan.yaml: a bool flag consumed the plan path", a.PlanPath)
	}
}

// TestHoistFlags_EqualsFormAndPositional covers both spellings
// of a value in one invocation.
func TestHoistFlags_EqualsFormAndPositional(t *testing.T) {
	a, err := Parse([]string{
		"aios", "resume", "--plan=p.yaml", "--task", "t1",
	}, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if a.PlanPath != "p.yaml" {
		t.Errorf("PlanPath = %q, want p.yaml", a.PlanPath)
	}
	if a.TaskID != "t1" {
		t.Errorf("TaskID = %q, want t1", a.TaskID)
	}
}

// TestHoistFlags_SeparatedValueAfterPositional is the
// combination that a naive fix gets wrong: a separated value
// flag written AFTER the positional it follows.
func TestHoistFlags_SeparatedValueAfterPositional(t *testing.T) {
	a, err := Parse([]string{"aios", "run", "plan.yaml", "--web-addr", "127.0.0.1:8123"}, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if a.PlanPath != "plan.yaml" {
		t.Errorf("PlanPath = %q, want plan.yaml", a.PlanPath)
	}
	if a.WebAddr != "127.0.0.1:8123" {
		t.Errorf("WebAddr = %q, want 127.0.0.1:8123", a.WebAddr)
	}
}

// TestValueFlagNames_ExcludesBools asserts the arity map is
// derived correctly, since a wrong answer here silently eats a
// positional on every invocation.
func TestValueFlagNames_ExcludesBools(t *testing.T) {
	fs := flag.NewFlagSet("t", flag.ContinueOnError)
	var s string
	var b bool
	fs.StringVar(&s, "plan", "", "")
	fs.BoolVar(&b, "web", false, "")
	got := valueFlagNames(fs)
	if !got["plan"] {
		t.Error("string flag `plan` not marked as taking a value")
	}
	if got["web"] {
		t.Error("bool flag `web` must not be marked as taking a value")
	}
}

// The defect: Go's flag.Parse stops at the first non-flag
// argument, so `aios run plan.yaml --web` left --web as a
// positional and never set the field. Every flag on every
// subcommand silently required the awkward
// `aios run --web plan.yaml` order, which is the opposite of
// what the usage text advertises and of what anyone types.
//
// These tests pin BOTH orders, because the whole point of
// the fix is that neither is privileged.

// TestHoistFlags_FlagAfterPositional is the regression gate
// for the reported bug.
func TestHoistFlags_FlagAfterPositional(t *testing.T) {
	a, err := Parse([]string{"aios", "run", "plan.yaml", "--web"}, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if !a.WebLive {
		t.Error("WebLive = false, want true: a flag after the positional was ignored")
	}
	if a.PlanPath != "plan.yaml" {
		t.Errorf("PlanPath = %q, want plan.yaml", a.PlanPath)
	}
}

// TestHoistFlags_FlagBeforePositional is the order that
// always worked; the fix must not regress it.
func TestHoistFlags_FlagBeforePositional(t *testing.T) {
	a, err := Parse([]string{"aios", "run", "--web", "plan.yaml"}, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if !a.WebLive {
		t.Error("WebLive = false, want true")
	}
	if a.PlanPath != "plan.yaml" {
		t.Errorf("PlanPath = %q, want plan.yaml", a.PlanPath)
	}
}

// TestHoistFlags_MixedOrder covers flags on both sides of
// the positional, which is what an operator types when
// combining --web with an existing habit of putting --format
// last.
func TestHoistFlags_MixedOrder(t *testing.T) {
	a, err := Parse([]string{
		"aios", "run", "--web", "plan.yaml",
		"--web-addr=127.0.0.1:9999",
	}, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if !a.WebLive {
		t.Error("WebLive = false, want true")
	}
	if a.WebAddr != "127.0.0.1:9999" {
		t.Errorf("WebAddr = %q, want 127.0.0.1:9999", a.WebAddr)
	}
	if a.PlanPath != "plan.yaml" {
		t.Errorf("PlanPath = %q, want plan.yaml", a.PlanPath)
	}
}

// TestHoistFlags_OtherSubcommands covers the defect's real
// blast radius: it was never just the new --web flag.
func TestHoistFlags_OtherSubcommands(t *testing.T) {
	cases := []struct {
		name string
		argv []string
		want func(*testing.T, *Args)
	}{
		{
			name: "report format after the task id",
			argv: []string{"aios", "report", "t1", "--format=html"},
			want: func(t *testing.T, a *Args) {
				if a.ReportFmt != "html" {
					t.Errorf("ReportFmt = %q, want html", a.ReportFmt)
				}
				if a.TaskID != "t1" {
					t.Errorf("TaskID = %q, want t1", a.TaskID)
				}
			},
		},
		{
			name: "audit task flag after nothing",
			argv: []string{"aios", "audit", "--task=t7"},
			want: func(t *testing.T, a *Args) {
				if a.TaskID != "t7" {
					t.Errorf("TaskID = %q, want t7", a.TaskID)
				}
			},
		},
		{
			name: "web addr flag",
			argv: []string{"aios", "web", "--addr=127.0.0.1:8123"},
			want: func(t *testing.T, a *Args) {
				if a.WebAddr != "127.0.0.1:8123" {
					t.Errorf("WebAddr = %q, want 127.0.0.1:8123", a.WebAddr)
				}
			},
		},
		{
			name: "fix plan-out after the objective",
			argv: []string{"aios", "fix", "count the go files", "--plan-out=p.yaml"},
			want: func(t *testing.T, a *Args) {
				if a.PlanOut != "p.yaml" {
					t.Errorf("PlanOut = %q, want p.yaml", a.PlanOut)
				}
				if a.Objective != "count the go files" {
					t.Errorf("Objective = %q", a.Objective)
				}
			},
		},
		{
			name: "resume plan flag after the task id",
			argv: []string{"aios", "resume", "t1", "--plan=plan.yaml"},
			want: func(t *testing.T, a *Args) {
				if a.PlanPath != "plan.yaml" {
					t.Errorf("PlanPath = %q, want plan.yaml", a.PlanPath)
				}
				if a.TaskID != "t1" {
					t.Errorf("TaskID = %q, want t1", a.TaskID)
				}
			},
		},
		{
			name: "audit tail flag after the task id",
			argv: []string{"aios", "audit", "t7", "--tail"},
			want: func(t *testing.T, a *Args) {
				if a.TaskID != "t7" {
					t.Errorf("TaskID = %q, want t7", a.TaskID)
				}
				if !a.Tail {
					t.Error("Tail = false, want true")
				}
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a, err := Parse(tc.argv, &bytes.Buffer{})
			if err != nil {
				t.Fatalf("Parse(%v): %v", tc.argv, err)
			}
			tc.want(t, a)
		})
	}
}

// TestHoistFlags_DoubleDashIsRespected covers the escape
// hatch: after "--", everything is positional. Without this,
// a value beginning with a dash (a filename like
// -- -weird.yaml) would be eaten as a flag.
func TestHoistFlags_DoubleDashIsRespected(t *testing.T) {
	got := hoistFlags([]string{"plan.yaml", "--", "--not-a-flag", "-x"}, nil)
	// The flags before "--" are hoisted; nothing after "--"
	// is moved, because those are values, not flags.
	if len(got) == 0 {
		t.Fatal("hoistFlags returned empty")
	}
	joined := ""
	for _, g := range got {
		joined += g + "\x00"
	}
	// "--not-a-flag" must remain after the "--" marker.
	idxMarker, idxValue := -1, -1
	for i, g := range got {
		if g == "--" && idxMarker < 0 {
			idxMarker = i
		}
		if g == "--not-a-flag" && idxValue < 0 {
			idxValue = i
		}
	}
	if idxMarker < 0 {
		t.Fatalf("-- marker missing from %v", got)
	}
	if idxValue < idxMarker {
		t.Errorf("--not-a-flag moved before the -- marker: %v", got)
	}
}

// TestHoistFlags_NoFlagsIsUnchanged asserts the identity
// case: an argument list with no flags comes back as it
// went in, so the common path allocates nothing extra.
func TestHoistFlags_NoFlagsIsUnchanged(t *testing.T) {
	in := []string{"plan.yaml"}
	got := hoistFlags(in, nil)
	if len(got) != 1 || got[0] != "plan.yaml" {
		t.Errorf("got %v, want [plan.yaml]", got)
	}
}

// TestHoistFlags_EmptyAndNil are the guards.
func TestHoistFlags_EmptyAndNil(t *testing.T) {
	if got := hoistFlags(nil, nil); len(got) != 0 {
		t.Errorf("hoistFlags(nil) = %v, want empty", got)
	}
	if got := hoistFlags([]string{}, nil); len(got) != 0 {
		t.Errorf("hoistFlags([]) = %v, want empty", got)
	}
}

// TestHoistFlags_PreservesOrderWithinGroups asserts the
// reorder is stable: flags keep their relative order and so
// do positionals, so "--a x --b y" never becomes
// "--b y --a x".
//
// Both flags are value-free here on purpose: a flag that takes
// a value absorbs the next argument, which is covered by
// TestHoistFlags_SeparatedValueStaysWithItsFlag.
func TestHoistFlags_PreservesOrderWithinGroups(t *testing.T) {
	got := hoistFlags([]string{"x", "--alpha", "y", "--beta"}, nil)
	want := []string{"--alpha", "--beta", "x", "y"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

// TestHoistFlags_ValueFlagCarriesItsValue asserts that a
// value-taking flag and its separated value stay adjacent and
// in order, rather than the value being sorted into the
// positional group.
func TestHoistFlags_ValueFlagCarriesItsValue(t *testing.T) {
	got := hoistFlags([]string{"--alpha", "y", "x", "--beta"}, map[string]bool{"alpha": true})
	want := []string{"--alpha", "y", "--beta", "x"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

// TestHoistFlags_SingleDashValue covers a bare "-" (stdin),
// which is a positional, not a flag.
func TestHoistFlags_SingleDashValue(t *testing.T) {
	got := hoistFlags([]string{"-", "--web"}, map[string]bool{"web": true})
	if got[0] != "--web" {
		t.Errorf("got %v, want the flag hoisted ahead of \"-\"", got)
	}
	if got[1] != "-" {
		t.Errorf("got %v, want \"-\" kept as a positional", got)
	}
}
