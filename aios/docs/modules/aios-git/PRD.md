# aios-git — Product Requirements Document

## 1. Purpose

This document states the product requirements for the
aios-git module. The module is the git tool set. The
tool set lets the agent run git operations through
the policy gateway. The tool set covers status, diff,
log, commit, branch, and worktree operations.

## 2. Scope

The aios-git module is in scope for this document.

- The GitStatus tool.
- The GitDiff tool.
- The GitLog tool.
- The GitCommit tool.
- The GitBranch tool.
- The GitWorktree tool.

The aios-git module is not in scope for this
document.

- The shell tool. The shell tool is in aios-tools.
- The git CLI itself. The CLI is an external binary.

## 3. Reference Documents

- Project design document, section 7.
- aios-core PRD. See `docs/modules/aios-core/PRD.md`.
- aios-tools PRD. See `docs/modules/aios-tools/PRD.md`.
- aios-policy PRD. See `docs/modules/aios-policy/PRD.md`.
- aios-sandbox PRD. See `docs/modules/aios-sandbox/PRD.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Repo | A git repository. |
| Commit | A snapshot of the repo. |
| Branch | A named pointer to a commit. |
| Worktree | A separate working copy linked to the same repo. |
| Staging | The set of files ready for the next commit. |
| Untracked | A file that is not tracked by git. |
| Modified | A tracked file that has local changes. |


## 5. Determinism

The whole point of aios is to make complex coding and large systems deterministic. The determinism stance of this module is **Bounded non-determinism**.

The remote repository and the file system are the only sources of non-determinism. The git client is deterministic for a fixed repository state.

The module provides the following determinism guarantees.

- Git operations are pure functions of the repository state and the command.
## 6. User Personas

The user persona is the agent that needs to inspect
and change the git state of the project. The agent
needs safe, auditable, and approval-gated git
operations.

## 7. User Stories

An agent wants to check the repo status. The agent
calls the GitStatus tool. The tool returns the
status of the working tree.

An agent wants to view the diff for a file. The
agent calls the GitDiff tool. The tool returns the
diff.

An agent wants to view the recent commits. The
agent calls the GitLog tool. The tool returns the
log.

An agent wants to commit the changes. The agent
calls the GitCommit tool. The tool stages the
files. The tool creates the commit. The tool
returns the commit hash.

An agent wants to switch to a new branch. The
agent calls the GitBranch tool. The tool creates
the branch. The tool checks out the branch.

An agent wants to run a task on a separate worktree.
The agent calls the GitWorktree tool. The tool
creates the worktree. The tool returns the
worktree path.

## 8. Functional Requirements Summary

The aios-git module provides the following
functions.

- The GitStatus tool.
- The GitDiff tool.
- The GitLog tool.
- The GitCommit tool.
- The GitBranch tool.
- The GitWorktree tool.

## 9. Non-Functional Requirements Summary

- The module runs on Go 1.23.
- The module depends on aios-core, aios-tools,
  aios-policy, aios-sandbox, and the `go-git`
  Go library or the `git` CLI.
- The module has 100 percent test coverage for the
  public functions.
- The commit operation requires a tier-3 approval
  by default.

## 10. Success Criteria

- The status tool returns the working tree status.
- The diff tool returns the diff.
- The log tool returns the log.
- The commit tool creates the commit.
- The branch tool creates and checks out the
  branch.
- The worktree tool creates the worktree.
- The test suite has a 100 percent pass rate.

## 11. Out of Scope

- The user interface for git operations.
- The remote access to git operations.
- The pull request creation.
- The merge and rebase operations.
