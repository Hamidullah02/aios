package edit

import (
	"github.com/Hamidullah02/aios/aios-system/internal/tools"
)

// WriteTool re-exports the write-file tool from aios-tools
// under a unified name (REQ-EDT-WRT-001 to 003). The bus
// consults the policy gateway for tier-3 approval; the tool
// itself uses the atomic-write path.
type WriteTool struct {
	Inner tools.WriteFileTool
}

func (WriteTool) Name() string { return "write" }

func (w WriteTool) Run(args map[string]any, ctx tools.Context) (tools.Result, error) {
	return w.Inner.Run(args, ctx)
}
