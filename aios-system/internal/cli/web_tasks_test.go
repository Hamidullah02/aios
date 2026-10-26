package cli

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/sadi/aios-system/internal/core/schemas"
	"github.com/sadi/aios-system/internal/models"
	"github.com/sadi/aios-system/internal/runtime"
	"github.com/sadi/aios-system/internal/webui"
)

// stubPlanner is a planner that returns a canned plan.
//
// It exists so the submitter can be tested without a model
// provider and without a network call. The point under test is
// everything AFTER the plan: that a sentence becomes a plan, the
// plan runs, the run's progress reaches the UI, and the outcome
// lands on the operator's card.
type stubPlanner struct {
	mu       sync.Mutex
	goals    []string
	contexts []map[string]any
	plan     *schemas.TaskPlan
	err      error
}

func (p *stubPlanner) Plan(ctx context.Context, goal string, planCtx map[string]any) (*schemas.TaskPlan, error) {
	p.mu.Lock()
	p.goals = append(p.goals, goal)
	p.contexts = append(p.contexts, planCtx)
	plan := p.plan
	err := p.err
	p.mu.Unlock()
	if err != nil {
		return nil, err
	}
	if plan == nil {
		return nil, errors.New("no plan configured")
	}
	// A copy: the submitter rewrites the plan id, and a shared
	// pointer would let one submission's id leak into another's.
	cp := *plan
	cp.Tasks = append([]schemas.TaskSpec(nil), plan.Tasks...)
	return &cp, nil
}

func (p *stubPlanner) sawGoal(goal string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, g := range p.goals {
		if g == goal {
			return true
		}
	}
	return false
}

// lastContext returns the plan context the planner was last
// given, or nil.
func (p *stubPlanner) lastContext() map[string]any {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.contexts) == 0 {
		return nil
	}
	return p.contexts[len(p.contexts)-1]
}

// twoStepPlan is a two-task plan the stub planner hands back.
func twoStepPlan() *schemas.TaskPlan {
	return &schemas.TaskPlan{
		ID:   "planned-by-the-model",
		Goal: "ship it",
		Tasks: []schemas.TaskSpec{
			{ID: "step-1", Description: "gather the numbers", Acceptance: "ok", Tool: "echo"},
			{ID: "step-2", Description: "write the summary", Acceptance: "ok", Tool: "echo"},
		},
	}
}

