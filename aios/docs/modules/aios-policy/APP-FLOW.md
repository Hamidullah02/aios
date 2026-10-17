# aios-policy — Application Flow

## 1. Purpose

This document describes the application flow for the
aios-policy module.

## 2. Scope

The scope of this document is the aios-policy module.
The module is a library. The flows are programming
flows.

## 3. Reference Documents

- aios-policy PRD. See `docs/modules/aios-policy/PRD.md`.
- aios-policy SRS. See `docs/modules/aios-policy/SRS.md`.
- aios-policy TRD. See `docs/modules/aios-policy/TRD.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Caller | The code that calls the policy gateway. |
| Tool call | A request to invoke a tool with arguments. |

## 5. User Flow

The aios-policy module has no user-facing flow.

## 6. System Flow

### 6.1 Policy Load Flow

The policy load flow runs as follows.

1. The caller calls `load_policy` with the path.
2. The component reads the file.
3. The component parses the YAML.
4. The component validates the rules against the
   schema.
5. The component returns the list of rules.
6. The caller creates the gateway with the rules.

### 6.2 Policy Decision Flow

The policy decision flow runs as follows.

1. The caller calls `gateway.decide` with the tool
   name, the arguments, the actor, and the task id.
2. The component iterates over the rules in order.
3. For each rule, the component checks the match
   conditions.
4. When a rule matches, the component constructs the
   decision.
5. When the action is `ask`, the component
   substitutes the template values in the approval
   payload.
6. The component writes an audit row for the
   decision.
7. The component returns the decision.
8. When no rule matches, the component returns the
   default deny decision.

## 7. Error Flows

### 7.1 Policy Parse Error

The policy parse error flow runs as follows.

1. The caller calls `load_policy` with an invalid
   file.
2. The component raises a `PolicyParseError`.
3. The caller catches the error.
4. The caller reports the error.

### 7.2 Rule Validation Error

The rule validation error flow runs as follows.

1. The caller passes an invalid rule to the gateway.
2. The gateway raises a `RuleValidationError`.
3. The caller catches the error.
4. The caller reports the error.

## 8. Sequence Diagrams

### 8.1 Decision Sequence

The decision sequence has the steps below.

1. The orchestrator calls `gateway.decide`.
2. The gateway iterates over the rules.
3. A rule matches.
4. The gateway constructs the decision.
5. The gateway writes the audit row.
6. The gateway returns the decision.
7. The orchestrator processes the decision.
