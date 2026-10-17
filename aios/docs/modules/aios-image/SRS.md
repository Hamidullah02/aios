# aios-image — Software Requirements Specification

## 1. Purpose

This document states the software requirements for the
aios-image module.

## 2. Scope

The scope of this document is the aios-image module.
The module is a library.

## 3. Reference Documents

- aios-image PRD. See `docs/modules/aios-image/PRD.md`.
- aios-core SRS. See `docs/modules/aios-core/SRS.md`.
- aios-tools SRS. See `docs/modules/aios-tools/SRS.md`.
- aios-models SRS. See `docs/modules/aios-models/SRS.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Image | A typed object that holds the image data. |
| Image block | A typed object that holds the model-side image. |
| Schema | A JSON Schema (github.com/santhosh-tekuri/jsonschema/v5) that describes the output shape. |

## 5. Functional Requirements

### 5.1 Image Read Tool

**REQ-IMG-RDT-001** The module shall provide the
`ReadImageTool` class. The type shall implement the
`Tool` protocol.

**REQ-IMG-RDT-002** The tool shall accept a `path`
argument. The tool shall return the image data.

**REQ-IMG-RDT-003** The tool shall support the
following formats.

- PNG
- JPEG
- GIF
- WebP

**REQ-IMG-RDT-004** The tool shall enforce a
maximum size of 20 MB. The tool shall raise
`ImageTooLargeError` when the file is too large.

**REQ-IMG-RDT-005** The tool shall write an audit
row. The row type shall be `image_read`.

### 5.2 Image Attach API

**REQ-IMG-ATT-001** The module shall provide the
`ImageAttachment` class. The type shall accept the
image data and the mime type.

**REQ-IMG-ATT-002** The type shall support the
`to_anthropic_block` method. The method shall
return the image block in the Anthropic format.

**REQ-IMG-ATT-003** The type shall support the
`to_openai_block` method. The method shall return
the image block in the OpenAI format.

**REQ-IMG-ATT-004** The type shall support a
`resize` method. The method shall resize the
image to the maximum dimension.

### 5.3 Structured Output Validator

**REQ-IMG-SOV-001** The module shall provide the
`StructuredOutputValidator` class. The class
shall accept the schema.

**REQ-IMG-SOV-002** The validator shall provide
the `validate` method. The method shall accept
the response. The method shall return the
validated object or raise an error.

**REQ-IMG-SOV-003** The validator shall use the
`github.com/santhosh-tekuri/github.com/santhosh-tekuri/jsonschema/v5/v5` library. The validator shall support
JSON Schema (github.com/santhosh-tekuri/jsonschema/v5) 2020-12 via github.com/santhosh-tekuri/jsonschema/v5.

**REQ-IMG-SOV-004** The validator shall support
structs. The validator shall convert the
struct to a JSON Schema (github.com/santhosh-tekuri/jsonschema/v5) and validate the
response.

### 5.4 Schema-Constrained Generation

**REQ-IMG-SCG-001** The module shall provide the
`SchemaConstrainedCompletion` function. The
function shall accept the model router, the
prompt, the model, and the schema.

**REQ-IMG-SCG-002** The function shall call the
model with the schema as a constraint. The
function shall use the provider-specific
constraint format.

**REQ-IMG-SCG-003** The function shall support
the Anthropic tool-use format. The function shall
pass the schema as a tool.

**REQ-IMG-SCG-004** The function shall support
the OpenAI strict mode. The function shall set
`strict: true` on the tool.

**REQ-IMG-SCG-005** The function shall validate
the response through the validator.

## 6. Non-Functional Requirements

**REQ-IMG-NFR-001** The module shall run on Go 1.23
3.12.

**REQ-IMG-NFR-002** The module shall have 100 percent
coverage for the public functions.

**REQ-IMG-NFR-003** The image read tool shall have
a maximum latency of 500 milliseconds for a 5 MB
image.

## 7. Interface Requirements

**REQ-IMG-IFC-001** The module shall export the
following items from `aios.image`.

- `ReadImageTool`
- `ImageAttachment`
- `StructuredOutputValidator`
- `SchemaConstrainedCompletion`
- `Image`
- `ImageTooLargeError`

## 8. Data Requirements

**REQ-IMG-DAT-001** The `Image` schema shall
include the fields below.

- `path` (string)
- `mime_type` (string)
- `data` (bytes)
- `width` (int)
- `height` (int)

## 9. Error Handling

**REQ-IMG-ERR-001** The module shall raise
`ImageTooLargeError` when the image is too
large.

**REQ-IMG-ERR-002** The module shall raise
`UnsupportedFormatError` when the image format
is not supported.

**REQ-IMG-ERR-003** The module shall raise
`ValidationFailure` when the structured output
fails validation.

## 10. Acceptance Criteria

1. Every REQ in this document has a test case.
2. The test suite has a 100 percent pass rate.
3. The test suite has 100 percent coverage for the
   public functions.
4. The image read tool returns the image data.
5. The image attach API returns the provider
   block.
6. The structured output validator validates the
   response.
7. The schema-constrained generation works.
