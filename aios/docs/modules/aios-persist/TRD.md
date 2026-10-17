# aios-persist — Technical Requirements Document

## 1. Purpose

This document states the technical approach for the
aios-persist module.

## 2. Scope

The scope of this document is the aios-persist module.
The module is a Go package.

## 3. Reference Documents

- aios-persist PRD. See
  `docs/modules/aios-persist/PRD.md`.
- aios-persist SRS. See
  `docs/modules/aios-persist/SRS.md`.
- aios-core TRD. See `docs/modules/aios-core/TRD.md`.
- aios-policy TRD. See `docs/modules/aios-policy/TRD.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| JSON | The JavaScript Object Notation. |
| Glob | A pattern for filename matching. |

## 5. Architecture

The aios-persist module has four components.

- `internal/persist/rules` — the rule storage.
- `internal/persist/checkpoint` — the session
  checkpoint.
- `internal/persist/handoff` — the structured
  handoff.
- `internal/persist/history` — the session history.

The four components depend on aios-core and
aios-policy. The components depend on each other in
this order. The rule storage has no dependencies.
The checkpoint has no dependencies. The handoff
has no dependencies. The history has no
dependencies.

## 6. Components

### 6.1 Rule Storage

The rule storage component uses JSON files. The
component loads the user-level rules from
`~/.aios/rules.json`. The component loads the
project-level rules from `./var/rules.json`. The
component merges the rules. The project-level rules
override the user-level rules.

The component provides the `add`, `remove`,
`list`, and `match` methods. The `match` method
iterates over the rules. The method returns the
first matching rule.

The component supports the `session` scope. The
session rules are stored in memory and discarded
on exit.

### 6.2 Session Checkpoint

The session checkpoint component uses JSON files.
The component stores the session state in
`./var/sessions/<session_id>.json`. The component
provides the `save`, `load`, `list`, and `delete`
methods.

The component supports compression. The component
compresses the state with gzip when the size is
above 1 MB.

### 6.3 Handoff

The handoff component uses JSON files. The
component stores the handoffs in
`./var/handoffs/<task_id>/<key>.json`. The
component provides the `put` and `get` methods.

The component supports file handoffs. When the
value is a path, the component stores the path.
When the next task reads the path, the component
returns the path.

The component supports value handoffs. When the
value is a JSON-serializable object, the
component stores the value as JSON.

### 6.4 Session History

The session history component uses the audit log.
The component writes the history events to the
audit log. The component provides the `add`,
`list`, and `search` methods.

The component supports search through a simple
grep on the audit log. The component uses the
audit log query API.

## 7. Data Structures

### 7.1 Rule

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `id` | string | yes | The unique identifier. |
| `match` | RuleMatch | yes | The match conditions. |
| `action` | string | yes | The action: allow or deny. |
| `scope` | string | yes | The scope: session, project, user. |
| `created_at` | string | yes | The ISO 8601 timestamp. |

### 7.2 Handoff

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `task_id` | string | yes | The task id. |
| `key` | string | yes | The key. |
| `value` | any | yes | The value. |
| `is_path` | bool | no | Whether the value is a path. |

## 8. Interfaces

### 8.1 RuleStorage

```go
type RuleStorage:
    """The rule storage."""

    def __init__(
        self,
        user_path: str = "~/.aios/rules.json",
        project_path: str = "./var/rules.json",
    ) -> nil:
        """Create the rule storage."""

    def add(self, rule: Rule) -> nil:
        """Add a rule."""

    def remove(self, rule_id: str) -> nil:
        """Remove a rule."""

    def list(self) -> list[Rule]:
        """List the rules."""

    def match(self, tool_call: ToolCall) -> Rule | nil:
        """Return the matching rule.

        Args:
            tool_call: The tool call to match.

        Returns:
            The matching rule, or nil.
        """
```

### 8.2 Handoff

```go
type Handoff:
    """The structured handoff."""

    def __init__(self, path: str = "./var/handoffs/") -> nil:
        """Create the handoff."""

    def put(self, task_id: str, key: str, value: any) -> nil:
        """Put a handoff value.

        Args:
            task_id: The task id.
            key: The key.
            value: The value.
        """

    def get(self, task_id: str, key: str) -> any:
        """Get a handoff value.

        Args:
            task_id: The task id.
            key: The key.

        Returns:
            The value.
        """
```

## 9. Algorithms

### 9.1 Rule Match Algorithm

The rule match algorithm works as follows.

1. Iterate over the rules in priority order.
2. For each rule, check the match conditions.
3. When the rule matches, return the rule.
4. When no rule matches, return nil.

### 9.2 Handoff Put Algorithm

The handoff put algorithm works as follows.

1. Build the path: `<path>/<task_id>/<key>.json`.
2. Create the directory if it does not exist.
3. Serialize the value to JSON.
4. Write the file.

## 10. Dependencies

### 10.1 Runtime Dependencies

- Go 1.23
- aios-core
- aios-policy

### 10.2 Module Dependencies

The aios-persist module depends on the following
internal modules.

- aios-core
- aios-policy

The aios-persist module is depended on by the
following internal modules.

- aios-approvals
- aios-runtime
- aios-cli

## 11. Deployment

The module compiles into the monorepo under `internal/<module>/`. The
package name is `github.com/<org>/aios/internal/persist`. The default paths
are configurable through the `AIOS_DATA_DIR`
environment variable.

## 12. Risks and Mitigations

| Risk | Impact | Probability | Mitigation |
| --- | --- | --- | --- |
| The rule file is corrupted. | High | Low | The file is JSON-validated. The file is backed up. |
| The handoff file is too large. | Medium | Low | The handoff supports compression. The handoff supports the `is_path` flag. |
| The session checkpoint is stale. | Medium | Low | The checkpoint has a TTL. The checkpoint is refreshed on every save. |
