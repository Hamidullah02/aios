# aios-image — Product Requirements Document

## 1. Purpose

This document states the product requirements for the
aios-image module. The module is the multimodal
input and structured output layer. The module
covers two areas: the image input tool and the
structured output validation.

## 2. Scope

The aios-image module is in scope for this document.

- The image read tool.
- The image attach API.
- The structured output validator.
- The schema-constrained generation.

The aios-image module is not in scope for this
document.

- The model adapter. The adapter is in aios-models.
- The file tools. The tools are in aios-tools.

## 3. Reference Documents

- Project design document, section 7.
- aios-core PRD. See `docs/modules/aios-core/PRD.md`.
- aios-tools PRD. See `docs/modules/aios-tools/PRD.md`.
- aios-models PRD. See `docs/modules/aios-models/PRD.md`.
- Claude Code image-paste feature reference.
- OpenAI vision input reference.
- Anthropic image blocks API reference.

## 4. Definitions

| Term | Definition |
| --- | --- |
| Image | A typed object that holds the image data. |
| Image block | A typed object that holds the image for the model. |
| Structured output | A typed object that holds the schema-constrained response. |
| JSON Schema (github.com/santhosh-tekuri/jsonschema/v5) | A specification for the shape of a JSON value. |

## 5. User Personas

The user persona is the agent that needs to read an
image. The agent wants to attach the image to the
next model call. The agent wants the model to
return a structured response.

## 6. User Stories

An agent wants to read an image. The agent calls
the image read tool. The tool returns the image
data. The agent attaches the image to the next
model call.

An agent wants a structured response. The agent
passes the schema to the model. The model returns
the response in the schema. The agent validates
the response.

## 7. Functional Requirements Summary

The aios-image module provides the following
functions.

- The image read tool.
- The image attach API.
- The structured output validator.
- The schema-constrained generation.

## 8. Non-Functional Requirements Summary

- The module runs on Go 1.23.
- The module depends on aios-core, aios-tools, and
  aios-models.
- The module has 100 percent test coverage for the
  public functions.

## 9. Success Criteria

- The image read tool returns the image data.
- The image attach API attaches the image to the
  model call.
- The structured output validator validates the
  response.
- The schema-constrained generation works.
- The test suite has a 100 percent pass rate.

## 10. Out of Scope

- The image generation.
- The image editing.
- The video input.
