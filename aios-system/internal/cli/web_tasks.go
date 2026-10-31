package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Hamidullah02/aios/aios-system/internal/core/audit"
	"github.com/Hamidullah02/aios/aios-system/internal/core/schemas"
	"github.com/Hamidullah02/aios/aios-system/internal/models"
	"github.com/Hamidullah02/aios/aios-system/internal/runtime"
	"github.com/Hamidullah02/aios/aios-system/internal/webui"
)

// WebTaskSubmitter turns an operator's plain-language goal into
// work.
//
// This is the seam that makes the product usable by someone who
// does not read YAML, write a plan, or know what a policy file
// is. The operator types one sentence; the system decides whether
// that sentence becomes a plan as written, a repaired plan, or a
// plan the conversation refined, and then runs it.
//
// The submit is asynchronous on purpose. The HTTP request that
// carried the goal returns as soon as the work is accepted, and
// the progress arrives through the hub's event stream. A submit
// that blocked for the length of a run would hold a request open
// for minutes and time out in every proxy in between.
//
// Concurrency. Several operators (or one operator in two tabs)
// can submit at once. The counter is atomic, the plan runner
// owns one run at a time per submitter, and the hub is safe for
// concurrent use, so the only real constraint is that a run
// holds the orchestrator's sequence.
type WebTaskSubmitter struct {
	// Hub is where progress is published.
	Hub *webui.Hub
	// Planner turns the goal into a plan.
	Planner plannerRef
	// Worker runs each task of the plan.
	Worker runtime.Worker
	// Verify checks each task's result.
	Verify runtime.Verifier
	// Resume persists checkpoints so a run can be picked up.
	Resume *runtime.ResumeHandler
	// Budget is the per-run limit envelope.
	Budget runtime.Budget
	// Approval resolves a gated tool call. It parks the request
	// in the UI and waits for the operator to decide there.
	Approval ApprovalHandler
	// Grants records the approvals given during a run so the
	// bus lets the retried call through.
	Grants *grantStore
	// Audit is the durable record.
	Audit *audit.Log
	// Out receives a one-line note per outcome, so an operator
	// watching the server's terminal can see what happened to
	// work they started in a browser.
	Out io.Writer
	// BaseCtx is the parent context for submitted runs.
	BaseCtx context.Context

	// seq numbers the submitted tasks.
	seq atomic.Int64
	// mu guards busy and the running task.
	mu sync.Mutex
	// busy is true while a run is in flight.
	busy bool
	// running is the id of the task in flight.
	running string
}

// plannerRef is the planner's surface the submitter needs.
//
// It is an interface rather than *planner.Planner so a test can
// drive the submitter with a stub planner and no model provider.
// The production *planner.Planner satisfies it as-is.
type plannerRef interface {
	// Plan turns a goal into a verified plan.
	Plan(ctx context.Context, goal string, planCtx map[string]any) (*schemas.TaskPlan, error)
}

// CanSubmit implements webui.TaskSubmitter.
//
// It reports false when the server cannot do the work: no
// planner means no way to turn a sentence into a plan, and no
// worker means no way to run one. A UI that offers a submit
// button it cannot honour is worse than one that says where to
// go instead.
func (s *WebTaskSubmitter) CanSubmit() bool {
	if s == nil {
		return false
	}
	return s.Planner != nil && s.Worker != nil && s.Verify != nil
}

// Submit implements webui.TaskSubmitter.
//
// It returns the id of the task the goal became. The goal is
// echoed into the hub before this returns, so the operator sees
// their own words appear on screen immediately rather than after
// the plan is built.
func (s *WebTaskSubmitter) Submit(goal string) (string, error) {
	if s == nil {
		return "", webui.ErrNoSubmitter
	}
	if !s.CanSubmit() {
		return "", errors.New("this server cannot run tasks: the planner is not configured")
	}
	goal = strings.TrimSpace(goal)
	if goal == "" {
		return "", errors.New("say what you want done")
	}

	// One run at a time. The orchestrator is sequential by
	// design, and the run state the hub shows is one plan's
	// worth. Accepting a second goal while the first is in
	// flight would make the agent panels ambiguous: two plans'
	// steps with the same task ids would collide on one panel.
	//
	// The operator is told plainly and can wait. That is
	// better than silently interleaving two runs into one
	// dashboard.
	s.mu.Lock()
	if s.busy {
		s.mu.Unlock()
		return "", errors.New("a task is already running; wait for it to finish")
	}
	s.busy = true
	s.mu.Unlock()

	id := fmt.Sprintf("task-%d", s.seq.Add(1))
	store := s.Hub.TaskStore()
	store.Upsert(id, func(t *webui.Task) {
		t.Goal = goal
		t.Status = webui.TaskPlanning
		t.Detail = "working out what to do"
	})

	go s.run(id, goal)

	return id, nil
}

