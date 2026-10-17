# aios-image — Application Flow

## 1. Purpose

This document describes the application flow for the
aios-image module.

## 2. Scope

The scope of this document is the aios-image module.
The module is a library.

## 3. Reference Documents

- aios-image PRD. See `docs/modules/aios-image/PRD.md`.
- aios-image SRS. See `docs/modules/aios-image/SRS.md`.
- aios-image TRD. See `docs/modules/aios-image/TRD.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Agent | The code that calls the image tool. |

## 5. User Flow

The aios-image module has no user-facing flow.

## 6. System Flow

### 6.1 Read Image Flow

The read image flow runs as follows.

1. The agent calls the read image tool.
2. The tool reads the file.
3. The tool detects the MIME type.
4. The tool returns the image data.

### 6.2 Attach Image Flow

The attach image flow runs as follows.

1. The agent creates an `ImageAttachment`.
2. The agent calls `to_anthropic_block` or
   `to_openai_block`.
3. The API returns the provider-specific block.
4. The agent passes the block to the model.

### 6.3 Schema-Constrained Completion Flow

The schema-constrained completion flow runs as
follows.

1. The agent calls the function with the schema.
2. The function builds the provider request.
3. The function calls the model.
4. The function validates the response.
5. When the validation fails, the function
   retries.
6. The function returns the validated object.

## 7. Error Flows

### 7.1 Image Too Large

The image too large flow runs as follows.

1. The image is larger than 20 MB.
2. The tool raises `ImageTooLargeError`.
3. The agent catches the error.
4. The agent resizes the image.

### 7.2 Validation Failure

The validation failure flow runs as follows.

1. The model returns an invalid response.
2. The validator raises `ValidationFailure`.
3. The function retries.
4. When the retries are exhausted, the function
   raises the error.

## 8. Sequence Diagrams

### 8.1 Schema-Constrained Completion Sequence

The schema-constrained completion sequence has the
steps below.

1. The agent calls the function.
2. The function calls the model.
3. The model returns the response.
4. The function validates the response.
5. The function returns the validated object.
