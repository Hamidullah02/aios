package verify

import (
	"errors"
	"strings"
	"testing"
)

func TestDefaultPrompt_ContainsDocSet(t *testing.T) {
	for _, want := range []string{
		"The agent did the following work",
		"The goal was",
		"status=\"pass\"",
		"status=\"fail\"",
		"status=\"inconclusive\"",
		"{{.Result}}",
		"{{.Goal}}",
	} {
		if !strings.Contains(DefaultPrompt, want) {
			t.Errorf("default prompt missing %q", want)
		}
	}
}

func TestPromptTemplate_New(t *testing.T) {
	if _, err := NewPromptTemplate(""); err == nil {
		t.Errorf("expected error for empty")
	}
	if _, err := NewPromptTemplate("{{"); err == nil {
		t.Errorf("expected parse error")
	}
	p, err := NewPromptTemplate("hello {{.Goal}}")
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if p.Text() != "hello {{.Goal}}" {
		t.Errorf("text mismatch: %q", p.Text())
	}
	if p.String() != "hello {{.Goal}}" {
		t.Errorf("stringer mismatch: %q", p.String())
	}
}

func TestPromptTemplate_Default(t *testing.T) {
	p, err := NewDefaultPromptTemplate()
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if p.Text() != DefaultPrompt {
		t.Errorf("default mismatch")
	}
}

func TestPromptTemplate_Render(t *testing.T) {
	p, err := NewPromptTemplate("hi {{.Goal}}")
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	got, err := p.Render(VerificationInput{Goal: "alice"})
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if got != "hi alice" {
		t.Errorf("got %q", got)
	}
}

func TestPromptTemplate_Render_Nil(t *testing.T) {
	var p *PromptTemplate
	if _, err := p.Render(VerificationInput{}); err == nil {
		t.Errorf("expected error for nil")
	}
}

func TestPromptTemplate_Render_ExecError(t *testing.T) {
	// Template that references a missing field
	// produces an execution error.
	p, err := NewPromptTemplate("{{.Missing}}")
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	_, err = p.Render(VerificationInput{})
	if err == nil {
		t.Errorf("expected exec error")
	}
	if !errors.Is(err, ErrVerification) {
		t.Errorf("expected ErrVerification, got %v", err)
	}
}
