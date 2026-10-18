package audit

import "testing"

// TestTypeAliases_Assignable confirms the doc-set-named types are
// real aliases: an AuditLog is a Log, an AuditRow is a Row, and a
// RedactionMarker is a redact.Match. The test is essentially a
// compile-time guard; if the aliases break, this file fails to
// compile.
//
// AuditLog, AuditRow, and RedactionMarker are value-type aliases of
// struct types declared in this package (or, for RedactionMarker,
// in the redact package). Assigning *Log to AuditLog would require
// a pointer-type alias; instead we use blank-declared values.
func TestTypeAliases_Assignable(t *testing.T) {
	var _ Log // AuditLog is Log (value).
	var _ AuditLog
	var _ Row // AuditRow is Row (value).
	var _ AuditRow
	var _ RedactionMarker
	// DuplicateWriteError must equal ErrDuplicate.
	if DuplicateWriteError != ErrDuplicate {
		t.Fatal("DuplicateWriteError must equal ErrDuplicate")
	}
}
