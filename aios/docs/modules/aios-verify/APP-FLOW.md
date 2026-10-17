# aios-verify — Application Flow

## 1. Purpose

This document describes the application flow for the
aios-verify module.

## 2. Scope

The scope of this document is the aios-verify module.
The module is a library.

## 3. Reference Documents

- aios-verify PRD. See
  `docs/modules/aios-verify/PRD.md`.
- aios-verify SRS. See
  `docs/modules/aios-verify/SRS.md`.
- aios-verify TRD. See
  `docs/modules/aios-verify/TRD.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Agent | The code that calls the verify tool. |

## 5. User Flow

The aios-verify module has no user-facing flow.

## 6. System Flow

### 6.1 Self-Check Flow

The self-check flow runs as follows.

1. The agent calls the self-check tool.
2. The tool builds the verification prompt.
3. The tool calls the model.
4. The model returns the verification result.
5. The tool parses the result.
6. The tool returns the result.

### 6.2 Test Run Flow

The test run flow runs as follows.

1. The agent calls the run_tests tool.
2. The tool determines the runner.
3. The tool runs the test command.
4. The tool parses the output.
5. The tool returns the test result.

## 7. Error Flows

### 7.1 Test Runner Error

The test runner error flow runs as follows.

1. The test command fails.
2. The tool raises `TestRunnerError`.
3. The agent catches the error.
4. The agent reports the error.

### 7.2 Verification Error

The verification error flow runs as follows.

1. The model call fails.
2. The tool raises `VerificationError`.
3. The agent catches the error.
4. The agent reports the error.

## 8. Sequence Diagrams

### 8.1 Self-Check Sequence

The self-check sequence has the steps below.

1. The agent calls the self-check tool.
2. The tool calls the model.
3. The model returns the verification result.
4. The tool returns the result.
