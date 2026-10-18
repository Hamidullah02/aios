package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/sadi/aios-system/internal/core/audit"
	"github.com/sadi/aios-system/internal/core/clock"
	"github.com/sadi/aios-system/internal/core/state"
	"github.com/sadi/aios-system/internal/report"
	"github.com/sadi/aios-system/internal/runtime"
)

// TestReport_BridgeFromRows exercises the audit-log to
// report bridge. Three rows in, JSON / Markdown / HTML
// out, each non-empty and parseable.
func TestReport_BridgeFromRows(t *testing.T) {
	log, err := audit.Open(audit.Config{
		Path:  ":memory:",
		Clock: clock.FixedDecision{T: time.Unix(1700000000, 0)},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer log.Close()
	for _, kind := range []string{"policy_decision", "tool_call", "decision"} {
		if _, err := log.Write(&audit.Row{
			TaskID: "bridge-1", Kind: kind, Actor: "tester",
		}); err != nil {
			t.Fatal(err)
		}
	}
	rows, _ := log.GetByTask("bridge-1")
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(rows))
	}
	for _, format := range []string{"json", "markdown", "html"} {
		var buf bytes.Buffer
		rep := bridgeReportFromRows("bridge-1", rows)
		if err := renderReport(format, rep, &buf); err != nil {
			t.Fatalf("render %s: %v", format, err)
		}
		if buf.Len() == 0 {
			t.Errorf("%s: empty output", format)
		}
		if format == "json" {
			var got report.Report
			if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
				t.Errorf("json: %v", err)
			}
			if got.TaskID != "bridge-1" {
				t.Errorf("json TaskID = %q", got.TaskID)
			}
			if len(got.Steps) != 3 {
				t.Errorf("json Steps = %d", len(got.Steps))
			}
		}
	}
}

// TestReport_BridgeFromRun exercises the runtime.Report
// to report.Report bridge. A real orchestrator run
// produces a runtime.Report; the bridge converts and
// the renderer prints.
func TestReport_BridgeFromRun(t *testing.T) {
	rr := &runtime.Report{
		PlanID:        "p1",
		TaskCount:     3,
		StartedAt:     "2024-01-01T00:00:00Z",
		FinishedAt:    "2024-01-01T00:00:05Z",
		StoppedReason: "completed",
		TaskStates: map[string]state.State{
			"t1": state.StateCompleted,
			"t2": state.StateCompleted,
			"t3": state.StateCompleted,
		},
	}
	rep := bridgeReport(rr, "p1")
	if rep.Status != report.StatusSucceeded {
		t.Errorf("status = %s, want succeeded", rep.Status)
	}
	if len(rep.Steps) != 3 {
		t.Errorf("steps = %d, want 3", len(rep.Steps))
	}
	var buf bytes.Buffer
	if err := renderReport("json", rep, &buf); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), `"task_id": "p1"`) {
		t.Errorf("json missing task_id: %s", buf.String())
	}
}

// The test below covers the new failure path in the
// render: an unknown format must return an error.
func TestReportBridge_UnknownFormat(t *testing.T) {
	var buf bytes.Buffer
	err := renderReport("xml", &report.Report{TaskID: "x", PlanID: "x"}, &buf)
	if err == nil {
		t.Fatal("expected error for xml format")
	}
	if !strings.Contains(err.Error(), "unknown report format") {
		t.Errorf("err = %v, want unknown format", err)
	}
}

// Ensure the bridge's TaskID column is preserved.
func TestReport_TaskIDPreserved(t *testing.T) {
	rep := bridgeReportFromRows("the-task", nil)
	if rep.TaskID != "the-task" || rep.PlanID != "the-task" {
		t.Errorf("TaskID=%q PlanID=%q", rep.TaskID, rep.PlanID)
	}
	if rep.Validate() != nil && rep.StartedAt == "" {
		// Empty rows means zero timestamps; the report
		// is invalid. Confirm the bridge does not
		// paper over it.
		rep2 := bridgeReportFromRows("the-task", []audit.PersistedRow{{RowID: "r1", TSNano: 1, Kind: "tool_call"}})
		if err := rep2.Validate(); err != nil {
			t.Errorf("rep2 invalid: %v", err)
		}
	}
}
