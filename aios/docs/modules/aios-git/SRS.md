# aios-git — Software Requirements Specification

## 1. Purpose

This document states the software requirements for the
aios-git module.

## 2. Scope

The scope of this document is the aios-git module.
The module is a library.

## 3. Reference Documents

- aios-git PRD. See `docs/modules/aios-git/PRD.md`.
- aios-core SRS. See `docs/modules/aios-core/SRS.md`.
- aios-tools SRS. See `docs/modules/aios-tools/SRS.md`.
- aios-policy SRS. See `docs/modules/aios-policy/SRS.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Status | A typed object that holds the working tree state. |
| Diff | A typed object that holds the diff text. |

## 5. Functional Requirements

### 5.1 GitStatus Tool

**REQ-GIT-STA-001** The module shall provide the
`GitStatusTool` class. The type shall implement the
`Tool` protocol.

**REQ-GIT-STA-002** The tool shall return the
working tree status. The status includes the staged
files, the modified files, and the untracked
files.

**REQ-GIT-STA-003** The tool shall not require an
approval. The tool is read-only.

### 5.2 GitDiff Tool

**REQ-GIT-DIF-001** The module shall provide the
`GitDiffTool` class. The type shall implement the
`Tool` protocol.

**REQ-GIT-DIF-002** The tool shall accept a `path`
argument. The tool shall return the diff for the
path.

**REQ-GIT-DIF-003** The tool shall accept a
`staged` flag. The flag returns the staged diff
when true.

**REQ-GIT-DIF-004** The tool shall support a
`commit` argument. The argument is a commit hash.
The tool returns the diff between the commit and
the working tree.

**REQ-GIT-DIF-005** The tool shall not require an
approval. The tool is read-only.

### 5.3 GitLog Tool

**REQ-GIT-LOG-001** The module shall provide the
`GitLogTool` class. The type shall implement the
`Tool` protocol.

**REQ-GIT-LOG-002** The tool shall accept a `max`
argument. The default is 10. The tool shall return
the last `max` commits.

**REQ-GIT-LOG-003** The tool shall accept a `path`
argument. The tool shall return the commits that
touched the path.

**REQ-GIT-LOG-004** The tool shall return the
hash, the author, the date, and the subject for
every commit.

**REQ-GIT-LOG-005** The tool shall not require an
approval. The tool is read-only.

### 5.4 GitCommit Tool

**REQ-GIT-CMT-001** The module shall provide the
`GitCommitTool` class. The type shall implement
the `Tool` protocol.

**REQ-GIT-CMT-002** The tool shall accept a
`message` argument. The tool shall accept a
`files` argument. The `files` argument is a list
of paths. The default is the staged files.

**REQ-GIT-CMT-003** The tool shall require a
tier-3 approval by default.

**REQ-GIT-CMT-004** The tool shall stage the
files. The tool shall create the commit. The
tool shall return the commit hash.

**REQ-GIT-CMT-005** The tool shall reject an
empty commit. The tool shall raise
`EmptyCommitError` when no files are staged.

### 5.5 GitBranch Tool

**REQ-GIT-BRH-001** The module shall provide the
`GitBranchTool` class. The type shall implement
the `Tool` protocol.

**REQ-GIT-BRH-002** The tool shall accept an
`action` argument. The supported actions are
`list`, `create`, `checkout`, and `delete`.

**REQ-GIT-BRH-003** The tool shall require a
tier-3 approval for `create`, `checkout`, and
`delete`.

**REQ-GIT-BRH-004** The tool shall return the
list of branches on `list`. The tool shall return
the new branch name on `create`. The tool shall
return the current branch on `checkout`.

### 5.6 GitWorktree Tool

**REQ-GIT-WRT-001** The module shall provide the
`GitWorktreeTool` class. The type shall implement
the `Tool` protocol.

**REQ-GIT-WRT-002** The tool shall accept an
`action` argument. The supported actions are
`list`, `add`, and `remove`.

**REQ-GIT-WRT-003** The tool shall accept a
`path` argument and a `branch` argument on
`add`. The tool shall create the worktree at
the path on the branch.

**REQ-GIT-WRT-004** The tool shall require a
tier-3 approval for `add` and `remove`.

## 6. Non-Functional Requirements

**REQ-GIT-NFR-001** The module shall run on Go 1.23
3.12.

**REQ-GIT-NFR-002** The module shall have 100 percent
coverage for the public functions.

**REQ-GIT-NFR-003** The module shall use the
`git` CLI through the os/exec wrapper from
aios-sandbox. The module shall not use the
`dulwich` library in v1.

## 7. Interface Requirements

**REQ-GIT-IFC-001** The module shall export the
following items from `aios.git`.

- `GitStatusTool`
- `GitDiffTool`
- `GitLogTool`
- `GitCommitTool`
- `GitBranchTool`
- `GitWorktreeTool`
- `GitStatus`
- `GitCommit`
- `EmptyCommitError`

## 8. Data Requirements

**REQ-GIT-DAT-001** The `GitStatus` schema shall
include the fields below.

- `branch` (string)
- `staged` (list of string)
- `modified` (list of string)
- `untracked` (list of string)

**REQ-GIT-DAT-002** The `GitCommit` schema shall
include the fields below.

- `hash` (string)
- `author` (string)
- `date` (string, ISO 8601)
- `subject` (string)

## 9. Error Handling

**REQ-GIT-ERR-001** The module shall raise
`NotARepositoryError` when the workspace is not a
git repo.

**REQ-GIT-ERR-002** The module shall raise
`GitCommandError` when the git command fails.

## 10. Acceptance Criteria

1. Every REQ in this document has a test case.
2. The test suite has a 100 percent pass rate.
3. The test suite has 100 percent coverage for the
   public functions.
4. The status tool returns the status.
5. The diff tool returns the diff.
6. The log tool returns the log.
7. The commit tool creates the commit.
8. The branch tool creates and checks out the
   branch.
9. The worktree tool creates the worktree.
10. The commit, branch create, and worktree add
    operations require approval.
