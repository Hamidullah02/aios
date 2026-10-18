// Package clock provides the decision and deadline clock interfaces.
//
// Per the global PRD §11.11:
//
//   - The decision clock is used in any code path that affects a decision
//     (ranking, retry choice, routing, model selection, scope matching, audit
//     row content hashing, audit row timestamps). It is injected. Tests and
//     replay pass a fixed clock.
//
//   - The deadline clock is used for timeouts, budgets, lease expirations,
//     and any path that ends a run. It is allowed to read the wall clock.
//
// aios-core uses the decision clock for the audit row's `ts` field, because
// the row is part of the content_hash (decision-path data). The deadline
// clock is exposed for callers that want a timeout on the SQLite write.
package clock

import "time"

// Decision is the injected clock used in any decision path.
//
// Implementations:
//   - SystemDecision reads time.Now. This is the default for production.
//   - FixedDecision returns a constant time. This is the only safe clock
//     for tests and replay.
type Decision interface {
	Now() time.Time
}

// Deadline is the wall-clock clock used for timeouts and budgets.
//
// This is intentionally a separate interface from Decision so that the
// caller can wire a different implementation per path. In production
// both default to time.Now. In tests the Deadline clock can be the
// system clock while the Decision clock is fixed.
type Deadline interface {
	Now() time.Time
}

// SystemDecision is the production decision clock. It reads the wall
// clock. Use FixedDecision in tests.
type SystemDecision struct{}

// Now returns the current local time.
func (SystemDecision) Now() time.Time { return time.Now() }

// SystemDeadline is the production deadline clock. It reads the wall clock.
type SystemDeadline struct{}

// Now returns the current local time.
func (SystemDeadline) Now() time.Time { return time.Now() }

// FixedDecision returns a constant time. The zero value is a clock that
// always returns the Unix epoch, which is the right "no clock injected"
// default for deterministic tests.
type FixedDecision struct {
	T time.Time
}

// Now returns the fixed time, or the Unix epoch if the
// receiver is the zero value. The epoch fallback makes
// the zero value a usable "no clock injected" default
// for deterministic tests; without it, callers would have
// to thread an explicit epoch through every helper.
func (c FixedDecision) Now() time.Time {
	if c.T.IsZero() {
		return time.Unix(0, 0)
	}
	return c.T
}

// FixedDeadline returns a constant time. The zero value is
// a clock that always returns the Unix epoch. Used in tests
// that want a deterministic deadline.
type FixedDeadline struct {
	T time.Time
}

// Now returns the fixed time, or the Unix epoch if the
// receiver is the zero value.
func (c FixedDeadline) Now() time.Time {
	if c.T.IsZero() {
		return time.Unix(0, 0)
	}
	return c.T
}
