package runtime

// Worker is the interface an agent implements
// (REQ-RT-WRK-001). The worker runs one task; the orchestrator
// calls it once per task and passes the running context and
// the live budget.
type Worker interface {
	// Run executes the task. The context is the per-task
	// scratch space; the budget is the live tracker the
	// worker should consult before any expensive operation.
	// The returned result feeds the verifier.
	//
	// The worker MUST raise *WaitingForApproval when a tool
	// call hits the policy gateway's `ask` action. The
	// orchestrator surfaces that to the operator.
	Run(task TaskInput, ctx *TaskContext, meter *BudgetMeter) (WorkerResult, error)
}

// TaskInput is the per-task view. The full TaskPlan is on
// the orchestrator; the worker only needs the current task.
type TaskInput struct {
	ID          string
	Description string
	Acceptance  string
	Tool        string
	Inputs      []string
}

// TaskContext is the per-task scratch space. The worker
// reads and writes it; the orchestrator snapshots it into
// the checkpoint between tasks.
type TaskContext struct {
	// Vars is a free-form key/value bag. The orchestrator
	// does not interpret it; the worker and the verifier do.
	Vars map[string]any
}

// WorkerResult is what Run returns. The Output feeds the
// no-progress detector; the Data feeds the verifier; the
// ToolCalls and CostUSD feed the report.
type WorkerResult struct {
	Output    string         // human-readable summary
	Data      map[string]any // structured result for the verifier
	ToolCalls int            // count of tool calls the worker made
	CostUSD   float64        // USD cost of model calls in this task
}

// Verifier checks a WorkerResult and returns a verdict.
// REQ-RT-ORC-005 / 006: on pass the orchestrator moves on;
// on fail the orchestrator stops the plan.
//
// The Verifier interface is not named in the doc set's 6
// components but the orchestrator's constructor takes a
// `verifier: Verifierinterface`. Defining it here is the
// natural place.
type Verifier interface {
	Verify(task TaskInput, result WorkerResult) (VerifierVerdict, error)
}

// VerifierVerdict is the result of a Verify call.
type VerifierVerdict string

const (
	VerdictPass         VerifierVerdict = "pass"
	VerdictFail         VerifierVerdict = "fail"
	VerdictInconclusive VerifierVerdict = "inconclusive" // per global PRD §11.7 = fail
)

// IsFail returns true for `fail` and `inconclusive`. The
// global PRD §11.7 says inconclusive is treated as fail.
func (v VerifierVerdict) IsFail() bool {
	return v == VerdictFail || v == VerdictInconclusive
}
