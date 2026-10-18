package report

import "strings"

// Renderer is the renderer interface every concrete
// renderer implements.
//
// The interface lets tests substitute a fake renderer
// and lets the CLI depend on the contract rather than
// the concrete type.
//
// Per REQ-RPT-ERR-001 a renderer returns a
// *ReportRenderError when the render fails.
type Renderer interface {
	// Render turns a report into a string. The
	// function returns a *ReportRenderError when the
	// render fails; the caller can match with
	// errors.As.
	Render(r *Report) (string, error)
}

// ReportRenderError is raised by a renderer when the
// render fails (REQ-RPT-ERR-001). The error wraps
// the underlying failure (e.g. a template parse
// error).
type ReportRenderError struct {
	Format string
	Err    error
}

func (e *ReportRenderError) Error() string {
	return formatErrorMessage("report: render: "+e.Format, e.Err)
}

func (e *ReportRenderError) Unwrap() error { return e.Err }

// Is reports whether target is also a
// ReportRenderError. The method lets callers use
// errors.Is to detect a render failure regardless of
// the concrete value.
func (e *ReportRenderError) Is(target error) bool {
	_, ok := target.(*ReportRenderError)
	return ok
}

// formatErrorMessage formats a wrapped error message
// that ends with a period. The function appends a
// period when the inner error message does not end
// with one.
func formatErrorMessage(prefix string, err error) string {
	if err == nil {
		return prefix + "."
	}
	msg := prefix + ": " + err.Error()
	if !strings.HasSuffix(msg, ".") {
		msg += "."
	}
	return msg
}
