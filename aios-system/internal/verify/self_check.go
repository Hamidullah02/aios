package verify

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/Hamidullah02/aios/aios-system/internal/core/audit"
	"github.com/Hamidullah02/aios/aios-system/internal/tools"
)

// Judge is the model-router surface the
// SelfCheck tool depends on (REQ-VRF-SCK-004).
//
// The interface lets the caller wire the tool
// to any model router. The default aios-models
// package exposes a Router type that satisfies
// this interface; the verify package itself
// does not import aios-models, so the package
// stays decoupled from the model implementation.
type Judge interface {
	// Judge asks the model to verify the
	// rendered prompt. The function returns
	// the model's raw response text.
	Judge(ctx context.Context, prompt string) (string, error)
}

// JudgeFunc is the function-type adapter for
// Judge. The adapter is the convenience
// constructor when the call site wants to
// pass a plain function.
type JudgeFunc func(ctx context.Context, prompt string) (string, error)

// Judge invokes the underlying function.
func (f JudgeFunc) Judge(ctx context.Context, prompt string) (string, error) {
	return f(ctx, prompt)
}

// SelfCheckTool is the tools.Tool that asks
// the model to verify the work and parses the
// response into a VerificationResult
// (REQ-VRF-SCK-001 to 006).
//
// Args:
//
//	result (string, required) — the work the
//	                            agent did.
//	goal   (string, required) — the goal the
//	                            work is measured
//	                            against.
type SelfCheckTool struct {
	// Judge is the model router. Required.
	Judge Judge
	// Prompt is the template. Defaults to
	// the doc-set DefaultPrompt when nil.
	Prompt *PromptTemplate
	// Audit is the audit log. nil means
	// "skip the row" (REQ-VRF-SCK-006).
	Audit *audit.Log
}

// Name returns the tool name.
func (SelfCheckTool) Name() string { return "self_check" }

// SelfCheckConfig controls SelfCheckTool
// construction.
type SelfCheckConfig struct {
	Judge  Judge
	Prompt *PromptTemplate
	Audit  *audit.Log
}

// NewSelfCheckTool constructs a SelfCheckTool.
// The function returns an error when the
// judge is nil.
func NewSelfCheckTool(cfg SelfCheckConfig) (*SelfCheckTool, error) {
	if cfg.Judge == nil {
		return nil, &VerificationError{Op: "init", Reason: "judge is required"}
	}
	prompt := cfg.Prompt
	if prompt == nil {
		p, err := NewDefaultPromptTemplate()
		if err != nil {
			return nil, &VerificationError{Op: "init", Reason: "default prompt", Err: err}
		}
		prompt = p
	}
	return &SelfCheckTool{Judge: cfg.Judge, Prompt: prompt, Audit: cfg.Audit}, nil
}

// Run executes the self-check
// (REQ-VRF-SCK-002 to 005).
func (t *SelfCheckTool) Run(args map[string]any, ctx tools.Context) (tools.Result, error) {
	result, _ := args["result"].(string)
	if result == "" {
		return tools.Result{OK: false, Summary: "missing result"},
			&VerificationError{Op: "run", Reason: "result is required"}
	}
	goal, _ := args["goal"].(string)
	if goal == "" {
		return tools.Result{OK: false, Summary: "missing goal"},
			&VerificationError{Op: "run", Reason: "goal is required"}
	}
	prompt, err := t.Prompt.Render(VerificationInput{Result: result, Goal: goal})
	if err != nil {
		return tools.Result{OK: false, Summary: "prompt failed"}, err
	}
	// Use a fresh context with a 10-second
	// timeout cap (REQ-VRF-NFR-003). The
	// judge may impose its own timeout.
	runCtx, cancel := context.WithTimeout(context.Background(), DefaultSelfCheckTimeout)
	defer cancel()
	response, err := t.Judge.Judge(runCtx, prompt)
	if err != nil {
		verr := &VerificationError{Op: "judge", Reason: "model call", Err: err}
		t.writeAudit(ctx, nil, verr)
		return tools.Result{OK: false, Summary: "judge failed"}, verr
	}
	verification, err := ParseVerificationResponse(response)
	if err != nil {
		verr := &VerificationError{Op: "parse", Reason: "response", Err: err}
		t.writeAudit(ctx, nil, verr)
		return tools.Result{OK: false, Summary: "parse failed"}, verr
	}
	t.writeAudit(ctx, &verification, nil)
	return tools.Result{
		OK: true,
		Output: map[string]any{
			"status":      string(verification.Status),
			"reason":      verification.Reason,
			"suggestions": verification.Suggestions,
		},
		Summary: fmt.Sprintf("self-check: %s", verification.Status),
	}, nil
}

