package cli

import (
	"sort"
	"time"

	"github.com/sadi/aios-system/internal/core/audit"
	"github.com/sadi/aios-system/internal/core/state"
	"github.com/sadi/aios-system/internal/report"
	"github.com/sadi/aios-system/internal/runtime"
)

// bridgeReportFromRows converts audit log rows for a
// task to a report.Report. The audit log is the
// canonical source; the bridge turns a row set into
// the human-readable projection the renderers expect.
//
// Each audit row becomes a step. The first row's
// timestamp is the start; the last row's timestamp is
// the end. The kind column tells the renderer what
// the step did (decision, tool_call, policy_decision).
func bridgeReportFromRows(taskID string, rows []audit.PersistedRow) *report.Report {
	steps := make([]report.Step, 0, len(rows))
	var startNS, endNS int64
	for _, r := range rows {
		steps = append(steps, report.Step{
			ID:        r.RowID,
			Tool:      r.Kind,
			StartedAt: formatTS(r.TSNano),
			EndedAt:   formatTS(r.TSNano),
			Status:    kindToStatus(r.Kind),
			Input:     map[string]any{"kind": r.Kind},
		})
		if startNS == 0 || r.TSNano < startNS {
			startNS = r.TSNano
		}
		if r.TSNano > endNS {
			endNS = r.TSNano
		}
	}
	return &report.Report{
		TaskID:       taskID,
		PlanID:       taskID,
		Status:       report.StatusSucceeded,
		StartedAt:    formatTS(startNS),
		EndedAt:      formatTS(endNS),
		Steps:        steps,
		InputTokens:  0,
		OutputTokens: 0,
	}
}

// kindToStatus maps an audit row kind to the report
// status enum. The audit row does not carry an
// outcome, so the bridge defaults to succeeded.
func kindToStatus(kind string) report.Status {
	switch kind {
	case "approval_denied", "approval_expired":
		return report.StatusFailed
	default:
		return report.StatusSucceeded
	}
}

// bridgeReport converts a runtime.Report to the
// report package's Report so the renderers (JSON /
// Markdown / HTML) can format it. The bridge is the
// only place in the CLI that knows the two shapes.
func bridgeReport(rr *runtime.Report, taskID string) *report.Report {
	if rr == nil {
		return &report.Report{
			TaskID: taskID,
			PlanID: taskID,
			Status: report.StatusSucceeded,
		}
	}
	ids := make([]string, 0, len(rr.TaskStates))
	for id := range rr.TaskStates {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	steps := make([]report.Step, 0, len(ids))
	for _, id := range ids {
		steps = append(steps, report.Step{
			ID:        id,
			Tool:      "task",
			StartedAt: rr.StartedAt,
			EndedAt:   rr.FinishedAt,
			Status:    stateToStatus(rr.TaskStates[id]),
			Input:     map[string]any{"task_id": id},
		})
	}
	status := report.StatusSucceeded
	switch rr.StoppedReason {
	case "completed":
		status = report.StatusSucceeded
	case "canceled":
		status = report.StatusCanceled
	default:
		status = report.StatusFailed
	}
	return &report.Report{
		TaskID:       taskID,
		PlanID:       taskID,
		Status:       status,
		StartedAt:    rr.StartedAt,
		EndedAt:      rr.FinishedAt,
		Steps:        steps,
		InputTokens:  0,
		OutputTokens: 0,
	}
}

// stateToStatus maps the runtime state to the report
// status enum.
func stateToStatus(s state.State) report.Status {
	switch s {
	case state.StateCompleted:
		return report.StatusSucceeded
	case state.StateCancelled:
		return report.StatusCanceled
	default:
		return report.StatusFailed
	}
}

// formatTS formats a Unix-nanosecond timestamp as
// RFC 3339 nano. Zero renders as the zero time so
// the report stays valid.
func formatTS(ns int64) string {
	if ns == 0 {
		return time.Time{}.UTC().Format(time.RFC3339Nano)
	}
	return time.Unix(0, ns).UTC().Format(time.RFC3339Nano)
}
