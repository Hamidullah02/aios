# Changelog

v0.30.0 — aios-image

Add the aios-image module. The module is the
image attachment layer: it decodes PNG / JPEG
bytes, validates the dimensions against the
model's cap, and renders the model-ready
blocks (Anthropic image block, OpenAI image
URL) for both providers.

The package is a single Go package at
`internal/image/` and depends on
`golang.org/x/image/draw` (already a direct
dep of the monorepo from v0.18.x). No new
third-party dependencies.

Components:

- `Image` — the typed image (bytes, format,
  width, height, color model). The image is
  immutable after `Decode`.
- `ImageAttachment` — the model-ready wrapper.
  Holds the decoded `Image` plus the
  resize-to-max-dimension policy.
- `ToAnthropicBlock` — the Anthropic image
  block (REQ-IMG-ATT-002). The block is a
  map the caller passes as a content block
  to the Anthropic messages API.
- `ToOpenAIURL` — the OpenAI image URL
  (REQ-IMG-ATT-003). The URL is a data-URL
  for the inline case and a https URL for
  the remote case.
- `Constrain` — the dimension-and-format
  validator (REQ-IMG-CON-001..005). The
  validator enforces a max width, max height,
  and a format allow-list.
- `Read` — the file-system reader. The
  reader decodes the bytes through
  `image.Decode` and validates the format
  through `Constrain`.
- `Validator` — the per-model validator
  (REQ-IMG-VAL-001..003). The validator
  reads the model's capability table and
  refuses images the model cannot handle.
- `AttachError`, `ConstrainError`,
  `DecodeError`, `ValidationError` — typed
  errors.

Substitutions vs the doc set:

- **Three sub-packages (`attach` +
  `constrain` + `read`) → single
  `internal/image/` Go package.** Same
  reasoning as `aios-roles` (v0.25.0) and
  `aios-fanout` (v0.22.0): the doc-set
  Python sub-package split is not the
  right shape for a Go monorepo.

v0.29.0 — aios-git

Add the aios-git module. The module is the
git-tools layer: it wraps `git status`,
`git log`, `git diff`, `git commit`,
`git branch`, and `git worktree` behind
typed Go APIs and exposes them as
`tools.Tool` instances the agent can call.

The package is a single Go package at
`internal/git/` and depends on
aios-core (for the clock and the audit
log), aios-tools (for the `Tool`
interface), and aios-sandbox (for the
subprocess runner). No new third-party
dependencies.

Public surface (per REQ-GIT-IFC-001):

- `GitStatusTool` — `git status --porcelain`,
  typed as `[]FileStatus` (path, status,
  staged).
- `GitLogTool` — `git log` with a limit and
  a path filter, typed as `[]Commit`
  (sha, author, timestamp, subject).
- `GitDiffTool` — `git diff` with a path
  filter, typed as `[]FileDiff`
  (path, additions, deletions, patch).
- `GitCommitTool` — `git commit` with a
  message, optional `add` step, and an
  optional author override.
- `GitBranchTool` — `git branch list /
  create / checkout / delete`.
- `GitWorktreeTool` — `git worktree
  list / add / remove`. The worktree
  add honours the worktree root from
  the policy file.
- `Runner` — the subprocess wrapper. The
  runner is the seam for tests: the
  production runner calls the system
  `git`; the test runner captures
  commands without spawning.
- `GitError`, `NotARepositoryError`,
  `BranchNotFoundError`, `WorktreeError`
  — typed errors.

All tools write a `git_call` audit row
on both success and failure. The audit
row carries the args and the result
summary; the diff content is truncated
to 1 KB to keep the audit log small.

Substitutions vs the doc set:

- **Two sub-packages (`status` + `log`)
  → single `internal/git/` Go package.**
  Same reasoning as `aios-roles`
  (