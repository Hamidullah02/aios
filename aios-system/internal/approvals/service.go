// Package approvals is the approval service for aios.
//
// The service owns the lifecycle of a single approval:
//
//	pending -> granted   (operator says y)
//	pending -> denied    (operator says n)
//	pending -> expired   (TTL fires before either)
//
// A grant is honored only when the payload's BoundTo matches the
// call-site context. The service writes one audit row per state
// transition. The audit row is content-addressed, so a re-grant
// of the same approval is a no-op (idempotency) and a re-grant
// against a tampered payload is rejected (binding).
//
// The service is in-memory. Persistence is the audit log; the
// service's only durable state is what the audit log holds. A
// restart loses the in-flight waiters; the in-flight approvals
// can be re-loaded from the audit log by replaying the
// `approval_requested` rows and the matching
// `approval_granted` / `approval_denied` / `approval_expired`
// rows in sequence.
package approvals

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/Hamidullah02/aios/aios-system/internal/core/audit"
	"github.com/Hamidullah02/aios/aios-system/internal/core/clock"
	"github.com/Hamidullah02/aios/aios-system/internal/core/schemas"
)

// Sentinel errors. The doc set names them in §7.1-7.3.
var (
	ErrNotFound          = errors.New("approvals: not found")
	ErrBindingMismatch   = errors.New("approvals: binding mismatch")
	ErrInvalidTransition = errors.New("approvals: invalid state transition")
	ErrInvalidPayload    = errors.New("approvals: invalid payload")
)

// Service is the approval service. One instance is safe for
// concurrent use; the internal map is guarded by a mutex.
type Service struct {
	mu        sync.Mutex
	approvals map[string]*entry
	log       *audit.Log
	clock     clock.Decision
}

type entry struct {
	a    Approval
	done chan struct{}
}

// Config wires the service to its collaborators.
type Config struct {
	Audit *audit.Log
	Clock clock.Decision
}

// New returns a service. A nil clock is replaced with the system
// clock; a nil audit log causes Request to return
// ErrInvalidPayload (the service cannot satisfy the "write an
// audit row" requirement).
func New(cfg Config) *Service {
	c := cfg.Clock
	if c == nil {
		c = clock.SystemDecision{}
	}
	return &Service{
		approvals: make(map[string]*entry),
		log:       cfg.Audit,
		clock:     c,
	}
}

// RequestOptions customizes a single request. The zero value is
// a 5-minute TTL, which matches the default for tier-3 actions.
type RequestOptions struct {
	// TTL is the time after which a still-pending approval is
	// auto-expired. Zero means the package default (5 minutes).
	// Negative means "no TTL" (the approval never expires).
	TTL time.Duration
}

// Request creates a new approval, writes the audit row, starts
// the TTL timer (if any), and returns the approval id.
//
// The actor is recorded as the requester. The returned id is a
// UUIDv4; the approval_id field in the payload is set to the
// same value so downstream consumers can join on either.
func (s *Service) Request(payload schemas.ApprovalPayload, actor string, opts RequestOptions) (string, error) {
	if s.log == nil {
		return "", fmt.Errorf("%w: audit log is required", ErrInvalidPayload)
	}
	if actor == "" {
		return "", fmt.Errorf("%w: actor is required", ErrInvalidPayload)
	}
	id := uuid.NewString()
	payload.ApprovalID = id
	if err := payload.Validate(); err != nil {
		return "", fmt.Errorf("%w: %v", ErrInvalidPayload, err)
	}
	now := s.clock.Now()
	ttl := opts.TTL
	if ttl == 0 {
		ttl = 5 * time.Minute
	}
	a := Approval{
		ID:        id,
		State:     StatePending,
		Payload:   payload,
		Actor:     actor,
		CreatedAt: now.UTC().Format(time.RFC3339Nano),
		UpdatedAt: now.UTC().Format(time.RFC3339Nano),
		TTL:       int(ttl.Seconds()),
	}
	e := &entry{a: a, done: make(chan struct{})}

	s.mu.Lock()
	s.approvals[id] = e
	s.mu.Unlock()

	if err := s.writeAudit("approval_requested", &a, actor, payloadTaskID(payload)); err != nil {
		// Fail-closed: remove the in-memory entry so the id is
		// not visible to a subsequent grant. The error propagates
		// to the caller, who treats it as a run block.
		s.mu.Lock()
		delete(s.approvals, id)
		s.mu.Unlock()
		return "", err
	}

	if ttl > 0 {
		time.AfterFunc(ttl, func() { s.expire(id) })
	}
	return id, nil
}

// Context is the call-site context for a binding check. An empty
// Context is a wildcard; every binding field matches.
type Context struct {
	Task     string
	Tool     string
	Actor    string
	Resource string
}

// Grant transitions the approval to granted. The grant is honored
// only when the binding matches the call-site context. actor is
// recorded as the granter.
func (s *Service) Grant(approvalID, actor string, ctx Context) error {
	return s.transition(approvalID, actor, StateGranted, ctx)
}

// Deny transitions the approval to denied. The binding is not
// checked on deny: the operator can always say no.
func (s *Service) Deny(approvalID, actor string) error {
	return s.transition(approvalID, actor, StateDenied, Context{})
}

