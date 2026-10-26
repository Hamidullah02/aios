package git

// Describe implements tools.Spec for the git package's tools.
//
// The schemas live here, next to the Run methods that read the
// args. A git tool is the one an operator is least likely to
// want to get wrong: git_commit writes history, and a plan
// that guesses its argument names produces a commit the
// operator did not ask for or, worse, a failed commit that
// half-stages a tree.
func (GitStatusTool) Describe() (string, string) {
	return "Show the working tree status of the git repository in the workspace. " +
			"Read-only. Args: none.",
		`{"type":"object","properties":{}}`
}

// Risk implements tools.RiskTier for GitStatusTool.
func (GitStatusTool) Risk() int { return 0 }

// Describe implements tools.Spec for GitDiffTool.
func (GitDiffTool) Describe() (string, string) {
	return "Show the git diff in the workspace. Read-only. " +
			"Args: path (optional, limit to one path), staged (optional bool), " +
			"commit (optional, a revision to diff against).",
		`{"type":"object","properties":{"path":{"type":"string"},"staged":{"type":"boolean"},` +
			`"commit":{"type":"string"}}}`
}

// Risk implements tools.RiskTier for GitDiffTool.
func (GitDiffTool) Risk() int { return 0 }

// Describe implements tools.Spec for GitLogTool.
func (GitLogTool) Describe() (string, string) {
	return "Show recent git commits in the workspace. Read-only. " +
			"Args: max (optional int, default 10), path (optional, limit to one path).",
		`{"type":"object","properties":{"max":{"type":"integer"},"path":{"type":"string"}}}`
}

// Risk implements tools.RiskTier for GitLogTool.
func (GitLogTool) Risk() int { return 0 }

// Describe implements tools.Spec for GitCommitTool.
func (GitCommitTool) Describe() (string, string) {
	return "Stage files and create a git commit in the workspace. " +
			"Args: message (required, the commit message), " +
			"files (optional list of paths to stage; all tracked changes when omitted).",
		`{"type":"object","required":["message"],"properties":{"message":{"type":"string"},` +
			`"files":{"type":"array","items":{"type":"string"}}}}`
}

// Risk implements tools.RiskTier for GitCommitTool.
//
// Committing writes history the operator cannot trivially undo,
// so it is tier 2: the policy gateway may ask before it runs.
func (GitCommitTool) Risk() int { return 2 }

// Describe implements tools.Spec for GitBranchTool.
func (GitBranchTool) Describe() (string, string) {
	return "List, create, or delete a git branch in the workspace. " +
			"Args: action (optional: list, create, or delete; default list), " +
			"name (required for create and delete, the branch name).",
		`{"type":"object","properties":{"action":{"type":"string"},"name":{"type":"string"}}}`
}

// Risk implements tools.RiskTier for GitBranchTool.
func (GitBranchTool) Risk() int { return 1 }

// Describe implements tools.Spec for GitWorktreeTool.
func (GitWorktreeTool) Describe() (string, string) {
	return "List, add, or remove a git worktree in the workspace. " +
			"Args: action (optional: list, add, or remove; default list), " +
			"path (required for add and remove), branch (required for add).",
		`{"type":"object","properties":{"action":{"type":"string"},"path":{"type":"string"},` +
			`"branch":{"type":"string"}}}`
}

// Risk implements tools.RiskTier for GitWorktreeTool.
func (GitWorktreeTool) Risk() int { return 1 }
