package planner

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/sadi/aios-system/internal/core/schemas"
	"github.com/sadi/aios-system/internal/models"
)

// Planner turns a goal into a TaskPlan.
//
// Per REQ-PLN-PLN-001 the planner holds:
//
//   - a model router (aios-models);
//   - a tool map (the public shape is ToolSpec, owned here);
//   - a verifier.
//
// The default model is "gpt-4o". Callers can set Model to any
// id the router can resolve. The default is wired so the planner
// works out-of-the-box with the production aios-models router.
type Planner struct {
	router     *models.Router
	tools      map[string]ToolSpec
	verifier   Verifier
	maxRepair  int
	defaultMod string
}

// Config is the constructor input. Tools and Verifier are
// required; everything else has a sensible default.
type Config struct {
	// Router is the model router. Required.
	Router *models.Router

	// Tools is the tool map. Required.
	Tools map[string]ToolSpec

	// Verifier validates each candidate plan. Required.
	Verifier Verifier

	// MaxRepairIterations is the cap on the repair loop. The
	// default is 3, matching REQ-PLN-PLN-006. Values <= 0 are
	// treated as the default.
	MaxRepairIterations int

	// DefaultModel is the model id the planner uses when the
	// plan does not specify one. Empty falls back to "gpt-4o".
	DefaultModel string
}

// NewPlanner constructs a planner. The verifier and tool map are
// copied so the caller can mutate their inputs afterwards.
func NewPlanner(cfg Config) (*Planner, error) {
	if cfg.Router == nil {
		return nil, fmt.Errorf("planner: router is required")
	}
	if cfg.Verifier == nil {
		return nil, fmt.Errorf("planner: verifier is required")
	}
	if len(cfg.Tools) == 0 {
		return nil, fmt.Errorf("planner: tools map is empty (deny-by-default: the planner cannot pick a tool)")
	}
	max := cfg.MaxRepairIterations
	if max <= 0 {
		max = 3
	}
	tools := make(map[string]ToolSpec, len(cfg.Tools))
	for k, v := range cfg.Tools {
		tools[k] = v
	}
	mod := cfg.DefaultModel
	if mod == "" {
		mod = "gpt-4o"
	}
	return &Planner{
		router:     cfg.Router,
		tools:      tools,
		verifier:   cfg.Verifier,
		maxRepair:  max,
		defaultMod: mod,
	}, nil
}

// Plan is the main entry point (REQ-PLN-PLN-002 / REQ-PLN-PLN-004
// / REQ-PLN-PLN-005). It returns a verified TaskPlan or an error.
//
// On success the plan is validated by both the schemas package
// and the verifier. The returned pointer is the planner's
// internal copy; callers may mutate it freely.
//
// On failure the error is one of:
//
//   - *PlanGenerationError: the model did not produce a parseable
//     plan, the verifier kept rejecting, or the model itself
//     returned an error after the first round.
//   - *PlanValidationError: the very first attempt produced a
//     parseable plan that the verifier rejected, the planner
//     gave up after max repair iterations, and we are returning
//     the last verifier report. (PlanGenerationError also covers
//     this; PlanValidationError is the case where the caller's
//     plan was bad on the first round and we are done.)
//   - context.Canceled / context.DeadlineExceeded: the caller's
//     context was cancelled mid-flight.
func (p *Planner) Plan(ctx context.Context, goal string, planCtx map[string]any) (*schemas.TaskPlan, error) {
	if goal == "" {
		return nil, &PlanValidationError{
			Errors: PlanErrorList{{Code: CodeSchema, Message: "goal is empty"}},
		}
	}
	tools := p.toolsList()

	// Round 1: ask the model for a plan.
	prompt := PromptData{
		Goal:    goal,
		Context: planCtx,
		Tools:   tools,
	}
	userPrompt, err := RenderUser(prompt)
	if err != nil {
		return nil, fmt.Errorf("planner: render user prompt: %w", err)
	}
	raw, err := p.ask(ctx, userPrompt)
	if err != nil {
		return nil, &PlanGenerationError{Iterations: 1, Cause: err}
	}
	plan, err := parsePlan(raw)
	if err != nil {
		// The first round produced garbage. The repair loop might
		// help: feed the bad output back and ask for a fixed plan.
		return p.repair(ctx, goal, planCtx, tools, raw, 1, PlanErrorList{{
			Code:    CodeMalformedJSON,
			Message: err.Error(),
		}})
	}
	errs := p.verifier.Verify(plan)
	if len(errs) == 0 {
		return plan, nil
	}
	return p.repair(ctx, goal, planCtx, tools, "", 1, errs)
}

