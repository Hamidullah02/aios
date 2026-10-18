package cli

import (
	"bytes"
	"errors"
	"testing"
)

func TestCliError_Error(t *testing.T) {
	e := &CliError{Kind: "user", What: "bad input"}
	if e.Error() == "" {
		t.Fatal("non-empty")
	}
	e2 := &CliError{Kind: "user", What: "bad input", Cause: errors.New("inner")}
	if e2.Error() == "" {
		t.Fatal("non-empty with cause")
	}
}

func TestCliError_Unwrap(t *testing.T) {
	inner := errors.New("inner")
	e := &CliError{Cause: inner}
	if errors.Unwrap(e) != inner {
		t.Fatal("Unwrap should return cause")
	}
}

func TestConfigError_Error(t *testing.T) {
	e := &ConfigError{Path: "c.yaml", Err: errors.New("parse")}
	if e.Error() == "" {
		t.Fatal("non-empty")
	}
	if errors.Unwrap(e) == nil {
		t.Fatal("Unwrap should return err")
	}
}

func TestPlanLoadError_Error(t *testing.T) {
	e := &PlanLoadError{Path: "p.yaml", Err: errors.New("invalid")}
	if e.Error() == "" {
		t.Fatal("non-empty")
	}
	if errors.Unwrap(e) == nil {
		t.Fatal("Unwrap should return err")
	}
}

func TestDisplayError_Nil(t *testing.T) {
	if DisplayError(nil, false) != "" {
		t.Fatal("nil should give empty string")
	}
}

func TestDisplayError_CliError(t *testing.T) {
	e := &CliError{Kind: "user", What: "bad"}
	s := DisplayError(e, false)
	if s == "" {
		t.Fatal("non-empty")
	}
	if !contains(s, "error:") {
		t.Errorf("got %q", s)
	}
}

func TestDisplayError_ConfigError(t *testing.T) {
	e := &ConfigError{Path: "c.yaml", Err: errors.New("parse")}
	s := DisplayError(e, false)
	if !contains(s, "config") {
		t.Errorf("got %q", s)
	}
}

func TestDisplayError_PlanError(t *testing.T) {
	e := &PlanLoadError{Path: "p.yaml", Err: errors.New("invalid")}
	s := DisplayError(e, false)
	if !contains(s, "plan") {
		t.Errorf("got %q", s)
	}
}

func TestDisplayError_Generic(t *testing.T) {
	e := errors.New("system blew up")
	s := DisplayError(e, false)
	if !contains(s, "system blew up") {
		t.Errorf("got %q", s)
	}
}

func TestDisplayError_Debug(t *testing.T) {
	inner := errors.New("root cause")
	e := &CliError{Kind: "user", What: "bad", Cause: inner}
	s := DisplayError(e, true)
	if !contains(s, "root cause") {
		t.Errorf("debug should include cause: %q", s)
	}
}

func TestExitCodeFor(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want int
	}{
		{"nil", nil, ExitOK},
		{"cli", &CliError{Kind: "user"}, ExitUserError},
		{"cfg", &ConfigError{Path: "x"}, ExitUserError},
		{"plan", &PlanLoadError{Path: "y"}, ExitUserError},
		{"generic", errors.New("system"), ExitSystemError},
	}
	for _, c := range cases {
		if got := ExitCodeFor(c.err); got != c.want {
			t.Errorf("%s: %d, want %d", c.name, got, c.want)
		}
	}
}

func TestMain_NoArgs(t *testing.T) {
	code := Main([]string{}, &bytes.Buffer{}, &bytes.Buffer{})
	if code != ExitUserError {
		t.Errorf("code: %d", code)
	}
}

func TestMain_Help(t *testing.T) {
	code := Main([]string{"aios", "--help"}, &bytes.Buffer{}, &bytes.Buffer{})
	if code != ExitOK {
		t.Errorf("code: %d", code)
	}
}

func TestMain_UnknownCommand(t *testing.T) {
	code := Main([]string{"aios", "frobnicate"}, &bytes.Buffer{}, &bytes.Buffer{})
	if code != ExitUserError {
		t.Errorf("code: %d", code)
	}
}

func contains(s, sub string) bool {
	if len(sub) == 0 {
		return true
	}
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
