package webui

import (
	"time"

	"github.com/sadi/aios-system/internal/core/state"
	"github.com/sadi/aios-system/internal/runtime"
)

// Bridge adapts runtime lifecycle events into hub state.
//
// It lives in webui rather than in runtime because it knows
// about panels and status strings; the runtime must not
// know about either. runtime defines Observer, webui
// implements it.
//
// Concurrency: TaskStarted runs on the orchestrator's task
// loop while the HTTP handlers read the hub, so every
// method here is safe to call from another goroutine.
type Bridge struct {
	hub *Hub
}

// NewBridge returns a bridge publishing into hub.
func NewBridge(hub *Hub) *Bridge {
	return &Bridge{hub: hub}
}

// TaskStarted implements runtime.Observer.
//
// The agent ID is the task id. That is a deliberate
// one-to-one mapping today: the orchestrator runs one task
// at a time, so a task IS the unit of work an operator
// watches. When fanout runs sub-agents in parallel each
// sub-agent will carry its own id here and become its own
// panel, with no change to this file.
//
// The tool and task id are set inside one Upsert because
// SetStatus is what maintains TaskCount and StartedAt;
// splitting the write across two calls would advance the
// counter for a transition that did not happen.
func (b *Bridge) TaskStarted(e runtime.TaskEvent) {
	if b == nil || b.hub == nil {
		return
	}
	b.hub.Upsert(e.TaskID, func(a *Agent) {
		if a.Status != StatusRunning {
			if e.At.IsZero() {
				a.StartedAt = time.Now().UTC()
			} else {
				a.StartedAt = e.At
			}
			a.TaskCount++
		}
		a.Status = StatusRunning
		a.Detail = ""
		a.Tool = e.Tool
		a.TaskID = e.TaskID
	})
}

// TaskFinished implements runtime.Observer.
func (b *Bridge) TaskFinished(e runtime.TaskEvent) {
	if b == nil || b.hub == nil {
		return
	}
	b.hub.SetStatus(e.TaskID, StatusDone, "")
}

// TaskFailed implements runtime.Observer.
//
// The hub's status vocabulary is smaller than the state
// machine's, so the mapping is explicit: a task waiting
// for approval is "blocked" rather than "failed" because
// the operator's next action is to grant it, not to
// investigate a fault.
func (b *Bridge) TaskFailed(e runtime.TaskEvent) {
	if b == nil || b.hub == nil {
		return
	}
	st := StatusFailed
	detail := e.Detail
	switch e.State {
	case state.StateAwaitingApproval:
		st = StatusBlocked
	case state.StatePaused:
		st = StatusIdle
	}
	if detail == "" && e.Err != nil {
		detail = e.Err.Error()
	}
	b.hub.SetStatus(e.TaskID, st, detail)
}

// Inbox is the agent-to-agent mailbox.
//
// Agents do not currently call each other: the runtime is
// sequential and a plan's tasks are connected through the
// $vars bag, not through messages. This type exists so the
// communication panel is a real transport rather than a
// log the UI writes to and nothing reads.
//
// The semantics are deliberately narrow so it is obvious
// when the meaning changes:
//
//   - Delivery is by ID, and a broadcast (To == "") is
//     visible to every agent.
//   - Delivery is NOT an execution guarantee. Nothing in
//     the runtime consumes a message, so a delivered
//     message does not mean a task acted on it. The UI
//     labels these "sent", not "delivered".
//   - History is bounded (Hub.DefaultMsgLimit) so a long
//     run cannot grow the mailbox without limit.
//
// When a real inter-agent channel lands, it reads from
// Inbox and this type stays the same.
type Inbox struct {
	hub *Hub
}

// NewInbox returns an inbox backed by hub.
func NewInbox(hub *Hub) *Inbox { return &Inbox{hub: hub} }

// Send posts a message and returns it with its assigned id.
func (i *Inbox) Send(m Message) (Message, bool) {
	if i == nil || i.hub == nil {
		return Message{}, false
	}
	return i.hub.Post(m)
}

// For returns the messages addressed to an agent, newest
// last, including broadcasts.
//
// A snapshot rather than a subscription: the UI reads the
// full history on connect and appends deltas as they
// arrive, which is simpler than per-agent streams and
// cheap at the message volume a plan produces.
func (i *Inbox) For(agentID string) []Message {
	if i == nil || i.hub == nil {
		return nil
	}
	all := i.hub.Messages()
	out := make([]Message, 0, len(all))
	for _, m := range all {
		if m.To == "" || m.To == agentID {
			out = append(out, m)
		}
	}
	return out
}
