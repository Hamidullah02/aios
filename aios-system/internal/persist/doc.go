// Package persist is the session persistence layer for aios.
//
// The package provides four things:
//
//   - RuleStorage: the always-allow rule storage. The
//     storage loads the user-level rules from
//     $AIOS_HOME/rules.json and the project-level rules
//     from $AIOS_DATA_DIR/rules.json. The project rules
//     override the user rules.
//   - SessionCheckpoint: the session state checkpoint.
//     The checkpoint stores the state of a session in
//     $AIOS_DATA_DIR/sessions/<session_id>.json. The
//     checkpoint compresses the state with gzip when the
//     size is above 1 MB.
//   - Handoff: the structured handoff. The handoff
//     stores an artifact under
//     $AIOS_DATA_DIR/handoffs/<task_id>/<key>.json. The
//     handoff supports both file and JSON value handoffs.
//   - SessionHistory: the session history. The history
//     writes the history events to the audit log. The
//     history is append-only with a hash chain.
//
// The package depends on aios-core for the audit log and
// the clock, and on aios-policy for the match semantics.
// The package is depended on by aios-approvals,
// aios-runtime, and aios-cli.
package persist
