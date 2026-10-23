package cli

import (
	"bytes"
	"context"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/sadi/aios-system/internal/core/audit"
	"github.com/sadi/aios-system/internal/core/schemas"
	"github.com/sadi/aios-system/internal/runtime"
	"github.com/sadi/aios-system/internal/webui"
)

// freeAddr returns a loopback address that was free a moment
// ago.
//
// It binds :0, reads the assigned port, and closes. There is
// an inherent race with any other process claiming the port in
// between; that is acceptable in a test that only needs a
// loopback address to talk to, and a retry loop would add
// complexity without removing the race.
func freeAddr(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := l.Addr().String()
	_ = l.Close()
	return addr
}

// liveTestToken is the session token the live UI runs with.
//
// The UI refuses an unauthenticated request, so the test pins the
// token in the environment and presents it on every call. Pinning
// it (rather than scraping it out of the printed URL) is what an
// operator does too, and it keeps the test independent of how the
// URL happens to be formatted.
const liveTestToken = "live-test-token"

// waitForServer polls until the server answers, or fails.
//
// runRun sleeps 150ms before starting the plan so the URL it
// prints is reachable, but a test must not depend on a sleep
// being long enough.
func waitForServer(t *testing.T, base string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		c := &http.Client{Timeout: 500 * time.Millisecond}
		req, err := http.NewRequest(http.MethodGet, base+"/api/state", nil)
		if err != nil {
			t.Fatalf("request: %v", err)
		}
		req.Header.Set(webui.TokenHeader, liveTestToken)
		resp, err := c.Do(req)
		if err == nil {
			_ = resp.Body.Close()
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("server at %s never became reachable", base)
}

func httpGet(t *testing.T, url string) ([]byte, int) {
	t.Helper()
	c := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	req.Header.Set(webui.TokenHeader, liveTestToken)
	resp, err := c.Do(req)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read %s: %v", url, err)
	}
	return body, resp.StatusCode
}

// twoTaskPlan is a minimal two-task plan.
func twoTaskPlan() *schemas.TaskPlan {
	return &schemas.TaskPlan{
		ID: "web-live", Goal: "ship",
		Tasks: []schemas.TaskSpec{
			{ID: "t1", Description: "first", Acceptance: "ok", Tool: "echo"},
			{ID: "t2", Description: "second", Acceptance: "ok", Tool: "echo"},
		},
	}
}

// okWorker is a worker that always succeeds, so a run reaches
// completion without touching the real bus, policy gateway, or
// filesystem.
type okWorker struct{}

func (okWorker) Run(runtime.TaskInput, *runtime.TaskContext, *runtime.BudgetMeter) (runtime.WorkerResult, error) {
	return runtime.WorkerResult{Output: "ok"}, nil
}

// slowWorker succeeds but takes d per task. It exists so a
// test can observe the UI while work is genuinely in flight:
// without a delay the plan finishes in microseconds and any
// "the panels update live" assertion is really just asserting
// on the final state.
type slowWorker struct{ d time.Duration }

func (s slowWorker) Run(runtime.TaskInput, *runtime.TaskContext, *runtime.BudgetMeter) (runtime.WorkerResult, error) {
	time.Sleep(s.d)
	return runtime.WorkerResult{Output: "ok"}, nil
}

// webTestDeps builds Deps that run a plan successfully.
func webTestDeps(t *testing.T, log *audit.Log) *Deps {
	t.Helper()
	return webTestDepsWithWorker(t, log, okWorker{})
}

// slowTestDeps is webTestDeps with a worker that takes d per
// task, so a run lasts long enough to observe.
func slowTestDeps(t *testing.T, log *audit.Log, d time.Duration) *Deps {
	t.Helper()
	return webTestDepsWithWorker(t, log, slowWorker{d: d})
}

func webTestDepsWithWorker(t *testing.T, log *audit.Log, w runtime.Worker) *Deps {
	t.Helper()
	return &Deps{
		Audit:    log,
		Resume:   runtime.NewResumeHandler(t.TempDir()),
		Worker:   w,
		Verifier: &e2eVerifier{verdict: runtime.VerdictPass},
	}
}

