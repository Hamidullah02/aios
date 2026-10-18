# Changelog

## v0.27.0 — aios-verify

Add the aios-verify module. The module is the
self-evaluation layer: the model-judgement
self-check tool and the test-runner tool.
The module lets the agent check the result
before the agent reports the task as done.

The package is a single Go package at
`internal/verify/` and depends on aios-core
(for the audit log), aios-tools (for the
`Tool` interface), aios-sandbox (for the
subprocess runner), and the Go standard
library. **No new third-party dependencies.**
The implementation does not import
`aios-models`: the self-check tool depends
on a `Judge` interface the caller provides
(the same shape as the `Summarizer`
interface in `aios-fanout`; the real
`aios-models.Router` satisfies it).

#