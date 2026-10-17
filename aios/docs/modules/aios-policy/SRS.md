# aios-policy — Software Requirements Specification

## 1. Purpose

This document states the software requirements for the
aios-policy module.

## 2. Scope

The scope of this document is the aios-policy module.
The module is a library. The module exports the policy
gateway and the policy types.

## 3. Reference Documents

- aios-policy PRD. See `docs/modules/aios-policy/PRD.md`.
- aios-core SRS. See `docs/modules/aios-core/SRS.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Rule | A typed object with match conditions and an action. |
| Condition | A predicate that selects a subset of tool calls. |
| Decision | A typed object that holds the action and the reason. |

## 5. Functional Requirements

**REQ-POL-FUN-001** The module shall provide the
`Rule` schema. The schema shall validate the match
conditions and the action.

**REQ-POL-FUN-002** The module shall provide the
`Decision` schema. The schema shall validate the
action, the reason, the matched rule id, and the
approval payload.

**REQ-POL-FUN-003** The module shall provide the
`PolicyGateway` class. The type shall accept a list
of rules.

**REQ-POL-FUN-004** The gateway shall return the
decision for a tool call. The method shall accept the
tool name, the tool arguments, the actor, and the
task id.

**REQ-POL-FUN-005** The gateway shall match the rules
in the order the rules are defined. The gateway shall
return the decision of the first matching rule.

**REQ-POL-FUN-006** When no rule matches, the gateway
shall return the default decision. The default action
is `deny`.

**REQ-POL-FUN-007** The gateway shall support the
following match conditions.

- `tool` (string, exact match)
- `tool_glob` (string, glob match)
- `path_prefix` (string, prefix match)
- `path_glob` (string, glob match)
- `host_glob` (string, glob match)
- `tier` (integer, exact match)
- `actor` (string, exact match)
- `task` (string, exact match)

**REQ-POL-FUN-008** The gateway shall support the
following actions.

- `allow`
- `deny`
- `ask`

**REQ-POL-FUN-009** When the action is `ask`, the
decision shall include the approval payload. The
payload shall include the action, the resource, the
reason, and the bound_to fields.

**REQ-POL-FUN-010** The gateway shall write an audit
row for every decision. The row type shall be
`policy_decision`. The row shall include the tool
name, the actor, the task id, the decision action,
and the matched rule id.

## 6. Non-Functional Requirements

**REQ-POL-NFR-001** The module shall run on Go 1.23
3.12.

**REQ-POL-NFR-002** The module shall depend only on
aios-core and the Go standard library.

**REQ-POL-NFR-003** The gateway decision shall have a
maximum latency of 1 millisecond.

**REQ-POL-NFR-004** The module shall have 100 percent
coverage for the public functions.

## 7. Interface Requirements

**REQ-POL-IFC-001** The module shall export the
following items from `aios.policy`.

- `Rule`
- `Decision`
- `PolicyGateway`

## 8. Data Requirements

**REQ-POL-DAT-001** The policy file shall be a YAML
file. The default path is `./var/policy.yaml`. The
path is configurable.

**REQ-POL-DAT-002** The policy file shall have the
following structure.

```yaml
rules:
  - id: rule-1
    match:
      tool: read_file
    action: allow
  - id: rule-2
    match:
      tool_glob: "write_*"
    action: ask
    approval:
      action: write
      resource: "{path}"
      reason: "Write to a project file"
      bound_to:
        task: "{task}"
        tool: "{tool}"
  - id: rule-3
    match:
      path_prefix: "/etc/"
    action: deny
```

## 9. Error Handling

**REQ-POL-ERR-001** The module shall raise a
`PolicyParseError` when the policy file is invalid.
The error shall include the file path and the line
number.

**REQ-POL-ERR-002** The module shall raise a
`RuleValidationError` when a rule is invalid. The
error shall include the rule id and the failure
reason.

## 10. Acceptance Criteria

The aios-policy module is accepted when the following
conditions are met.

1. Every REQ in this document has a test case.
2. The test suite has a 100 percent pass rate.
3. The test suite has 100 percent coverage for the
   public functions.
4. The gateway returns the correct decision for every
   test case.
5. The gateway writes the correct audit row.
6. The module is importable from aios-policy.
