package ask

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// PromptCallback is the hook the tool uses to show the question
// to the operator. The doc set describes it as
// `Callable[[Question], Awaitable[Answer]]`. The Go version is
// synchronous: the callback runs the prompt UI and returns the
// answer (or the zero Answer when the operator did not pick one).
//
// The callback MUST respect the context. The tool sets a
// timeout on the context before calling the callback; the
// callback should return ctx.Err() == context.DeadlineExceeded
// (or any non-nil error) when the deadline fires.
//
// The callback may also return a default answer on its own.
// The tool only falls back to a default when the callback
// returns an error AND the question has a default option.
type PromptCallback func(ctx context.Context, q Question) (Answer, error)

// Clock is the injection point for tests. The tool reads
// time.Now() through this. The default is time.Now (real clock).
type Clock interface {
	Now() time.Time
	NewTimer(d time.Duration) (<-chan time.Time, func())
}

type realClock struct{}

func (realClock) Now() time.Time { return time.Now() }
func (realClock) NewTimer(d time.Duration) (<-chan time.Time, func()) {
	t := time.NewTimer(d)
	return t.C, func() { t.Stop() }
}

// ErrTimeout is the sentinel error the tool returns (wrapped
// in a *TimeoutError) when the prompt callback did not return
// an answer within the question's timeout and the question
// has no default option. Per REQ-ASK-ERR-001 the tool raises
// AskTimeout. In Go that is a sentinel error and a typed
// error for error-type matching.
var ErrTimeout = errors.New("ask: timeout")

// TimeoutError is the typed wrapper for the timeout case. It
// implements the Error, Is, and Unwrap interfaces so callers
// can match with errors.Is(err, ErrTimeout) and read the
// question id off the typed value.
type TimeoutError struct {
	QuestionID string
}

func (e *TimeoutError) Error() string {
	return fmt.Sprintf("ask: question %q timed out without an answer", e.QuestionID)
}

func (e *TimeoutError) Is(target error) bool { return target == ErrTimeout }
func (e *TimeoutError) Unwrap() error        { return ErrTimeout }

// AskUserTool is the tool the agent calls. Per REQ-ASK-FUN-003
// it implements the `tools.Tool` interface — Name and Run. The
// prompt callback is configured at construction time. The
// tool is otherwise stateless and safe for concurrent use.
type AskUserTool struct {
	prompt PromptCallback
	clock  Clock
}

// New returns an AskUserTool. prompt is required; nil is
// rejected. clock is optional; nil becomes the real clock.
func New(prompt PromptCallback, clock Clock) (*AskUserTool, error) {
	if prompt == nil {
		return nil, errors.New("ask: prompt callback is required")
	}
	if clock == nil {
		clock = realClock{}
	}
	return &AskUserTool{prompt: prompt, clock: clock}, nil
}

// Name returns the tool name. Per REQ-ASK-FUN-003 the tool
// is registered in the bus under "ask_user".
func (a *AskUserTool) Name() string { return "ask_user" }

