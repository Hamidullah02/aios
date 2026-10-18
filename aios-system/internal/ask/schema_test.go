package ask

import (
	"encoding/json"
	"strings"
	"testing"
)

func goodQuestion() Question {
	return Question{
		ID:     "q-1",
		Prompt: "Which database?",
		Options: []AskOption{
			{ID: "pg", Label: "PostgreSQL", Description: "production"},
			{ID: "sqlite", Label: "SQLite", Description: "dev"},
		},
	}
}

func TestQuestion_Validate_Good(t *testing.T) {
	q := goodQuestion()
	if err := q.Validate(); err != nil {
		t.Fatalf("want nil, got %v", err)
	}
}

func TestQuestion_Validate_MissingID(t *testing.T) {
	q := goodQuestion()
	q.ID = ""
	if err := q.Validate(); err == nil {
		t.Fatal("expected error for empty id")
	}
}

func TestQuestion_Validate_MissingPrompt(t *testing.T) {
	q := goodQuestion()
	q.Prompt = ""
	if err := q.Validate(); err == nil {
		t.Fatal("expected error for empty prompt")
	}
}

func TestQuestion_Validate_NoOptions(t *testing.T) {
	q := goodQuestion()
	q.Options = nil
	if err := q.Validate(); err == nil {
		t.Fatal("expected error for empty options")
	}
}

func TestQuestion_Validate_NegativeTimeout(t *testing.T) {
	q := goodQuestion()
	q.TimeoutSeconds = -1
	if err := q.Validate(); err == nil {
		t.Fatal("expected error for negative timeout")
	}
}

func TestQuestion_Validate_DefaultUnknown(t *testing.T) {
	q := goodQuestion()
	q.DefaultOption = "mysql"
	if err := q.Validate(); err == nil {
		t.Fatal("expected error for unknown default")
	}
}

func TestQuestion_Validate_DefaultOK(t *testing.T) {
	q := goodQuestion()
	q.DefaultOption = "pg"
	if err := q.Validate(); err != nil {
		t.Fatalf("want nil, got %v", err)
	}
}

func TestQuestion_Validate_DuplicateOptionID(t *testing.T) {
	q := goodQuestion()
	q.Options = append(q.Options, AskOption{ID: "pg", Label: "x"})
	if err := q.Validate(); err == nil {
		t.Fatal("expected error for duplicate option id")
	}
}

func TestQuestion_Validate_EmptyOptionID(t *testing.T) {
	q := goodQuestion()
	q.Options[0].ID = ""
	if err := q.Validate(); err == nil {
		t.Fatal("expected error for empty option id")
	}
}

func TestQuestion_Validate_EmptyOptionLabel(t *testing.T) {
	q := goodQuestion()
	q.Options[0].Label = ""
	if err := q.Validate(); err == nil {
		t.Fatal("expected error for empty option label")
	}
}

func TestQuestion_Marshal_RoundTrip(t *testing.T) {
	q := goodQuestion()
	raw, err := json.Marshal(q)
	if err != nil {
		t.Fatal(err)
	}
	var back Question
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatal(err)
	}
	if back.ID != q.ID || back.Prompt != q.Prompt || len(back.Options) != len(q.Options) {
		t.Fatalf("round-trip mismatch: %+v", back)
	}
}

func TestQuestion_Marshal_KeysSorted(t *testing.T) {
	q := goodQuestion()
	raw, err := json.Marshal(q)
	if err != nil {
		t.Fatal(err)
	}
	// The first key in the object should be "id" (sorted).
	if !strings.HasPrefix(string(raw), `{"id":`) {
		t.Errorf("keys not sorted: %s", string(raw))
	}
}

func TestQuestion_Unmarshal_RejectsUnknownField(t *testing.T) {
	raw := []byte(`{"id":"q-1","prompt":"x","options":[{"id":"a","label":"A"}],"unknown":"x"}`)
	var q Question
	if err := json.Unmarshal(raw, &q); err == nil {
		t.Fatal("expected error for unknown field")
	}
}

func TestAnswer_Validate_Good(t *testing.T) {
	a := Answer{QuestionID: "q-1", SelectedOption: "pg", Timestamp: "2024-01-01T00:00:00Z"}
	if err := a.Validate(); err != nil {
		t.Fatalf("want nil, got %v", err)
	}
}

func TestAnswer_Validate_MissingQuestionID(t *testing.T) {
	a := Answer{SelectedOption: "pg", Timestamp: "x"}
	if err := a.Validate(); err == nil {
		t.Fatal("expected error for missing question_id")
	}
}

func TestAnswer_Validate_MissingSelectedOption(t *testing.T) {
	a := Answer{QuestionID: "q-1", Timestamp: "x"}
	if err := a.Validate(); err == nil {
		t.Fatal("expected error for missing selected_option")
	}
}

func TestAnswer_Validate_MissingTimestamp(t *testing.T) {
	a := Answer{QuestionID: "q-1", SelectedOption: "pg"}
	if err := a.Validate(); err == nil {
		t.Fatal("expected error for missing timestamp")
	}
}

func TestAnswer_Marshal_RoundTrip(t *testing.T) {
	a := Answer{QuestionID: "q-1", SelectedOption: "pg", Timestamp: "t"}
	raw, err := json.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	var back Answer
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatal(err)
	}
	if back != a {
		t.Fatalf("round-trip mismatch: %+v", back)
	}
}

func TestAnswer_Unmarshal_RejectsUnknownField(t *testing.T) {
	raw := []byte(`{"question_id":"q-1","selected_option":"pg","timestamp":"t","extra":1}`)
	var a Answer
	if err := json.Unmarshal(raw, &a); err == nil {
		t.Fatal("expected error for unknown field")
	}
}

func TestJoinStrings(t *testing.T) {
	cases := map[string]string{
		"":            "",
		"a":           "a",
		"a;b;c":       "a;b;c", // single element with separator chars
		"a,b,c → sep": "a,b,c", // not the function
	}
	// Use real calls instead of the map since the keys are
	// shape values, not inputs.
	if got := joinStrings(nil, ";"); got != "" {
		t.Errorf("nil: %q", got)
	}
	if got := joinStrings([]string{"x"}, ";"); got != "x" {
		t.Errorf("one: %q", got)
	}
	if got := joinStrings([]string{"a", "b", "c"}, ";"); got != "a;b;c" {
		t.Errorf("many: %q", got)
	}
	if got := joinStrings([]string{"a,b,c"}, ";"); got != "a,b,c" {
		t.Errorf("single with separator: %q", got)
	}
	_ = cases // silence unused
}
