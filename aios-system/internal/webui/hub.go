package webui

import (
	"encoding/json"
	"sort"
	"strconv"
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
	// StartedAt is when the current task began, absent when
	// idle. It serialises as null rather than being omitted:
	// the UI sorts on it, so the key must be present even when
	// the value is not. encoding/json ignores omitempty on a
	// struct field, so the tag would be a promise it cannot
	// keep.
	StartedAt Time `json:"started_at"`
	// UpdatedAt is the last mutation time.
	UpdatedAt Time `json:"updated_at"`
}

// Time is a JSON timestamp that can be absent.
//
// This exists instead of a time.Time with `json:",omitzero"`
// because omitzero is a Go 1.24 encoding/json addition. On
// the Go 1.23 toolchain this module targets, the option is
// parsed as an unknown format verb and SILENTLY ignored, so
// an idle agent's zero time was serialised as
// "0001-01-01T00:00:00Z". The UI sorted its activity feed on
// that string, so a placeholder timestamp sorted above real
// activity instead of below it.
//
// A nil Time marshals to JSON null and a set Time to an
// RFC 3339 string, which is the distinction the UI actually
// needs: "never started" rather than "started at year one".
type Time struct {
	time.Time
}

// NewTime wraps a time, treating the zero time as absent.
func NewTime(t time.Time) Time {
	if t.IsZero() {
		return Time{}
	}
	return Time{Time: t}
}

// MarshalJSON writes null for an absent timestamp.
func (t Time) MarshalJSON() ([]byte, error) {
	if t.IsZero() {
		return []byte("null"), nil
	}
	return json.Marshal(t.Time)
}

// UnmarshalJSON reads either null or an RFC 3339 string.
//
// A client that sends a string, and a server that round-trips
// one, both work; an absent field leaves the zero Time, which
// re-marshals as null.
func (t *Time) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		t.Time = time.Time{}
		return nil
	}
	return t.Time.UnmarshalJSON(b)
}

// IsZero reports whether the timestamp is absent.
func (t Time) IsZero() bool { return t.Time.IsZero() }

// Or returns the wrapped time, or zero when absent. It keeps
// callers that need the raw value from reaching through the
// embedded field at every site.
func (t Time) Or() time.Time { return t.Time }

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
	At Time `json:"at"`
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

	// version counts mutations. See Snapshot.Version.
	version int

	// store persists the layout across process restarts. It is
	// optional: a hub with no store keeps the layout in memory
	// only, which is what a test or a throwaway server wants.
	store *LayoutStore

	// approvals holds the pending approval requests an operator
	// can decide from the browser.
	//
	// This is the one place the UI does more than read. It is
	// safe because the registry forwards every decision to
	// whichever process owns the approvals rather than
	// deciding anything itself, so there is still exactly one
	// source of truth for run state.
	approvals *approvalRegistry

	// tasks holds the things an operator asked for from the
	// browser, and what became of them.
	//
	// It is separate from the agent panels on purpose. An agent
	// panel is the machinery doing the work; a task is the job
	// the operator asked for. The task exists before any agent
	// does and outlives them, so it cannot be derived from the
	// agent list -- and an operator who typed one sentence and
	// watched it fail needs to see that sentence next to the
	// failure.
	tasks *TaskStore

	// submitter turns an operator's goal into work. It is nil
	// when this server is read-only over run state.
	submitter TaskSubmitter
}

