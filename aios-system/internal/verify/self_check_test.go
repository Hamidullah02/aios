package verify

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Hamidullah02/aios/aios-system/internal/core/audit"
	"github.com/Hamidullah02/aios/aios-system/internal/core/clock"
	"github.com/Hamidullah02/aios/aios-system/internal/tools"
)

func TestSelfCheckTool_Name(t *testing.T) {
	if (SelfCheckTool{}).Name() != "self_check" {
		t.Errorf("name mismatch")
	}
}

func TestNewSelfCheckTool_NoJudge(t *testing.T) {
	if _, err := NewSelfCheckTool(SelfCheckConfig{}); err == nil {
		t.Errorf("expected error for nil judge")
	}
}

func TestNewSelfCheckTool_DefaultPrompt(t *testing.T) {
	tl := newVerifyTestLog(t)
	tool, err := NewSelfCheckTool(SelfCheckConfig{
		Judge: JudgeFunc(func(_ context.Context, _ string) (string, error) {
			return `{"status":"pass","reason":"ok"}`, nil
		}),
		Audit: tl,
	})
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if tool.Prompt == nil {
		t.Errorf("default prompt not applied")
	}
	if tool.Audit == nil {
		t.Errorf("audit not propagated")
	}
}

func TestSelfCheckTool_Run_MissingArgs(t *testing.T) {
	tool, _ := NewSelfCheckTool(SelfCheckConfig{
		Judge: JudgeFunc(func(_ context.Context, _ string) (string, error) { return "", nil }),
	})
	if _, err := tool.Run(map[string]any{}, tools.Context{}); err == nil {
		t.Errorf("expected missing result error")
	}
	if _, err := tool.Run(map[string]any{"result": "x"}, tools.Context{}); err == nil {
		t.Errorf("expected missing goal error")
	}
}

func TestSelfCheckTool_Run_Happy(t *testing.T) {
	tool, _ := NewSelfCheckTool(SelfCheckConfig{
		Judge: JudgeFunc(func(_ context.Context, prompt string) (string, error) {
			// Confirm the prompt contains both
			// the result and the goal.
			if !strings.Contains(prompt, "agent did the following work") {
				t.Errorf("prompt missing result section")
			}
			if !strings.Contains(prompt, "goal was") {
				t.Errorf("prompt missing goal section")
			}
			return `{"status":"pass","reason":"looks good"}`, nil
		}),
	})
	out, err := tool.Run(map[string]any{
		"result": "the work",
		"goal":   "the goal",
	}, tools.Context{})
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if !out.OK {
		t.Errorf("expected OK")
	}
	if got := out.Output["status"]; got != "pass" {
		t.Errorf("status: %v", got)
	}
}

func TestSelfCheckTool_Run_CustomPrompt(t *testing.T) {
	custom, err := NewPromptTemplate("verifier: {{.Result}} vs {{.Goal}}")
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	tool, _ := NewSelfCheckTool(SelfCheckConfig{
		Judge: JudgeFunc(func(_ context.Context, prompt string) (string, error) {
			if !strings.HasPrefix(prompt, "verifier:") {
				t.Errorf("custom prompt not used: %q", prompt[:30])
			}
			return `{"status":"fail","reason":"bad","suggestions":["fix it"]}`, nil
		}),
		Prompt: custom,
	})
	out, err := tool.Run(map[string]any{"result": "x", "goal": "y"}, tools.Context{})
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if got := out.Output["status"]; got != "fail" {
		t.Errorf("status: %v", got)
	}
}

func TestSelfCheckTool_Run_JudgeError(t *testing.T) {
	tool, _ := NewSelfCheckTool(SelfCheckConfig{
		Judge: JudgeFunc(func(_ context.Context, _ string) (string, error) {
			return "", errors.New("network down")
		}),
	})
	_, err := tool.Run(map[string]any{"result": "x", "goal": "y"}, tools.Context{})
	if err == nil {
		t.Fatalf("expected error")
	}
	if !errors.Is(err, ErrVerification) {
		t.Errorf("expected ErrVerification, got %v", err)
	}
}

