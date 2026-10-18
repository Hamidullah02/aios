# aios-persist

The aios-persist module is the session persistence
layer. The module is a single Go package at
`internal/persist/` and depends on aios-core and
aios-policy.

## Public surface

- `RuleStorage` — the always-allow rule storage. The
  storage loads the user-level rules from
  `$AIOS_HOME/rules.json` and the project-level rules
  from `$AIOS_DATA_DIR/rules.json`. The project rules
  override the user rules.
- `SessionCheckpoint` — the session state checkpoint.
  The checkpoint stores the state of a session in
  `$AIOS_DATA_DIR/sessions/<session_id>.json`. The
  checkpoint compresses the state with gzip when the
  size is above 1 MB.
- `Handoff` — the structured handoff. The handoff
  stores an artifact under
  `$AIOS_DATA_DIR/handoffs/<task_id>/<key>.json`. The
  handoff supports both file and JSON value handoffs.
- `SessionHistory` — the session history. The history
  writes the history events to the audit log under
  kind `session_event`. The history is append-only with
  a hash chain (the audit log's chain).
- `Rule`, `RuleMatch`, `RuleAction`, `RuleScope` — the
  rule types.
- `SessionState`, `StepState` — the session types.
- `HandoffRecord` — the on-disk form of a handoff.
- `HistoryEvent` — the typed payload of one history
  event.
- `RuleNotFoundError`, `SessionNotFoundError`,
  `HandoffNotFoundError` — typed errors.

## Environment variables

- `AIOS_HOME` — the user-level home. Defaults to
  `$HOME/.aios/` (or `%USERPROFILE%\.aios\` on
  Windows).
- `AIOS_DATA_DIR` — the project-level data directory.
  Defaults to `./var/`.

## Paths

| Component | Default path |
| --- | --- |
| Rules (user) | `$AIOS_HOME/rules.json` |
| Rules (project) | `$AIOS_DATA_DIR/rules.json` |
| Sessions | `$AIOS_DATA_DIR/sessions/` |
| Handoffs | `$AIOS_DATA_DIR/handoffs/` |

The user can override the paths through the
constructors' `Config.Path` fields.

## Tests

The tests use a fixed clock and temp directories. The
history tests open a real SQLite audit log in a temp
directory. Coverage on the package is 83.8% on
Windows.
