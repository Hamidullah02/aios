package policy

import (
	"strings"
	"testing"
)

func TestSubstitute_Known(t *testing.T) {
	c := Call{Tool: "edit", TaskID: "t1", Path: "/tmp/x", Host: "h", Actor: "a", RiskTier: 2}
	cases := map[string]string{
		"{task}":  "t1",
		"{tool}":  "edit",
		"{path}":  "/tmp/x",
		"{host}":  "h",
		"{actor}": "a",
		"{tier}":  "2",
	}
	for in, want := range cases {
		got, err := Substitute(in, c)
		if err != nil {
			t.Fatalf("%s: %v", in, err)
		}
		if got != want {
			t.Errorf("%s: got %q want %q", in, got, want)
		}
	}
}

func TestSubstitute_Arg(t *testing.T) {
	c := Call{Args: map[string]any{"branch": "main"}}
	got, err := Substitute("{arg.branch}", c)
	if err != nil {
		t.Fatal(err)
	}
	if got != "main" {
		t.Fatalf("got %q", got)
	}
}

func TestSubstitute_ArgMissing(t *testing.T) {
	c := Call{Args: map[string]any{}}
	_, err := Substitute("{arg.x}", c)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSubstitute_ArgNoArgs(t *testing.T) {
	c := Call{}
	_, err := Substitute("{arg.x}", c)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSubstitute_Unknown(t *testing.T) {
	_, err := Substitute("{unknown}", Call{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSubstitute_Unterminated(t *testing.T) {
	_, err := Substitute("hello {", Call{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSubstitute_Literal(t *testing.T) {
	got, err := Substitute("hello world", Call{})
	if err != nil {
		t.Fatal(err)
	}
	if got != "hello world" {
		t.Fatalf("got %q", got)
	}
}

func TestSubstitute_Empty(t *testing.T) {
	got, err := Substitute("", Call{})
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Fatalf("got %q", got)
	}
}

func TestSubstitute_Mixed(t *testing.T) {
	c := Call{Tool: "edit", Path: "/tmp/x"}
	got, err := Substitute("Write to {path} via {tool}", c)
	if err != nil {
		t.Fatal(err)
	}
	if got != "Write to /tmp/x via edit" {
		t.Fatalf("got %q", got)
	}
}

func TestSubstituteMap(t *testing.T) {
	c := Call{TaskID: "t1"}
	out, err := SubstituteMap(map[string]string{
		"task": "{task}",
		"tool": "{tool}",
	}, c)
	if err != nil {
		t.Fatal(err)
	}
	if out["task"] != "t1" {
		t.Fatalf("task: %q", out["task"])
	}
	if out["tool"] != "" {
		t.Fatalf("tool: %q", out["tool"])
	}
}

func TestSubstituteMap_BadPlaceholder(t *testing.T) {
	_, err := SubstituteMap(map[string]string{"x": "{unknown}"}, Call{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "unknown") {
		t.Fatalf("error: %v", err)
	}
}
