package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/sadi/aios-system/internal/core/audit"
	"github.com/sadi/aios-system/internal/core/schemas"
	"github.com/sadi/aios-system/internal/core/state"
)

// Orchestrator is the for-loop that drives a TaskPlan
// (REQ-RT-ORC-001 to 010). It is sequential: one task at a
// time (REQ-RT-ORC-003). The parallel-workers semaphore in
// the BudgetMeter is a no-op in the sequential path but is
// wired so a future parallel path can use it without
// re-plumbing.
type Orchestrator struct {
	worker   Worker
	verifier Verifier
	meter    *BudgetMeter
	noProg   *NoProgressDetector
	log      *audit.Log
	resume   *ResumeHandler
	tracer   *Tracer

	mu     sync.Mutex
	cancel context.CancelFunc
}

// Config wires the orchestrator. Worker, Verifier, Meter,
// and Resume are required; NoProgress may be nil (the
// orchestrator uses a no-op default); Log and Tracer may be
// nil (writes become no-ops).
type Config struct {
	Worker   Worker
	Verifier Verifier
	Meter    *BudgetMeter
	NoProg   *NoProgressDetector
	Audit    *audit.Log
	Resume   *ResumeHandler
	Tracer   *Tracer
}

// New returns an Orchestrator.
func New(cfg Config) *Orchestrator {
	if cfg.NoProg == nil {
		cfg.NoProg = NewNoProgressDetector(0) // disabled
	}
	if cfg.Tracer == nil && cfg.Audit != nil {
		cfg.Tracer = NewTracer(cfg.Audit)
	}
	return &Orchestrator{
		worker:   cfg.Worker,
		verifier: cfg.Verifier,
		meter:    cfg.Meter,
		noProg:   cfg.NoProg,
		log:      cfg.Audit,
		resume:   cfg.Resume,
		tracer:   cfg.Tracer,
	}
}

// Run executes the plan. When checkpoint is non-nil, the
// orchestrator resumes from checkpoint.TaskIndex.
//
// The return is a Report. On any stop condition the report
// carries the reason and the last task index reached.
func (o *Orchestrator) Run(parentCtx context.Context, plan *schemas.TaskPlan, checkpoint *Checkpoint) (*Report, error) {
	if plan == nil {
		return nil, errors.New("runtime: orchestrator: plan is required")
	}
	if err := plan.Validate(); err != nil {
		return nil, fmt.Errorf("runtime: orchestrator: plan: %w", err)
	}
	startIdx := 0
	taskCtx := &TaskContext{Vars: map[string]any{}}
	if checkpoint != nil {
		startIdx = checkpoint.TaskIndex
		if len(checkpoint.Context) > 0 {
			_ = json.Unmarshal(checkpoint.Context, taskCtx)
		}
	}

	ctx, cancel := context.WithCancel(parentCtx)
	o.mu.Lock()
	o.cancel = cancel
	o.mu.Unlock()
	defer func() {
		o.mu.Lock()
		o.cancel = nil
		o.mu.Unlock()
	}()

	report := &Report{
		PlanID:     plan.ID,
		TaskCount:  len(plan.Tasks),
		StartedAt:  time.Now().UTC().Format(time.RFC3339Nano),
		FromIndex:  startIdx,
		TaskStates: make(map[string]state.State, len(plan.Tasks)),
	}
	defer func() {
		report.FinishedAt = time.Now().UTC().Format(time.RFC3339Nano)
		report.Budget = o.meter.Snapshot()
	}()

	o.noProg.Reset()

	for i := startIdx; i < len(plan.Tasks); i++ {
		select {
		case <-ctx.Done():
			report.StoppedReason = "canceled"
			_ = o.writeStateRow(plan.ID, "task_cancelled", i, taskCtx)
			return report, ctx.Err()
		default:
		}

		task := plan.Tasks[i]
		input := TaskInput{
			ID:          task.ID,
			Description: task.Description,
			Acceptance:  task.Acceptance,
			Tool:        task.Tool,
			Inputs:      task.Inputs,
		}

		// Check budget before each task.
		if err := o.meter.CanProceed(); err != nil {
			report.StoppedReason = "budget_exhausted"
			report.LastError = err
			_ = o.saveCheckpoint(plan, i, taskCtx)
			_ = o.writeStateRow(plan.ID, "task_paused", i, taskCtx)
			report.TaskStates[task.ID] = state.StatePaused
			return report, err
		}

		// Run the worker. The worker may raise *WaitingForApproval,
		// in which case we propagate the error and stop the plan
		// (the operator is expected to grant and re-run).
		result, err := o.worker.Run(input, taskCtx, o.meter)
		if err != nil {
			var wfa *WaitingForApproval
			if errors.As(err, &wfa) {
				report.StoppedReason = "waiting_for_approval"
				report.LastError = err
				report.Waiting = wfa
				report.TaskStates[task.ID] = state.StateAwaitingApproval
				// Persist a checkpoint so the operator can grant
				// the approval and resume from the same step.
				_ = o.saveCheckpoint(plan, i, taskCtx)
				_ = o.writeStateRow(plan.ID, "task_awaiting_approval", i, taskCtx)
				return report, err
			}
			report.StoppedReason = "worker_error"
			report.LastError = err
			report.TaskStates[task.ID] = state.StateFailed
			return report, err
		}

		// No-progress detection on the worker's output.
		if err := o.noProg.Observe(result.Output); err != nil {
			report.StoppedReason = "no_progress"
			report.LastError = err
			report.TaskStates[task.ID] = state.StateFailed
			_ = o.writeStateRow(plan.ID, "task_failed", i, taskCtx)
			return report, err
		}

		// Verify the result.
		verdict, verr := o.verifier.Verify(input, result)
		if verr != nil {
			report.StoppedReason = "verifier_error"
			report.LastError = verr
			report.TaskStates[task.ID] = state.StateFailed
			return report, verr
		}
		if verdict.IsFail() {
			report.StoppedReason = "verifier_fail"
			report.TaskStates[task.ID] = state.StateFailed
			_ = o.writeStateRow(plan.ID, "task_failed", i, taskCtx)
			return report, nil
		}

		// Trace the decision.
		_ = o.tracer.Trace(plan.ID, "orchestrator", 0, schemas.DecisionTrace{
			StepID:   task.ID,
			Chosen:   "task_passed",
			Basis:    fmt.Sprintf("verifier=%s tool_calls=%d cost_usd=%.6f", verdict, result.ToolCalls, result.CostUSD),
			RiskTier: 0,
		})

		report.TaskStates[task.ID] = state.StateCompleted
		// Save a checkpoint after every successful task.
		_ = o.saveCheckpoint(plan, i+1, taskCtx)
	}

	report.StoppedReason = "completed"
	return report, nil
}

