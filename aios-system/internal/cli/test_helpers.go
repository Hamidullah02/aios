package cli

import (
	"context"
	"testing"

	"github.com/Hamidullah02/aios/aios-system/internal/ask"
	"github.com/Hamidullah02/aios/aios-system/internal/core/audit"
	"github.com/Hamidullah02/aios/aios-system/internal/policy"
	"github.com/Hamidullah02/aios/aios-system/internal/sandbox"
	"github.com/Hamidullah02/aios/aios-system/internal/tools"
)

// buildTestAgentLoop constructs the production wire
// (tools.Bus, BusWorker, CommandVerifier) using a
// pre-built audit log and policy gateway, and a fake
// ask callback so the test does not block on stdin.
//
// This is the test seam: production code uses
// wireAgentLoop which loads the policy from disk and
// uses a real StdinPromptCallback. Tests call this
// helper to inject their own audit log, policy, and
// ask callback.
func buildTestAgentLoop(
	t *testing.T,
	workspace string,
	gw *policy.Gateway,
	log *audit.Log,
) (*tools.Bus, *BusWorker, CommandVerifier) {
	t.Helper()
	askTool, err := ask.New(func(_ context.Context, q ask.Question) (ask.Answer, error) {
		if len(q.Options) == 0 {
			return ask.Answer{}, nil
		}
		return ask.Answer{QuestionID: q.ID, SelectedOption: q.Options[0].ID}, nil
	}, nil)
	if err != nil {
		t.Fatalf("ask.New: %v", err)
	}
	toolSet := map[string]tools.Tool{
		"read_file":    tools.ReadFileTool{},
		"search_files": tools.SearchFilesTool{},
		"write_file":   tools.WriteFileTool{},
		"run_shell":    tools.RunShellTool{Runner: sandbox.NewSubprocessRunner()},
		"ask_user":     askBusAdapter{tool: askTool},
	}
	bus := tools.New(tools.Config{
		Tools:  toolSet,
		Policy: gw,
		Audit:  log,
	})
	worker := NewBusWorker(bus, askTool, workspace)
	verifier := NewCommandVerifier()
	return bus, worker, verifier
}

// testBus returns a bus carrying the full production tool
// set, with no policy and no audit log.
//
// It exists so a test can ask the bus what tools exist
// without setting up a policy or an audit database. The
// planner's tool list is derived from the bus, so a test that
// wires the planner against the production set is testing the
// same list production uses.
func testBus(t *testing.T) *tools.Bus {
	t.Helper()
	askTool, err := ask.New(func(_ context.Context, q ask.Question) (ask.Answer, error) {
		if len(q.Options) == 0 {
			return ask.Answer{}, nil
		}
		return ask.Answer{QuestionID: q.ID, SelectedOption: q.Options[0].ID}, nil
	}, nil)
	if err != nil {
		t.Fatalf("ask.New: %v", err)
	}
	return tools.New(tools.Config{
		Tools: productionToolSet(askBusAdapter{tool: askTool}, nil),
	})
}