// TestRunRun_WebPublishesLiveState is the v0.41.1 gate.
//
// Before this change `aios web` could only ever show the audit
// log's history: it is a separate process from `aios run`, so
// it never saw a task start. This asserts the run publishes
// into a hub the UI can serve, and that the UI served it over
// HTTP.
func TestRunRun_WebPublishesLiveState(t *testing.T) {
	t.Setenv(webui.EnvToken, liveTestToken)
	log := openTestAudit(t)
	addr := freeAddr(t)

	// runRun builds its own hub; the only way to observe it is
	// through the server it binds, so this test asserts on the
	// HTTP surface rather than on internal state.
	//
	// The plan is deliberately slow enough that the server is
	// queryable while it runs: the whole point of --web is
	// watching work as it happens, so a test that only looks
	// after the run proves nothing.
	slowPlan := &schemas.TaskPlan{
		ID: "web-live", Goal: "ship",
		Tasks: []schemas.TaskSpec{
			{ID: "t1", Description: "first", Acceptance: "ok", Tool: "echo"},
			{ID: "t2", Description: "second", Acceptance: "ok", Tool: "echo"},
			{ID: "t3", Description: "third", Acceptance: "ok", Tool: "echo"},
		},
	}

	type runResult struct {
		code   int
		stdout string
		stderr string
	}
	done := make(chan runResult, 1)
	var stdout, stderr bytes.Buffer
	go func() {
		code := runRun(context.Background(), &Args{
			Command:  CmdRun,
			PlanPath: writePlanFile(t, slowPlan),
			WebLive:  true,
			WebAddr:  addr,
		}, slowTestDeps(t, log, 400*time.Millisecond), &stdout, &stderr)
		done <- runResult{code, stdout.String(), stderr.String()}
	}()

	base := "http://" + addr
	waitForServer(t, base)

	// The server is reachable ~150ms before the first task
	// starts (runRun yields so the URL it prints is live), so a
	// single sample here would race the run. Poll instead, and
	// require the thing that makes --web worth having: a task
	// observed in `running` while later tasks are still absent,
	// which is only true if the UI is fed during the run rather
	// than filled in afterwards.
	var (
		sawRunning   bool
		ranAhead     bool
		lastState    []byte
		sawAnything  bool
		deadlineIdle = time.Now().Add(10 * time.Second)
	)
	for time.Now().Before(deadlineIdle) {
		body, st := httpGet(t, base+"/api/state")
		if st != http.StatusOK {
			t.Fatalf("GET /api/state = %d", st)
		}
		lastState = body
		if bytes.Contains(body, []byte(`"t1"`)) {
			sawAnything = true
			if bytes.Contains(body, []byte(`"running"`)) {
				sawRunning = true
				// t1 running while t3 has not been announced is
				// the proof of a live feed.
				if !bytes.Contains(body, []byte(`"t3"`)) {
					ranAhead = true
					break
				}
			}
		}
		// Once every task is done there is nothing left to
		// observe.
		if bytes.Contains(body, []byte(`"t3"`)) && bytes.Contains(body, []byte(`"done"`)) {
			break
		}
		time.Sleep(25 * time.Millisecond)
	}
	if !sawAnything {
		t.Fatalf("the UI never reported a task during the run; last state:\n%s", lastState)
	}
	if !sawRunning {
		t.Errorf("no task was ever observed in `running`; last state:\n%s", lastState)
	}
	if !ranAhead {
		t.Errorf("the UI only ever showed settled state, so it is not a live feed:\n%s", lastState)
	}

	res := <-done
	if res.code != ExitOK {
		t.Fatalf("runRun exit = %d; stderr=%s", res.code, res.stderr)
	}
	if !bytes.Contains(stdout.Bytes(), []byte("serving the operator UI")) {
		t.Errorf("stdout did not announce the UI:\n%s", stdout.String())
	}

	// After the run, every task is present and done.
	final, code := httpGet(t, base+"/api/state")
	if code != http.StatusOK {
		t.Fatalf("GET /api/state = %d", code)
	}
	for _, want := range []string{`"t1"`, `"t2"`, `"t3"`, `"done"`} {
		if !bytes.Contains(final, []byte(want)) {
			t.Errorf("final state missing %s:\n%s", want, final)
		}
	}
}

