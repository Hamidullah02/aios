package cli

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sadi/aios-system/internal/core/schemas"
	"github.com/sadi/aios-system/internal/policy"
	"github.com/sadi/aios-system/internal/runtime"
	"github.com/sadi/aios-system/internal/tools"
)

// asWaitingForApproval is errors.As specialised for the
// approval sentinel.
func asWaitingForApproval(err error, target **runtime.WaitingForApproval) bool {
	return errors.As(err, target)
}

func pointerWFA(id, task, action, target string) *runtime.WaitingForApproval {
	return &runtime.WaitingForApproval{
		ApprovalID: id,
		Decision:   "ask",
		// A POINTER, which is what the bus worker actually
		// stores. The earlier version of this file asserted
		// the value type and silently got a zero payload.
		Payload: &schemas.ApprovalPayload{
			ApprovalID:  id,
			RequesterID: task,
			Action:      action,
			Target:      target,
		},
	}
}

// TestApprovalPayload_AcceptsBothShapes is the regression
// test for the infinite approval loop.
//
// The worker sets Payload to *schemas.ApprovalPayload, but
// the prompt and the grant store asserted the value type.
// The assertion failed, so every payload field read back as
// "": the operator saw a blank prompt, and the grant was
// keyed on task/action/target with all three empty. That key
// never matched the retried call, so the bus re-asked and
// `aios run --yes` printed 100+ approvals before dying on
// "too many approval loops".
func TestApprovalPayload_AcceptsBothShapes(t *testing.T) {
	want := schemas.ApprovalPayload{
		ApprovalID:  "apr-1",
		RequesterID: "t1",
		Action:      "shell",
		Target:      "rm -rf /",
		Reason:      "gated",
	}

	ptr := &runtime.WaitingForApproval{Payload: &want}
	val := &runtime.WaitingForApproval{Payload: want}

	for name, wfa := range map[string]*runtime.WaitingForApproval{"pointer": ptr, "value": val} {
		got, ok := approvalPayload(wfa)
		if !ok {
			t.Fatalf("%s: payload not recognised", name)
		}
		if got != want {
			t.Errorf("%s: payload = %+v, want %+v", name, got, want)
		}
	}
}

// TestApprovalPayload_Missing covers the no-payload case so a
// caller cannot crash on it.
func TestApprovalPayload_Missing(t *testing.T) {
	if _, ok := approvalPayload(nil); ok {
		t.Error("nil request should yield no payload")
	}
	if _, ok := approvalPayload(&runtime.WaitingForApproval{}); ok {
		t.Error("empty payload should yield no payload")
	}
	// A wrong type must not panic and must not claim success.
	wfa := &runtime.WaitingForApproval{Payload: 42}
	if _, ok := approvalPayload(wfa); ok {
		t.Error("unrelated payload type should yield no payload")
	}
}

// TestGrantStore_MatchesRetry is the other half of the loop
// regression: a grant recorded from the request must satisfy
// the check the bus makes on the retried call.
func TestGrantStore_MatchesRetry(t *testing.T) {
	g := newGrantStore(nil)
	g.Grant(pointerWFA("uuid-1", "t1", "shell", "echo hi"))

	// The gateway mints a FRESH uuid on the retry, so the
	// grant must not depend on the id matching.
	retried := &policy.Decision{Action: policy.ActionAsk, Approval: &schemas.ApprovalPayload{
		ApprovalID:  "uuid-2-different",
		RequesterID: "t1",
		Action:      "shell",
		Target:      "echo hi",
	}}
	if !g.IsApproved(retried) {
		t.Error("an approved call must be recognised on retry despite a new approval id")
	}
}

// TestGrantStore_DoesNotOverreach is the safety half: a grant
// authorises exactly one call.
func TestGrantStore_DoesNotOverreach(t *testing.T) {
	g := newGrantStore(nil)
	g.Grant(pointerWFA("uuid-1", "t1", "shell", "echo hi"))

	cases := []struct {
		name                 string
		task, action, target string
	}{
		{"different target", "t1", "shell", "rm -rf /"},
		{"different task", "t2", "shell", "echo hi"},
		{"different action", "t1", "write", "echo hi"},
	}
	for _, c := range cases {
		d := &policy.Decision{Approval: &schemas.ApprovalPayload{
			RequesterID: c.task, Action: c.action, Target: c.target,
		}}
		if g.IsApproved(d) {
			t.Errorf("%s: grant must not carry over", c.name)
		}
	}
}

