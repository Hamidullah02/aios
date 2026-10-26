package cli

import (
	"github.com/sadi/aios-system/internal/core/audit"
	"github.com/sadi/aios-system/internal/edit"
	"github.com/sadi/aios-system/internal/git"
	"github.com/sadi/aios-system/internal/image"
	"github.com/sadi/aios-system/internal/memory"
	"github.com/sadi/aios-system/internal/memorytool"
	"github.com/sadi/aios-system/internal/search"
	"github.com/sadi/aios-system/internal/tools"
)

// productionToolSet returns the complete set of tools the
// bus dispatches, keyed by canonical name.
//
// The set is assembled here, in one place, so the bus and the
// planner can never disagree about which tools exist. Before
// this function the tool map was assembled inline in
// wireAgentLoop and the planner had a separate hand-written
// copy of a fifth of it; the two drifted, and the tools that
// were missing from the planner's copy were unreachable from
// a plan.
//
// The ask tool is passed in because it needs a prompt
// callback, which differs between production (stdin) and
// tests (a fake).
//
// The memory tools are constructed here on a best-effort
// basis: a workspace whose data directory is unwritable still
// starts, and the memory tools return a typed error rather
// than panicking. Failing the whole CLI over an unwritable
// memory directory would make the problem strictly worse than
// not having memory at all.
func productionToolSet(askTool tools.Tool, auditLog *audit.Log) map[string]tools.Tool {
	toolSet := map[string]tools.Tool{
		"read_file":    tools.ReadFileTool{},
		"search_files": tools.SearchFilesTool{},
		"write_file":   tools.WriteFileTool{},
		// run_shell gets NO explicit runner. The tool falls
		// back to sandbox.DefaultRunner(), and the egress
		// guard installs its network policy there.
		//
		// Passing a fresh runner here was the reason the
		// egress allow list was not enforced: the tool held
		// its own runner with no policy, so every command the
		// agent ran reached the network directly. The proxy
		// was running the whole time and had recorded nothing,
		// because no request ever went through it.
		"run_shell": tools.RunShellTool{},
		"ask_user":  askTool,
	}
	// Verify-package tools. The RunTestsTool is always
	// wired (no model dependency). SelfCheckTool needs a
	// router and is registered later, in wirePlanner.
	for name, tool := range wireVerifyTools(nil, auditLog) {
		toolSet[name] = tool
	}
	// The rest of the doc-set tool set: the edit / git /
	// search / image packages each ship a working
	// tools.Tool implementation.
	//
	// Never shadow a tool that is already wired. The verify
	// set wins on a name collision so a future rename in
	// another package cannot silently change what run_tests
	// resolves to.
	for name, tool := range wireDocSetTools() {
		if _, exists := toolSet[name]; exists {
			continue
		}
		toolSet[name] = tool
	}
	// The memory adapters need a constructed *memory.Memory,
	// and both tools share one instance so a write is visible
	// to a read in the same process.
	mem, err := memory.NewMemory(memory.MemoryConfig{})
	if err != nil {
		// Leave mem nil. wireMemoryTools still registers the
		// two tools so `aios tools` shows them and a plan
		// gets a clear "memory is not wired" error instead of
		// ToolNotFoundError, which would send the operator
		// looking for a policy problem.
		mem = nil
	}
	for name, tool := range wireMemoryTools(mem, auditLog) {
		if _, exists := toolSet[name]; exists {
			continue
		}
		toolSet[name] = tool
	}
	return toolSet
}

// wireDocSetTools registers the doc-set tool set that
// wireAgentLoop does not already wire.
//
// v0.38.0. The v0.32.0 CLI shipped only the five
// REQ-DOC-MIN tools (read_file, search_files, write_file,
// run_shell, ask_user). Every other tool the packages
// already implement was dead code: the packages compiled,
// had their own tests, and were never reachable from a
// plan. A user writing `tool: edit` in a TaskPlan got
// `ToolNotFoundError` even though the implementation
// sat in the same binary.
//
// This function returns the tools that already satisfy
// tools.Tool, keyed by their canonical names:
//
//	edit package:   edit, multi_edit, apply_patch, write,
//	                notebook_edit
//	git package:    git_status, git_diff, git_log,
//	                git_commit, git_branch, git_worktree
//	search package: grep, glob, code_search, diagnostics
//	image package:  read_image
//
// Every one of these is a zero-value struct or takes no
// constructor arguments, so the map is a plain literal.
// The git tools' Runner field is unexported and nil means
// "use the default subprocess runner" (see
// GitStatusTool.runner), so `git.GitStatusTool{}` is the
// correct production construction.
//
// Not registered here:
//
//   - benchmark, fanout, optimizer, persist, roles:
//     these packages are libraries with no tools.Tool
//     implementation. Wiring them needs new adapter
//     types, not just registration; tracked separately.
//   - memory: also a library, but v0.39.0 added the
//     internal/memorytool adapters, so it is registered
//     through wireMemoryTools instead of the literal
//     below (it needs a constructed *memory.Memory, and
//     the two tools must share one instance).
//   - task, file_watcher (internal/watch): wired under
//     the daemon subcommand, which builds a SubAgentFactory.
//   - self_check: needs a model router; see
//     wireVerifyTools.
//
// Every tool here goes through the same policy gateway
// as the rest of the bus, so a default-deny workspace
// still denies them until the operator adds a rule.
func wireDocSetTools() map[string]tools.Tool {
	return map[string]tools.Tool{
		// edit package (REQ-EDT-*)
		"edit":          edit.EditTool{},
		"multi_edit":    edit.MultiEditTool{},
		"apply_patch":   edit.ApplyPatchTool{},
		"write":         edit.WriteTool{},
		"notebook_edit": edit.NotebookEditTool{},

		// git package (REQ-GIT-*)
		"git_status":   git.GitStatusTool{},
		"git_diff":     git.GitDiffTool{},
		"git_log":      git.GitLogTool{},
		"git_commit":   git.GitCommitTool{},
		"git_branch":   git.GitBranchTool{},
		"git_worktree": git.GitWorktreeTool{},

		// search package (REQ-SRC-*)
		"grep":        search.GrepTool{},
		"glob":        search.GlobTool{},
		"code_search": search.CodeSearchTool{},
		"diagnostics": search.DiagnosticsTool{},

		// image package (REQ-IMG-*)
		"read_image": image.ReadImageTool{},
	}
}

// wireMemoryTools registers the memory adapters
// (v0.39.0).
//
// Unlike wireDocSetTools these need a constructed
// *memory.Memory, and BOTH tools must receive the same
// instance: the read tool has to see what the write tool
// stored. A nil mem means "memory is not available"; the
// tools then return a typed error rather than panicking,
// so a workspace whose data dir is unwritable still
// starts and every other tool still works.
func wireMemoryTools(mem *memory.Memory, auditLog *audit.Log) map[string]tools.Tool {
	return map[string]tools.Tool{
		// Read-only surface. Separate from memory_write so
		// a policy rule can grant recall without also
		// granting persist / delete.
		"memory": memorytool.ReadTool{Mem: mem, Audit: auditLog},
		// Mutating surface.
		"memory_write": memorytool.WriteTool{Mem: mem, Audit: auditLog},
	}
}
