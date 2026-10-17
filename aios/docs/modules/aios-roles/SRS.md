# aios-roles — Software Requirements Specification

## 1. Purpose

This document states the software requirements for the
aios-roles module.

## 2. Scope

The scope of this document is the aios-roles module.
The module is a library.

## 3. Reference Documents

- aios-roles PRD. See `docs/modules/aios-roles/PRD.md`.
- aios-core SRS. See `docs/modules/aios-core/SRS.md`.
- aios-models SRS. See `docs/modules/aios-models/SRS.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Role | A typed object that holds the agent configuration. |
| Role config | A typed object that holds the role definitions. |

## 5. Functional Requirements

### 5.1 Role Registry

**REQ-ROL-REG-001** The module shall provide the
`RoleRegistry` class. The type shall accept the
list of roles.

**REQ-ROL-REG-002** The registry shall provide the
`get` method. The method shall accept a role name.
The method shall return the role.

**REQ-ROL-REG-003** The registry shall provide the
`set` method. The method shall accept a role. The
method shall update the registry.

**REQ-ROL-REG-004** The registry shall provide the
`list` method. The method shall return the list
of role names.

**REQ-ROL-REG-005** The registry shall load the
roles from a YAML file. The default path is
`./var/roles.yaml`.

### 5.2 Role Configuration

**REQ-ROL-CFG-001** A role shall have a `name`, a
`system_prompt`, a `tools` list, a `model_tier`,
and an `output_format`.

**REQ-ROL-CFG-002** The `output_format` shall be
`text`, `json`, or `code`.

**REQ-ROL-CFG-003** The role shall support a
`temperature` argument. The default is 0.

**REQ-ROL-CFG-004** The role shall support a
`max_tokens` argument. The default is the model
maximum.

**REQ-ROL-CFG-005** The role shall support a
`stop_sequences` list. The list is the stop
sequences for the model.

### 5.3 Role Resolver

**REQ-ROL-RES-001** The module shall provide the
`RoleResolver` class. The type shall accept the
role registry.

**REQ-ROL-RES-002** The resolver shall accept a
role name. The resolver shall return the
configuration.

**REQ-ROL-RES-003** The resolver shall raise
`UnknownRoleError` when the name is not in the
registry.

**REQ-ROL-RES-004** The resolver shall support a
fallback role. The resolver returns the fallback
when the name is not in the registry and the
fallback is set.

### 5.4 Default Roles

**REQ-ROL-DEF-001** The module shall provide the
following default roles.

- `planner` — the planner role. The role uses the
  strong tier. The role uses the plan output
  format.
- `coder` — the coder role. The role uses the
  strong tier. The role uses the code output
  format.
- `editor` — the editor role. The role uses the
  medium tier. The role uses the code output
  format.
- `explorer` — the explorer role. The role uses
  the weak tier. The role uses the text output
  format. The role uses the read-only tools.
- `reviewer` — the reviewer role. The role uses
  the strong tier. The role uses the text output
  format.
- `verifier` — the verifier role. The role uses
  the strong tier. The role uses the json output
  format.

## 6. Non-Functional Requirements

**REQ-ROL-NFR-001** The module shall run on Go 1.23
3.12.

**REQ-ROL-NFR-002** The module shall have 100 percent
coverage for the public functions.

**REQ-ROL-NFR-003** The resolver decision shall
have a maximum latency of 1 millisecond.

## 7. Interface Requirements

**REQ-ROL-IFC-001** The module shall export the
following items from `aios.roles`.

- `RoleRegistry`
- `RoleResolver`
- `Role`
- `OutputFormat`
- `UnknownRoleError`

## 8. Data Requirements

**REQ-ROL-DAT-001** The `Role` schema shall include
the fields below.

- `name` (string)
- `system_prompt` (string)
- `tools` (list of string)
- `model_tier` (string: strong, medium, weak)
- `output_format` (string: text, json, code)
- `temperature` (float, default 0)
- `max_tokens` (int, optional)
- `stop_sequences` (list of string, optional)

**REQ-ROL-DAT-002** The default roles file shall
be loaded from `./var/roles.yaml`. The file is
created on `aios init`.

## 9. Error Handling

**REQ-ROL-ERR-001** The module shall raise
`UnknownRoleError` when the role name is not in
the registry.

**REQ-ROL-ERR-002** The module shall raise
`InvalidRoleError` when the role definition is
invalid.

## 10. Acceptance Criteria

1. Every REQ in this document has a test case.
2. The test suite has a 100 percent pass rate.
3. The test suite has 100 percent coverage for the
   public functions.
4. The role registry stores the roles.
5. The role configuration is loaded from a YAML
   file.
6. The role resolver returns the configuration.
7. The default roles cover the common cases.