// run does the work for one submitted goal.
//
// Every failure path ends with the task marked failed and a
// detail an operator can act on. A task left in "planning" for
// ever is the worst outcome: the operator has no idea whether
// anything is happening.
func (s *WebTaskSubmitter) run(id, goal string) {
	defer func() {
		s.mu.Lock()
		s.busy = false
		s.running = ""
		s.mu.Unlock()
	}()

	ctx := s.BaseCtx
	if ctx == nil {
		ctx = context.Background()
	}
	store := s.Hub.TaskStore()

	// 1. Turn the sentence into a plan.
	//
	// This is where the "make a perfect plan from the
	// conversation" promise lives: the planner asks the model
	// for a plan, checks it against the registered tools, and
	// repairs it until it passes. The operator never sees a
	// plan file and never has to approve one.
	//
	// The conversation goes with it. The goal is what the
	// operator wants NOW; the earlier messages are the
	// context that makes the goal make sense ("summarise it"
	// is not a goal on its own, but it is after the operator
	// said which file). A planner that only sees the last
	// sentence guesses at what "it" refers to; a planner that
	// sees the thread does not.
	plan, err := s.Planner.Plan(ctx, goal, s.planContext(goal))
	if err != nil {
		s.fail(store, id, describePlanError(err))
		return
	}
	if plan == nil {
		s.fail(store, id, "the planner did not produce a plan")
		return
	}
	// The plan's id becomes the task's id so the checkpoint,
	// the audit rows, and the agent panels all name the same
	// thing. The model chose the original id and it is not
	// unique: two submissions would fight over one checkpoint
	// file.
	plan.ID = id

	store.Upsert(id, func(t *webui.Task) {
		t.Status = webui.TaskRunning
		t.Detail = "working on it"
		t.PlanSummary = summarizePlan(plan)
	})

	// 2. Run it, with the UI watching.
	//
	// The observer is the same bridge `aios run --web` uses, so
	// a submitted task's steps appear as agent panels exactly
	// like a plan run from the terminal.
	report, runErr := s.execute(ctx, id, plan)
	if runErr != nil {
		s.fail(store, id, describeRunError(runErr))
		return
	}
	if report == nil {
		s.fail(store, id, "the run finished without a report")
		return
	}
	if report.StoppedReason != "completed" {
		s.fail(store, id, describeStop(report))
		return
	}

	store.Upsert(id, func(t *webui.Task) {
		t.Status = webui.TaskDone
		t.Detail = "done"
		t.Result = describeResult(report)
		t.FinishedAt = webui.NewTime(time.Now().UTC())
	})
	if s.Out != nil {
		fmt.Fprintf(s.Out, "task %s: done (%d step(s))\n", id, report.TaskCount)
	}
}

// execute runs the plan through the shared approval loop.
//
// It is the same path `aios run` takes, so the approval gate, the
// grant store, the checkpointing, and the budget meter all behave
// identically whether a plan came from a YAML file or from a
// sentence in a browser.
func (s *WebTaskSubmitter) execute(ctx context.Context, id string, plan *schemas.TaskPlan) (*runtime.Report, error) {
	return runWithApprovals(ctx, plan, nil, ResumeDeps{
		Audit:      s.Audit,
		Resume:     s.Resume,
		Worker:     s.Worker,
		Verify:     s.Verify,
		Budget:     s.Budget,
		Approve:    &taskApprovalHandler{inner: s.Approval, store: s.Hub.TaskStore(), taskID: id},
		GrantStore: s.Grants,
		Observer:   webui.NewBridge(s.Hub),
	}, "web task")
}

// taskApprovalHandler wraps the submitter's approval handler so
// the task card says what is actually happening.
//
// The card said "working on it" for the whole run, including the
// minutes the run spent waiting for a person to click Allow. That
// is the one moment the operator most needs to see: the task
// looks stuck, the agent panels say nothing useful, and the only
// sign that the run needs them is the approval request in
// another panel.
//
// The wrapper knows the task id, so it can move the card to
// "blocked" while the request is outstanding and back to "running"
// once the decision arrives. A denied request returns with an
// error, and the card is left where the run puts it: failed.
type taskApprovalHandler struct {
	inner  ApprovalHandler
	store  *webui.TaskStore
	taskID string
}

// Approve implements ApprovalHandler.
func (h *taskApprovalHandler) Approve(ctx context.Context, wfa *runtime.WaitingForApproval) (bool, error) {
	if h == nil || h.inner == nil {
		return false, errors.New("approval: no handler configured")
	}
	h.setBlocked(wfa)
	approved, err := h.inner.Approve(ctx, wfa)
	if err != nil {
		// The decision failed (a timeout, a refused click). The
		// run is about to fail the task, so the card is left for
		// that to describe.
		return approved, err
	}
	// The decision arrived. The run continues, so the card goes
	// back to running.
	if h.store != nil {
		h.store.SetStatus(h.taskID, webui.TaskRunning, "working on it")
	}
	return approved, nil
}