// Cancel stops a running orchestrator. It is a no-op when
// the orchestrator is not running.
func (o *Orchestrator) Cancel() {
	o.mu.Lock()
	c := o.cancel
	o.mu.Unlock()
	if c != nil {
		c()
	}
}

// Report is the result of a Run. The fields are populated
// even on early stop.
type Report struct {
	PlanID        string                 `json:"plan_id"`
	TaskCount     int                    `json:"task_count"`
	FromIndex     int                    `json:"from_index"`
	StartedAt     string                 `json:"started_at"`
	FinishedAt    string                 `json:"finished_at"`
	StoppedReason string                 `json:"stopped_reason"`
	LastError     error                  `json:"-"`
	Waiting       *WaitingForApproval    `json:"waiting,omitempty"`
	TaskStates    map[string]state.State `json:"task_states"`
	Budget        BudgetSnapshot         `json:"budget"`
}

func (o *Orchestrator) saveCheckpoint(plan *schemas.TaskPlan, idx int, taskCtx *TaskContext) error {
	if o.resume == nil {
		return nil
	}
	planJSON, err := json.Marshal(plan)
	if err != nil {
		return err
	}
	ctxJSON, err := json.Marshal(taskCtx)
	if err != nil {
		return err
	}
	cp := &Checkpoint{
		TaskID:    plan.ID,
		Plan:      planJSON,
		TaskIndex: idx,
		Context:   ctxJSON,
		Budget:    o.meter.budget,
	}
	return o.resume.Save(cp)
}

func (o *Orchestrator) writeStateRow(planID, kind string, idx int, taskCtx *TaskContext) error {
	if o.log == nil {
		return nil
	}
	row := &audit.Row{
		TaskID:   planID,
		Kind:     kind,
		Actor:    "orchestrator",
		RiskTier: 0,
		Payload: map[string]any{
			"task_index": idx,
			"context":    taskCtx.Vars,
		},
	}
	_, err := o.log.Write(row)
	return err
}
