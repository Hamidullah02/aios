package webui

import (
	"errors"
	"testing"

	"github.com/sadi/aios-system/internal/core/state"
	"github.com/sadi/aios-system/internal/runtime"
)

// --- bridge ---

// TestBridge_StartedMarksRunning is the panel's core
// transition: a task starting must light up as running
// rather than waiting for completion.
func TestBridge_StartedMarksRunning(t *testing.T) {
	hub := NewHub()
	b := NewBridge(hub)
	b.TaskStarted(runtime.TaskEvent{TaskID: "t1", Tool: "read_file"})

	a, ok := hub.Agent("t1")
	if !ok {
		t.Fatal("agent t1 was not created")
	}
	if a.Status != StatusRunning {
		t.Errorf("status = %q, want running", a.Status)
	}
	if a.Tool != "read_file" {
		t.Errorf("tool = %q, want read_file", a.Tool)
	}
	if a.TaskID != "t1" {
		t.Errorf("task_id = %q, want t1", a.TaskID)
	}
}

func TestBridge_FinishedMarksDone(t *testing.T) {
	hub := NewHub()
	b := NewBridge(hub)
	b.TaskStarted(runtime.TaskEvent{TaskID: "t1"})
	b.TaskFinished(runtime.TaskEvent{TaskID: "t1", State: state.StateCompleted})

	a, _ := hub.Agent("t1")
	if a.Status != StatusDone {
		t.Errorf("status = %q, want done", a.Status)
	}
}

// TestBridge_AwaitingApprovalIsBlockedNotFailed is the
// mapping that matters most to an operator. A task waiting
// on approval is not a fault: the next action is to grant
// it, and showing it red would send the operator looking for
// a bug instead of to the approval prompt.
func TestBridge_AwaitingApprovalIsBlockedNotFailed(t *testing.T) {
	hub := NewHub()
	b := NewBridge(hub)
	b.TaskFailed(runtime.TaskEvent{
		TaskID: "t1",
		State:  state.StateAwaitingApproval,
		Detail: "waiting_for_approval",
	})

	a, _ := hub.Agent("t1")
	if a.Status != StatusBlocked {
		t.Errorf("status = %q, want blocked", a.Status)
	}
	if a.Detail != "waiting_for_approval" {
		t.Errorf("detail = %q", a.Detail)
	}
}

func TestBridge_PausedIsIdle(t *testing.T) {
	hub := NewHub()
	b := NewBridge(hub)
	b.TaskFailed(runtime.TaskEvent{TaskID: "t1", State: state.StatePaused})
	a, _ := hub.Agent("t1")
	if a.Status != StatusIdle {
		t.Errorf("status = %q, want idle (paused is not a failure)", a.Status)
	}
}

func TestBridge_FailureFallsBackToErrorText(t *testing.T) {
	hub := NewHub()
	b := NewBridge(hub)
	b.TaskFailed(runtime.TaskEvent{
		TaskID: "t1",
		State:  state.StateFailed,
		Err:    errors.New("policy denied run_shell"),
	})
	a, _ := hub.Agent("t1")
	if a.Status != StatusFailed {
		t.Errorf("status = %q, want failed", a.Status)
	}
	if a.Detail != "policy denied run_shell" {
		t.Errorf("detail = %q, want the error text", a.Detail)
	}
}

// TestBridge_TaskCountTracksStatusChanges covers the counter
// the UI shows so an operator can tell a fresh agent from
// one that has done work.
func TestBridge_TaskCountTracksStatusChanges(t *testing.T) {
	hub := NewHub()
	b := NewBridge(hub)
	b.TaskStarted(runtime.TaskEvent{TaskID: "t1"})
	b.TaskFinished(runtime.TaskEvent{TaskID: "t1"})
	b.TaskStarted(runtime.TaskEvent{TaskID: "t1"})
	b.TaskFinished(runtime.TaskEvent{TaskID: "t1"})

	a, _ := hub.Agent("t1")
	if a.TaskCount != 2 {
		t.Errorf("task_count = %d, want 2", a.TaskCount)
	}
}

func TestBridge_NilSafe(t *testing.T) {
	var b *Bridge
	b.TaskStarted(runtime.TaskEvent{TaskID: "t1"})
	b.TaskFinished(runtime.TaskEvent{TaskID: "t1"})
	b.TaskFailed(runtime.TaskEvent{TaskID: "t1"})
	// A bridge with a nil hub must also be inert.
	NewBridge(nil).TaskStarted(runtime.TaskEvent{TaskID: "t1"})
}

// --- inbox ---