// NewHub returns an empty hub with the default layout.
func NewHub() *Hub {
	return &Hub{
		agents:    map[string]*Agent{},
		subs:      map[int]chan event{},
		msgLimit:  DefaultMsgLimit,
		layout:    DefaultLayout(),
		approvals: newApprovalRegistry(nil),
		tasks:     NewTaskStore(),
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
	a.UpdatedAt = NewTime(time.Now().UTC())
	h.version++
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
			a.StartedAt = NewTime(time.Now().UTC())
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
	msg.ID = strconv.Itoa(h.order)
	msg.At = NewTime(msg.At.UTC())
	h.msgs = append(h.msgs, msg)
	h.version++
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
//
// When the hub has a store, the layout is written through to
// it. A persistence failure is returned, but the in-memory
// layout is still updated: the operator's drag took effect in
// the open tab, and reporting an error while silently
// discarding it would be worse than the reverse. The in-memory
// state is the source of truth for this process; the store is
// how it survives the next one.
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
	h.version++
	ev := mustEvent("layout", norm)
	store := h.store
	h.mu.Unlock()
	h.broadcast(ev)
	if store == nil {
		return nil
	}
	return store.Save(norm)
}

// SetStore attaches a layout store and loads whatever it has
// saved into the hub.
//
// The load is reported separately from the attachment: a
// corrupt layout file must not stop the server from starting,
// so the hub keeps a usable default and the caller decides
// whether to mention the failure.
func (h *Hub) SetStore(s *LayoutStore) error {
	if h == nil {
		return nil
	}
	l, found, err := s.Load()
	h.mu.Lock()
	h.store = s
	h.mu.Unlock()
	if !found {
		// Nothing saved yet. Leave the current layout alone:
		// Attach may be called after a caller has already set
		// one deliberately.
		return err
	}
	if err != nil {
		return err
	}
	// Assign directly rather than through SetLayout, so loading
	// the saved layout does not immediately rewrite the file
	// that it just came from.
	h.mu.Lock()
	h.layout = l
	h.version++
	h.mu.Unlock()
	return nil
}

// AddApproval records a pending approval and notifies
// subscribers, so an open browser sees the request without
// waiting for its next poll.
func (h *Hub) AddApproval(a PendingApproval) {
	if h == nil {
		return
	}
	h.approvals.Add(a)
	h.mu.Lock()
	h.version++
	h.mu.Unlock()
	h.broadcast(mustEvent("approvals", h.approvals.Pending()))
}

// RemoveApproval clears a pending approval and notifies
// subscribers.
func (h *Hub) RemoveApproval(id string) {
	if h == nil {
		return
	}
	if !h.approvals.Has(id) {
		return
	}
	h.approvals.Remove(id)
	h.mu.Lock()
	h.version++
	h.mu.Unlock()
	h.broadcast(mustEvent("approvals", h.approvals.Pending()))
}

// PendingApprovals returns the approval requests waiting on a
// decision, oldest first.
func (h *Hub) PendingApprovals() []PendingApproval {
	if h == nil {
		return nil
	}
	return h.approvals.Pending()
}

// CanResolveApprovals reports whether this hub can apply a
// decision. False means the UI must point the operator
// somewhere else rather than showing a button that cannot
// work.
func (h *Hub) CanResolveApprovals() bool {
	if h == nil {
		return false
	}
	return h.approvals.CanResolve()
}

// SetApprovalResolver attaches the owner of the approvals, so
// a decision made in the browser reaches the run.
func (h *Hub) SetApprovalResolver(r ApprovalResolver) {
	if h == nil {
		return
	}
	h.approvals.SetResolver(r)
}

// DecideApproval applies an operator's decision.
func (h *Hub) DecideApproval(id string, approve bool) (string, error) {
	if h == nil {
		return "", ErrNoResolver
	}
	state, err := h.approvals.Decide(id, approve)
	if err != nil {
		return "", err
	}
	h.mu.Lock()
	h.version++
	h.mu.Unlock()
	h.broadcast(mustEvent("approvals", h.approvals.Pending()))
	return state, nil
}

// Tasks returns the submitted tasks, oldest first.
func (h *Hub) Tasks() []Task {
	if h == nil {
		return nil
	}
	return h.tasks.Tasks()
}

// TaskStore returns the hub's task store.
func (h *Hub) TaskStore() *TaskStore {
	if h == nil {
		return nil
	}
	return h.tasks
}

// SetSubmitter attaches the thing that turns a goal into work.
//
// It is set after NewHub rather than in the constructor
// because the run builds its planner and its hub in either
// order, and the hub is also created by a server that never
// runs anything.
func (h *Hub) SetSubmitter(s TaskSubmitter) {
	if h == nil {
		return
	}
	h.mu.Lock()
	h.submitter = s
	h.mu.Unlock()
}

// Submitter returns the attached submitter, or nil.
func (h *Hub) Submitter() TaskSubmitter {
	if h == nil {
		return nil
	}
	h.mu.RLock()
	s := h.submitter
	h.mu.RUnlock()
	return s
}

// CanSubmitTasks reports whether this hub can start work.
//
// False means the operator must be pointed somewhere else
// rather than shown a button that cannot work.
func (h *Hub) CanSubmitTasks() bool {
	if h == nil {
		return false
	}
	s := h.Submitter()
	return s != nil && s.CanSubmit()
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
	// Send while holding the read lock rather than snapshotting
	// the channel list and releasing first.
	//
	// The previous version copied the channels, dropped the
	// lock, and then sent. A subscriber whose cancel() ran in
	// that window closed the channel between the two steps and
	// the send panicked: "send on closed channel". Closing and
	// sending are now both under h.mu, so they cannot interleave.
	//
	// The lock is not held across a blocking operation: every
	// send is in a select with a default, so a subscriber that
	// has stopped reading costs a dropped event and no wait.
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, ch := range h.subs {
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
	// Tasks are the things an operator asked for from the
	// browser. Always present, never null: the UI checks
	// length, and a null that is sometimes an array is a
	// footgun in JavaScript.
	Tasks []Task `json:"tasks"`
	// CanSubmitTasks says whether this server can start work.
	// False means the operator must use another route, and the
	// UI must not offer a submit button that cannot work.
	CanSubmitTasks bool `json:"can_submit_tasks"`
	// Approvals are the requests waiting on an operator
	// decision. Always present, never null: the UI checks
	// length, and a null that is sometimes an array is a
	// footgun in JavaScript.
	Approvals []PendingApproval `json:"approvals"`
	// CanResolveApprovals says whether this server can apply a
	// decision. False means an operator must resolve at the
	// terminal or over the daemon, and the UI must not offer
	// buttons that cannot work.
	CanResolveApprovals bool `json:"can_resolve_approvals"`
	// Version increases by one on every mutation, so a client
	// can tell "the snapshot I hold is current" from "the
	// world moved on".
	//
	// It previously carried nextSub + len(msgs), which is not
	// a version of anything: it went backwards when a browser
	// disconnected, and moved on a message post without any
	// agent changing. A client comparing it across snapshots
	// would have seen spurious equality.
	Version int `json:"version"`
}

// Snapshot returns the complete current state.
func (h *Hub) Snapshot() Snapshot {
	if h == nil {
		return Snapshot{Version: 0}
	}
	// The task store has its own lock. Read it before taking
	// the hub lock so the two are never held at once: the hub
	// lock is the outer one everywhere else, and a reader that
	// acquires them in the other order is a deadlock waiting
	// for a writer to arrive between the two.
	tasks := h.tasks.Tasks()
	h.mu.RLock()
	defer h.mu.RUnlock()
	agents := make([]Agent, 0, len(h.agents))
	for _, a := range h.agents {
		agents = append(agents, *a)
	}
	sort.Slice(agents, func(i, j int) bool { return agents[i].ID < agents[j].ID })
	msgs := make([]Message, len(h.msgs))
	copy(msgs, h.msgs)
	// Read the submitter under the lock already held rather
	// than calling CanSubmitTasks, which would take the read
	// lock a second time. Go's RWMutex blocks a recursive read
	// lock when a writer is queued, so the nested call could
	// deadlock this very snapshot.
	canSubmit := h.submitter != nil && h.submitter.CanSubmit()
	return Snapshot{
		Agents:              agents,
		Messages:            msgs,
		Layout:              h.layout,
		Tasks:               tasks,
		CanSubmitTasks:      canSubmit,
		Approvals:           h.approvals.Pending(),
		CanResolveApprovals: h.approvals.CanResolve(),
		Version:             h.version,
	}
}

// mustEvent marshals a payload that is known to be
// JSON-safe (every payload type is a struct of scalars,
// slices, and Time). A marshal failure would be a
// programming error, so it panics here rather than
// propagating an error through every publish path.
func mustEvent(kind string, payload any) event {
	b, err := json.Marshal(payload)
	if err != nil {
		panic("webui: cannot marshal " + kind + " event: " + err.Error())
	}
	return event{Kind: kind, Payload: b}
}
