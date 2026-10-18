package ask

import (
	"context"
	"testing"
	"time"

	"github.com/sadi/aios-system/internal/tools"
)

// askAdapter is a thin shim that exposes AskUserTool to the
// tools.Tool interface. The shim exists in the test (rather
// than the package) because the tools package would create an
// import cycle (tools does not import ask; ask does not import
// tools). The shim translates the ask Result into a tools
// Result; the bus then audits it like any other tool call.
type askAdapter struct {
	tool *AskUserTool
}

func (a askAdapter) Name() string { return a.tool.Name() }

func (a askAdapter) Run(args map[string]any, ctx tools.Context) (tools.Result, error) {
	res, err := a.tool.Run(context.Background(), args)
	return tools.Result{OK: res.OK, Output: res.Output, Summary: res.Summary}, err
}

func TestAskUserTool_IntegratesWithBus(t *testing.T) {
	cb := scriptedCB(func(s *scriptedCallback) {
		s.resp = Answer{SelectedOption: "pg", Timestamp: "t"}
	})
	tool, _ := New(cb.asCallback(), nil)
	bus := tools.New(tools.Config{
		Tools: map[string]tools.Tool{"ask_user": askAdapter{tool: tool}},
	})
	q := mkQuestion()
	res, err := bus.Dispatch("ask_user", map[string]any{"question": q}, tools.Context{
		Actor: "tester", TaskID: "task-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.OK {
		t.Fatalf("OK: %v, Summary: %s", res.OK, res.Summary)
	}
	if res.Output["selected_option"] != "pg" {
		t.Errorf("output: %+v", res.Output)
	}
}

func TestAskUserTool_BusTimeoutError(t *testing.T) {
	cb := scriptedCB(func(s *scriptedCallback) {
		s.err = context.DeadlineExceeded
		s.delay = 50 * time.Millisecond
	})
	tool, _ := New(cb.asCallback(), nil)
	bus := tools.New(tools.Config{
		Tools: map[string]tools.Tool{"ask_user": askAdapter{tool: tool}},
	})
	q := mkQuestion()
	q.TimeoutSeconds = 1
	res, err := bus.Dispatch("ask_user", map[string]any{"question": q}, tools.Context{
		Actor: "tester", TaskID: "task-1",
	})
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if res.OK {
		t.Fatal("OK should be false on timeout with no default")
	}
}
