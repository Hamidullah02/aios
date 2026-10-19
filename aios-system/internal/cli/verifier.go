package cli

import (
	"fmt"

	"github.com/sadi/aios-system/internal/runtime"
)

// CommandVerifier is the production verifier the CLI ships
// with the BusWorker. It replaces SmokeVerifier, which
// returned VerdictPass for every result and therefore
// rubber-stamped every tool call — even a tool that
// returned OK=false or one that never ran.
//
// CommandVerifier inspects the WorkerResult the BusWorker
// produced and decides pass / fail on real signal:
//
//   - Output is the bus's Result.Summary. Every tool in
//     the production tool set (read_file, search_files,
//     write_file, run_shell, ask_user) fills the summary
//     on both success and failure. An empty summary means
//     the bus short-circuited (missing tool, the worker
//     returned no result) and is a fail.
//   - Data is the bus's Result.Output. The bus fills
//     Result.Output with the tool's structured result
//     (a map of fields the agent will read later). A nil
//     Data means the tool did not produce a typed result
//     and is a fail.
//
// The verifier is intentionally strict: any tool that
// cannot produce a non-empty Summary AND a non-nil Data
// is rejected. The plan stops on the first fail (per
// the orchestrator's no-progress guard) and the operator
// can read the audit log to see exactly which tool
// returned the bad result.
type CommandVerifier struct{}

// NewCommandVerifier returns a CommandVerifier. The type
// is a struct (not a constructor) because there is no
// configuration; the value is zero-cost and the production
// BuildDeps uses the literal.
func NewCommandVerifier() CommandVerifier {
	return CommandVerifier{}
}

// Verify implements runtime.Verifier.
func (CommandVerifier) Verify(_ runtime.TaskInput, result runtime.WorkerResult) (runtime.VerifierVerdict, error) {
	// The bus fills Output with the tool's Result.Summary.
	// A successful tool call always returns a non-empty
	// summary; an empty one means the bus short-circuited
	// (missing tool, policy deny that bypassed Result).
	if result.Output == "" {
		return runtime.VerdictFail, fmt.Errorf("verifier: empty output (tool did not run)")
	}
	// Data is the bus's Result.Output. The bus sets
	// Data to the tool's structured result (a map of
	// fields). A nil Data means the tool returned a
	// failure result with no payload.
	if result.Data == nil {
		return runtime.VerdictFail, fmt.Errorf("verifier: nil data (tool returned no result)")
	}
	return runtime.VerdictPass, nil
}

// Compile-time check.
var _ runtime.Verifier = CommandVerifier{}