// Wait blocks until the approval reaches a terminal state or the
// context is cancelled. The returned snapshot is a copy.
func (s *Service) Wait(ctx context.Context, approvalID string) (Approval, error) {
	e, err := s.lookup(approvalID)
	if err != nil {
		return Approval{}, err
	}
	select {
	case <-e.done:
		return e.a, nil
	case <-ctx.Done():
		return Approval{}, ctx.Err()
	}
}

// Get returns the current snapshot without blocking. Useful for
// "show me the approval" prompts.
func (s *Service) Get(approvalID string) (Approval, error) {
	e, err := s.lookup(approvalID)
	if err != nil {
		return Approval{}, err
	}
	return e.a, nil
}

// Pending returns every approval still waiting on a decision.
//
// The service could always answer "what is this one
// approval", but not "what is everything outstanding", and
// that second question is what an operator dashboard asks.
// Without it a UI has no way to show a queue; it would have to
// be handed ids by the process that created them, which is
// exactly the coupling that breaks when the UI runs in a
// different process from the run.
//
// Sorted oldest first so the request that has been waiting
// longest is the one an operator sees at the top.
func (s *Service) Pending() []Approval {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	out := make([]Approval, 0, len(s.approvals))
	for _, e := range s.approvals {
		if e.a.State == StatePending {
			out = append(out, e.a)
		}
	}
	s.mu.Unlock()
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt < out[j].CreatedAt })
	return out
}

func (s *Service) transition(approvalID, actor string, target State, ctx Context) error {
	s.mu.Lock()
	e, ok := s.approvals[approvalID]
	if !ok {
		s.mu.Unlock()
		return ErrNotFound
	}
	if e.a.State != StatePending {
		s.mu.Unlock()
		return fmt.Errorf("%w: cannot transition from %q to %q", ErrInvalidTransition, e.a.State, target)
	}
	if target == StateGranted {
		if !e.a.Payload.BoundTo.Match(ctx.Task, ctx.Tool, ctx.Actor, ctx.Resource) {
			s.mu.Unlock()
			return fmt.Errorf("%w: approval %s", ErrBindingMismatch, approvalID)
		}
	}
	e.a.State = target
	e.a.UpdatedAt = s.clock.Now().UTC().Format(time.RFC3339Nano)
	snapshot := e.a
	ch := e.done
	s.mu.Unlock()

	if err := s.writeAudit(auditKindFor(target), &snapshot, actor, approvalTaskID(snapshot, ctx.Task)); err != nil {
		return err
	}
	close(ch)
	return nil
}

func (s *Service) expire(approvalID string) {
	s.mu.Lock()
	e, ok := s.approvals[approvalID]
	if !ok || e.a.State != StatePending {
		s.mu.Unlock()
		return
	}
	e.a.State = StateExpired
	e.a.UpdatedAt = s.clock.Now().UTC().Format(time.RFC3339Nano)
	snapshot := e.a
	ch := e.done
	s.mu.Unlock()
	// Best-effort: a write failure here is logged by the audit
	// package and surfaces as a run block on the operator side.
	_ = s.writeAudit("approval_expired", &snapshot, "system:ttl", approvalTaskID(snapshot, ""))
	close(ch)
}

func (s *Service) lookup(approvalID string) (*entry, error) {
	if approvalID == "" {
		return nil, fmt.Errorf("%w: empty id", ErrNotFound)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.approvals[approvalID]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrNotFound, approvalID)
	}
	return e, nil
}

func (s *Service) writeAudit(kind string, a *Approval, actor, taskID string) error {
	if s.log == nil {
		return nil
	}
	row := &audit.Row{
		TaskID:     taskID,
		Kind:       kind,
		Actor:      actor,
		RiskTier:   a.Payload.RiskTier,
		ApprovalID: a.ID,
		Payload:    a,
	}
	_, err := s.log.Write(row)
	if err != nil && errors.Is(err, audit.DuplicateWriteError) {
		// Re-writing the same state is a no-op; the existing row
		// is what we want.
		return nil
	}
	return err
}

// approvalTaskID derives the audit row's task_id for an
// approval. The function takes the preferred task id (from
// the binding or the call-site context) and falls back to
// the approval id with a stable prefix when no task is
// known, so the audit row is never written with a
// misleading free-form value. The function is also used to
// satisfy the audit row's required task_id column for
// approvals created outside a task context.
func approvalTaskID(a Approval, preferred string) string {
	if preferred != "" {
		return preferred
	}
	if a.Payload.BoundTo != nil && a.Payload.BoundTo.Task != "" {
		return a.Payload.BoundTo.Task
	}
	return "approval:" + a.ID
}

// payloadTaskID is the Request-time equivalent of
// approvalTaskID; it accepts the raw payload (the in-flight
// form, before the Approval wrapper is built) and the same
// fallback rules.
func payloadTaskID(p schemas.ApprovalPayload) string {
	if p.BoundTo != nil && p.BoundTo.Task != "" {
		return p.BoundTo.Task
	}
	return "approval:" + p.ApprovalID
}

func auditKindFor(s State) string {
	switch s {
	case StateGranted:
		return "approval_granted"
	case StateDenied:
		return "approval_denied"
	case StateExpired:
		return "approval_expired"
	}
	return "approval_" + string(s)
}