// TestRunRun_WithoutWebDoesNotServeUI is the regression guard:
// --web is opt-in, so a default run must be unchanged and must
// neither bind a port nor print the UI banner.
func TestRunRun_WithoutWebDoesNotServeUI(t *testing.T) {
	log := openTestAudit(t)
	var stdout, stderr bytes.Buffer
	code := runRun(context.Background(), &Args{
		Command:  CmdRun,
		PlanPath: writePlanFile(t, twoTaskPlan()),
		WebLive:  false,
	}, webTestDeps(t, log), &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("runRun exit = %d; stderr=%s", code, stderr.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte("completed")) {
		t.Errorf("stdout missing the completion summary:\n%s", stdout.String())
	}
	if bytes.Contains(stdout.Bytes(), []byte("operator UI")) {
		t.Error("a run without --web announced the operator UI")
	}
}

// TestRunRun_WebBadAddrStillRunsThePlan is a safety gate: the
// plan is the point, the UI is a convenience. A UI that cannot
// bind must not fail the work.
func TestRunRun_WebBadAddrStillRunsThePlan(t *testing.T) {
	log := openTestAudit(t)
	var stdout, stderr bytes.Buffer
	// An address that cannot be bound.
	code := runRun(context.Background(), &Args{
		Command:  CmdRun,
		PlanPath: writePlanFile(t, twoTaskPlan()),
		WebLive:  true,
		WebAddr:  "256.256.256.256:1",
	}, webTestDeps(t, log), &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("runRun exit = %d; a UI bind failure must not fail the plan (stderr=%s)",
			code, stderr.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte("completed")) {
		t.Errorf("the plan did not finish:\n%s", stdout.String())
	}
}

// TestRunDeps_ObserverIsOptional asserts a nil observer does
// not panic: every pre-existing Run call site omits it.
func TestRunDeps_ObserverIsOptional(t *testing.T) {
	log := openTestAudit(t)
	report, err := Run(context.Background(),
		&Args{PlanPath: writePlanFile(t, twoTaskPlan())},
		RunDeps{
			Audit:    log,
			Resume:   runtime.NewResumeHandler(t.TempDir()),
			Worker:   okWorker{},
			Verifier: &e2eVerifier{verdict: runtime.VerdictPass},
		})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if report.StoppedReason != "completed" {
		t.Errorf("StoppedReason = %q, want completed", report.StoppedReason)
	}
}

// TestResumeDeps_ObserverDoesNotPanic covers the resume path:
// the observer hook is registered on the shared orchestrator,
// so a resumed plan feeds the UI too.
func TestResumeDeps_ObserverDoesNotPanic(t *testing.T) {
	log := openTestAudit(t)
	hub := webui.NewHub()
	planPath := writePlanFile(t, twoTaskPlan())
	_, err := Resume(context.Background(), &Args{PlanPath: planPath}, ResumeDeps{
		Audit:    log,
		Resume:   runtime.NewResumeHandler(t.TempDir()),
		Worker:   okWorker{},
		Verify:   &e2eVerifier{verdict: runtime.VerdictPass},
		Observer: webui.NewBridge(hub),
	})
	// Resume without a checkpoint is an error in some states;
	// the point of the test is that wiring the observer never
	// panics on the way there.
	if err == nil && len(hub.Agents()) == 0 {
		t.Error("a completed resume published nothing to the hub")
	}
}

// --- flag surface ---

func TestParse_RunWebFlag(t *testing.T) {
	a, err := Parse([]string{"aios", "run", "plan.yaml", "--web"}, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if !a.WebLive {
		t.Error("--web did not set WebLive")
	}
}

func TestParse_RunWebAddrFlag(t *testing.T) {
	a, err := Parse([]string{"aios", "run", "plan.yaml", "--web", "--web-addr=127.0.0.1:8123"}, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if a.WebAddr != "127.0.0.1:8123" {
		t.Errorf("WebAddr = %q, want 127.0.0.1:8123", a.WebAddr)
	}
}

// TestUsageText_MentionsWebFlags keeps the help honest.
func TestUsageText_MentionsWebFlags(t *testing.T) {
	var buf bytes.Buffer
	_, _ = Parse([]string{"aios", "run", "--help"}, &buf)
	if !bytes.Contains(buf.Bytes(), []byte("-web")) {
		t.Errorf("run --help does not document --web:\n%s", buf.String())
	}
}
