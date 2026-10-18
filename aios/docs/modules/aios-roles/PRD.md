# aios-roles — Product Requirements Document

## 1. Purpose

This document states the product requirements for the
aios-roles module. The module is the agent role
registry. The module maps roles to system prompts,
tool sets, and model classes. The module lets the
planner and the orchestrator use the right
configuration for every task.

## 2. Scope

The aios-roles module is in scope for this document.

- The role registry.
- The role configuration.
- The role resolver.
- The default roles.

The aios-roles module is not in scope for this
document.

- The model adapter. The adapter is in aios-models.
- The orchestrator. The orchestrator is in
  aios-runtime.
- The planner. The planner is in aios-planner.

## 3. Reference Documents

- Project design document, section 6.
- aios-core PRD. See `docs/modules/aios-core/PRD.md`.
- aios-models PRD. See `docs/modules/aios-models/PRD.md`.
- aios-planner PRD. See `docs/modules/aios-planner/PRD.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Role | A named configuration for an agent. |
| System prompt | The prompt that sets the agent behavior. |
| Tool set | The list of tools the agent can use. |
| Model class | The model capability level for the agent: cheap, mid, or strong. |


## 5. Determinism

The whole point of aios is to make complex coding and large systems deterministic. The determinism stance of this module is **Highly deterministic**.

The role registry is content addressed. The resolver is a pure function of the role and the request.

The module provides the following determinism guarantees.

- Roles are content addressed.

- Resolver decisions are pure functions of the registry and the request.
## 6. User Personas

The user persona is the planner or the operator
who wants to set the agent behavior. The user
defines the roles through the config.

## 7. User Stories

A planner wants to assign a role to a task. The
planner passes the role name to the role resolver.
The resolver returns the role configuration.

An operator wants to define a new role. The
operator adds the role to the config. The
operator reloads the config. The role is
available.

## 8. Functional Requirements Summary

The aios-roles module provides the following
functions.

- The role registry.
- The role configuration.
- The role resolver.
- The default roles.

## 9. Non-Functional Requirements Summary

- The module runs on Go 1.23.
- The module depends on aios-core and aios-models.
- The module has 100 percent test coverage for the
  public functions.

## 10. Success Criteria

- The role registry stores the roles.
- The role configuration is loaded from a YAML
  file.
- The role resolver returns the configuration.
- The default roles cover the common cases.
- The behavioural test suite has a 100 percent pass rate.

## 11. Out of Scope

- The user interface for the roles.
- The remote access to the roles.
- The role inheritance.
