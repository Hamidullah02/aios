package cli

import (
	"github.com/sadi/aios-system/internal/edit"
	"github.com/sadi/aios-system/internal/git"
	"github.com/sadi/aios-system/internal/image"
	"github.com/sadi/aios-system/internal/search"
	"github.com/sadi/aios-system/internal/tools"
)

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
//   - benchmark, fanout, optimizer, persist, roles,
//     memory: these packages are libraries with no
//     tools.Tool implementation. Wiring them needs new
//     adapter types, not just registration; tracked
//     separately.
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