// setBlocked moves the card to "waiting for you" and says what is
// being asked for.
func (h *taskApprovalHandler) setBlocked(wfa *runtime.WaitingForApproval) {
	if h.store == nil {
		return
	}
	detail := "waiting for your approval"
	if payload, ok := approvalPayload(wfa); ok && payload.Action != "" {
		detail = "waiting for your approval: " + payload.Action
	}
	h.store.SetStatus(h.taskID, webui.TaskBlocked, detail)
}

// fail marks a task as failed with a detail the operator can act
// on.
func (s *WebTaskSubmitter) fail(store *webui.TaskStore, id, detail string) {
	store.SetStatus(id, webui.TaskFailed, detail)
	if s.Out != nil {
		fmt.Fprintf(s.Out, "task %s: failed: %s\n", id, detail)
	}
}

// planContext builds the context the planner receives alongside
// the goal: the conversation so far, minus the goal itself.
//
// The goal is the LAST thing the operator said, so it is
// already in the prompt as the goal. Repeating it in the
// context would only make the model read it twice.
//
// The conversation is the operator's own messages. An agent's
// chatter is machinery talking to itself and would only add
// noise to the prompt.
//
// The thread is trimmed to the most recent messages: a long
// session would otherwise grow the prompt without bound, and
// the messages that matter for "what did they mean" are the
// recent ones.
func (s *WebTaskSubmitter) planContext(goal string) map[string]any {
	if s == nil || s.Hub == nil {
		return nil
	}
	msgs := s.Hub.Messages()
	lines := make([]string, 0, len(msgs))
	for _, m := range msgs {
		if m.From != webui.OperatorSender {
			continue
		}
		body := strings.TrimSpace(m.Body)
		if body == "" || body == strings.TrimSpace(goal) {
			continue
		}
		lines = append(lines, body)
	}
	if len(lines) == 0 {
		return nil
	}
	if len(lines) > maxConversationLines {
		lines = lines[len(lines)-maxConversationLines:]
	}
	return map[string]any{"conversation": strings.Join(lines, "\n")}
}

// maxConversationLines caps how much of the thread goes into the
// planner's context.
//
// Ten lines is roughly a paragraph of back-and-forth. It is
// enough to carry a referent ("it", "the second file") and
// short enough that the prompt does not drown the goal.
const maxConversationLines = 10

// summarizePlan renders a plan as one line per step.
//
// The lines are the task descriptions, not the task ids and not
// the tool names. An operator asked for a job, not a list of
// dispatch names, and the ids are already on the agent panels
// below.
func summarizePlan(plan *schemas.TaskPlan) []string {
	if plan == nil {
		return nil
	}
	out := make([]string, 0, len(plan.Tasks))
	for _, t := range plan.Tasks {
		line := strings.TrimSpace(t.Description)
		if line == "" {
			line = t.ID
		}
		out = append(out, line)
	}
	return out
}

// describePlanError turns a planner failure into something a
// non-coder can act on.
//
// The planner's own error is a developer message ("plan
// generation failed after 2 iteration(s)"). The common cause by
// far is a missing API key, and "the model provider is not
// configured" is actionable where "OPENAI_API_KEY is not set" is
// not, so the key error is named in words and the underlying
// message is kept underneath it.
//
// The key named in the advice is the one the provider actually
// asked for, not a fixed list. A workspace set up for OpenRouter
// needs OPENROUTER_API_KEY; telling that operator to set
// OPENAI_API_KEY sends them down a path that does not help.
func describePlanError(err error) string {
	if err == nil {
		return "could not work out what to do"
	}
	var pe *models.ProviderError
	if errors.As(err, &pe) && pe.Code == "no_api_key" {
		return "no AI model key is configured, so nothing can be planned. " +
			"Set " + pe.Message + " and try again."
	}
	return "could not work out what to do: " + err.Error()
}

// describeRunError turns a run failure into an operator-facing
// line.
func describeRunError(err error) string {
	if err == nil {
		return "the run failed"
	}
	var wfa *runtime.WaitingForApproval
	if errors.As(err, &wfa) {
		return "waiting for your approval"
	}
	return "the run failed: " + err.Error()
}

// describeStop explains a plan that stopped without an error.
//
// The orchestrator's StoppedReason is a machine word. "verifier
// fail" tells an operator nothing; "a step did not produce the
// result it promised" tells them what happened.
func describeStop(report *runtime.Report) string {
	if report == nil {
		return "the run stopped"
	}
	switch report.StoppedReason {
	case "verifier_fail":
		return "a step did not produce the result it promised"
	case "budget_exhausted":
		return "the run used up its budget before finishing"
	case "no_progress":
		return "the run stopped because it stopped making progress"
	case "worker_error":
		return "a step could not be completed"
	case "waiting_for_approval":
		return "waiting for your approval"
	case "canceled":
		return "the run was cancelled"
	default:
		return "the run stopped: " + report.StoppedReason
	}
}

// describeResult renders the outcome of a finished task.
func describeResult(report *runtime.Report) string {
	if report == nil {
		return ""
	}
	return fmt.Sprintf("All %d step(s) completed.", report.TaskCount)
}