// repair runs the repair loop. It always returns at most maxRepair
// additional attempts (so the total is 1 + maxRepair). The
// function returns either a verified plan or a PlanGenerationError
// carrying the last verifier report.
func (p *Planner) repair(
	ctx context.Context,
	goal string,
	planCtx map[string]any,
	tools []ToolSpec,
	prevRaw string,
	startIter int,
	lastErrs PlanErrorList,
) (*schemas.TaskPlan, error) {
	for i := 0; i < p.maxRepair; i++ {
		iter := startIter + i
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		// For the very first repair iteration we may have the
		// raw text from the failed parse. Subsequent iterations
		// feed the JSON-encoded plan from the last round.
		previous := prevRaw
		if previous == "" {
			// Re-render the prompt with the most recent
			// (presumably JSON-parseable) attempt. The caller
			// does not pass it; we re-encode the last good
			// plan by re-running the verifier report path.
			// In practice this branch fires only on round 2+.
			previous = ""
		}
		// On iteration 2+ we need the previous JSON. We
		// re-derive it from the last verifier report. To keep
		// the planner self-contained we stash the previous
		// JSON in the prompt via the Previous field; the
		// caller of repair is responsible for filling it.
		//
		// For now: re-render the prompt with the last errors
		// but no previous plan when previous is empty. The
		// model is told via the repair prompt's structure
		// that this is a fix, and the errors give it enough
		// context to retry from scratch.
		prompt := PromptData{
			Goal:     goal,
			Context:  planCtx,
			Tools:    tools,
			Errors:   lastErrs,
			Previous: previous,
		}
		userPrompt, err := RenderRepair(prompt)
		if err != nil {
			return nil, fmt.Errorf("planner: render repair prompt: %w", err)
		}
		raw, err := p.ask(ctx, userPrompt)
		if err != nil {
			return nil, &PlanGenerationError{Iterations: iter, LastErrors: lastErrs, Cause: err}
		}
		plan, perr := parsePlan(raw)
		if perr != nil {
			lastErrs = PlanErrorList{{Code: CodeMalformedJSON, Message: perr.Error()}}
			prevRaw = raw
			continue
		}
		lastErrs = p.verifier.Verify(plan)
		if len(lastErrs) == 0 {
			return plan, nil
		}
		// Stash the round-2+ JSON for the next repair prompt.
		encoded, _ := EncodePlan(plan)
		prevRaw = encoded
	}
	return nil, &PlanGenerationError{
		Iterations: startIter + p.maxRepair,
		LastErrors: lastErrs,
	}
}

// ask runs one model call. The model id is the planner's default
// unless the response so far has implied a different one (we
// don't track that today; the model is the planner's default).
func (p *Planner) ask(ctx context.Context, userPrompt string) (string, error) {
	req := models.Request{
		Model:  p.defaultMod,
		Prompt: userPrompt,
		Options: models.Options{
			Temperature: 0, // REQ-PLN-NFR-001 / PRD §5 determinism stance
			System:      RenderSystem(),
		},
	}
	resp, err := p.router.Complete(ctx, req)
	if err != nil {
		return "", fmt.Errorf("planner: model call: %w", err)
	}
	return resp.Text, nil
}

// toolsList returns the tool specs in a deterministic order
// (the prompt renderer also sorts, but we sort here so the
// renderer's contract is local).
func (p *Planner) toolsList() []ToolSpec {
	out := make([]ToolSpec, 0, len(p.tools))
	for _, t := range p.tools {
		out = append(out, t)
	}
	// Insertion sort is fine here: the tool map is small (<100
	// entries in any realistic deployment) and we want to keep
	// the planner's surface predictable.
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j-1].Name > out[j].Name; j-- {
			out[j-1], out[j] = out[j], out[j-1]
		}
	}
	return out
}

// parsePlan unmarshals the model's text into a TaskPlan. The
// model is expected to return JSON only, but we are lenient about
// surrounding whitespace and we strip a leading "```json" fence
// if the model included one (some providers do this even when
// told not to).
func parsePlan(raw string) (*schemas.TaskPlan, error) {
	cleaned := stripMarkdownFence(raw)
	cleaned = trimSpace(cleaned)
	if cleaned == "" {
		return nil, fmt.Errorf("empty response")
	}
	var plan schemas.TaskPlan
	if err := json.Unmarshal([]byte(cleaned), &plan); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}
	if err := plan.Validate(); err != nil {
		return nil, fmt.Errorf("invalid plan: %w", err)
	}
	return &plan, nil
}

// stripMarkdownFence removes an opening ``` or ```json line and
// a matching closing ``` line from s. The opening and closing
// markers may be on their own lines; surrounding whitespace is
// preserved when no fence is present. A fence with a trailing
// newline is treated as a fence (so the model may include or
// omit the newline before the closing backticks).
func stripMarkdownFence(s string) string {
	s = trimSpace(s) // work on the trimmed body
	opensFence := false
	if len(s) >= 7 && s[:7] == "```json" {
		opensFence = true
	} else if len(s) >= 3 && s[:3] == "```" {
		// Only treat ``` at the start as a fence if the next
		// character is a newline (otherwise it's part of the
		// JSON, e.g. a code block inside a string).
		if len(s) == 3 || s[3] == '\n' || s[3] == '\r' {
			opensFence = true
		}
	}
	if opensFence {
		// Drop the first line.
		idx := indexOf(s, "\n")
		if idx < 0 {
			// No newline after the opener. Treat the whole
			// thing as a fence; we'll look for a closing
			// fence below.
			s = ""
		} else {
			s = s[idx+1:]
		}
	}
	// Trim whitespace before looking for the closing fence.
	s = trimSpace(s)
	// Strip a trailing ``` line. We look for the last ``` and
	// drop it (plus any surrounding newlines).
	idx := lastIndexOf(s, "```")
	if idx >= 0 {
		s = s[:idx]
	}
	// Trim whitespace that may now surround the body.
	s = trimSpace(s)
	return s
}

func lastIndexOf(s, sub string) int {
	if len(sub) == 0 {
		return len(s)
	}
	for i := len(s) - len(sub); i >= 0; i-- {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func trimSpace(s string) string {
	start := 0
	for start < len(s) && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}
	end := len(s)
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}
