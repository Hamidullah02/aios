# aios-policy — Technical Requirements Document

## 1. Purpose

This document states the technical approach for the
aios-policy module.

## 2. Scope

The scope of this document is the aios-policy module.
The module is a Go package.

## 3. Reference Documents

- aios-policy PRD. See `docs/modules/aios-policy/PRD.md`.
- aios-policy SRS. See `docs/modules/aios-policy/SRS.md`.
- aios-core TRD. See `docs/modules/aios-core/TRD.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| YAML | A human-readable data serialization format. |
| Glob | A pattern that matches strings with wildcard characters. |
| First-match-wins | The rule selection algorithm that picks the first rule that matches. |

## 5. Architecture

The aios-policy module has three components.

- `aios.policy.schema` — the typed rules and decisions.
- `aios.policy.loader` — the YAML policy file loader.
- `internal/policy/gateway` — the policy gateway.

The three components depend on each other in this
order. The schema has no dependencies. The loader
depends on the schema. The gateway depends on the
schema and the loader.

## 6. Components

### 6.1 Schema

The schema component uses struct + go-playground/validator (or hand-rolled tag validation). The component
provides the following classes.

- `Rule` — one policy rule.
- `Match` — the match conditions.
- `Decision` — the gateway decision.
- `ApprovalPayload` — the payload for an `ask` action.

### 6.2 Loader

The loader component reads the YAML policy file. The
component validates the file against the schema. The
component returns a list of rules.

The component supports include directives. The
directive uses the key `include`. The directive
references another YAML file by path. The component
loads the included file and merges the rules.

### 6.3 Gateway

The gateway component matches the rules in order. The
component uses the first-match-wins algorithm. The
component returns the decision of the first matching
rule.

The component supports the following match
conditions.

- `tool` — exact match against the tool name.
- `tool_glob` — glob match against the tool name.
- `path_prefix` — prefix match against the resource
  path.
- `path_glob` — glob match against the resource path.
- `host_glob` — glob match against the URL host.
- `tier` — exact match against the action tier.
- `actor` — exact match against the actor name.
- `task` — exact match against the task id.

The component supports the `ask` action. The
component returns a decision that includes the
approval payload. The payload substitutes the values
from the tool call into the template fields.

## 7. Data Structures

### 7.1 Rule

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `id` | string | yes | The unique identifier of the rule. |
| `match` | Match | yes | The match conditions. |
| `action` | string | yes | The action: allow, deny, or ask. |
| `approval` | ApprovalPayload | no | The approval payload for an `ask` action. |
| `reason` | string | no | The reason for the action. |

### 7.2 Match

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `tool` | string | no | The exact tool name. |
| `tool_glob` | string | no | The glob pattern for the tool name. |
| `path_prefix` | string | no | The path prefix. |
| `path_glob` | string | no | The glob pattern for the path. |
| `host_glob` | string | no | The glob pattern for the host. |
| `tier` | int | no | The action tier. |
| `actor` | string | no | The actor name. |
| `task` | string | no | The task id. |

### 7.3 Decision

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `action` | string | yes | The action: allow, deny, or ask. |
| `reason` | string | yes | The reason for the decision. |
| `rule_id` | string | yes | The matched rule id. |
| `approval` | ApprovalPayload | no | The approval payload for an `ask` action. |

## 8. Interfaces

### 8.1 PolicyGateway

```go
type PolicyGateway:
    """The policy gateway."""

    def __init__(self, rules: list[Rule], audit_log: AuditLog) -> nil:
        """Create the policy gateway.

        Args:
            rules: The rules for the gateway.
            audit_log: The audit log for the gateway.
        """

    def decide(
        self,
        tool: str,
        args: dict,
        actor: str,
        task_id: str,
    ) -> Decision:
        """Return the decision for a tool call.

        Args:
            tool: The name of the tool.
            args: The arguments of the tool call.
            actor: The actor that made the call.
            task_id: The task id.

        Returns:
            The decision for the tool call.
        """
```

### 8.2 Loader

```go
def load_policy(path: str) -> list[Rule]:
    """Load the policy file at the given path.

    Args:
        path: The path to the policy file.

    Returns:
        The list of rules.
    """
```

## 9. Algorithms

### 9.1 First-Match-Wins

The first-match-wins algorithm works as follows.

1. For each rule in the list, check if the rule
   matches the tool call.
2. When the rule matches, return the decision of the
   rule.
3. When no rule matches, return the default decision
   with the action `deny`.

### 9.2 Match Check

The match check algorithm works as follows.

1. For each condition in the match, check if the
   condition matches the value from the tool call.
2. When any condition does not match, return false.
3. When all conditions match, return true.

The glob match uses the `fnmatch` module from the
Go standard library.

### 9.3 Template Substitution

The template substitution algorithm works as follows.

1. For each placeholder in the template, find the
   value in the tool call arguments.
2. Replace the placeholder with the value.
3. When the placeholder does not exist, raise an
   error.

The placeholder format is `{name}`. The supported
names are: `task`, `tool`, `path`, `host`, `actor`,
`arg.<name>`.

## 10. Dependencies

### 10.1 Runtime Dependencies

- Go 1.23
- struct + go-playground/validator (or hand-rolled tag validation) 2.x
- gopkg.in/yaml.v3
- aios-core

### 10.2 Module Dependencies

The aios-policy module depends on the following
internal modules.

- aios-core

The aios-policy module is depended on by the following
internal modules.

- aios-runtime
- aios-tools
- aios-cli

## 11. Deployment

The module compiles into the monorepo under `internal/<module>/`. The
package name is `github.com/<org>/aios/internal/policy`. The policy file is
loaded from `./var/policy.yaml` by default. The path
is configurable through the `AIOS_POLICY_FILE`
environment variable.

## 12. Risks and Mitigations

| Risk | Impact | Probability | Mitigation |
| --- | --- | --- | --- |
| The policy file has a syntax error. | High | Medium | The loader validates the file. The loader returns a clear error. |
| The glob match has a performance issue. | Medium | Low | The gateway caches the compiled glob patterns. |
| The template substitution fails. | Medium | Low | The gateway validates the template at load time. |