func TestSelfCheckTool_Run_ParseError(t *testing.T) {
	tool, _ := NewSelfCheckTool(SelfCheckConfig{
		Judge: JudgeFunc(func(_ context.Context, _ string) (string, error) {
			return "no JSON here", nil
		}),
	})
	_, err := tool.Run(map[string]any{"result": "x", "goal": "y"}, tools.Context{})
	if err == nil {
		t.Fatalf("expected parse error")
	}
	if !errors.Is(err, ErrVerification) {
		t.Errorf("expected ErrVerification, got %v", err)
	}
}

func TestSelfCheckTool_Run_ValidateError(t *testing.T) {
	tool, _ := NewSelfCheckTool(SelfCheckConfig{
		Judge: JudgeFunc(func(_ context.Context, _ string) (string, error) {
			return `{"status":"bad","reason":"x"}`, nil
		}),
	})
	_, err := tool.Run(map[string]any{"result": "x", "goal": "y"}, tools.Context{})
	if err == nil {
		t.Fatalf("expected validate error")
	}
	if !errors.Is(err, ErrVerification) {
		t.Errorf("expected ErrVerification, got %v", err)
	}
}

func TestSelfCheckTool_Run_Audit(t *testing.T) {
	tl := newVerifyTestLog(t)
	tool, _ := NewSelfCheckTool(SelfCheckConfig{
		Judge: JudgeFunc(func(_ context.Context, _ string) (string, error) {
			return `{"status":"pass","reason":"ok"}`, nil
		}),
		Audit: tl,
	})
	if _, err := tool.Run(map[string]any{"result": "x", "goal": "y"}, tools.Context{Actor: "actor", TaskID: "task"}); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	rows, err := tl.GetByTask("task")
	if err != nil {
		t.Fatalf("audit read: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if rows[0].Kind != "self_check" {
		t.Errorf("kind: %s", rows[0].Kind)
	}
	if rows[0].TaskID != "task" {
		t.Errorf("task id: %s", rows[0].TaskID)
	}
	if rows[0].Actor != "actor" {
		t.Errorf("actor: %s", rows[0].Actor)
	}
	if len(rows[0].Payload) == 0 {
		t.Errorf("expected payload bytes")
	}
	// Confirm the payload contains the
	// status we set. The payload is stored
	// as canonical JSON.
	if !bytesContains(rows[0].Payload, []byte(`"status":"pass"`)) {
		t.Errorf("expected pass in payload, got: %s", rows[0].Payload)
	}
}

func TestSelfCheckTool_Run_AuditOnError(t *testing.T) {
	tl := newVerifyTestLog(t)
	tool, _ := NewSelfCheckTool(SelfCheckConfig{
		Judge: JudgeFunc(func(_ context.Context, _ string) (string, error) {
			return "", errors.New("boom")
		}),
		Audit: tl,
	})
	_, _ = tool.Run(map[string]any{"result": "x", "goal": "y"}, tools.Context{TaskID: "task"})
	rows, _ := tl.GetByTask("task")
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if !bytesContains(rows[0].Payload, []byte(`"error"`)) {
		t.Errorf("expected error key in payload, got: %s", rows[0].Payload)
	}
}

func TestSelfCheckTool_NoAuditWhenNil(t *testing.T) {
	tool, _ := NewSelfCheckTool(SelfCheckConfig{
		Judge: JudgeFunc(func(_ context.Context, _ string) (string, error) {
			return `{"status":"pass","reason":"ok"}`, nil
		}),
	})
	if _, err := tool.Run(map[string]any{"result": "x", "goal": "y"}, tools.Context{}); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
}

func TestParseVerificationResponse_Bare(t *testing.T) {
	r, err := ParseVerificationResponse(`{"status":"pass","reason":"x"}`)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if r.Status != StatusPass {
		t.Errorf("status: %s", r.Status)
	}
}

func TestParseVerificationResponse_Fenced(t *testing.T) {
	r, err := ParseVerificationResponse("```json\n{\"status\":\"fail\",\"reason\":\"x\",\"suggestions\":[\"a\"]}\n```")
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if r.Status != StatusFail {
		t.Errorf("status: %s", r.Status)
	}
	if len(r.Suggestions) != 1 {
		t.Errorf("suggestions: %v", r.Suggestions)
	}
}

func TestParseVerificationResponse_ProseWrapped(t *testing.T) {
	r, err := ParseVerificationResponse("Sure! Here you go: {\"status\":\"inconclusive\",\"reason\":\"maybe\"}. Done.")
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if r.Status != StatusInconclusive {
		t.Errorf("status: %s", r.Status)
	}
}

func TestParseVerificationResponse_Empty(t *testing.T) {
	if _, err := ParseVerificationResponse(""); err == nil {
		t.Errorf("expected error")
	}
}

func TestParseVerificationResponse_NoJSON(t *testing.T) {
	if _, err := ParseVerificationResponse("nothing here"); err == nil {
		t.Errorf("expected error")
	}
}

func TestParseVerificationResponse_Unbalanced(t *testing.T) {
	if _, err := ParseVerificationResponse("{\"status\":\"pass\""); err == nil {
		t.Errorf("expected error")
	}
}

func TestParseVerificationResponse_InvalidJSON(t *testing.T) {
	if _, err := ParseVerificationResponse("{status: pass}"); err == nil {
		t.Errorf("expected error")
	}
}

func TestParseVerificationResponse_ValidateFail(t *testing.T) {
	if _, err := ParseVerificationResponse(`{"status":"bad","reason":"x"}`); err == nil {
		t.Errorf("expected error")
	}
}

func TestExtractJSON_Nested(t *testing.T) {
	// Nested objects inside strings should not
	// confuse the simple depth counter because
	// the test response never contains nested
	// braces in the JSON itself. We still
	// confirm a deeper object is recovered.
	in := `prefix {"status":"pass","meta":{"k":1}} suffix`
	got := extractJSON(in)
	if got != `{"status":"pass","meta":{"k":1}}` {
		t.Errorf("got %q", got)
	}
}

func TestStripFences_NoFences(t *testing.T) {
	if got := stripFences("hello\nworld"); got != "hello\nworld" {
		t.Errorf("got %q", got)
	}
}

func TestStripFences_Opens(t *testing.T) {
	// Unbalanced fences: the stripper toggles
	// on every line that starts with ```.
	if got := stripFences("```\nhello\nworld"); got != "hello\nworld" {
		t.Errorf("got %q", got)
	}
}

func TestJudgeFunc_Adapter(t *testing.T) {
	var calls int32
	f := JudgeFunc(func(_ context.Context, p string) (string, error) {
		atomic.AddInt32(&calls, 1)
		return p, nil
	})
	got, err := f.Judge(context.Background(), "hi")
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if got != "hi" {
		t.Errorf("got %q", got)
	}
	if atomic.LoadInt32(&calls) != 1 {
		t.Errorf("calls: %d", calls)
	}
}

// newVerifyTestLog is the audit-log fixture the
// verify tests share. The fixture is a copy of
// the helper in the audit package's test file;
// we re-declare it here so the verify tests
// stay self-contained.
func newVerifyTestLog(t *testing.T) *audit.Log {
	t.Helper()
	log, err := audit.Open(audit.Config{
		Path:  ":memory:",
		Clock: clock.FixedDecision{T: time.Unix(1700000000, 0)},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { log.Close() })
	return log
}

// bytesContains reports whether haystack
// contains needle. The helper is the
// payload-content check the audit tests
// use.
func bytesContains(haystack, needle []byte) bool {
	return bytes.Contains(haystack, needle)
}
