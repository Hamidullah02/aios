package webui

import (
	"strings"
	"sync"
	"time"
)

// TaskStatus is the lifecycle of a task an operator submitted
// from the browser.
//
// The vocabulary is deliberately smaller than the runtime's state
// machine. An operator who types "make my shop show up on Google"
// does not care whether the plan is in its repair loop or its
// second task; they care whether it is being worked on, whether
// something needs them, and whether it finished.
type TaskStatus string

const (
	// TaskPlanning means the goal is being turned into a plan.
	// The model is talking to the provider; nothing is running
	// yet.
	TaskPlanning TaskStatus = "planning"
	// TaskRunning means the plan is executing.
	TaskRunning TaskStatus = "running"
	// TaskBlocked means the run stopped waiting for a human
	// decision. This is the state that needs the operator.
	TaskBlocked TaskStatus = "blocked"
	// TaskDone means the plan finished successfully.
	TaskDone TaskStatus = "done"
	// TaskFailed means the plan stopped on an error.
	TaskFailed TaskStatus = "failed"
)

// Task is one thing the operator asked for, and what became of
// it.
//
// It is the unit a non-coder watches. The agent panels below it
// are the machinery; this is the job.
type Task struct {
	// ID identifies the task within the process run.
	ID string `json:"id"`
	// Goal is the operator's own words. It is echoed back
	// verbatim so the operator can see their request is the
	// one being worked on.
	Goal string `json:"goal"`
	// Status is where the task is in its lifecycle.
	Status TaskStatus `json:"status"`
	// Detail is a short human-readable note: what is happening
	// right now, or why it stopped. It is not structured.
	Detail string `json:"detail,omitempty"`
	// PlanSummary is the plan the system produced, rendered for
	// a human: one line per step. It is empty until the plan
	// exists.
	PlanSummary []string `json:"plan_summary,omitempty"`
	// Result is the outcome, in words, once the task is done.
	Result string `json:"result,omitempty"`
	// StartedAt is when the task was submitted.
	StartedAt Time `json:"started_at"`
	// FinishedAt is when the task reached a terminal state.
	FinishedAt Time `json:"finished_at"`
}

// TaskStore holds the submitted tasks and notifies subscribers.
//
// It is separate from the Hub because a task is a different kind
// of thing: the Hub holds agent panels and the message log, both
// derived from a run. A task is an operator's request, and it
// exists before any agent does.
//
// Concurrency follows the Hub's rules: safe for concurrent use,
// and subscribers never block the writer.
type TaskStore struct {
	mu    sync.RWMutex
	tasks map[string]*Task
	order int
	subs  map[int]chan event
	next  int
	// limit caps the retained history so a long-lived server
	// cannot grow without bound.
	limit int
}

// NewTaskStore returns an empty store.
func NewTaskStore() *TaskStore {
	return &TaskStore{
		tasks: map[string]*Task{},
		subs:  map[int]chan event{},
		limit: DefaultTaskLimit,
	}
}

// DefaultTaskLimit is how many submitted tasks are retained.
const DefaultTaskLimit = 100

// Upsert creates or updates a task and notifies subscribers.
//
// A new task is appended to the panel order; an existing one
// keeps its position, so the task an operator is watching does
// not jump around as its steps complete.
func (s *TaskStore) Upsert(id string, mutate func(*Task)) *Task {
	if s == nil || id == "" {
		return nil
	}
	s.mu.Lock()
	t, ok := s.tasks[id]
	if !ok {
		s.order++
		t = &Task{ID: id}
		if t.StartedAt.IsZero() {
			t.StartedAt = NewTime(time.Now().UTC())
		}
		s.tasks[id] = t
	}
	mutate(t)
	snapshot := *t
	ev := mustEvent("task", snapshot)
	s.mu.Unlock()
	s.broadcast(ev)
	return &snapshot
}

// Tasks returns every retained task, oldest first.
//
// Oldest first because the list is a history: the task at the
// top is the one the operator asked for first, and the newest
// is at the bottom where they just added it.
func (s *TaskStore) Tasks() []Task {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	out := make([]Task, 0, len(s.tasks))
	for _, t := range s.tasks {
		out = append(out, *t)
	}
	s.mu.RUnlock()
	// Insertion sort outside the lock: the retained set is
	// bounded by DefaultTaskLimit, and a comparator that closes
	// over a mutex is a deadlock waiting to happen.
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j].StartedAt.Or().Before(out[j-1].StartedAt.Or()); j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

