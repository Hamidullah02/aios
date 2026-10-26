package cli

import (
	"context"

	"github.com/sadi/aios-system/internal/ask"
	"github.com/sadi/aios-system/internal/tools"
)

// askBusAdapter exposes the ask.AskUserTool to the
// tools.Tool interface so the bus can dispatch it. The
// shim lives here (in the cli package) because:
//
//  1. The tools package does not import ask (and must not
//     — that would be a cycle, ask imports nothing from
//     the rest of aios).
//  2. The cli package already imports both ask and tools.
//
// The shim translates the ask Result into a tools Result
// in a way the bus understands: OK=true when the answer
// was recorded, Output carries the answer fields the
// verifier inspects, Summary is the human-readable string.
type askBusAdapter struct {
	tool *ask.AskUserTool
}

// nameAskTool is the canonical tool name. The BusWorker
// looks it up in the bus's tool map.
const nameAskTool = "ask_user"

func (a askBusAdapter) Name() string { return a.tool.Name() }

// Describe implements tools.Describer for the ask adapter.
//
// The adapter delegates to the ask package's own description
// so the schema lives next to the code that decodes the
// question, not in a second place that can drift from it.
func (a askBusAdapter) Describe() (string, string) {
	return a.tool.Describe()
}

func (a askBusAdapter) Run(args map[string]any, _ tools.Context) (tools.Result, error) {
	res, err := a.tool.Run(context.Background(), args)
	return tools.Result{
		OK:      res.OK,
		Output:  res.Output,
		Summary: res.Summary,
	}, err
}
