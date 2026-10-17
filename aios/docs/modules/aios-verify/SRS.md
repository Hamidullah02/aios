# aios-verify — Software Requirements Specification

## 1. Purpose

This document states the software requirements for the
aios-verify module.

## 2. Scope

The scope of this document is the aios-verify module.
The module is a library.

## 3. Reference Documents

- aios-verify PRD. See
  `docs/modules/aios-verify/PRD.md`.
- aios-core SRS. See `docs/modules/aios-core/SRS.md`.
- aios-tools SRS. See `docs/modules/aios-tools/SRS.md`.
- aios-models SRS. See `docs/modules/aios-models/SRS.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Verification result | A typed object that holds the verification outcome. |
| Test result | A typed object that holds the test suite outcome. |

## 5. Functional Requirements

### 5.1 Self-Check Tool

**REQ-VRF-SCK-001** The module shall provide the
`SelfCheckTool` class. The type shall implement the
`Tool` protocol.

**REQ-VRF-SCK-002** The tool shall accept a `result`
argument. The result is the work the agent did.

**REQ-VRF-SCK-003** The tool shall build the
verification prompt. The prompt asks the model to
check the result against the goal.

**REQ-VRF-SCK-004** The tool shall call the model
with the verification prompt. The tool shall parse
the response.

**REQ-VRF-SCK-005** The tool shall return the
verification result. The result includes the
`status` (pass or fail), the `reason`, and the
`suggestions`.

**REQ-VRF-SCK-006** The tool shall write an audit
row. The row type shall be `self_check`. The row
shall include the result, the status, and the
reason.

### 5.2 run_tests Tool

**REQ-VRF-RNT-001** The module shall provide the
`RunTestsTool` class. The type shall implement the
`Tool` protocol.

**REQ-VRF-RNT-002** The tool shall accept a
`test_command` argument. The default is the command
in `AGENTS.md`.

**REQ-VRF-RNT-003** The tool shall run the test
command through the os/exec wrapper. The tool
shall parse the output.

**REQ-VRF-RNT-004** The tool shall support the
following test runners.

- go test
- jest
- mocha
- cargo test
- go test
- npm test

**REQ-VRF-RNT-005** The tool shall return the test
result. The result includes the `passed`, the
`failed`, the `skipped`, the `duration`, and the
`failures` (list of failure messages).

**REQ-VRF-RNT-006** The tool shall write an audit
row. The row type shall be `run_tests`. The row
shall include the test command, the result, and
the failures.

### 5.3 Verification Prompt

**REQ-VRF-PRM-001** The module shall provide the
`VerificationPrompt` class. The type shall accept
the goal and the result.

**REQ-VRF-PRM-002** The prompt shall include the
goal. The prompt shall include the result. The
prompt shall include the verification
instructions.

**REQ-VRF-PRM-003** The prompt shall be a text/template (stdlib)
template. The template is loaded from
`./var/verify_prompt.txt`.

### 5.4 Verification Result

**REQ-VRF-RES-001** The module shall provide the
`VerificationResult` schema. The schema shall
validate the `status`, the `reason`, and the
`suggestions`.

**REQ-VRF-RES-002** The status shall be `pass`,
`fail`, or `inconclusive`.

**REQ-VRF-RES-003** The reason shall be a string.
The suggestions shall be a list of strings.

## 6. Non-Functional Requirements

**REQ-VRF-NFR-001** The module shall run on Go 1.23
3.12.

**REQ-VRF-NFR-002** The module shall have 100 percent
coverage for the public functions.

**REQ-VRF-NFR-003** The self-check tool shall have
a maximum latency of 10 seconds.

## 7. Interface Requirements

**REQ-VRF-IFC-001** The module shall export the
following items from `aios.verify`.

- `SelfCheckTool`
- `RunTestsTool`
- `VerificationPrompt`
- `VerificationResult`
- `TestResult`
- `TestFailure`

## 8. Data Requirements

**REQ-VRF-DAT-001** The `TestResult` schema shall
include the fields below.

- `passed` (int)
- `failed` (int)
- `skipped` (int)
- `duration_ms` (int)
- `failures` (list of TestFailure)

**REQ-VRF-DAT-002** The `TestFailure` schema
shall include the fields below.

- `name` (string)
- `message` (string)
- `traceback` (string, optional)

**REQ-VRF-DAT-003** The default verification
prompt shall be:

```
The agent did the following work:
{result}

The goal was:
{goal}

Check whether the work satisfies the goal.
- If yes, return status="pass" and a short reason.
- If no, return status="fail", a short reason, and
  a list of suggestions.
- If you cannot decide, return status="inconclusive"
  and explain why.
```

## 9. Error Handling

**REQ-VRF-ERR-001** The module shall raise
`TestRunnerError` when the test runner fails.

**REQ-VRF-ERR-002** The module shall raise
`VerificationError` when the model call fails.

## 10. Acceptance Criteria

1. Every REQ in this document has a test case.
2. The test suite has a 100 percent pass rate.
3. The test suite has 100 percent coverage for the
   public functions.
4. The self-check tool verifies the result.
5. The run_tests tool runs the test suite.
6. The verification prompt builds the prompt.
7. The verification result is typed.
