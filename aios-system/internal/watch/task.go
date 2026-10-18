package watch

import (
	"fmt"
	"sync"

	"github.com/sadi/aios-system/internal/core/audit"
	"github.com/sadi/aios-system/internal/runtime"
	"github.com/sadi/aios-system/internal/tools"
)

// AgentType is the kind of sub-agent the Task tool spawns
// (REQ-WTC-DAT-002).
type AgentType string

const (
	AgentGeneral AgentType = "general"
	AgentExplore AgentType = "explore"
	AgentPlan    AgentType = "plan"
	AgentCode    AgentType = "code"
)

// AllAgentTypes is the set of valid agent types, used for
// validation.
var AllAgentTypes = []AgentType{AgentGeneral, AgentExplore, AgentPlan, AgentCode}

// IsValid reports whether t is one of the supported agent
// types.
func (t AgentType) IsValid() bool {
	for _, a := range AllAgentTypes {
		if a == t {
			return true
		}
	}
	return false
}

// SubAgentFactory builds a runtime.Worker for a sub-agent
// invocation. The factory lets callers (and tests) inject the
// sub-agent implementation. Callers pass a bare function
// literal — the type is a function type, not an interface.
type SubAgentFactory func(agentType AgentType) (runtime.Worker, error)

// TaskTool spawns sub-agents (REQ-WTC-SBT-001 to 007).
//
// Args:
//
//	prompt       (string, required) — the prompt to send.
//	agent_type   (string, required) — one of general, explore,
//	                                 plan, code.
//	description  (string, optional) — short label for the
//	                                 audit log.
//	calls        (list,   optional) — when present, an array
//	                                 of objects, each with
//	                                 its own prompt / agent_type,
//	                                 and the tool runs them in
//	                                 parallel (REQ-WTC-SBT-007).
//
// The tool returns a string per call. When calls is absent the
// result has one entry. When calls is present the result has
// one entry per call, in input order.
type TaskTool struct {
	// Factory builds the sub-agent. Required.
	Factory SubAgentFactory
	// MaxParallel caps the number of sub-agents running
	// at once. Zero or negative means 4.
	MaxParallel int
	// Audit is the audit log. nil means "skip the row".
	Audit *audit.Log
}

func (TaskTool) Name() string { return "task" }

// Run executes the call (REQ-WTC-SBT-002 to 006).
func (t *TaskTool) Run(args map[string]any, ctx tools.Context) (tools.Result, error) {
	if t.Factory == nil {
		return tools.Result{OK: false, Summary: "no factory"}, &SubAgentError{Reason: "factory is required"}
	}
	calls := normalizeCalls(args)
	if err := validateCalls(calls); err != nil {
		return tools.Result{OK: false, Summary: "invalid args"}, err
	}
	results, errs := t.runParallel(calls)
	if hasErrors(errs) {
		return tools.Result{OK: false, Summary: "sub-agent failed"}, firstError(errs)
	}
	for i, c := range calls {
		t.writeAudit(ctx, c, results[i])
	}
	return tools.Result{
		OK: true,
		Output: map[string]any{
			"results": results,
		},
		Summary: fmt.Sprintf("ran %d sub-agent call(s)", len(results)),
	}, nil
}

// call is the normalized form of a single Task invocation.
type call struct {
	Prompt      string
	AgentType   AgentType
	Description string
}

// normalizeCalls accepts the args as either {prompt, agent_type,
// description} for a single call, or {calls: [{...}, ...]} for
// parallel calls. The function returns the call list in
// input order.
func normalizeCalls(args map[string]any) []call {
	if list, ok := args["calls"].([]any); ok {
		out := make([]call, 0, len(list))
		for _, raw := range list {
			m, _ := raw.(map[string]any)
			out = append(out, callFromMap(m))
		}
		return out
	}
	return []call{callFromMap(args)}
}

func callFromMap(m map[string]any) call {
	c := call{
		Prompt:      stringArg(m, "prompt"),
		AgentType:   AgentType(stringArg(m, "agent_type")),
		Description: stringArg(m, "description"),
	}
	if c.AgentType == "" {
		c.AgentType = AgentGeneral
	}
	return c
}

func stringArg(m map[string]any, k string) string {
	if m == nil {
		return ""
	}
	s, _ := m[k].(string)
	return s
}

// validateCalls checks every call has the required fields.
func validateCalls(calls []call) error {
	for i, c := range calls {
		if c.Prompt == "" {
			return &SubAgentError{Reason: fmt.Sprintf("call %d: prompt is required", i)}
		}
		if !c.AgentType.IsValid() {
			return &SubAgentError{AgentType: string(c.AgentType), Reason: fmt.Sprintf("call %d: agent_type must be one of %v", i, AllAgentTypes)}
		}
	}
	return nil
}

// runParallel runs the calls in parallel through a semaphore
// of size maxParallel. The returned slice has one result per
// call, in input order.
func (t *TaskTool) runParallel(calls []call) ([]string, []error) {
	results := make([]string, len(calls))
	errs := make([]error, len(calls))
	if len(calls) == 0 {
		return results, errs
	}
	limit := t.MaxParallel
	if limit <= 0 {
		limit = 4
	}
	sem := make(chan struct{}, limit)
	var wg sync.WaitGroup
	for i := range calls {
		i := i
		call := calls[i]
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			results[i], errs[i] = t.runOne(call)
		}()
	}
	wg.Wait()
	return results, errs
}

// runOne invokes the sub-agent for a single call. The
// factory builds a runtime.Worker; the worker is run with a
// fresh TaskContext and a no-progress-free TaskInput.
func (t *TaskTool) runOne(c call) (string, error) {
	worker, err := t.Factory(c.AgentType)
	if err != nil {
		return "", &SubAgentError{AgentType: string(c.AgentType), Reason: "factory", Err: err}
	}
	in := runtime.TaskInput{
		ID:          "subagent-" + string(c.AgentType),
		Description: c.Prompt,
		Acceptance:  "",
		Tool:        string(c.AgentType),
		Inputs:      nil,
	}
	tctx := &runtime.TaskContext{Vars: map[string]any{
		"agent_type": string(c.AgentType),
		"prompt":     c.Prompt,
	}}
	res, err := worker.Run(in, tctx, nil)
	if err != nil {
		return "", &SubAgentError{AgentType: string(c.AgentType), Prompt: truncate(c.Prompt, 200), Reason: "run", Err: err}
	}
	return res.Output, nil
}

// writeAudit writes a "subagent_task" audit row
// (REQ-WTC-SBT-006). One row per call.
func (t *TaskTool) writeAudit(ctx tools.Context, c call, result string) {
	if t.Audit == nil {
		return
	}
	taskID := ctx.TaskID
	if taskID == "" {
		taskID = "subagent"
	}
	actor := ctx.Actor
	if actor == "" {
		actor = "subagent"
	}
	_, _ = t.Audit.Write(&audit.Row{
		TaskID:   taskID,
		Kind:     "subagent_task",
		Actor:    actor,
		RiskTier: 0,
		Payload: map[string]any{
			"prompt":      c.Prompt,
			"agent_type":  string(c.AgentType),
			"description": c.Description,
			"result":      result,
		},
	})
}

// hasErrors reports whether the error slice contains any
// non-nil error.
func hasErrors(errs []error) bool {
	for _, e := range errs {
		if e != nil {
			return true
		}
	}
	return false
}

// firstError returns the first non-nil error.
func firstError(errs []error) error {
	for _, e := range errs {
		if e != nil {
			return e
		}
	}
	return nil
}

// truncate trims s to n runes (safe on invalid UTF-8).
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