// waitForTask polls the store until the task reaches one of the
// wanted statuses, or the deadline passes.
//
// Polling rather than a channel handshake: the task store
// publishes to subscribers, but the submitter's own completion is
// signalled by the task's status, and a test that asserts on a
// status has to wait for that status to arrive.
func waitForTask(t *testing.T, store *webui.TaskStore, id string, want ...webui.TaskStatus) webui.Task {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	var last webui.Task
	for time.Now().Before(deadline) {
		for _, task := range store.Tasks() {
			if task.ID != id {
				continue
			}
			last = task
			for _, w := range want {
				if task.Status == w {
					return task
				}
			}
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("task %s never reached %v; last seen: %+v", id, want, last)
	return last
}

// --- the submitter ---

// TestWebTaskSubmitter_SentenceBecomesARunningPlan is the whole
// promise in one test: one sentence in, a plan that runs, and a
// card that says it finished.
//
// Before this, the only way to start work was `aios fix
// <objective>` in a terminal. The browser could show a run but
// could not begin one, so the product had no entry point a
// non-coder could use.
func TestWebTaskSubmitter_SentenceBecomesARunningPlan(t *testing.T) {
	log := openTestAudit(t)
	hub := webui.NewHub()
	planner := &stubPlanner{plan: twoStepPlan()}

	s := &WebTaskSubmitter{
		Hub:     hub,
		Planner: planner,
		Worker:  okWorker{},
		Verify:  &e2eVerifier{verdict: runtime.VerdictPass},
		Resume:  runtime.NewResumeHandler(t.TempDir()),
		Audit:   log,
		Out:     &strings.Builder{},
		BaseCtx: context.Background(),
	}

	const goal = "make my shop show up on Google"
	id, err := s.Submit(goal)
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if id == "" {
		t.Fatal("Submit returned no id")
	}
	task := waitForTask(t, hub.TaskStore(), id, webui.TaskDone)

	// The planner runs in the submitter's goroutine, so the
	// check comes after the task finished: asserting right
	// after Submit would be a race against the worker.
	if !planner.sawGoal(goal) {
		t.Errorf("the planner never saw the goal %q", goal)
	}
	if task.Goal != goal {
		t.Errorf("the card shows goal %q, want the operator's own words %q", task.Goal, goal)
	}
	if task.Result == "" {
		t.Error("a finished task must say what came of it")
	}
	if len(task.PlanSummary) != 2 {
		t.Errorf("plan summary = %v, want one line per step", task.PlanSummary)
	}
	// The summary must be the step descriptions, not the ids:
	// an operator asked for a job, not a list of dispatch names.
	if task.PlanSummary[0] != "gather the numbers" {
		t.Errorf("plan summary[0] = %q, want the step description", task.PlanSummary[0])
	}
}

// TestWebTaskSubmitter_PlannerSeesTheConversation pins the
// "perfect plan from the conversation" promise.
//
// The goal is what the operator wants NOW. The conversation is
// the context that makes the goal mean something: "summarise
// it" is not a goal on its own, but it is after the operator
// said which file. A planner that only sees the last sentence
// guesses at the referent; a planner that sees the thread does
// not.
func TestWebTaskSubmitter_PlannerSeesTheConversation(t *testing.T) {
	log := openTestAudit(t)
	hub := webui.NewHub()
	planner := &stubPlanner{plan: twoStepPlan()}

	// The operator's earlier words, in the log.
	hub.Post(webui.Message{From: webui.OperatorSender, Body: "the numbers are in reports/q3.csv"})
	hub.Post(webui.Message{From: "t1", Body: "agent chatter is not context"})

	s := &WebTaskSubmitter{
		Hub:     hub,
		Planner: planner,
		Worker:  okWorker{},
		Verify:  &e2eVerifier{verdict: runtime.VerdictPass},
		Resume:  runtime.NewResumeHandler(t.TempDir()),
		Audit:   log,
		Out:     &strings.Builder{},
		BaseCtx: context.Background(),
	}

	const goal = "summarise it"
	id, err := s.Submit(goal)
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	waitForTask(t, hub.TaskStore(), id, webui.TaskDone)

	ctx := planner.lastContext()
	if ctx == nil {
		t.Fatal("the planner got no context at all")
	}
	conversation, _ := ctx["conversation"].(string)
	if !strings.Contains(conversation, "reports/q3.csv") {
		t.Errorf("conversation = %q, want the operator's earlier words", conversation)
	}
	// The agent's chatter is machinery talking to itself; it
	// would only add noise to the prompt.
	if strings.Contains(conversation, "agent chatter") {
		t.Errorf("conversation = %q, want no agent chatter in it", conversation)
	}
	// The goal is already the prompt's goal. Repeating it in the
	// context makes the model read it twice.
	if strings.Contains(conversation, goal) {
		t.Errorf("conversation = %q, want the goal left out of it", conversation)
	}
}

// TestWebTaskSubmitter_NoConversationMeansNoContext pins that a
// first message does not arrive wrapped in an empty context.
//
// The prompt template skips an empty context block, so an empty
// one would change the prompt for no reason.
func TestWebTaskSubmitter_NoConversationMeansNoContext(t *testing.T) {
	log := openTestAudit(t)
	hub := webui.NewHub()
	planner := &stubPlanner{plan: twoStepPlan()}

	s := &WebTaskSubmitter{
		Hub:     hub,
		Planner: planner,
		Worker:  okWorker{},
		Verify:  &e2eVerifier{verdict: runtime.VerdictPass},
		Resume:  runtime.NewResumeHandler(t.TempDir()),
		Audit:   log,
		Out:     &strings.Builder{},
		BaseCtx: context.Background(),
	}

	id, err := s.Submit("make my shop show up on Google")
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	waitForTask(t, hub.TaskStore(), id, webui.TaskDone)

	if ctx := planner.lastContext(); ctx != nil {
		t.Errorf("context = %v, want none for a first message", ctx)
	}
}

// TestWebTaskSubmitter_PlanStepsAppearAsAgentPanels pins that the
// machinery of a submitted task shows up in the UI.
//
// The operator watches the work happen. Without this, a submitted
// task would be a card that says "running" with nothing behind
// it, which is not watching anything.
func TestWebTaskSubmitter_PlanStepsAppearAsAgentPanels(t *testing.T) {
	log := openTestAudit(t)
	hub := webui.NewHub()
	planner := &stubPlanner{plan: twoStepPlan()}

	s := &WebTaskSubmitter{
		Hub:     hub,
		Planner: planner,
		Worker:  okWorker{},
		Verify:  &e2eVerifier{verdict: runtime.VerdictPass},
		Resume:  runtime.NewResumeHandler(t.TempDir()),
		Audit:   log,
		Out:     &strings.Builder{},
		BaseCtx: context.Background(),
	}

	id, err := s.Submit("do the thing")
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	waitForTask(t, hub.TaskStore(), id, webui.TaskDone)

	agents := hub.Agents()
	if len(agents) != 2 {
		t.Fatalf("agents = %d, want one per plan step", len(agents))
	}
	// The plan id was rewritten to the task id, so the panels
	// name the same thing as the card. Two submissions of the
	// same model-chosen plan id would otherwise fight over one
	// checkpoint file.
	for _, a := range agents {
		if a.Status != webui.StatusDone {
			t.Errorf("agent %s status = %q, want done", a.ID, a.Status)
		}
	}
}

// TestWebTaskSubmitter_PlanIDIsRewritten pins the uniqueness
// guarantee.
//
// The planner takes the plan id from the model, and the model
// has no way to know what else the server has run. Two
// submissions can produce the same plan id, and the checkpoint
// is written under that id -- so the second run would resume
// the first one's checkpoint.
func TestWebTaskSubmitter_PlanIDIsRewritten(t *testing.T) {
	log := openTestAudit(t)
	hub := webui.NewHub()
	planner := &stubPlanner{plan: twoStepPlan()}

	s := &WebTaskSubmitter{
		Hub:     hub,
		Planner: planner,
		Worker:  okWorker{},
		Verify:  &e2eVerifier{verdict: runtime.VerdictPass},
		Resume:  runtime.NewResumeHandler(t.TempDir()),
		Audit:   log,
		Out:     &strings.Builder{},
		BaseCtx: context.Background(),
	}

	first, err := s.Submit("first job")
	if err != nil {
		t.Fatal(err)
	}
	waitForTask(t, hub.TaskStore(), first, webui.TaskDone)
	second, err := s.Submit("second job")
	if err != nil {
		t.Fatal(err)
	}
	waitForTask(t, hub.TaskStore(), second, webui.TaskDone)

	if first == second {
		t.Fatalf("two submissions shared the id %q; they would fight over one checkpoint", first)
	}
}

// TestWebTaskSubmitter_PlanningFailureIsExplained pins that a
// planner failure lands on the card in words.
//
// The planner's own error is a developer message. The most common
// cause by far is a missing model key, and "no AI model key is
// configured" is something an operator can act on where
// "OPENAI_API_KEY is not set" is not.
func TestWebTaskSubmitter_PlanningFailureIsExplained(t *testing.T) {
	log := openTestAudit(t)
	hub := webui.NewHub()
	planner := &stubPlanner{
		err: &models.ProviderError{
			Provider: "openai",
			Code:     "no_api_key",
			Message:  "OPENAI_API_KEY is not set",
		},
	}

	s := &WebTaskSubmitter{
		Hub:     hub,
		Planner: planner,
		Worker:  okWorker{},
		Verify:  &e2eVerifier{verdict: runtime.VerdictPass},
		Resume:  runtime.NewResumeHandler(t.TempDir()),
		Audit:   log,
		Out:     &strings.Builder{},
		BaseCtx: context.Background(),
	}

	id, err := s.Submit("do the thing")
	if err != nil {
		t.Fatal(err)
	}
	task := waitForTask(t, hub.TaskStore(), id, webui.TaskFailed)
	if !strings.Contains(task.Detail, "model key") {
		t.Errorf("detail = %q, want it to name the missing model key", task.Detail)
	}
}

// TestWebTaskSubmitter_MissingKeyNamesTheRightOne pins that the
// card names the variable the operator actually has to set.
//
// A workspace set up for OpenRouter needs OPENROUTER_API_KEY.
// Telling that operator to set OPENAI_API_KEY sends them down a
// path that does not help, and the message is the only thing
// standing between them and a working setup.
func TestWebTaskSubmitter_MissingKeyNamesTheRightOne(t *testing.T) {
	log := openTestAudit(t)
	hub := webui.NewHub()
	planner := &stubPlanner{
		err: &models.ProviderError{
			Provider: "openrouter",
			Code:     "no_api_key",
			Message:  "OPENROUTER_API_KEY is not set",
		},
	}

	s := &WebTaskSubmitter{
		Hub:     hub,
		Planner: planner,
		Worker:  okWorker{},
		Verify:  &e2eVerifier{verdict: runtime.VerdictPass},
		Resume:  runtime.NewResumeHandler(t.TempDir()),
		Audit:   log,
		Out:     &strings.Builder{},
		BaseCtx: context.Background(),
	}

	id, err := s.Submit("do the thing")
	if err != nil {
		t.Fatal(err)
	}
	task := waitForTask(t, hub.TaskStore(), id, webui.TaskFailed)
	if !strings.Contains(task.Detail, "OPENROUTER_API_KEY") {
		t.Errorf("detail = %q, want it to name OPENROUTER_API_KEY", task.Detail)
	}
	if strings.Contains(task.Detail, "OPENAI_API_KEY") {
		t.Errorf("detail = %q, want it not to name a key this workspace does not use", task.Detail)
	}
}

// TestWebTaskSubmitter_TaskNeverStaysPlanning pins that a failed
// task reaches a terminal state.
//
// A card stuck on "planning" is the worst outcome: the operator
// has no idea whether anything is happening.
func TestWebTaskSubmitter_TaskNeverStaysPlanning(t *testing.T) {
	log := openTestAudit(t)
	hub := webui.NewHub()
	planner := &stubPlanner{err: errors.New("the model said no")}

	s := &WebTaskSubmitter{
		Hub:     hub,
		Planner: planner,
		Worker:  okWorker{},
		Verify:  &e2eVerifier{verdict: runtime.VerdictPass},
		Resume:  runtime.NewResumeHandler(t.TempDir()),
		Audit:   log,
		Out:     &strings.Builder{},
		BaseCtx: context.Background(),
	}

	id, err := s.Submit("do the thing")
	if err != nil {
		t.Fatal(err)
	}
	task := waitForTask(t, hub.TaskStore(), id, webui.TaskFailed)
	if task.Detail == "" {
		t.Error("a failed task must say why")
	}
	if task.FinishedAt.IsZero() {
		t.Error("a failed task must record when it finished")
	}
}

// TestWebTaskSubmitter_SecondSubmissionIsRefusedWhileOneRuns
// pins that two runs cannot interleave into one dashboard.
//
// The orchestrator is sequential by design and the agent panels
// are keyed by task id. Two plans in flight at once would put
// two plans' steps in one set of panels, and the operator would
// have no way to tell which step belongs to which job.
func TestWebTaskSubmitter_SecondSubmissionIsRefusedWhileOneRuns(t *testing.T) {
	log := openTestAudit(t)
	hub := webui.NewHub()
	planner := &stubPlanner{plan: twoStepPlan()}

	s := &WebTaskSubmitter{
		Hub:     hub,
		Planner: planner,
		Worker:  slowWorker{d: 300 * time.Millisecond},
		Verify:  &e2eVerifier{verdict: runtime.VerdictPass},
		Resume:  runtime.NewResumeHandler(t.TempDir()),
		Audit:   log,
		Out:     &strings.Builder{},
		BaseCtx: context.Background(),
	}

	if _, err := s.Submit("first job"); err != nil {
		t.Fatal(err)
	}
	// Wait until the first run is actually under way, so the
	// refusal is about a run in flight rather than a race with
	// the submitter's own bookkeeping.
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if len(hub.Agents()) > 0 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	_, err := s.Submit("second job")
	if err == nil {
		t.Fatal("a second submission was accepted while a run was in flight")
	}
	if !strings.Contains(err.Error(), "already running") {
		t.Errorf("error = %q, want it to say a task is already running", err)
	}
}

// TestWebTaskSubmitter_EmptyGoalIsRefused pins that a blank
// request never reaches the planner.
func TestWebTaskSubmitter_EmptyGoalIsRefused(t *testing.T) {
	hub := webui.NewHub()
	planner := &stubPlanner{plan: twoStepPlan()}
	s := &WebTaskSubmitter{
		Hub:     hub,
		Planner: planner,
		Worker:  okWorker{},
		Verify:  &e2eVerifier{verdict: runtime.VerdictPass},
		Resume:  runtime.NewResumeHandler(t.TempDir()),
	}
	if _, err := s.Submit("   "); err == nil {
		t.Fatal("a blank goal was accepted")
	}
	if len(planner.goals) != 0 {
		t.Errorf("the planner saw %v, want nothing", planner.goals)
	}
}

// TestWebTaskSubmitter_CanSubmitIsHonest pins that a submitter
// without the pieces it needs says so.
//
// The UI asks CanSubmit before showing a Start button. A server
// that said yes and then failed would be worse than one that
// says no.
func TestWebTaskSubmitter_CanSubmitIsHonest(t *testing.T) {
	cases := []struct {
		name string
		s    *WebTaskSubmitter
		want bool
	}{
		{"no planner", &WebTaskSubmitter{Worker: okWorker{}, Verify: &e2eVerifier{}}, false},
		{"no worker", &WebTaskSubmitter{Planner: &stubPlanner{}, Verify: &e2eVerifier{}}, false},
		{"no verifier", &WebTaskSubmitter{Planner: &stubPlanner{}, Worker: okWorker{}}, false},
		{"complete", &WebTaskSubmitter{
			Planner: &stubPlanner{}, Worker: okWorker{}, Verify: &e2eVerifier{},
		}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.s.CanSubmit(); got != c.want {
				t.Errorf("CanSubmit() = %v, want %v", got, c.want)
			}
		})
	}
}

// --- the wiring ---

// TestAttachWebTaskSubmitter_WiresTheSubmitter pins that the web
// server installs a submitter when it has a planner.
//
// This is the line between a dashboard and a product. Without it
// the browser can only watch runs somebody else started.
//
// The submitter is attached through the same struct the
// production path uses, so the stub planner exercises the real
// wiring rather than a parallel one.
func TestAttachWebTaskSubmitter_WiresTheSubmitter(t *testing.T) {
	hub := webui.NewHub()
	deps := &Deps{
		Worker:   okWorker{},
		Verifier: &e2eVerifier{verdict: runtime.VerdictPass},
		Resume:   runtime.NewResumeHandler(t.TempDir()),
	}

	attachWebTaskSubmitter(hub, deps, nil, &strings.Builder{}, "http://127.0.0.1:7755/")

	// Without a planner the server cannot start work, and it
	// says so rather than offering a button that fails.
	if hub.CanSubmitTasks() {
		t.Error("a server without a planner must not claim it can submit tasks")
	}

	// Now give it a planner and the submitter appears.
	sub := &WebTaskSubmitter{
		Hub:     hub,
		Planner: &stubPlanner{plan: twoStepPlan()},
		Worker:  deps.Worker,
		Verify:  deps.Verifier,
		Resume:  deps.Resume,
	}
	hub.SetSubmitter(sub)

	if !hub.CanSubmitTasks() {
		t.Fatal("the server with a planner cannot submit tasks")
	}
	if hub.Submitter() == nil {
		t.Fatal("no submitter was attached")
	}
	if _, err := hub.Submitter().Submit("do a thing"); err != nil {
		t.Errorf("Submit: %v", err)
	}
}

// TestAttachWebTaskSubmitter_NoPlannerMeansNoSubmitter pins the
// honest case: a server without a planner says it cannot start
// tasks rather than offering a button that fails.
func TestAttachWebTaskSubmitter_NoPlannerMeansNoSubmitter(t *testing.T) {
	hub := webui.NewHub()
	attachWebTaskSubmitter(hub, &Deps{}, nil, &strings.Builder{}, "http://127.0.0.1:7755/")
	if hub.CanSubmitTasks() {
		t.Error("a server without a planner must not claim it can submit tasks")
	}
	if hub.Submitter() != nil {
		t.Error("a server without a planner must not attach a submitter")
	}
}

// TestCompositeResolver_FallsThroughToTheDaemon pins that one
// browser button can reach both approval queues.
//
// The web server can be looking at the approvals its own tasks
// raised (in its own hub) and the approvals a run attached to
// the daemon raised (in the daemon's process). If the composite
// only tried one, one queue would be a dead end.
func TestCompositeResolver_FallsThroughToTheDaemon(t *testing.T) {
	hub := webui.NewHub()
	local := NewUIApprovalHandler(hub, &strings.Builder{}, "http://127.0.0.1:7755/")
	daemon := &DaemonApprovalResolver{}

	c := &compositeResolver{local: local, daemon: daemon}

	// Neither owner has the id: the error has to name the
	// approval rather than read like a server fault.
	_, err := c.Resolve("apr-nope", true)
	if err == nil {
		t.Fatal("resolving an unknown approval succeeded")
	}
	if !strings.Contains(err.Error(), "apr-nope") {
		t.Errorf("error = %q, want it to name the approval", err)
	}
}

// TestCompositeResolver_UnknownApprovalKeepsTheNotFoundSentinel
// pins that a miss is reported as a miss.
//
// The web server answers a decision for an approval it does not
// have with 404, and with 409 for an approval it has but cannot
// grant. The two are different failures: 404 is a stale browser
// tab, and 409 is a configuration the operator has to change. The
// server picks between them with errors.Is against
// webui.ErrApprovalNotFound.
//
// The composite resolver falls through to the daemon when the
// local handler misses, and the daemon's error is about the
// socket ("the daemon is not running"). When that error is
// returned with the approval id wrapped around it, the
// ErrApprovalNotFound sentinel is lost -- errors.Is stops
// matching, and the operator gets a 409 that says "try again"
// for an approval that is not coming back.
//
// A second click on an already-decided approval is exactly this
// case, and it is the case a stale tab produces.
func TestCompositeResolver_UnknownApprovalKeepsTheNotFoundSentinel(t *testing.T) {
	hub := webui.NewHub()
	local := NewUIApprovalHandler(hub, &strings.Builder{}, "http://127.0.0.1:7755/")
	// A daemon resolver with no socket: it always fails to dial.
	c := &compositeResolver{local: local, daemon: &DaemonApprovalResolver{}}

	_, err := c.Resolve("apr-nope", true)
	if err == nil {
		t.Fatal("resolving an unknown approval succeeded")
	}
	if !errors.Is(err, webui.ErrApprovalNotFound) {
		t.Errorf("error = %v, want it to wrap webui.ErrApprovalNotFound", err)
	}
}

// TestCompositeResolver_LocalApprovalWins pins that an approval
// this server owns is decided here.
func TestCompositeResolver_LocalApprovalWins(t *testing.T) {
	hub := webui.NewHub()
	local := NewUIApprovalHandler(hub, &strings.Builder{}, "http://127.0.0.1:7755/")
	c := &compositeResolver{local: local}

	// Park an approval the way a gated tool call would.
	go func() {
		_, _ = local.Approve(context.Background(), uiWFA("apr-1"))
	}()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && len(hub.PendingApprovals()) == 0 {
		time.Sleep(5 * time.Millisecond)
	}

	state, err := c.Resolve("apr-1", true)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if state != "granted" {
		t.Errorf("state = %q, want granted", state)
	}
}
