# aios-git — Application Flow

## 1. Purpose

This document describes the application flow for the
aios-git module.

## 2. Scope

The scope of this document is the aios-git module.
The module is a library.

## 3. Reference Documents

- aios-git PRD. See `docs/modules/aios-git/PRD.md`.
- aios-git SRS. See `docs/modules/aios-git/SRS.md`.
- aios-git TRD. See `docs/modules/aios-git/TRD.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Agent | The code that calls the git tool. |

## 5. User Flow

The aios-git module has no user-facing flow.

## 6. System Flow

### 6.1 Status Flow

The status flow runs as follows.

1. The agent calls the GitStatus tool.
2. The tool runs `git status --porcelain`.
3. The tool parses the output.
4. The tool returns the `GitStatus` object.

### 6.2 Diff Flow

The diff flow runs as follows.

1. The agent calls the GitDiff tool.
2. The tool runs `git diff` with the path.
3. The tool returns the diff text.

### 6.3 Log Flow

The log flow runs as follows.

1. The agent calls the GitLog tool.
2. The tool runs `git log` with the max and the
   path.
3. The tool parses the output.
4. The tool returns the list of commits.

### 6.4 Commit Flow

The commit flow runs as follows.

1. The agent calls the GitCommit tool.
2. The tool checks the policy gateway.
3. On allow, the tool runs `git add` and
   `git commit`.
4. The tool returns the commit hash.
5. On ask, the tool raises `WaitingForApproval`.

## 7. Error Flows

### 7.1 Not A Repository

The not a repository flow runs as follows.

1. The workspace is not a git repo.
2. The tool raises `NotARepositoryError`.
3. The agent catches the error.
4. The agent initializes the repo with
   `git init`.

### 7.2 Git Command Error

The git command error flow runs as follows.

1. The git command fails.
2. The tool raises `GitCommandError`.
3. The agent catches the error.
4. The agent reports the error.

## 8. Sequence Diagrams

### 8.1 Commit Sequence

The commit sequence has the steps below.

1. The agent calls the GitCommit tool.
2. The tool checks the policy gateway.
3. The gateway returns allow.
4. The tool runs `git add`.
5. The tool runs `git commit`.
6. The tool returns the commit hash.