// TestGrantStore_NilAndEmpty is defensive: the bus runs with a
// nil store in several subcommands.
func TestGrantStore_NilAndEmpty(t *testing.T) {
	var g *grantStore
	g.Grant(pointerWFA("a", "t1", "shell", "x")) // must not panic
	if g.IsApproved(&policy.Decision{Approval: &schemas.ApprovalPayload{}}) {
		t.Error("nil store approves nothing")
	}
	empty := newGrantStore(nil)
	if empty.IsApproved(&policy.Decision{}) {
		t.Error("a decision with no approval cannot be approved")
	}
	if empty.IsApproved(nil) {
		t.Error("a nil decision cannot be approved")
	}
}

// TestGrantStore_NoLoopAfterApproval is the end-to-end guard:
// with a grant store consulted by the bus, the same gated call
// must succeed on the retry instead of asking again.
//
// Without this the run loop spins to MaxApprovalLoops and
// fails with "too many approval loops", which is the exact
// symptom the operator sees.
func TestGrantStore_NoLoopAfterApproval(t *testing.T) {
	log := openTestAudit(t)
	defer log.Close()

	gw := policy.New([]policy.Rule{
		{ID: "ask-read", Match: policy.Match{Tool: "read_file"},
			Action: policy.ActionAsk, Reason: "operator review",
			Approval: &policy.ApprovalTemplate{Action: "read", Resource: "{tool}"}},
	}, log)

	grants := newGrantStore(nil)
	bus := tools.New(tools.Config{
		Tools:  map[string]tools.Tool{"read_file": tools.ReadFileTool{}},
		Policy: gw,
		Audit:  log,
		Grants: grants,
	})
	// A real file inside the workspace: the point of this test
	// is the policy gate, so the tool itself must succeed.
	ws := t.TempDir()
	if err := os.WriteFile(filepath.Join(ws, "go.mod"), []byte("module x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	w := NewBusWorker(bus, nil, ws)

	// First attempt: gated, no grant yet.
	_, err := w.Run(runtime.TaskInput{ID: "t1", Tool: "read_file",
		Inputs: []string{"path=go.mod"}},
		&runtime.TaskContext{Vars: map[string]any{}}, nil)
	if err == nil {
		t.Fatal("first attempt should be gated")
	}
	var wfa *runtime.WaitingForApproval
	if !asWaitingForApproval(err, &wfa) {
		t.Fatalf("expected *WaitingForApproval, got %T: %v", err, err)
	}

	// Operator says yes.
	grants.Grant(wfa)

	// Second attempt: identical call, must now be allowed.
	res, err := w.Run(runtime.TaskInput{ID: "t1", Tool: "read_file",
		Inputs: []string{"path=go.mod"}},
		&runtime.TaskContext{Vars: map[string]any{}}, nil)
	if err != nil {
		t.Fatalf("approved retry should succeed, got %v", err)
	}
	if res.Output == "" {
		t.Error("approved retry should return the tool's summary")
	}
}

// TestApprovalPrompt_ShowsRealPayload pins that the operator
// sees the command they are approving, not blank fields.
func TestApprovalPrompt_ShowsRealPayload(t *testing.T) {
	var out bytes.Buffer
	p := &ApprovalPrompt{In: strings.NewReader("y\n"), Out: &out}
	ok, err := p.Approve(context.Background(), pointerWFA("apr-9", "t1", "shell", "rm -rf /"))
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected approval")
	}
	s := out.String()
	for _, want := range []string{"apr-9", "shell", "rm -rf /"} {
		if !strings.Contains(s, want) {
			t.Errorf("prompt missing %q:\n%s", want, s)
		}
	}
}
