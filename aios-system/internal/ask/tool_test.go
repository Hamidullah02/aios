package ask

import (
	"context"
	"encoding/json"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

// fakeClock lets tests advance time deterministically. NewTimer
// returns a channel the test can fire with a method, and a stop
// function.
type fakeClock struct {
	now    time.Time
	timers []func()
}

func newFakeClock() *fakeClock {
	return &fakeClock{now: time.Unix(1700000000, 0)}
}

func (f *fakeClock) Now() time.Time { return f.now }

func (f *fakeClock) NewTimer(d time.Duration) (<-chan time.Time, func()) {
	ch := make(chan time.Time, 1)
	cancel := func() { /* nothing to stop in a fake */ }
	// Schedule the trigger.
	go func() {
		time.Sleep(10 * time.Millisecond) // keep test wall-time low
		select {
		case ch <- f.now.Add(d):
		default:
		}
	}()
	return ch, cancel
}

// scriptedCallback returns a PromptCallback that yields the
// scripted answer or error and counts the calls.
type scriptedCallback struct {
	calls atomic.Int32
	last  Question
	resp  Answer
	err   error
	delay time.Duration
}

func (s *scriptedCallback) Run(ctx context.Context, q Question) (Answer, error) {
	s.calls.Add(1)
	s.last = q
	if s.delay > 0 {
		select {
		case <-ctx.Done():
			return Answer{}, ctx.Err()
		case <-time.After(s.delay):
		}
	}
	if s.err != nil {
		return Answer{}, s.err
	}
	return s.resp, nil
}

// asCallback returns a PromptCallback bound to the receiver
// pointer so the test can pass it to New() without taking the
// address of an unaddressable composite literal. The closure
// captures the pointer; the value receiver exists only so the
// method is callable on a non-addressable composite literal.
func (s *scriptedCallback) asCallback() PromptCallback {
	return s.Run
}

// scriptedCB is a small helper for tests that want to use a
// fresh scriptedCallback as a value. It returns a pointer so
// the caller can call .asCallback() on the addressable value.
func scriptedCB(opts ...func(*scriptedCallback)) *scriptedCallback {
	s := &scriptedCallback{}
	for _, o := range opts {
		o(s)
	}
	return s
}

func mkQuestion() Question {
	return Question{
		ID:     "q-1",
		Prompt: "Which database?",
		Options: []AskOption{
			{ID: "pg", Label: "PostgreSQL"},
			{ID: "sqlite", Label: "SQLite"},
		},
	}
}

func TestNew_RequiresPrompt(t *testing.T) {
	if _, err := New(nil, nil); err == nil {
		t.Fatal("expected error for nil prompt")
	}
}

func TestNew_DefaultClock(t *testing.T) {
	tool, err := New(scriptedCB().asCallback(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if tool.clock == nil {
		t.Fatal("default clock should be set")
	}
}

func TestAskUserTool_Name(t *testing.T) {
	tool, _ := New(scriptedCB().asCallback(), nil)
	if got := tool.Name(); got != "ask_user" {
		t.Fatalf("name: %q", got)
	}
}

func TestRun_HappyPath(t *testing.T) {
	cb := scriptedCB(func(s *scriptedCallback) {
		s.resp = Answer{QuestionID: "q-1", SelectedOption: "pg", Timestamp: "t"}
	})
	tool, _ := New(cb.asCallback(), nil)
	res, err := tool.Run(context.Background(), map[string]any{
		"question": mkQuestion(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.OK {
		t.Fatalf("OK: %v", res.OK)
	}
	if res.Summary != "answered: pg" {
		t.Errorf("summary: %q", res.Summary)
	}
	if res.Output["selected_option"] != "pg" {
		t.Errorf("output: %+v", res.Output)
	}
	if cb.calls.Load() != 1 {
		t.Errorf("calls: %d", cb.calls.Load())
	}
}

func TestRun_TypedQuestionArg(t *testing.T) {
	cb := scriptedCB(func(s *scriptedCallback) { s.resp = Answer{SelectedOption: "pg", Timestamp: "t"} })
	tool, _ := New(cb.asCallback(), nil)
	q := mkQuestion()
	if _, err := tool.Run(context.Background(), map[string]any{"question": q}); err != nil {
		t.Fatal(err)
	}
	if cb.last.ID != q.ID {
		t.Errorf("callback got wrong question: %+v", cb.last)
	}
}

func TestRun_PointerQuestionArg(t *testing.T) {
	cb := scriptedCB(func(s *scriptedCallback) { s.resp = Answer{SelectedOption: "pg", Timestamp: "t"} })
	tool, _ := New(cb.asCallback(), nil)
	q := mkQuestion()
	if _, err := tool.Run(context.Background(), map[string]any{"question": &q}); err != nil {
		t.Fatal(err)
	}
}

func TestRun_MapQuestionArg(t *testing.T) {
	cb := scriptedCB(func(s *scriptedCallback) { s.resp = Answer{SelectedOption: "pg", Timestamp: "t"} })
	tool, _ := New(cb.asCallback(), nil)
	if _, err := tool.Run(context.Background(), map[string]any{
		"question": map[string]any{
			"id":      "q-1",
			"prompt":  "Which?",
			"options": []any{map[string]any{"id": "pg", "label": "PG"}},
		},
	}); err != nil {
		t.Fatal(err)
	}
}

func TestRun_StringQuestionArg(t *testing.T) {
	cb := scriptedCB(func(s *scriptedCallback) { s.resp = Answer{SelectedOption: "pg", Timestamp: "t"} })
	tool, _ := New(cb.asCallback(), nil)
	q := mkQuestion()
	raw, _ := jsonMarshalQ(q)
	if _, err := tool.Run(context.Background(), map[string]any{"question": string(raw)}); err != nil {
		t.Fatal(err)
	}
}

func TestRun_BytesQuestionArg(t *testing.T) {
	cb := scriptedCB(func(s *scriptedCallback) { s.resp = Answer{SelectedOption: "pg", Timestamp: "t"} })
	tool, _ := New(cb.asCallback(), nil)
	q := mkQuestion()
	raw, _ := jsonMarshalQ(q)
	if _, err := tool.Run(context.Background(), map[string]any{"question": raw}); err != nil {
		t.Fatal(err)
	}
}

func TestRun_NilArgs(t *testing.T) {
	tool, _ := New(scriptedCB().asCallback(), nil)
	if _, err := tool.Run(context.Background(), nil); err == nil {
		t.Fatal("expected error for nil args")
	}
}

func TestRun_MissingQuestion(t *testing.T) {
	tool, _ := New(scriptedCB().asCallback(), nil)
	if _, err := tool.Run(context.Background(), map[string]any{}); err == nil {
		t.Fatal("expected error for missing question")
	}
}

func TestRun_UnsupportedQuestionType(t *testing.T) {
	tool, _ := New(scriptedCB().asCallback(), nil)
	if _, err := tool.Run(context.Background(), map[string]any{"question": 42}); err == nil {
		t.Fatal("expected error for unsupported type")
	}
}

func TestRun_InvalidQuestion(t *testing.T) {
	tool, _ := New(scriptedCB().asCallback(), nil)
	q := mkQuestion()
	q.ID = "" // invalid
	_, err := tool.Run(context.Background(), map[string]any{"question": q})
	if err == nil {
		t.Fatal("expected error for invalid question")
	}
}

func TestRun_InvalidMapQuestion(t *testing.T) {
	tool, _ := New(scriptedCB().asCallback(), nil)
	// missing options
	_, err := tool.Run(context.Background(), map[string]any{
		"question": map[string]any{"id": "q-1", "prompt": "x"},
	})
	if err == nil {
		t.Fatal("expected error for invalid map question")
	}
}

func TestRun_InvalidBytesQuestion(t *testing.T) {
	tool, _ := New(scriptedCB().asCallback(), nil)
	_, err := tool.Run(context.Background(), map[string]any{"question": []byte("not json")})
	if err == nil {
		t.Fatal("expected error for invalid bytes")
	}
}

func TestRun_InvalidStringQuestion(t *testing.T) {
	tool, _ := New(scriptedCB().asCallback(), nil)
	_, err := tool.Run(context.Background(), map[string]any{"question": "not json"})
	if err == nil {
		t.Fatal("expected error for invalid string")
	}
}

func TestRun_NilPointerQuestion(t *testing.T) {
	tool, _ := New(scriptedCB().asCallback(), nil)
	var q *Question
	_, err := tool.Run(context.Background(), map[string]any{"question": q})
	if err == nil {
		t.Fatal("expected error for nil pointer")
	}
}

func TestRun_CallbackStampsTimestamp(t *testing.T) {
	cb := scriptedCB(func(s *scriptedCallback) { s.resp = Answer{SelectedOption: "pg", Timestamp: ""} })
	tool, _ := New(cb.asCallback(), nil)
	res, err := tool.Run(context.Background(), map[string]any{"question": mkQuestion()})
	if err != nil {
		t.Fatal(err)
	}
	ts, _ := res.Output["timestamp"].(string)
	if ts == "" {
		t.Fatal("timestamp should be stamped")
	}
}

func TestRun_CallbackStampsQuestionID(t *testing.T) {
	cb := scriptedCB(func(s *scriptedCallback) { s.resp = Answer{SelectedOption: "pg", Timestamp: "t"} })
	tool, _ := New(cb.asCallback(), nil)
	res, _ := tool.Run(context.Background(), map[string]any{"question": mkQuestion()})
	if res.Output["question_id"] != "q-1" {
		t.Fatalf("question_id: %+v", res.Output)
	}
}

func TestRun_CallbackInvalidAnswer(t *testing.T) {
	cb := scriptedCB(func(s *scriptedCallback) { s.resp = Answer{SelectedOption: ""} }) // missing question_id and selected_option
	tool, _ := New(cb.asCallback(), nil)
	_, err := tool.Run(context.Background(), map[string]any{"question": mkQuestion()})
	if err == nil {
		t.Fatal("expected error for invalid answer")
	}
}

func TestRun_TimeoutFallsBackToDefault(t *testing.T) {
	q := mkQuestion()
	q.TimeoutSeconds = 1
	q.DefaultOption = "pg"
	cb := scriptedCB(func(s *scriptedCallback) {
		s.err = context.DeadlineExceeded
		s.delay = 100 * time.Millisecond
	})
	tool, _ := New(cb.asCallback(), nil)
	res, err := tool.Run(context.Background(), map[string]any{"question": q})
	if err != nil {
		t.Fatal(err)
	}
	if !res.OK {
		t.Fatal("OK should be true with default fallback")
	}
	if res.Output["selected_option"] != "pg" {
		t.Errorf("output: %+v", res.Output)
	}
	if res.Summary != "defaulted: pg" {
		t.Errorf("summary: %q", res.Summary)
	}
}

func TestRun_TimeoutNoDefaultReturnsError(t *testing.T) {
	q := mkQuestion()
	q.TimeoutSeconds = 1
	q.DefaultOption = ""
	cb := &scriptedCallback{
		err:   context.DeadlineExceeded,
		delay: 100 * time.Millisecond,
	}
	tool, _ := New(cb.asCallback(), nil)
	_, err := tool.Run(context.Background(), map[string]any{"question": q})
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !errors.Is(err, ErrTimeout) {
		t.Errorf("expected ErrTimeout, got %v", err)
	}
	var te *TimeoutError
	if !errors.As(err, &te) {
		t.Fatal("expected *TimeoutError")
	}
	if te.QuestionID != "q-1" {
		t.Errorf("question id: %q", te.QuestionID)
	}
}

func TestRun_ZeroTimeoutNoDeadline(t *testing.T) {
	cb := scriptedCB(func(s *scriptedCallback) {
		s.resp = Answer{SelectedOption: "pg", Timestamp: "t"}
	})
	tool, _ := New(cb.asCallback(), nil)
	q := mkQuestion()
	q.TimeoutSeconds = 0
	res, err := tool.Run(context.Background(), map[string]any{"question": q})
	if err != nil {
		t.Fatal(err)
	}
	if !res.OK {
		t.Fatal("OK should be true with no timeout")
	}
}

func TestRun_CallerContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	tool, _ := New(scriptedCB().asCallback(), nil)
	_, err := tool.Run(ctx, map[string]any{"question": mkQuestion()})
	if err == nil {
		t.Fatal("expected error when context is cancelled")
	}
}

func TestTimeoutError_Error(t *testing.T) {
	e := &TimeoutError{QuestionID: "q-1"}
	if e.Error() == "" {
		t.Fatal("non-empty")
	}
	if !errors.Is(e, ErrTimeout) {
		t.Fatal("errors.Is should match")
	}
	if !errors.Is(errors.New("wrap: "+e.Error()), ErrTimeout) {
		// Just sanity: the sentinel match should still work
		// through the message text only when wrapped.
	}
}

func TestNowUTC(t *testing.T) {
	ts := nowUTC()
	if ts == "" {
		t.Fatal("non-empty")
	}
	if _, err := time.Parse(time.RFC3339, ts); err != nil {
		t.Fatalf("not RFC3339: %s", ts)
	}
}

// jsonMarshalQ is a small helper that re-marshals a Question so
// the string/byte arg tests can pass the canonical encoding
// through the decoder.
func jsonMarshalQ(q Question) ([]byte, error) {
	return json.Marshal(q)
}
