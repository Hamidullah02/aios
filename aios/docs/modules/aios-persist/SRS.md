# aios-persist — Software Requirements Specification

## 1. Purpose

This document states the software requirements for the
aios-persist module.

## 2. Scope

The scope of this document is the aios-persist module.
The module is a library.

## 3. Reference Documents

- aios-persist PRD. See
  `docs/modules/aios-persist/PRD.md`.
- aios-core SRS. See `docs/modules/aios-core/SRS.md`.
- aios-policy SRS. See `docs/modules/aios-policy/SRS.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Always-allow rule | A typed object that holds the rule definition. |
| Handoff | A typed object that holds the artifact data. |
| Session | A typed object that holds the session state. |

## 5. Functional Requirements

### 5.1 Always-Allow Rule Storage

**REQ-PRS-RUL-001** The module shall provide the
`RuleStorage` class. The type shall accept the
storage path.

**REQ-PRS-RUL-002** The storage shall provide the
`add` method. The method shall accept the rule.
The method shall save the rule.

**REQ-PRS-RUL-003** The storage shall provide the
`remove` method. The method shall accept the rule
id. The method shall remove the rule.

**REQ-PRS-RUL-004** The storage shall provide the
`list` method. The method shall return the list
of rules.

**REQ-PRS-RUL-005** The storage shall provide the
`match` method. The method shall accept a tool
call. The method shall return the matching rule.

**REQ-PRS-RUL-006** The storage shall support
user-level rules and project-level rules. The
project-level rules override the user-level rules.

### 5.2 Rule Schema

**REQ-PRS-RSC-001** A rule shall have an `id`, a
`match`, an `action`, a `scope`, and a `created_at`.

**REQ-PRS-RSC-002** The `match` shall have a
`tool`, a `path_glob`, a `host_glob`, and an
`actor`.

**REQ-PRS-RSC-003** The `action` shall be `allow`
or `deny`.

**REQ-PRS-RSC-004** The `scope` shall be `session`,
`project`, or `user`.

### 5.3 Session Checkpoint

**REQ-PRS-CHK-001** The module shall provide the
`SessionCheckpoint` class. The type shall accept
the storage path.

**REQ-PRS-CHK-002** The checkpoint shall provide
the `save` method. The method shall accept the
session id and the state. The method shall save
the state.

**REQ-PRS-CHK-003** The checkpoint shall provide
the `load` method. The method shall accept the
session id. The method shall return the state.

**REQ-PRS-CHK-004** The checkpoint shall provide
the `list` method. The method shall return the
list of past sessions.

**REQ-PRS-CHK-005** The checkpoint shall provide
the `delete` method. The method shall accept the
session id. The method shall delete the session.

### 5.4 Structured Handoff

**REQ-PRS-HND-001** The module shall provide the
`Handoff` class. The type shall accept the
storage path.

**REQ-PRS-HND-002** The handoff shall provide the
`put` method. The method shall accept the task
id, the key, and the value. The method shall save
the value.

**REQ-PRS-HND-003** The handoff shall provide the
`get` method. The method shall accept the task
id and the key. The method shall return the
value.

**REQ-PRS-HND-004** The handoff shall support
file handoffs. The handoff shall save the path
to the artifact.

**REQ-PRS-HND-005** The handoff shall support
value handoffs. The handoff shall save the value
as JSON.

### 5.5 Session History

**REQ-PRS-HST-001** The module shall provide the
`SessionHistory` class. The type shall accept the
storage path.

**REQ-PRS-HST-002** The history shall provide the
`add` method. The method shall accept the session
id, the task id, the action, and the timestamp.
The method shall save the event.

**REQ-PRS-HST-003** The history shall provide the
`list` method. The method shall accept the
session id. The method shall return the events
in time order.

**REQ-PRS-HST-004** The history shall provide the
`search` method. The method shall accept a query.
The method shall return the matching events.

## 6. Non-Functional Requirements

**REQ-PRS-NFR-001** The module shall run on Go 1.23
3.12.

**REQ-PRS-NFR-002** The module shall have 100 percent
coverage for the public functions.

**REQ-PRS-NFR-003** The rule match shall have a
maximum latency of 1 millisecond.

## 7. Interface Requirements

**REQ-PRS-IFC-001** The module shall export the
following items from `aios.persist`.

- `RuleStorage`
- `SessionCheckpoint`
- `Handoff`
- `SessionHistory`
- `Rule`
- `RuleMatch`
- `RuleAction`

## 8. Data Requirements

**REQ-PRS-DAT-001** The default rule file shall
be `~/.aios/rules.json`. The file is a JSON
array of rules.

**REQ-PRS-DAT-002** The default session
checkpoint path shall be `./var/sessions/`. The
file is a JSON object with the session id and
the state.

**REQ-PRS-DAT-003** The default handoff path
shall be `./var/handoffs/`. The file is a JSON
object with the task id, the key, and the value.

## 9. Error Handling

**REQ-PRS-ERR-001** The module shall raise
`RuleNotFoundError` when the rule id is not in
the storage.

**REQ-PRS-ERR-002** The module shall raise
`SessionNotFoundError` when the session id is
not in the checkpoint.

**REQ-PRS-ERR-003** The module shall raise
`HandoffNotFoundError` when the key is not in
the handoff.

## 10. Acceptance Criteria

1. Every REQ in this document has a test case.
2. The test suite has a 100 percent pass rate.
3. The test suite has 100 percent coverage for the
   public functions.
4. The rule storage saves the rules.
5. The session checkpoint saves the session.
6. The handoff passes the artifact.
7. The history lists the events.