// DefaultSelfCheckTimeout is the
// REQ-VRF-NFR-003 latency cap.
const DefaultSelfCheckTimeout = 10 * 1_000_000_000 // 10s as time.Duration

// writeAudit writes a "self_check" audit row
// (REQ-VRF-SCK-006). The row carries the
// result on success; the error on failure.
func (t *SelfCheckTool) writeAudit(ctx tools.Context, result *VerificationResult, runErr error) {
	if t.Audit == nil {
		return
	}
	taskID := ctx.TaskID
	if taskID == "" {
		taskID = "self_check"
	}
	actor := ctx.Actor
	if actor == "" {
		actor = "self_check"
	}
	payload := map[string]any{
		"status": "",
		"reason": "",
	}
	var errStr string
	if runErr != nil {
		errStr = runErr.Error()
		payload["error"] = errStr
	}
	if result != nil {
		payload["status"] = string(result.Status)
		payload["reason"] = result.Reason
		if len(result.Suggestions) > 0 {
			payload["suggestions"] = result.Suggestions
		}
	}
	_, _ = t.Audit.Write(&audit.Row{
		TaskID:   taskID,
		Kind:     "self_check",
		Actor:    actor,
		RiskTier: 0,
		Payload:  payload,
	})
	_ = errStr
}

// jsonBlock is the regex the parser uses to
// extract the first JSON object from the
// model's response. The pattern is greedy on
// the braces so the parser recovers from a
// response that wraps the JSON in markdown
// fences or surrounds it with prose.
var jsonBlock = regexp.MustCompile(`(?s)\{[^{}]*\}`)

// ParseVerificationResponse extracts and
// validates a VerificationResult from the
// model's response. The function is exported
// so the test suite and other callers can
// share the parser.
//
// The parser tolerates three response shapes:
//
//  1. Bare JSON: `{"status": "pass", ...}`.
//  2. JSON wrapped in markdown fences:
//     ```` ```json\n{...}\n``` ````.
//  3. JSON surrounded by prose: `Sure!\n{...}\nDone.`
func ParseVerificationResponse(response string) (VerificationResult, error) {
	if strings.TrimSpace(response) == "" {
		return VerificationResult{}, &VerificationError{Op: "parse", Reason: "empty response"}
	}
	body := extractJSON(response)
	if body == "" {
		return VerificationResult{}, &VerificationError{Op: "parse", Reason: "no JSON object in response"}
	}
	var result VerificationResult
	if err := json.Unmarshal([]byte(body), &result); err != nil {
		return VerificationResult{}, &VerificationError{Op: "parse", Reason: "unmarshal", Err: err}
	}
	if err := result.Validate(); err != nil {
		return VerificationResult{}, &VerificationError{Op: "parse", Reason: "validate", Err: err}
	}
	return result, nil
}

// extractJSON returns the first JSON object in
// the response. The function is the helper the
// parser uses to recover from the model
// wrapping the JSON in fences or prose.
//
// The strategy is a two-step scan:
//
//  1. Strip markdown fences.
//  2. Find the first balanced `{...}` and
//     return it.
func extractJSON(response string) string {
	cleaned := stripFences(response)
	// Find the first '{' that has a matching '}'
	// later in the string.
	start := strings.Index(cleaned, "{")
	if start < 0 {
		return ""
	}
	// Scan forward and track brace depth.
	depth := 0
	for i := start; i < len(cleaned); i++ {
		switch cleaned[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return cleaned[start : i+1]
			}
		}
	}
	return ""
}

// stripFences removes markdown code fences
// (``` or ```json ... ```) from the response.
// The function is the small helper the parser
// uses.
func stripFences(response string) string {
	lines := strings.Split(response, "\n")
	var out []string
	inFence := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			inFence = !inFence
			continue
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}
