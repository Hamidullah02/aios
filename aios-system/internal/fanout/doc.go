// Package fanout is the parallel execution layer for
// aios.
//
// The package provides four things:
//
//   - FanOutTool: the parallel task executor. The
//     tool accepts a list of tasks, spawns a
//     sub-agent for every task, runs the sub-agents
//     in parallel under a semaphore, and merges
//     the results through a pluggable strategy.
//   - WorktreeIsolation: the per-task worktree
//     wrapper. The type creates a git worktree
//     for every task when the isolate flag is set,
//     changes the working directory, runs the
//     callable, restores the directory, and
//     removes the worktree.
//   - ResultMerger: the result merger. The merger
//     supports three strategies: concat
//     (concatenation in order), summary
//     (caller-provided model summarization), and
//     vote (consensus by content similarity).
//   - Scheduler: the budgeted batch runner. The
//     scheduler runs the tasks in batches of
//     MaxParallel and stops when the budget is
//     exhausted.
//
// The package depends on aios-core (for the audit
// log and the canonical-JSON encoder), aios-runtime
// (for the Worker interface the sub-agents
// implement), and the Go standard library. The
// package is depended on by aios-runtime, aios-cli,
// and aios-fanout callers (the model router when
// the call site wires the Summary strategy).
//
// Per the doc set the package also depends on
// aios-git, aios-watch, and aios-models. The
// implementation substitutes:
//
//   - aios-git → a Worktree interface plus an
//     ExecGitBackend (a thin os/exec wrapper) that
//     any caller may replace with a real
//     aios-git implementation when that module
//     lands.
//   - aios-models (for the Summary strategy) → a
//     Summarizer interface the caller provides.
//     The default is Concat.
//   - aios-watch → no direct dep; the Scheduler
//     exposes a Progress callback the caller may
//     wire to the watch layer.
package fanout
