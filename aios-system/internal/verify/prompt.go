package verify

import (
	"bytes"
	"fmt"
	"text/template"
)

// DefaultPrompt is the doc-set REQ-VRF-DAT-003
// default verification prompt. The prompt is
// bundled with the package; the CLI is the
// layer that writes it to
// `./var/verify_prompt.txt` on `aios init`.
//
// The template variables are:
//
//	{{.Result}} — the work the agent did.
//	{{.Goal}}   — the goal the work is
//	              measured against.
//
// The response format expected from the model
// is JSON of the VerificationResult schema.
const DefaultPrompt = `The agent did the following work:

{{.Result}}

The goal was:

{{.Goal}}

Check whether the work satisfies the goal.
- If yes, return status="pass" and a short reason.
- If no, return status="fail", a short reason, and
  a list of suggestions.
- If you cannot decide, return status="inconclusive"
  and explain why.

Return the result as a single JSON object with this
exact shape (no prose around it):

{
  "status": "pass|fail|inconclusive",
  "reason": "<short reason>",
  "suggestions": ["<fix 1>", "<fix 2>"]
}
`

// PromptTemplate is the typed text/template
// wrapper the SelfCheck tool uses to render the
// verification prompt (REQ-VRF-PRM-001 to
// 003).
//
// The wrapper is the small object the
// VerificationPrompt class becomes in Go. The
// text/template stdlib is the engine (the doc
// set's `text/template (stdlib)` substitution).
type PromptTemplate struct {
	text string
	tmpl *template.Template
}

// NewPromptTemplate parses the given text as
// a Go text/template with the VerificationInput
// data shape. The function returns an error
// when the template is malformed.
func NewPromptTemplate(text string) (*PromptTemplate, error) {
	if text == "" {
		return nil, &VerificationError{Op: "new prompt", Reason: "template is empty"}
	}
	t, err := template.New("verify").Parse(text)
	if err != nil {
		return nil, &VerificationError{Op: "new prompt", Reason: "parse", Err: err}
	}
	return &PromptTemplate{text: text, tmpl: t}, nil
}

// NewDefaultPromptTemplate returns a
// PromptTemplate with the doc-set default
// text. The function is the convenience
// constructor the SelfCheck tool uses when
// the caller does not provide a custom
// prompt.
func NewDefaultPromptTemplate() (*PromptTemplate, error) {
	return NewPromptTemplate(DefaultPrompt)
}

// VerificationInput is the data shape the
// template renders.
type VerificationInput struct {
	Result string
	Goal   string
}

// Render produces the rendered prompt for the
// given input. The function is the entry point
// the SelfCheck tool calls.
func (p *PromptTemplate) Render(in VerificationInput) (string, error) {
	if p == nil || p.tmpl == nil {
		return "", &VerificationError{Op: "render", Reason: "template is nil"}
	}
	var buf bytes.Buffer
	if err := p.tmpl.Execute(&buf, in); err != nil {
		return "", &VerificationError{Op: "render", Reason: "execute", Err: err}
	}
	return buf.String(), nil
}

// Text returns the raw template text. The
// function is the getter the test suite and
// the CLI use to dump the default prompt to
// ./var/verify_prompt.txt.
func (p *PromptTemplate) Text() string {
	if p == nil {
		return ""
	}
	return p.text
}

// String implements fmt.Stringer for the
// prompt; the string is the raw template text.
func (p *PromptTemplate) String() string {
	if p == nil {
		return ""
	}
	return p.text
}

// compile-time check: PromptTemplate is
// fmt.Stringer.
var _ fmt.Stringer = (*PromptTemplate)(nil)
