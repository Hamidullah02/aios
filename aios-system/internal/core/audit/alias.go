package audit

import "github.com/Hamidullah02/aios/aios-system/internal/core/redact"

// Doc-set names. The doc set (SRS §7 IFC-002) names the audit types
// `AuditLog`, `AuditRow`, and `RedactionMarker`. The implementation
// uses the package-qualified `Log` / `Row` to avoid shadowing the
// standard `log` package, and keeps redaction in a separate
// `redact` package. These aliases give downstream modules a choice:
//
//   - use the short form:  audit.Log, audit.Row
//   - use the doc-set form: audit.AuditLog, audit.AuditRow,
//                            audit.RedactionMarker (= redact.Match)
//
// The two forms are interchangeable; aliasing in Go is purely
// textual. No runtime cost.

// AuditLog is the doc-set name for Log.
type AuditLog = Log

// AuditRow is the doc-set name for Row. It is the write-side row
// the caller fills in before passing to Log.Write.
type AuditRow = Row

// DuplicateWriteError is the doc-set name (SRS §9.4) for the
// sentinel returned by Write on an idempotency-key collision.
// It is interchangeable with ErrDuplicate.
var DuplicateWriteError = ErrDuplicate

// RedactionMarker is the doc-set name (SRS §7 IFC-002) for a
// single redaction event. The implementation defines this type in
// the `redact` package as `Match` (a single detection event with
// its stable placeholder). The audit package re-exports it here so
// callers that imported only `audit` can refer to it as
// `audit.RedactionMarker` without taking a dependency on `redact`.
type RedactionMarker = redact.Match