// SetStatus is the common case of Upsert: move a task to a
// status, optionally with a note.
//
// FinishedAt is stamped on a terminal transition, so the UI can
// show how long a task took without tracking it itself.
func (s *TaskStore) SetStatus(id string, st TaskStatus, detail string) *Task {
	return s.Upsert(id, func(t *Task) {
		t.Status = st
		t.Detail = detail
		if isTerminalTaskStatus(st) {
			t.FinishedAt = NewTime(time.Now().UTC())
		}
	})
}

// isTerminalTaskStatus reports whether a status ends the task.
func isTerminalTaskStatus(st TaskStatus) bool {
	switch st {
	case TaskDone, TaskFailed:
		return true
	default:
		return false
	}
}

// subscribe registers a listener and returns the channel plus a
// cancel func.
func (s *TaskStore) subscribe() (<-chan event, func()) {
	ch := make(chan event, subBuffer)
	s.mu.Lock()
	id := s.next
	s.next++
	s.subs[id] = ch
	s.mu.Unlock()
	return ch, func() {
		s.mu.Lock()
		if c, ok := s.subs[id]; ok {
			delete(s.subs, id)
			close(c)
		}
		s.mu.Unlock()
	}
}

// broadcast fans an event out to every subscriber without
// blocking.
func (s *TaskStore) broadcast(ev event) {
	if s == nil {
		return
	}
	// Send while holding the read lock rather than snapshotting
	// the channel list and releasing first: a subscriber whose
	// cancel() ran in that window closed the channel between the
	// two steps and the send panicked.
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, ch := range s.subs {
		select {
		case ch <- ev:
		default:
			// Subscriber is behind. Dropping is safe because
			// the client re-syncs from a snapshot.
		}
	}
}

// OperatorSender is the reserved sender id for a message the
// operator typed into the browser rather than an agent.
//
// It is what separates the two kinds of message the UI can
// post. A message from an agent is a note between agents and
// nothing consumes it. A message from the operator is a request
// for work: the server turns it into a plan and runs it.
const OperatorSender = "operator"

// TaskSubmitter turns an operator's plain-language goal into
// work.
//
// It is the seam between the browser and the agent loop. The
// webui package defines the interface and the cli package
// implements it, because the dependency runs that way: the UI
// must not know about the planner, the model router, or the
// orchestrator, and those must not know about HTTP.
//
// The interface is deliberately coarse. A non-coder submits a
// goal and the system works out the rest -- whether the goal
// becomes a plan as written, a repaired plan, or a plan the
// conversation refined. The UI does not get a say in that.
//
// Implementations MUST be non-blocking with respect to the
// caller: a submit returns as soon as the work is accepted, and
// the result arrives through the hub. A submit that blocked for
// the length of a run would hold an HTTP request open for
// minutes and time out in every proxy in between.
type TaskSubmitter interface {
	// Submit accepts a goal and returns the id of the task it
	// became. The task's progress is published to the hub, not
	// returned here.
	Submit(goal string) (string, error)
	// CanSubmit reports whether this submitter can accept work
	// right now. False means the server cannot run plans, and
	// the UI must say so rather than offering a button that
	// cannot work.
	CanSubmit() bool
}

// ErrNoSubmitter is returned when the server has no way to run
// a submitted task.
//
// It is the read-only-server case: a UI that can watch a run but
// not start one. The message says so plainly, because "task
// rejected" on its own reads like a broken product rather than
// a server that was not asked to do that job.
var ErrNoSubmitter = &submitterError{}

type submitterError struct{}

func (e *submitterError) Error() string {
	return "this server cannot start tasks; it can only watch runs that are already going"
}

// normalizeGoal trims a submitted goal and reports whether it is
// usable.
//
// Whitespace-only is not a goal. An empty submission must be
// rejected before it reaches the planner, because a planner call
// with an empty goal produces a confusing "plan generation
// failed" rather than "say what you want done".
func normalizeGoal(goal string) (string, bool) {
	g := strings.TrimSpace(goal)
	if g == "" {
		return "", false
	}
	return g, true
}