func TestInbox_SendAndDeliver(t *testing.T) {
	hub := NewHub()
	in := NewInbox(hub)
	hub.SetStatus("t1", StatusIdle, "")
	hub.SetStatus("t2", StatusIdle, "")

	if _, ok := in.Send(Message{From: "t1", To: "t2", Body: "handoff"}); !ok {
		t.Fatal("Send returned false for a valid message")
	}
	in.Send(Message{From: "orchestrator", Body: "everyone stop"})

	got := in.For("t2")
	if len(got) != 2 {
		t.Fatalf("t2 inbox = %d messages, want 2 (one direct, one broadcast)", len(got))
	}
	// t1 must not see the message addressed to t2.
	if got1 := in.For("t1"); len(got1) != 1 {
		t.Errorf("t1 inbox = %d, want 1 (the broadcast only)", len(got1))
	}
}

func TestInbox_EmptyBodyRejected(t *testing.T) {
	in := NewInbox(NewHub())
	if _, ok := in.Send(Message{From: "t1", Body: "  "}); ok {
		t.Error("a whitespace-only body must be rejected")
	}
}

// --- layout normalisation ---

func TestNormalize_AddsMissingPanels(t *testing.T) {
	// A layout saved before a panel existed must still work;
	// the new panel appears with its defaults.
	got, err := Layout{Panels: []PanelState{{ID: PanelComms, Visible: false, Width: 6}}}.Normalize()
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}
	if len(got.Panels) != len(AllPanels) {
		t.Errorf("panels = %d, want %d", len(got.Panels), len(AllPanels))
	}
}

func TestNormalize_RejectsUnknownPanel(t *testing.T) {
	// Silently dropping one would hide the operator's typo.
	_, err := Layout{Panels: []PanelState{{ID: "typo"}}}.Normalize()
	if err == nil {
		t.Fatal("expected an error for an unknown panel")
	}
}

func TestNormalize_DropsDuplicatePanel(t *testing.T) {
	got, err := Layout{Panels: []PanelState{
		{ID: PanelComms, Visible: true, Width: 4},
		{ID: PanelComms, Visible: false, Width: 8},
	}}.Normalize()
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}
	n := 0
	for _, p := range got.Panels {
		if p.ID == PanelComms {
			n++
			if p.Width != 4 {
				t.Errorf("width = %d, want the first occurrence (4)", p.Width)
			}
		}
	}
	if n != 1 {
		t.Errorf("comms appears %d times, want 1", n)
	}
}

func TestNormalize_ClampsWidth(t *testing.T) {
	for _, tc := range []struct{ in, want int }{{0, 1}, {-5, 1}, {99, MaxWidth}, {7, 7}} {
		got, err := Layout{Panels: []PanelState{{ID: PanelAgents, Width: tc.in}}}.Normalize()
		if err != nil {
			t.Fatalf("Normalize: %v", err)
		}
		for _, p := range got.Panels {
			if p.ID == PanelAgents && p.Width != tc.want {
				t.Errorf("width %d -> %d, want %d", tc.in, p.Width, tc.want)
			}
		}
	}
}

func TestNormalize_RejectsTooManyPanels(t *testing.T) {
	ps := make([]PanelState, MaxPanels+1)
	for i := range ps {
		ps[i] = PanelState{ID: PanelAgents}
	}
	if _, err := (Layout{Panels: ps}).Normalize(); err == nil {
		t.Error("expected an error past the panel cap")
	}
}

func TestNormalize_Theme(t *testing.T) {
	// An unknown theme becomes "follow the OS" rather than
	// being rejected: a stale saved value must not block the
	// rest of the layout from loading.
	got, err := Layout{Theme: "solarized"}.Normalize()
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}
	if got.Theme != "" {
		t.Errorf("theme = %q, want empty (follow the OS)", got.Theme)
	}
	for _, want := range []string{"dark", "light"} {
		got, _ := Layout{Theme: want}.Normalize()
		if got.Theme != want {
			t.Errorf("theme = %q, want %q", got.Theme, want)
		}
	}
}

func TestNormalize_SortsByOrder(t *testing.T) {
	got, err := Layout{Panels: []PanelState{
		{ID: PanelAudit, Order: 5},
		{ID: PanelComms, Order: 1},
		{ID: PanelAgents, Order: 9},
	}}.Normalize()
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}
	// Sorted list, agents added back at the end (order 0 is
	// taken, and ties break on ID).
	if got.Panels[0].ID != PanelComms {
		t.Errorf("first panel = %s, want comms (order 1)", got.Panels[0].ID)
	}
	if got.Panels[1].ID != PanelAudit {
		t.Errorf("second panel = %s, want audit (order 5)", got.Panels[1].ID)
	}
}
