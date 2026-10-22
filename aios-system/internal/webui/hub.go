package webui

import (
	"encoding/json"
	"sort"
	"strings"
	"sync"
	"time"
)

// Status is the lifecycle state of an agent panel.
//
// The values are the strings the browser sees, so they are
// part of the API contract: renaming one is a breaking
// change for any saved panel layout that filters by status.
type Status string

const (
	// StatusIdle means the agent exists but has no run
	// in flight.
	StatusIdle Status = "idle"
	// StatusRunning means a task is executing.
	StatusRunning Status = "running"
	// StatusBlocked means the agent is waiting on an
	// approval decision. This is distinct from
	// StatusRunning because it is the one state an
	// operator usually needs to act on.
	StatusBlocked Status = "blocked"
	// StatusDone means the last task finished
	// successfully.
	StatusDone Status = "done"
	// StatusFailed means the last task errored.
	StatusFailed Status = "failed"
)

// Agent is one panel's worth of state: a unit of work the
// operator watches. Today one agent maps to one task id;
// the indirection exists so a single agent can later carry
// several tasks without changing the wire format.
type Agent struct {
	// ID is stable for the lifetime of the process and is
	// what a saved layout keys on.
	ID string `json:"id"`
	// TaskID is the runtime task this agent is currently
	// executing, empty when idle.
	TaskID string `json:"task_id,omitempty"`
	// Label is the operator-facing name. It defaults to
	// the ID and can be renamed through the API.
	Label string `json:"label"`
	// Tool is the bus dispatch name of the current task.
	Tool string `json:"tool,omitempty"`
	// Status is the current lifecycle state.
	Status Status `json:"status"`
	// Detail is a short human-readable note, e.g. the
	// last error. It is not structured.
	Detail string `json:"detail,omitempty"`
	// TaskCount is how many tasks this agent has run.
	// The UI shows it so an operator can tell a fresh
	// agent from one that has done work.
	TaskCount int `json:"task_count"`
	// StartedAt is when the current task began, zero when
	// idle.
	StartedAt time.Time `json:"started_at,omitzero"`
	// UpdatedAt is the last mutation time.
	UpdatedAt time.Time `json:"updated_at,omitzero"`
}

// Message is one entry in the agent-to-agent
// communication panel.
type Message struct {
	// ID is unique within a process run.
	ID string `json:"id"`
	// From and To are agent IDs. To == "" is a broadcast
	// to every agent, which is how a coordinator
	// addresses the fleet.
	From string `json:"from"`
	To   string `json:"to,omitempty"`
	// Kind lets a consumer filter, e.g. "handoff" or
	// "question". Empty means a plain note.
	Kind string `json:"kind,omitempty"`
	// Body is the human-readable text.
	Body string `json:"body"`
	// At is when the message was posted.
	At time.Time `json:"at,omitzero"`
}

// event is what the hub pushes to subscribers. It is an
// internal type: the HTTP layer translates it into SSE.
type event struct {
	// Kind is "snapshot", "agent", or "message".
	Kind string `json:"kind"`
	// Payload is the JSON for that kind.
	Payload json.RawMessage `json:"payload"`
}

// Hub is the in-memory source of truth the web UI reads.
//
// Design constraints:
//
//   - It is safe for concurrent use. The runtime writes
//     from goroutines while HTTP handlers read.
//   - Subscribers never block the writer. A slow browser
//     must not stall the orchestrator, so each subscriber
//     gets a buffered channel and events are dropped for
//     a subscriber that has fallen behind. Dropping is the
//     right trade here: the UI reconciles from a fresh
//     snapshot on reconnect, so a missed intermediate
//     event costs a frame, not correctness.
//
// The hub holds no secrets. Payloads shown in the
// communication panel are whatever callers post, so
// callers must not post raw credentials.
type Hub struct {
	mu      sync.RWMutex
	agents  map[string]*Agent
	msgs    []Message
	layout  Layout
	order   int
	subs    map[int]chan event
	nextSub int
	// msgLimit caps the retained message history so a
	// long-running process cannot grow without bound.
	msgLimit int
}

