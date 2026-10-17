# aios-ask — Software Requirements Specification

## 1. Purpose

This document states the software requirements for the
aios-ask module.

## 2. Scope

The scope of this document is the aios-ask module.
The module is a library.

## 3. Reference Documents

- aios-ask PRD. See `docs/modules/aios-ask/PRD.md`.
- aios-core SRS. See `docs/modules/aios-core/SRS.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Question | A typed object that describes the ask request. |
| Answer | A typed object that describes the answer. |

## 5. Functional Requirements

**REQ-ASK-FUN-001** The module shall provide the
`Question` schema. The schema shall validate the
prompt, the options, and the timeout.

**REQ-ASK-FUN-002** The module shall provide the
`Answer` schema. The schema shall validate the
selected option and the timestamp.

**REQ-ASK-FUN-003** The module shall provide the
`AskUserTool` class. The type shall implement the
`Tool` protocol.

**REQ-ASK-FUN-004** The tool shall accept a
`question` argument. The argument is a `Question`.

**REQ-ASK-FUN-005** The tool shall display the
question through the prompt callback. The prompt
callback is configured at construction.

**REQ-ASK-FUN-006** The tool shall return the
answer. The answer is an `Answer`.

**REQ-ASK-FUN-007** The tool shall support a
timeout. The tool shall return a default answer
when the timeout expires.

## 6. Non-Functional Requirements

**REQ-ASK-NFR-001** The module shall run on Go 1.23
3.12.

**REQ-ASK-NFR-002** The module shall have 100 percent
coverage for the public functions.

## 7. Interface Requirements

**REQ-ASK-IFC-001** The module shall export the
following items from `aios.ask`.

- `Question`
- `Answer`
- `AskUserTool`
- `AskOption`

## 8. Data Requirements

**REQ-ASK-DAT-001** The `Question` schema shall
include the fields below.

- `id` (string, unique)
- `prompt` (string)
- `options` (list of AskOption)
- `timeout_seconds` (int, default 0 for no timeout)
- `default_option` (string, optional)

**REQ-ASK-DAT-002** The `AskOption` schema shall
include the fields below.

- `id` (string, unique within the question)
- `label` (string)
- `description` (string, optional)

## 9. Error Handling

**REQ-ASK-ERR-001** The module shall raise
`AskTimeout` when the timeout expires without an
answer.

## 10. Acceptance Criteria

1. Every REQ in this document has a test case.
2. The test suite has a 100 percent pass rate.
3. The test suite has 100 percent coverage for the
   public functions.
4. The tool displays the question.
5. The tool returns the answer.
6. The tool returns the default on timeout.
