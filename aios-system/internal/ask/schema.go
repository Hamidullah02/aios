// Package ask is the ask-user tool for aios.
//
// The tool lets the agent ask the operator a multiple-choice
// question. The operator picks an option. The tool returns
// the answer. The flow is:
//
//  1. The agent builds a Question with at least one Option.
//  2. The agent calls AskUserTool.Run(args, ctx).
//  3. The tool validates the args and the question.
//  4. The tool calls the prompt callback, which is configured
//     at construction time and which shows the question to the
//     operator (CLI, web, ...).
//  5. The callback returns an Answer.
//  6. The tool returns the Answer as a Result.
//
// Substitutions vs the doc set:
//
//   - Python filenames (.py) become Go files (.go). The
//     doc set's split between `aios/ask/schema.py` and
//     `aios/ask/tool.py` collapses into this single package.
//   - The doc set's "go-playground/validator (or hand-rolled
//     tag validation)" is hand-rolled. The aios-core schemas
//     package already does hand-rolled validation; this package
//     follows the same pattern so aios has no third-party
//     validation dependency.
//   - The doc set's `Callable[[Question], Awaitable[Answer]]`
//     becomes a synchronous `PromptCallback func(Question) Answer`.
//     The Go runtime does not need async to express the
//     operator-prompt flow; the timeout is the synchronization
//     primitive (a select on a context done channel).
//   - The doc set's `AskTimeout` is exported as `ErrTimeout`.
//     The error name in the doc is a class name; the Go
//     convention is a sentinel error.
package ask

import (
	"fmt"
	"sort"
	"time"

	"github.com/sadi/aios-system/internal/core/schemas"
)

// Question is the typed question the agent sends to the
// operator. Per REQ-ASK-DAT-001 the fields are id, prompt,
// options, timeout_seconds (default 0 for no timeout), and
// default_option (optional).
type Question struct {
	ID             string      `json:"id"`
	Prompt         string      `json:"prompt"`
	Options        []AskOption `json:"options"`
	TimeoutSeconds int         `json:"timeout_seconds,omitempty"`
	DefaultOption  string      `json:"default_option,omitempty"`
}

// AskOption is one choice inside a Question. Per REQ-ASK-DAT-002
// the fields are id, label, and an optional description.
type AskOption struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
}

// Answer is the operator's response. Per REQ-ASK-DAT-002
// the Answer carries the question id, the selected option id,
// and the ISO 8601 timestamp of the answer.
type Answer struct {
	QuestionID     string `json:"question_id"`
	SelectedOption string `json:"selected_option"`
	Timestamp      string `json:"timestamp"` // ISO 8601, set by the tool
}

// Validate implements schemas.Schema on Question. The same
// validation runs on the tool side, so the doc set's
// "schema validates the prompt, the options, and the timeout"
// is exercised by Run and is also available to callers that
// build a Question directly.
func (q *Question) Validate() error {
	return schemas.JoinValidationErrors(
		schemas.RequiredString("id", q.ID),
		schemas.RequiredString("prompt", q.Prompt),
		validateOptions("options", q.Options),
		validateTimeout("timeout_seconds", q.TimeoutSeconds),
		validateDefaultOption("default_option", q.DefaultOption, q.Options),
	)
}

// MarshalJSON produces a canonical encoding (sorted keys) so two
// questions with the same content have the same bytes. The
// schemas package's canonical encoder is the source of truth.
func (q *Question) MarshalJSON() ([]byte, error) {
	type alias Question
	return schemas.MarshalCanonical((*alias)(q))
}

// UnmarshalJSON parses with strict unknown-field rejection. The
// Question wire shape is small and stable; any drift surfaces here.
func (q *Question) UnmarshalJSON(data []byte) error {
	type alias Question
	var a alias
	if err := schemas.UnmarshalStrict(data, &a, []string{
		"id", "prompt", "options", "timeout_seconds", "default_option",
	}); err != nil {
		return err
	}
	*q = Question(a)
	return nil
}

func validateOptions(path string, opts []AskOption) error {
	if len(opts) == 0 {
		return schemas.NewValidationError(path, "required, at least one option")
	}
	seen := make(map[string]struct{}, len(opts))
	// Sort the option ids for deterministic error messages.
	ids := make([]string, 0, len(opts))
	for i, o := range opts {
		base := fmt.Sprintf("%s.%d", path, i)
		if err := schemas.JoinValidationErrors(
			schemas.RequiredString(base+".id", o.ID),
			schemas.RequiredString(base+".label", o.Label),
			schemas.OptionalString(base+".description", o.Description),
		); err != nil {
			return err
		}
		if _, dup := seen[o.ID]; dup {
			ids = append(ids, o.ID)
		}
		seen[o.ID] = struct{}{}
	}
	if len(ids) > 0 {
		sort.Strings(ids)
		return schemas.NewValidationError(path,
			"duplicate option ids: "+joinStrings(ids, ", "))
	}
	return nil
}

func validateTimeout(path string, sec int) error {
	if sec < 0 {
		return schemas.NewValidationError(path, "must be non-negative")
	}
	return nil
}

func validateDefaultOption(path, def string, opts []AskOption) error {
	if def == "" {
		return nil
	}
	for _, o := range opts {
		if o.ID == def {
			return nil
		}
	}
	return schemas.NewValidationError(path,
		fmt.Sprintf("default_option %q is not in the options list", def))
}

// Validate implements schemas.Schema on Answer. The answer
// always references a question id, so the field is required.
func (a *Answer) Validate() error {
	return schemas.JoinValidationErrors(
		schemas.RequiredString("question_id", a.QuestionID),
		schemas.RequiredString("selected_option", a.SelectedOption),
		schemas.RequiredString("timestamp", a.Timestamp),
	)
}

// MarshalJSON / UnmarshalJSON for Answer. Answer is small
// enough that the canonical encoding is mostly identical to
// the default; we still route through the canonical encoder
// for consistency.
func (a *Answer) MarshalJSON() ([]byte, error) {
	type alias Answer
	return schemas.MarshalCanonical((*alias)(a))
}

func (a *Answer) UnmarshalJSON(data []byte) error {
	type alias Answer
	var al alias
	if err := schemas.UnmarshalStrict(data, &al, []string{
		"question_id", "selected_option", "timestamp",
	}); err != nil {
		return err
	}
	*a = Answer(al)
	return nil
}

// joinStrings is a small helper for error messages. Pulled in
// here to avoid an extra dep on the egress package.
func joinStrings(parts []string, sep string) string {
	if len(parts) == 0 {
		return ""
	}
	if len(parts) == 1 {
		return parts[0]
	}
	out := parts[0]
	for i := 1; i < len(parts); i++ {
		out += sep + parts[i]
	}
	return out
}

// nowUTC is the timestamp helper. Pulled out so tests can swap
// in a fixed clock. The default is RFC 3339 (a strict subset
// of ISO 8601) in UTC.
func nowUTC() string {
	return time.Now().UTC().Format(time.RFC3339)
}