// NewHub returns an empty hub with the default layout.
func NewHub() *Hub {
	return &Hub{
		agents:   map[string]*Agent{},
		subs:     map[int]chan event{},
		msgLimit: DefaultMsgLimit,
		layout:   DefaultLayout(),
	}
}

// DefaultMsgLimit is how many messages the hub retains.
const DefaultMsgLimit = 500

// subBuffer is the per-subscriber queue depth. It is
// deliberately small: a burst of updates that outruns a
// browser is dropped rather than buffered, because the
// next full snapshot corrects the view.
const subBuffer = 32

// Upsert creates or updates an agent and notifies
// subscribers.
//
// A new agent is appended to the panel order; an existing
// one keeps its position, so an agent the operator pinned
// does not jump around as tasks come and go.
func (h *Hub) Upsert(id string, mutate func(*Agent)) *Agent {
	if h == nil || id == "" {
		return nil
	}
	h.mu.Lock()
	a, ok := h.agents[id]
	if !ok {
		h.order++
		a = &Agent{ID: id, Label: id}
		h.agents[id] = a
	}
	mutate(a)
	a.UpdatedAt = time.Now().UTC()
	snapshot := *a
	ev := mustEvent("agent", snapshot)
	h.mu.Unlock()
	h.broadcast(ev)
	return &snapshot
}

// SetStatus is the common case of Upsert: move an agent
// to a status, optionally with a detail note.
//
// It is a separate function because every caller does
// exactly this and the closure would otherwise be
// repeated at each call site.
//
// TaskCount advances only on a transition INTO running,
// because it counts tasks started, not state changes. A
// task that goes running -> done is one task; counting both
// transitions would report twice the work actually done.
// StartedAt is set on the same transition for the same
// reason.
func (h *Hub) SetStatus(id string, st Status, detail string) *Agent {
	return h.Upsert(id, func(a *Agent) {
		if a.Status != st && st == StatusRunning {
			a.StartedAt = time.Now().UTC()
			a.TaskCount++
		}
		a.Status = st
		a.Detail = detail
	})
}

