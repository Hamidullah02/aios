package runtime

import (
	"time"

	"github.com/sadi/aios-system/internal/core/state"
)

// Observer receives task lifecycle notifications from the
// orchestrator.
//
// It exists so an external consumer -- today, the web UI's
// live panels -- can watch a plan without the runtime
// importing a transport. The runtime defines the
// interface; the consumer implements it. That direction
// matters: if the runtime knew about the web UI, every
// future viewer (a TUI, a metrics exporter, a test) would
// either fork the orchestrator or wait for a runtime
// change.
//
// Implementations MUST be non-blocking. The
// orchestrator calls an observer inline, on the task
// loop, before moving to the next task: a slow observer
// stalls the plan. The web UI's implementation is a
// buffered channel with drop-on-full for exactly this
// reason.
//
// Observers are called after the corresponding state is
// durable (checkpoint written, audit row emitted), so an
// observer never reports a transition that a crash could
// roll back.
type Observer interface {
	// TaskStarted fires before the worker runs.
	TaskStarted(e TaskEvent)
	// TaskFinished fires after a task passes
	// verification, with State set to StateCompleted.
	TaskFinished(e TaskEvent)
	// TaskFailed fires when a task errors or the plan
	// stops, with State carrying the reason
	// (StateFailed, StatePaused, or
	// StateAwaitingApproval).
	TaskFailed(e TaskEvent)
}

// TaskEvent is one lifecycle notification.
type TaskEvent struct {
	// PlanID is the plan the task belongs to.
	PlanID string
	// TaskID identifies the task within the plan.
	TaskID string
	// Tool is the bus dispatch name.
	Tool string
	// State is the state the task moved to.
	State state.State
	// Detail is a short human-readable note. It may be
	// empty and is not part of the contract.
	Detail string
	// Err is the failure cause, when there was one.
	Err error
	// At is when the transition happened.
	At time.Time
}

// ObserverFuncs adapts plain functions to Observer so a
// consumer with one interest does not have to write four
// no-op methods.
//
// A nil field is skipped, so a consumer only supplies what
// it cares about.
type ObserverFuncs struct {
	Started  func(TaskEvent)
	Finished func(TaskEvent)
	Failed   func(TaskEvent)
}

// TaskStarted implements Observer.
func (o ObserverFuncs) TaskStarted(e TaskEvent) {
	if o.Started != nil {
		o.Started(e)
	}
}

// TaskFinished implements Observer.
func (o ObserverFuncs) TaskFinished(e TaskEvent) {
	if o.Finished != nil {
		o.Finished(e)
	}
}

// TaskFailed implements Observer.
func (o ObserverFuncs) TaskFailed(e TaskEvent) {
	if o.Failed != nil {
		o.Failed(e)
	}
}

// AddObserver registers an observer.
//
// The orchestrator takes a write lock to change the
// observer set, and the run loop reads the set once per
// task, so registering while a plan runs is safe: the new
// observer sees subsequent tasks, not the one in flight.
func (o *Orchestrator) AddObserver(obs Observer) {
	if o == nil || obs == nil {
		return
	}
	o.obsMu.Lock()
	o.observers = append(o.observers, obs)
	o.obsMu.Unlock()
}

// notify fans an event out to every observer.
//
// Errors from observers are ignored by design: an observer
// is a viewer, and a viewer that fails must not fail the
// plan it is watching.
func (o *Orchestrator) notify(fn func(Observer, TaskEvent), e TaskEvent) {
	if o == nil {
		return
	}
	o.obsMu.RLock()
	obs := append([]Observer(nil), o.observers...)
	o.obsMu.RUnlock()
	if e.At.IsZero() {
		e.At = time.Now().UTC()
	}
	for _, ob := range obs {
		fn(ob, e)
	}
}
