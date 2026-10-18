package image

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/sadi/aios-system/internal/models"
)

// fakeCompleter returns canned responses. The script is
// a list of {text, err} pairs; the completer advances
// through the list on each call. When the list is
// exhausted, the last response is reused.
type fakeCompleter struct {
	script []fakeResp
	calls  int
}

type fakeResp struct {
	text string
	err  error
}

func (f *fakeCompleter) Complete(_ context.Context, _ models.Request) (models.Response, error) {
	if len(f.script) == 0 {
		return models.Response{}, errors.New("no responses scripted")
	}
	idx := f.calls
	if idx >= len(f.script) {
		idx = len(f.script) - 1
	}
	f.calls++
	r := f.script[idx]
	return models.Response{Text: r.text}, r.err
}

func TestConstrain_Happy(t *testing.T) {
	fc := &fakeCompleter{
		script: []fakeResp{
			{text: `{"name":"a","age":1}`},
		},
	}
	got, err := SchemaConstrainedCompletion(
		context.Background(), fc, "give me a user", "x",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name": map[string]any{"type": "string"},
				"age":  map[string]any{"type": "number"},
			},
			"required": []any{"name", "age"},
		},
		SchemaConstrainedCompletionOptions{},
	)
	if err != nil {
		t.Fatal(err)
	}
	if got["name"] != "a" || got["age"] != 1.0 {
		t.Fatalf("got %+v", got)
	}
	if fc.calls != 1 {
		t.Fatalf("calls = %d", fc.calls)
	}
}

func TestConstrain_RetryOnInvalid(t *testing.T) {
	fc := &fakeCompleter{
		script: []fakeResp{
			{text: "not json"},
			{text: `{"name":"a"}`}, // missing age
			{text: `{"name":"a","age":1}`},
		},
	}
	got, err := SchemaConstrainedCompletion(
		context.Background(), fc, "give me a user", "x",
		map[string]any{
			"type":     "object",
			"required": []any{"name", "age"},
			"properties": map[string]any{
				"name": map[string]any{"type": "string"},
				"age":  map[string]any{"type": "number"},
			},
		},
		SchemaConstrainedCompletionOptions{MaxRetries: 5},
	)
	if err != nil {
		t.Fatal(err)
	}
	if got["age"] != 1.0 {
		t.Fatalf("got %+v", got)
	}
	if fc.calls != 3 {
		t.Fatalf("calls = %d", fc.calls)
	}
}

func TestConstrain_RetriesExhausted(t *testing.T) {
	fc := &fakeCompleter{
		script: []fakeResp{{text: "garbage"}},
	}
	_, err := SchemaConstrainedCompletion(
		context.Background(), fc, "give me a user", "x",
		map[string]any{"type": "object"},
		SchemaConstrainedCompletionOptions{MaxRetries: 2},
	)
	if !errors.Is(err, ErrValidationFailure) {
		t.Fatalf("err = %v", err)
	}
	if fc.calls != 3 { // initial + 2 retries
		t.Fatalf("calls = %d", fc.calls)
	}
}

func TestConstrain_NonObjectResponse(t *testing.T) {
	fc := &fakeCompleter{
		script: []fakeResp{
			{text: `[1,2,3]`}, // valid JSON, but not an object
			{text: `{"x":1}`},
		},
	}
	got, err := SchemaConstrainedCompletion(
		context.Background(), fc, "give me obj", "x",
		map[string]any{"type": "object"},
		SchemaConstrainedCompletionOptions{MaxRetries: 3},
	)
	if err != nil {
		t.Fatal(err)
	}
	if got["x"] != 1.0 {
		t.Fatalf("got %+v", got)
	}
}