// Agent returns a copy of one agent.
func (h *Hub) Agent(id string) (Agent, bool) {
	if h == nil {
		return Agent{}, false
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	a, ok := h.agents[id]
	if !ok {
		return Agent{}, false
	}
	return *a, true
}

// Agents returns every agent, sorted by ID so the UI
// order is stable across polls.
//
// Sorting by ID rather than insertion order is a
// deliberate trade: insertion order is friendlier when
// tasks run sequentially, but it makes a reloaded page
// disagree with a live one once agents finish out of
// order. The saved layout is what expresses the
// operator's preferred order, so the wire order stays
// deterministic and the UI re-applies the layout on top.
func (h *Hub) Agents() []Agent {
	if h == nil {
		return nil
	}
	h.mu.RLock()
	out := make([]Agent, 0, len(h.agents))
	for _, a := range h.agents {
		out = append(out, *a)
	}
	h.mu.RUnlock()
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// Post appends a message to the communication log and
// notifies subscribers.
//
// A message whose body is empty or only whitespace is
// dropped: it carries no information and would only clutter
// the panel. Trimming the check (rather than testing for an
// empty string) means a stray space from the compose box
// cannot post a blank message.
func (h *Hub) Post(msg Message) (Message, bool) {
	if h == nil || strings.TrimSpace(msg.Body) == "" {
		return Message{}, false
	}
	h.mu.Lock()
	h.order++
	msg.ID = itoa(h.order)
	if msg.At.IsZero() {
		msg.At = time.Now().UTC()
	}
	msg.At = msg.At.UTC()
	h.msgs = append(h.msgs, msg)
	if len(h.msgs) > h.msgLimit {
		// Drop from the front: the newest messages are the
		// ones an operator is reading.
		h.msgs = h.msgs[len(h.msgs)-h.msgLimit:]
	}
	snapshot := msg
	ev := mustEvent("message", snapshot)
	h.mu.Unlock()
	h.broadcast(ev)
	return snapshot, true
}

// Messages returns the retained log, newest last.
func (h *Hub) Messages() []Message {
	if h == nil {
		return nil
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	out := make([]Message, len(h.msgs))
	copy(out, h.msgs)
	return out
}

// Layout returns the operator's saved panel layout.
func (h *Hub) Layout() Layout {
	if h == nil {
		return DefaultLayout()
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.layout
}

// SetLayout replaces the saved layout and notifies
// subscribers so every open browser converges on the new
// one.
func (h *Hub) SetLayout(l Layout) error {
	if h == nil {
		return nil
	}
	norm, err := l.Normalize()
	if err != nil {
		return err
	}
	h.mu.Lock()
	h.layout = norm
	ev := mustEvent("layout", norm)
	h.mu.Unlock()
	h.broadcast(ev)
	return nil
}

// subscribe registers a listener and returns the channel
// plus a cancel func.
//
// The caller must drain the channel; a subscriber that
// stops reading is dropped from by the broadcaster rather
// than blocking the writer.
func (h *Hub) subscribe() (<-chan event, func()) {
	ch := make(chan event, subBuffer)
	h.mu.Lock()
	id := h.nextSub
	h.nextSub++
	h.subs[id] = ch
	h.mu.Unlock()
	return ch, func() {
		h.mu.Lock()
		if c, ok := h.subs[id]; ok {
			delete(h.subs, id)
			close(c)
		}
		h.mu.Unlock()
	}
}

// broadcast fans an event out to every subscriber without
// blocking.
//
// This is called after the hub lock is released, so a slow
// consumer cannot stall a writer.
func (h *Hub) broadcast(ev event) {
	if h == nil {
		return
	}
	h.mu.RLock()
	targets := make([]chan event, 0, len(h.subs))
	for _, ch := range h.subs {
		targets = append(targets, ch)
	}
	h.mu.RUnlock()
	for _, ch := range targets {
		select {
		case ch <- ev:
		default:
			// Subscriber is behind. Dropping is safe
			// because the client re-syncs from a
			// snapshot; see the Hub doc comment.
		}
	}
}

// Snapshot is the full state a freshly-connected browser
// needs before it can render or apply deltas.
type Snapshot struct {
	Agents   []Agent   `json:"agents"`
	Messages []Message `json:"messages"`
	Layout   Layout    `json:"layout"`
	Version  int       `json:"version"`
}

// Snapshot returns the complete current state.
func (h *Hub) Snapshot() Snapshot {
	if h == nil {
		return Snapshot{Version: 0}
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	agents := make([]Agent, 0, len(h.agents))
	for _, a := range h.agents {
		agents = append(agents, *a)
	}
	sort.Slice(agents, func(i, j int) bool { return agents[i].ID < agents[j].ID })
	msgs := make([]Message, len(h.msgs))
	copy(msgs, h.msgs)
	return Snapshot{
		Agents:   agents,
		Messages: msgs,
		Layout:   h.layout,
		Version:  h.nextSub + len(h.msgs),
	}
}

// mustEvent marshals a payload that is known to be
// JSON-safe (every payload type is a struct of scalars,
// slices, and time.Time). A marshal failure would be a
// programming error, so it panics here rather than
// propagating an error through every publish path.
func mustEvent(kind string, payload any) event {
	b, err := json.Marshal(payload)
	if err != nil {
		panic("webui: cannot marshal " + kind + " event: " + err.Error())
	}
	return event{Kind: kind, Payload: b}
}

// itoa avoids importing strconv for one call site.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
