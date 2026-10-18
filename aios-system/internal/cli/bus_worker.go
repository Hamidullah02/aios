package cli

import (
	"errors"
	"strings"

	"github.com/sadi/aios-system/internal/ask"
	"github.com/sadi/aios-system/internal/runtime"
	"github.com/sadi/aios-system/internal/tools"
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
func (w *BusWorker) Run(input runtime.TaskInput, _ *runtime.TaskContext, _ *runtime.BudgetMeter) (runtime.WorkerResult, error) {
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
	args := make(map[string]any, len(input.Inputs)+1)
	for _, in := range input.Inputs {
		k, v, ok := splitKV(in)
		if !ok {
			args[in] = nil
			continue
		}
		// Comma-separated values become []string for
		// tools like `run_shell` that expect argv. JSON
		// objects (which also contain commas) stay as
		// strings so the receiver can json.Unmarshal.
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
	if err != nil {
		// The bus returns this error for policy `ask`.
		// The orchestrator catches *WaitingForApproval
		// via errors.As, so the worker must surface
		// the same type.
		if strings.Contains(err.Error(), "requires approval") {
			return runtime.WorkerResult{}, &runtime.WaitingForApproval{
				ApprovalID: "ask:" + input.ID,
				Decision:   "ask",
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