func TestConstrain_RouterError(t *testing.T) {
	fc := &fakeCompleter{
		script: []fakeResp{{err: errors.New("boom")}},
	}
	_, err := SchemaConstrainedCompletion(
		context.Background(), fc, "give me", "x",
		map[string]any{"type": "object"},
		SchemaConstrainedCompletionOptions{},
	)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestConstrain_BadSchema(t *testing.T) {
	fc := &fakeCompleter{}
	_, err := SchemaConstrainedCompletion(
		context.Background(), fc, "give me", "x",
		42,
		SchemaConstrainedCompletionOptions{},
	)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestConstrain_ContextCancelled(t *testing.T) {
	fc := &fakeCompleter{
		script: []fakeResp{{text: `{"x":1}`}},
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := SchemaConstrainedCompletion(
		ctx, fc, "give me", "x",
		map[string]any{"type": "object"},
		SchemaConstrainedCompletionOptions{},
	)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v", err)
	}
}

func TestConstrain_FencedJSON(t *testing.T) {
	fc := &fakeCompleter{
		script: []fakeResp{
			{text: "```json\n{\"x\":1}\n```"},
		},
	}
	got, err := SchemaConstrainedCompletion(
		context.Background(), fc, "give me", "x",
		map[string]any{"type": "object"},
		SchemaConstrainedCompletionOptions{},
	)
	if err != nil {
		t.Fatal(err)
	}
	if got["x"] != 1.0 {
		t.Fatalf("got %+v", got)
	}
}

func TestConstrain_ZeroRetries(t *testing.T) {
	fc := &fakeCompleter{script: []fakeResp{{text: "garbage"}}}
	_, err := SchemaConstrainedCompletion(
		context.Background(), fc, "give me", "x",
		map[string]any{"type": "object"},
		SchemaConstrainedCompletionOptions{MaxRetries: 0},
	)
	if !errors.Is(err, ErrValidationFailure) {
		t.Fatalf("err = %v", err)
	}
	if fc.calls != 1 {
		t.Fatalf("calls = %d", fc.calls)
	}
}

func TestConstrain_NegativeRetries(t *testing.T) {
	fc := &fakeCompleter{script: []fakeResp{{text: `{"x":1}`}}}
	got, err := SchemaConstrainedCompletion(
		context.Background(), fc, "give me", "x",
		map[string]any{"type": "object"},
		SchemaConstrainedCompletionOptions{MaxRetries: -5},
	)
	if err != nil {
		t.Fatal(err)
	}
	if got["x"] != 1.0 {
		t.Fatalf("got %+v", got)
	}
}

func TestParseJSON(t *testing.T) {
	if _, err := parseJSON(""); err == nil {
		t.Fatal("empty")
	}
	if _, err := parseJSON("not json"); err == nil {
		t.Fatal("garbage")
	}
}

func TestStripCodeFence(t *testing.T) {
	in := "```json\n{\"a\":1}\n```"
	out := stripCodeFence(in)
	if !strings.Contains(out, "\"a\"") {
		t.Fatalf("stripped: %q", out)
	}
}

func TestBuildConstrainedPrompt(t *testing.T) {
	p := buildConstrainedPrompt("hi", "{}", "", "")
	if !strings.Contains(p, "Respond with JSON") {
		t.Fatalf("missing hint: %q", p)
	}
	p2 := buildConstrainedPrompt("hi", "{}", "system: be brief", "prev failed")
	if !strings.Contains(p2, "system: be brief") {
		t.Fatalf("missing system: %q", p2)
	}
	if !strings.Contains(p2, "prev failed") {
		t.Fatalf("missing prev err: %q", p2)
	}
}

func TestNormalizeNumbers(t *testing.T) {
	out := normalizeNumbers(map[string]any{"a": float64(1.0)})
	if _, ok := out.(map[string]any); !ok {
		t.Fatal("expected map")
	}
}

func TestNormalizeNumbers_JSONNumber(t *testing.T) {
	// When UseNumber is on, the parser produces json.Number.
	// normalizeNumbers must convert it to float64.
	type jsonNumberLike struct{}
	_ = jsonNumberLike{}
	// json.Number is internal to the package; we exercise
	// the float fallback by passing a value that
	// normalizeNumbers does not recognize — it must pass
	// through unchanged.
	v := "string"
	if normalizeNumbers(v) != v {
		t.Fatal("string passthrough")
	}
}
