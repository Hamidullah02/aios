# aios-roles — Technical Requirements Document

## 1. Purpose

This document states the technical approach for the
aios-roles module.

## 2. Scope

The scope of this document is the aios-roles module.
The module is a Go package.

## 3. Reference Documents

- aios-roles PRD. See `docs/modules/aios-roles/PRD.md`.
- aios-roles SRS. See `docs/modules/aios-roles/SRS.md`.
- aios-core TRD. See `docs/modules/aios-core/TRD.md`.
- aios-models TRD. See `docs/modules/aios-models/TRD.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Role | A typed object that holds the agent configuration. |
| YAML | A human-readable data serialization format. |

## 5. Architecture

The aios-roles module has three components.

- `internal/roles/schema` — the role schema.
- `internal/roles/registry` — the role registry.
- `internal/roles/resolver` — the role resolver.

The three components depend on aios-core and
aios-models. The components depend on each other in
this order. The schema has no dependencies. The
registry depends on the schema. The resolver
depends on the registry.

## 6. Components

### 6.1 Schema

The schema component uses struct + go-playground/validator (or hand-rolled tag validation). The component
defines the `Role` class. The component validates
the role definitions.

### 6.2 Registry

The registry component holds the roles. The
component loads the roles from a YAML file. The
component provides the `get`, `set`, and `list`
methods.

The component uses the `Role` schema to validate
the loaded roles.

### 6.3 Resolver

The resolver component returns the configuration
for a role name. The resolver looks up the role
in the registry. The resolver returns the
configuration.

The resolver supports a fallback role. The
resolver returns the fallback when the name is
not in the registry.

## 7. Data Structures

### 7.1 Role

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `name` | string | yes | The name of the role. |
| `system_prompt` | string | yes | The system prompt. |
| `tools` | list of string | yes | The list of tools. |
| `model_tier` | string | yes | The model tier. |
| `output_format` | string | yes | The output format. |
| `temperature` | float | no | The temperature. |
| `max_tokens` | int | no | The max tokens. |
| `stop_sequences` | list of string | no | The stop sequences. |

## 8. Interfaces

### 8.1 RoleResolver

```go
type RoleResolver:
    """The role resolver."""

    def __init__(
        self,
        registry: RoleRegistry,
        fallback: str | nil = nil,
    ) -> nil:
        """Create the role resolver."""

    def resolve(self, name: str) -> Role:
        """Resolve a role.

        Args:
            name: The name of the role.

        Returns:
            The role configuration.
        """
```

## 9. Algorithms

### 9.1 Resolution Algorithm

The resolution algorithm works as follows.

1. Look up the name in the registry.
2. When the name is found, return the role.
3. When the name is not found and the fallback is
   set, return the fallback.
4. When the name is not found and the fallback is
   not set, raise `UnknownRoleError`.

## 10. Dependencies

### 10.1 Runtime Dependencies

- Go 1.23
- gopkg.in/yaml.v3
- aios-core
- aios-models

### 10.2 Module Dependencies

The aios-roles module depends on the following
internal modules.

- aios-core
- aios-models

The aios-roles module is depended on by the
following internal modules.

- aios-planner
- aios-runtime
- aios-watch
- aios-cli

## 11. Deployment

The module compiles into the monorepo under `internal/<module>/`. The
package name is `github.com/<org>/aios/internal/roles`. The default roles
file is loaded from `./var/roles.yaml`.

## 12. Risks and Mitigations

| Risk | Impact | Probability | Mitigation |
| --- | --- | --- | --- |
| The role definition is invalid. | High | Low | The schema validates the role. The schema raises an error on invalid data. |
| The role name is misspelled. | Medium | Medium | The resolver supports a fallback. The test suite covers the typo cases. |
| The role file is missing. | High | Low | The CLI creates the file on `aios init`. The loader raises a clear error. |
