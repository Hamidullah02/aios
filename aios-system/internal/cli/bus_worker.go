package cli

import (
	"errors"
	"strings"

	"github.com/Hamidullah02/aios/aios-system/internal/ask"
	"github.com/Hamidullah02/aios/aios-system/internal/runtime"
	"github.com/Hamidullah02/aios/aios-system/internal/tools"
)

// BusWorker is a runtime.Worker that forwards each task to
// a tools.Bus. The bus, in turn, consults the policy
// gateway, runs the tool, and writes the audit row. The
// worker is the only place in aios where the runtime and
// the tools bus meet; without it the runtime and the tools
// are two unconnected libraries.
//
// The worker handles the three outcomes the bus returns:
//   - success: a Result with OK=true; the worker passes
//     the result through.
//   - tool error: a Result with OK=false and an error;
//     the worker surfaces both.
//   - policy ask: an error wrapping "requires approval";
//     the worker converts that into a
//     *runtime.WaitingForApproval so the orchestrator
//     can surface it to the operator.
type BusWorker struct {
	bus       *tools.Bus
	ask       *ask.AskUserTool
	workspace string
}

// NewBusWorker wires a BusWorker. The bus is required; the
// ask tool is optional. The workspace is the root for path
// validation in tools like read_file / write_file; empty
// means the package-level default.
func NewBusWorker(bus *tools.Bus, askTool *ask.AskUserTool, workspace string) *BusWorker {
	return &BusWorker{bus: bus, ask: askTool, workspace: workspace}
}

// Run implements runtime.Worker. The task's `tool` field
// is the bus dispatch name. The `inputs` are the args map.
func (w *BusWorker) Run(input runtime.TaskInput, taskCtx *runtime.TaskContext, meter *runtime.BudgetMeter) (runtime.WorkerResult, error) {
	if w.bus == nil {
		return runtime.WorkerResult{}, errors.New("cli: bus worker: bus is nil")
	}
	if input.Tool == "" {
		return runtime.WorkerResult{}, errors.New("cli: bus worker: task has no tool")
	}
	// The task inputs are the args the tool expects. The
	// doc set uses "key=value" strings; the worker splits
	// each one on the first '=' so the bus sees a real
	// map[string]any the tool can read. A value that is a
	// comma-separated list (e.g. "echo,hello") becomes a
	// []string, which `run_shell` requires for its
	// `command` argv argument.
	//
	// v0.40.0: each input is first run through
	// interpolateVars, so a task can reference an earlier
	// task's output ("hash=$vars.t1.hash"). Interpolation
	// happens BEFORE the split so the resolved value is
	// what lands in the args map; a reference that does not
	// resolve is left verbatim so the tool sees the literal
	// and reports the unknown argument, rather than the
	// worker failing the whole task on a typo.
	var vars map[string]any
	if taskCtx != nil {
		vars = taskCtx.Vars
	}
	args := make(map[string]any, len(input.Inputs)+1)
	for _, in := range input.Inputs {
		resolved := interpolateVars(in, vars)
		k, v, ok := splitKV(resolved)
		if !ok {
			args[in] = nil
			continue
		}
		// Comma-separated values become []string for
		// tools like `run_shell` that expect argv. JSON
		// objects (which also contain commas) stay as
		// strings so the receiver can json.Unmarshal.
		//
		// Interpolation runs first, so a resolved value
		// that happens to contain a comma still splits
		// correctly.
		if hasComma(v) && !looksLikeJSON(v) {
			args[k] = strings.Split(v, ",")
		} else {
			args[k] = v
		}
	}
	res, err := w.bus.Dispatch(input.Tool, args, tools.Context{
		Actor:     "bus-worker",
		TaskID:    input.ID,
		Workspace: w.workspace,
	})
	// Record the call on the meter. Without this the
	// meter never saw a tool call, so max_tool_calls and
	// max_tool_calls_per_minute could never be reached and
	// every report printed tool=0. The record happens for
	// the dispatch attempt, not only for success: a denied
	// or failed call is still work the operator paid for
	// and must be visible in the accounting.
	if meter != nil {
		meter.RecordToolCall()
	}
	if err != nil {
		// The bus returns *tools.ApprovalRequiredError for
		// policy `ask`. The orchestrator catches
		// *runtime.WaitingForApproval via errors.As, so the
		// worker converts one into the other.
		//
		// This used to sniff the error string for
		// "requires approval" and then fabricate an id
		// ("ask:"+taskID). That threw away the real
		// approval id the policy gateway issued and the
		// whole payload with it, so the operator was asked
		// about an opaque string and no grant could ever be
		// matched to the call it belonged to.
		var need *tools.ApprovalRequiredError
		if errors.As(err, &need) {
			id := need.ApprovalID
			if id == "" {
				id = "ask:" + input.ID
			}
			return runtime.WorkerResult{}, &runtime.WaitingForApproval{
				ApprovalID: id,
				Decision:   "ask",
				Payload:    need.Approval,
			}
		}
		return runtime.WorkerResult{Output: res.Summary}, err
	}
	return runtime.WorkerResult{
		Output:    res.Summary,
		Data:      res.Output,
		ToolCalls: 1,
	}, nil
}

// Compile-time check.
var _ runtime.Worker = (*BusWorker)(nil)

// splitKV splits "key=value" into (key, value, true). When
// the string has no '=' the bool is false and the caller
// falls back to a positional argument. When the '=' is the
// first byte the key is empty; the caller still uses the
// value as a positional.
func splitKV(s string) (string, string, bool) {
	for i := 0; i < len(s); i++ {
		if s[i] == '=' {
			return s[:i], s[i+1:], true
		}
	}
	return "", "", false
}

// hasComma is a tiny inline test to keep the call site
// readable; it avoids importing strings just for Contains.
func hasComma(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == ',' {
			return true
		}
	}
	return false
}

// looksLikeJSON is a heuristic that returns true for
// strings starting with '{' or '['. Such values are
// kept verbatim so json.Unmarshal on the receiving end
// can decode them.
func looksLikeJSON(s string) bool {
	return len(s) > 0 && (s[0] == '{' || s[0] == '[')
}
