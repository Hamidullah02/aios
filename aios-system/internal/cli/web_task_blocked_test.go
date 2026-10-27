package cli

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Hamidullah02/aios/aios-system/internal/core/schemas"
	"github.com/Hamidullah02/aios/aios-system/internal/runtime"
	"github.com/Hamidullah02/aios/aios-system/internal/webui"
)

// blockingApprovalHandler is an approval handler that never
// answers on its own.
//
// It records the request and then waits for the test to release
// it, so the test can observe the task card WHILE the run is
// waiting for a person. That is the window the bug lived in.
type blockingApprovalHandler struct {
	mu       sync.Mutex
	requests []*runtime.WaitingForApproval
	release  chan bool
	seen     chan struct{}
}

func newBlockingApprovalHandler() *blockingApprovalHandler {
	return &blockingApprovalHandler{
		release: make(chan bool, 1),
		seen:    make(chan struct{}, 1),
	}
}

func (h *blockingApprovalHandler) Approve(ctx context.Context, wfa *runtime.WaitingForApproval) (bool, error) {
	h.mu.Lock()
	h.requests = append(h.requests, wfa)
	h.mu.Unlock()

	select {
	case h.seen <- struct{}{}:
	default:
	}

	select {
	case decision := <-h.release:
		return decision, nil
	case <-ctx.Done():
		return false, ctx.Err()
	}
}

func (h *blockingApprovalHandler) count() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.requests)
}

// approvalWorker is a worker whose tool call needs a human.
//
// The worker is what raises the approval request, so a test that
// wants the run to stop and wait for a decision has to use a
// worker that asks. A worker that simply succeeds never produces
// an approval, and the test would wait for something that can
// never arrive.
//
// It asks ONCE. The run loop retries the step after the operator
// grants the request, and a worker that asked every time would
// put the run straight back into "waiting for a decision" -- so
// the card would be right but the run would never finish.
type approvalWorker struct {
	mu    sync.Mutex
	asked bool
}

func newApprovalWorker() *approvalWorker {
	return &approvalWorker{}
}

func (w *approvalWorker) Run(runtime.TaskInput, *runtime.TaskContext, *runtime.BudgetMeter) (runtime.WorkerResult, error) {
	w.mu.Lock()
	first := !w.asked
	w.asked = true
	w.mu.Unlock()
	if !first {
		// The operator granted it, so the retried step succeeds.
		return runtime.WorkerResult{Output: "ok"}, nil
	}
	return runtime.WorkerResult{}, &runtime.WaitingForApproval{
		ApprovalID: "apr-1",
		Decision:   "ask",
		Payload: &schemas.ApprovalPayload{
			ApprovalID:  "apr-1",
			RequesterID: "t1",
			Action:      "git_commit",
			Target:      "wip",
			Reason:      "gated",
		},
	}
}

// TestWebTaskSubmitter_CardIsBlockedWhileWaitingForApproval is the
// regression guard for a card that lied.
//
// The task card said "working on it" for the whole run, including
// the minutes the run spent waiting for the operator to click
// Allow. That is the one moment the operator most needs the card:
// the task looks stuck, the agent panels say nothing useful, and
// the only sign that the run needs a person is the approval
// request in a different panel.
//
// The test holds the approval open and reads the card while the
// request is outstanding.
func TestWebTaskSubmitter_CardIsBlockedWhileWaitingForApproval(t *testing.T) {
	log := openTestAudit(t)
	hub := webui.NewHub()
	handler := newBlockingApprovalHandler()

	s := &WebTaskSubmitter{
		Hub:      hub,
		Planner:  &stubPlanner{plan: twoStepPlan()},
		Worker:   newApprovalWorker(),
		Verify:   &e2eVerifier{verdict: runtime.VerdictPass},
		Resume:   runtime.NewResumeHandler(t.TempDir()),
		Audit:    log,
		Approval: handler,
		Out:      &strings.Builder{},
		BaseCtx:  context.Background(),
	}

	id, err := s.Submit("do the thing")
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}

	// Wait for the approval request to reach the handler.
	select {
	case <-handler.seen:
	case <-time.After(5 * time.Second):
		t.Fatal("the run never asked for an approval")
	}

	// The card must say it is waiting, not that it is working.
	deadline := time.Now().Add(5 * time.Second)
	var card webui.Task
	for time.Now().Before(deadline) {
		for _, task := range hub.TaskStore().Tasks() {
			if task.ID == id {
				card = task
			}
		}
		if card.Status == webui.TaskBlocked {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if card.Status != webui.TaskBlocked {
		t.Fatalf("card status = %q (%s), want %q", card.Status, card.Detail, webui.TaskBlocked)
	}
	if card.Detail == "" {
		t.Error("the blocked card carries no detail, so the operator cannot tell what is being asked")
	}

	// Let the run finish so the test does not leave a goroutine
	// waiting on a channel nobody reads.
	handler.release <- true
	waitForTask(t, hub.TaskStore(), id, webui.TaskDone, webui.TaskFailed)
}

// TestWebTaskSubmitter_CardReturnsToRunningAfterApproval is the
// other half: the card must not get stuck on "blocked" once the
// operator has answered.
//
// A card that stays blocked after the decision is worse than the
// original bug in one way: it tells the operator the run is still
// waiting when it is not, and the run may be about to finish.
func TestWebTaskSubmitter_CardReturnsToRunningAfterApproval(t *testing.T) {
	log := openTestAudit(t)
	hub := webui.NewHub()
	handler := newBlockingApprovalHandler()

	s := &WebTaskSubmitter{
		Hub:      hub,
		Planner:  &stubPlanner{plan: twoStepPlan()},
		Worker:   newApprovalWorker(),
		Verify:   &e2eVerifier{verdict: runtime.VerdictPass},
		Resume:   runtime.NewResumeHandler(t.TempDir()),
		Audit:    log,
		Approval: handler,
		Out:      &strings.Builder{},
		BaseCtx:  context.Background(),
	}

	id, err := s.Submit("do the thing")
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}

	select {
	case <-handler.seen:
	case <-time.After(5 * time.Second):
		t.Fatal("the run never asked for an approval")
	}

	// Approve, then let the run finish.
	handler.release <- true
	waitForTask(t, hub.TaskStore(), id, webui.TaskDone, webui.TaskFailed)

	// The card must not be left saying it is waiting for a
	// decision that has already been made.
	for _, task := range hub.TaskStore().Tasks() {
		if task.ID != id {
			continue
		}
		if task.Status == webui.TaskBlocked {
			t.Errorf("the card is still blocked after the decision: %+v", task)
		}
	}
}

// TestTaskApprovalHandler_NoInnerHandlerDoesNotPanic covers the
// wiring mistake: a submitter with no approval handler must
// report an error rather than crash.
func TestTaskApprovalHandler_NoInnerHandlerDoesNotPanic(t *testing.T) {
	h := &taskApprovalHandler{store: webui.NewTaskStore(), taskID: "task-1"}
	_, err := h.Approve(context.Background(), &runtime.WaitingForApproval{
		ApprovalID: "a1",
		Payload:    &schemas.ApprovalPayload{Action: "git_commit", Target: "wip"},
	})
	if err == nil {
		t.Fatal("expected an error when no handler is configured")
	}
}
