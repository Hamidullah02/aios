package cli

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/sadi/aios-system/internal/core/schemas"
	"github.com/sadi/aios-system/internal/runtime"
)

func testWFA() *runtime.WaitingForApproval {
	return &runtime.WaitingForApproval{
		ApprovalID: "ask:t1",
		Decision:   "ask",
		Payload: schemas.ApprovalPayload{
			Action:      "shell",
			Target:      "echo hello",
			Reason:      "shell access is gated",
			RequesterID: "bus-worker",
		},
	}
}

// TestApprovalPrompt_Yes approves when the operator types y.
func TestApprovalPrompt_Yes(t *testing.T) {
	var out bytes.Buffer
	p := &ApprovalPrompt{In: strings.NewReader("y\n"), Out: &out}
	ok, err := p.Approve(context.Background(), testWFA())
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected approval")
	}
	s := out.String()
	// The operator must SEE what they approved. Showing the
	// target is the whole point of asking.
	for _, want := range []string{"ask:t1", "shell", "echo hello", "shell access is gated"} {
		if !strings.Contains(s, want) {
			t.Errorf("prompt missing %q:\n%s", want, s)
		}
	}
	if !strings.Contains(s, "approved") {
		t.Errorf("prompt should confirm the decision:\n%s", s)
	}
}

// TestApprovalPrompt_No denies when the operator types n.
func TestApprovalPrompt_No(t *testing.T) {
	var out bytes.Buffer
	p := &ApprovalPrompt{In: strings.NewReader("n\n"), Out: &out}
	ok, err := p.Approve(context.Background(), testWFA())
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expected denial")
	}
}

// TestApprovalPrompt_EmptyLineDenies covers the bare Enter
// case, which must NOT be read as consent.
func TestApprovalPrompt_EmptyLineDenies(t *testing.T) {
	var out bytes.Buffer
	p := &ApprovalPrompt{In: strings.NewReader("\n"), Out: &out}
	ok, err := p.Approve(context.Background(), testWFA())
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("an empty answer must not approve")
	}
}

// TestApprovalPrompt_EOFDenies is the CI / piped-stdin case.
//
// A closed stdin is not an operator saying no; it is nobody
// being there. Treating it as a DENIAL is the safe reading:
// a gated action must not proceed without a human. Treating
// it as approval would make `echo | aios run plan.yaml`
// silently execute shell commands.
func TestApprovalPrompt_EOFDenies(t *testing.T) {
	var out bytes.Buffer
	p := &ApprovalPrompt{In: strings.NewReader(""), Out: &out}
	ok, err := p.Approve(context.Background(), testWFA())
	if err != nil {
		t.Fatalf("EOF should be a decision, not an error: %v", err)
	}
	if ok {
		t.Fatal("a run with nobody at the keyboard must not approve")
	}
}

// TestApprovalPrompt_TimeoutDenies asserts the prompt cannot
// hang forever on a terminal nobody is watching.
func TestApprovalPrompt_TimeoutDenies(t *testing.T) {
	// An io.Reader that never returns data and never EOFs.
	pr, pw := io.Pipe()
	defer pw.Close()

	var out bytes.Buffer
	p := &ApprovalPrompt{In: pr, Out: &out, Timeout: 60 * time.Millisecond}
	done := make(chan bool, 1)
	go func() {
		ok, _ := p.Approve(context.Background(), testWFA())
		done <- ok
	}()
	select {
	case ok := <-done:
		if ok {
			t.Fatal("a timeout must not approve")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("prompt did not honour its timeout")
	}
}

// TestApprovalPrompt_AssumeYesAnnounces pins the --yes
// behaviour: auto-approval must be VISIBLE, so a transcript
// or CI log shows no human decided.
func TestApprovalPrompt_AssumeYesAnnounces(t *testing.T) {
	var out bytes.Buffer
	p := &ApprovalPrompt{In: strings.NewReader(""), Out: &out, AssumeYes: true}
	ok, err := p.Approve(context.Background(), testWFA())
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("--yes should approve")
	}
	if !strings.Contains(out.String(), "AUTO-APPROVED") {
		t.Errorf("auto-approval must be announced:\n%s", out.String())
	}
}

// TestNewApprovalHandler_Precedence locks the selection order.
func TestNewApprovalHandler_Precedence(t *testing.T) {
	deps := &Deps{Stdin: strings.NewReader(""), Stderr: &bytes.Buffer{}}

	t.Run("yes wins", func(t *testing.T) {
		h := newApprovalHandler(&Args{AssumeYes: true, SocketExplicit: true}, deps)
		if _, ok := h.(*ApprovalPrompt); !ok {
			t.Fatalf("--yes should give a prompt handler, got %T", h)
		}
	})
	t.Run("explicit socket gives daemon", func(t *testing.T) {
		h := newApprovalHandler(&Args{SocketPath: "/tmp/x.sock", SocketExplicit: true}, deps)
		if _, ok := h.(*DaemonApprovalHandler); !ok {
			t.Fatalf("explicit --socket should give the daemon handler, got %T", h)
		}
	})
	t.Run("default socket gives prompt", func(t *testing.T) {
		// A default socket value is NOT the operator asking
		// for the daemon: they never mentioned it.
		h := newApprovalHandler(&Args{SocketPath: DefaultSocketPath()}, deps)
		if _, ok := h.(*ApprovalPrompt); !ok {
			t.Fatalf("default should give the prompt handler, got %T", h)
		}
	})
}