// Run executes one ask-user call. The args map is expected to
// carry a `question` entry whose value is a Question (the
// canonical encoding, a map[string]any, or the typed struct
// all work). The tool:
//
//  1. Validates the question.
//  2. Sets up the timeout context (per REQ-ASK-FUN-007).
//  3. Calls the prompt callback.
//  4. On error + default: returns the default answer.
//  5. On error + no default: returns a *TimeoutError.
//  6. On success: returns the answer (with timestamp).
//
// The tool returns a tools.Result with OK=true and the answer
// in the Output map. The call id is the question id so the
// audit row can join the result back to the question.
func (a *AskUserTool) Run(ctx context.Context, args map[string]any) (Result, error) {
	q, err := decodeQuestion(args)
	if err != nil {
		return Result{
			OK:      false,
			Summary: "invalid question",
		}, fmt.Errorf("ask: decode question: %w", err)
	}
	if err := q.Validate(); err != nil {
		return Result{
			OK:      false,
			Summary: "invalid question: " + err.Error(),
		}, fmt.Errorf("ask: invalid question: %w", err)
	}

	// Per TRD §6.2 the tool supports a timeout via
	// time.AfterFunc + select on context.Done. We use
	// context.WithTimeout which is the stdlib equivalent and
	// gives the callback a cancellable context for free.
	runCtx := ctx
	var cancel context.CancelFunc
	if q.TimeoutSeconds > 0 {
		runCtx, cancel = context.WithTimeout(ctx, time.Duration(q.TimeoutSeconds)*time.Second)
		defer cancel()
	}

	ans, err := a.prompt(runCtx, q)
	if err == nil {
		// Stamp the timestamp if the callback left it empty.
		if ans.Timestamp == "" {
			ans.Timestamp = nowUTC()
		}
		ans.QuestionID = q.ID
		if vErr := ans.Validate(); vErr != nil {
			return Result{
				OK:      false,
				Summary: "callback returned invalid answer: " + vErr.Error(),
			}, fmt.Errorf("ask: callback returned invalid answer: %w", vErr)
		}
		return Result{
			OK:      true,
			Output:  answerToOutput(ans),
			Summary: "answered: " + ans.SelectedOption,
		}, nil
	}

	// The callback failed. Distinguish timeout (the question
	// has a default) from a real failure (no default).
	if q.DefaultOption == "" {
		return Result{
			OK:      false,
			Summary: "no default and callback failed",
		}, &TimeoutError{QuestionID: q.ID}
	}
	// Build a default answer.
	ans = Answer{
		QuestionID:     q.ID,
		SelectedOption: q.DefaultOption,
		Timestamp:      nowUTC(),
	}
	return Result{
		OK:      true,
		Output:  answerToOutput(ans),
		Summary: "defaulted: " + ans.SelectedOption,
	}, nil
}

// Result is the typed return of the tool. It mirrors the
// tools.Result shape (OK, Output, Summary) so the bus can
// wrap it without translation. The ask package does not import
// internal/tools to keep the package tool-agnostic; the bus
// adapter translates this Result into tools.Result.
type Result struct {
	OK      bool
	Output  map[string]any
	Summary string
}

// decodeQuestion pulls the Question out of args["question"].
// The agent may pass a typed Question, a map[string]any, or
// the canonical JSON bytes. We accept all three for ergonomics.
func decodeQuestion(args map[string]any) (Question, error) {
	if args == nil {
		return Question{}, errors.New("args is nil")
	}
	raw, ok := args["question"]
	if !ok {
		return Question{}, errors.New("missing question argument")
	}
	switch v := raw.(type) {
	case Question:
		return v, nil
	case *Question:
		if v == nil {
			return Question{}, errors.New("question is nil")
		}
		return *v, nil
	case map[string]any:
		// Re-marshal then unmarshal through the typed decoder
		// to honour the strict unknown-field rejection.
		raw, err := json.Marshal(v)
		if err != nil {
			return Question{}, err
		}
		var q Question
		if err := json.Unmarshal(raw, &q); err != nil {
			return Question{}, err
		}
		return q, nil
	case []byte:
		var q Question
		if err := json.Unmarshal(v, &q); err != nil {
			return Question{}, err
		}
		return q, nil
	case string:
		var q Question
		if err := json.Unmarshal([]byte(v), &q); err != nil {
			return Question{}, err
		}
		return q, nil
	}
	return Question{}, fmt.Errorf("unsupported question type %T", raw)
}

// answerToOutput builds the Output map the bus consumes. The
// shape matches the schemas.Answer JSON encoding so the bus
// can serialise the result as a JSON-compatible map.
func answerToOutput(a Answer) map[string]any {
	return map[string]any{
		"question_id":     a.QuestionID,
		"selected_option": a.SelectedOption,
		"timestamp":       a.Timestamp,
	}
}
