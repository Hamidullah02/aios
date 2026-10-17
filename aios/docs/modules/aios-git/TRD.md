# aios-git — Technical Requirements Document

## 1. Purpose

This document states the technical approach for the
aios-git module.

## 2. Scope

The scope of this document is the aios-git module.
The module is a Go package.

## 3. Reference Documents

- aios-git PRD. See `docs/modules/aios-git/PRD.md`.
- aios-git SRS. See `docs/modules/aios-git/SRS.md`.
- aios-core TRD. See `docs/modules/aios-core/TRD.md`.
- aios-tools TRD. See `docs/modules/aios-tools/TRD.md`.
- aios-sandbox TRD. See `docs/modules/aios-sandbox/TRD.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| git | The distributed version control system. |
| CLI | The command-line interface. |

## 5. Architecture

The aios-git module has six components.

- `aios.git.status` — the GitStatus tool.
- `internal/git/diff` — the GitDiff tool.
- `aios.git.log` — the GitLog tool.
- `internal/git/commit` — the GitCommit tool.
- `aios.git.branch` — the GitBranch tool.
- `aios.git.worktree` — the GitWorktree tool.

The six components depend on aios-core, aios-tools,
aios-policy, and aios-sandbox. The components
depend on each other in this order. The status
tool has no dependencies. The diff tool has no
dependencies. The log tool has no dependencies. The
commit tool depends on the policy gateway. The
branch tool depends on the policy gateway. The
worktree tool depends on the policy gateway.

## 6. Components

### 6.1 Status Tool

The status tool runs `git status --porcelain`. The
tool parses the output. The tool returns the
staged, modified, and untracked files.

### 6.2 Diff Tool

The diff tool runs `git diff` with the path. The
tool returns the diff text. The tool supports the
`--staged` flag. The tool supports the commit
argument.

### 6.3 Log Tool

The log tool runs `git log` with the max and the
path. The tool parses the output. The tool returns
the list of commits.

### 6.4 Commit Tool

The commit tool runs `git add` for the files. The
tool runs `git commit -m` with the message. The
tool returns the commit hash.

The tool requires a tier-3 approval by default.
The tool checks the policy gateway before the
commit.

### 6.5 Branch Tool

The branch tool runs `git branch` for the action.
The tool returns the result. The tool requires a
tier-3 approval for `create`, `checkout`, and
`delete`.

### 6.6 Worktree Tool

The worktree tool runs `git worktree` for the
action. The tool returns the result. The tool
requires a tier-3 approval for `add` and
`remove`.

## 7. Data Structures

### 7.1 GitStatus

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `branch` | string | yes | The current branch. |
| `staged` | list of string | yes | The staged files. |
| `modified` | list of string | yes | The modified files. |
| `untracked` | list of string | yes | The untracked files. |

### 7.2 GitCommit

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `hash` | string | yes | The commit hash. |
| `author` | string | yes | The author. |
| `date` | string | yes | The commit date. |
| `subject` | string | yes | The commit subject. |

## 8. Interfaces

### 8.1 GitCommitTool

```go
type GitCommitTool:
    """The git commit tool."""

    func run(self, args: dict, context: ToolContext) -> ToolResult:
        """Run the git commit tool.

        Args:
            args: The arguments of the call.
            context: The context for the call.

        Returns:
            The result of the call.
        """
```

## 9. Algorithms

### 9.1 Status Algorithm

The status algorithm works as follows.

1. Run `git status --porcelain`.
2. Parse the output by line.
3. For each line, classify the file as staged,
   modified, or untracked.
4. Return the `GitStatus` object.

### 9.2 Commit Algorithm

The commit algorithm works as follows.

1. Validate the arguments.
2. Check the policy gateway for the approval.
3. On deny, return an error.
4. On ask, raise `WaitingForApproval`.
5. On allow, run `git add` for the files.
6. Run `git commit -m` with the message.
7. Run `git rev-parse HEAD` to get the commit
   hash.
8. Return the `GitCommit` object.

## 10. Dependencies

### 10.1 Runtime Dependencies

- Go 1.23
- aios-core
- aios-tools
- aios-policy
- aios-sandbox

### 10.2 Module Dependencies

The aios-git module depends on the following
internal modules.

- aios-core
- aios-tools
- aios-policy
- aios-sandbox

The aios-git module is depended on by the following
internal modules.

- aios-runtime
- aios-cli
- aios-fanout

## 11. Deployment

The module compiles into the monorepo under `internal/<module>/`. The
package name is `github.com/<org>/aios/internal/git`. The module requires
the `git` CLI to be on the PATH.

## 12. Risks and Mitigations

| Risk | Impact | Probability | Mitigation |
| --- | --- | --- | --- |
| The git command fails. | Medium | Low | The tool raises `GitCommandError`. The tool returns the stderr. |
| The commit is created without approval. | High | Low | The tool checks the policy gateway. The tool has a test case. |
| The worktree path collides. | Medium | Low | The tool checks the path. The tool raises an error on collision. |
