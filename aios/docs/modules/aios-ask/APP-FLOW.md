# aios-ask — Application Flow

## 1. Purpose

This document describes the application flow for the
aios-ask module.

## 2. Scope

The scope of this document is the aios-ask module.
The module is a library.

## 3. Reference Documents

- aios-ask PRD. See `docs/modules/aios-ask/PRD.md`.
- aios-ask SRS. See `docs/modules/aios-ask/SRS.md`.
- aios-ask TRD. See `docs/modules/aios-ask/TRD.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Agent | The code that calls the tool. |
| Operator | The user that answers the question. |

## 5. User Flow

The operator flow runs as follows.

1. The agent calls the ask-user tool.
2. The CLI shows the question to the operator.
3. The operator types the number of the option.
4. The CLI returns the answer to the tool.
5. The tool returns the answer to the agent.

## 6. System Flow

### 6.1 Ask Flow

The ask flow runs as follows.

1. The agent calls `ask.run` with the arguments.
2. The tool validates the arguments.
3. The tool builds the `Question`.
4. The tool calls the prompt callback.
5. The callback shows the question.
6. The operator answers the question.
7. The callback returns the answer.
8. The tool returns the result.

### 6.2 Timeout Flow

The timeout flow runs as follows.

1. The operator does not answer within the timeout.
2. The tool returns the default answer.
3. The tool returns the result.

## 7. Error Flows

### 7.1 Ask Timeout

The ask timeout flow runs as follows.

1. The timeout expires.
2. The tool raises `AskTimeout`.
3. The caller catches the error.
4. The caller uses the default answer.

## 8. Sequence Diagrams

### 8.1 Ask Sequence

The ask sequence has the steps below.

1. The agent calls `ask.run`.
2. The tool builds the question.
3. The tool calls the callback.
4. The callback shows the question.
5. The operator answers the question.
6. The callback returns the answer.
7. The tool returns the answer.
