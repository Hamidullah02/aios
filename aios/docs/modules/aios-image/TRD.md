# aios-image — Technical Requirements Document

## 1. Purpose

This document states the technical approach for the
aios-image module.

## 2. Scope

The scope of this document is the aios-image module.
The module is a Go package.

## 3. Reference Documents

- aios-image PRD. See `docs/modules/aios-image/PRD.md`.
- aios-image SRS. See `docs/modules/aios-image/SRS.md`.
- aios-core TRD. See `docs/modules/aios-core/TRD.md`.
- aios-tools TRD. See `docs/modules/aios-tools/TRD.md`.
- aios-models TRD. See `docs/modules/aios-models/TRD.md`.

## 4. Definitions

| Term | Definition |
| --- | --- |
| image/jpeg, image/png, image/gif (stdlib) | The Go standard-library image decoders. |
| github.com/santhosh-tekuri/jsonschema/v5 | The Go JSON Schema validator (draft 2020-12). |
| MIME | The Multipurpose Internet Mail Extensions type. |

## 5. Architecture

The aios-image module has four components.

- `internal/image/read` — the image read tool.
- `internal/image/attach` — the image attach API.
- `internal/image/validator` — the structured output
  validator.
- `internal/image/constrain` — the schema-constrained
  generation.

The four components depend on aios-core, aios-tools,
and aios-models. The components depend on each
other in this order. The read tool has no
dependencies. The attach API depends on the read
tool. The validator has no dependencies. The
constrain function depends on the validator and
the model router.

## 6. Components

### 6.1 Read Tool

The read tool accepts a path. The tool opens the
file. The tool reads the bytes. The tool detects
the MIME type from the file extension. The tool
uses the standard `image` package to read the dimensions.

The tool enforces the maximum size. The tool
raises `ImageTooLargeError` when the file is
larger than 20 MB.

### 6.2 Attach API

The attach API converts the image to the
provider-specific format. The API supports the
Anthropic image block format and the OpenAI image
input format.

The API supports the `resize` method. The method
uses the standard `image` package to resize the image. The method preserves
the aspect ratio. The method uses the LANCZOS
resampling.

### 6.3 Validator

The validator uses the `github.com/santhosh-tekuri/github.com/santhosh-tekuri/jsonschema/v5/v5` library. The
validator supports JSON Schema (github.com/santhosh-tekuri/jsonschema/v5) 2020-12 via github.com/santhosh-tekuri/jsonschema/v5.

The validator supports structs. The
validator uses the struct's `Schema()`
method to generate the JSON Schema (github.com/santhosh-tekuri/jsonschema/v5). The validator
uses the generated schema to validate the
response.

### 6.4 Schema-Constrained Generation

The function accepts the model router, the prompt,
the model, and the schema. The function builds the
provider-specific request. The request includes the
schema as a tool or as a constraint.

The function calls the model router. The function
validates the response through the validator. The
function returns the validated object.

The function supports a retry loop. When the
validation fails, the function calls the model
again with the error message. The function retries
up to 3 times.

## 7. Data Structures

### 7.1 Image

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `path` | string | yes | The path of the image. |
| `MIME_type` | string | yes | The MIME type. |
| `data` | bytes | yes | The image data. |
| `width` | int | yes | The width in pixels. |
| `height` | int | yes | The height in pixels. |

## 8. Interfaces

### 8.1 ReadImageTool

```go
type ReadImageTool:
    """The image read tool."""

    func run(self, args: dict, context: ToolContext) -> ToolResult:
        """Run the image read tool.

        Args:
            args: The arguments of the call.
            context: The context for the call.

        Returns:
            The result of the call.
        """
```

### 8.2 ImageAttachment

```go
type ImageAttachment:
    """The image attachment."""

    def __init__(self, image: Image) -> nil:
        """Create the image attachment.

        Args:
            image: The image to attach.
        """

    def to_anthropic_block(self) -> dict:
        """Return the Anthropic image block."""

    def to_openai_block(self) -> dict:
        """Return the OpenAI image block."""
```

### 8.3 SchemaConstrainedCompletion

```go
func schema_constrained_completion(
    router: ModelRouter,
    prompt: str,
    model: str,
    schema: type[BaseModel] | dict,
    max_retries: int = 3,
) -> BaseModel | dict:
    """Call the model with a schema constraint.

    Args:
        router: The model router.
        prompt: The prompt.
        model: The model id.
        schema: The schema (struct or dict).
        max_retries: The maximum number of retries.

    Returns:
        The validated object.
    """
```

## 9. Algorithms

### 9.1 Schema-Constrained Completion Algorithm

The algorithm works as follows.

1. Convert the schema to the provider format.
2. Call the model router.
3. Validate the response through the validator.
4. When the validation fails, build the retry
   prompt with the error message.
5. Call the model router again.
6. Repeat up to the maximum retries.
7. Return the validated object.

### 9.2 Image Resize Algorithm

The resize algorithm works as follows.

1. Open the image with the standard `image` package.
2. Compute the new dimensions. The new dimension
   is the minimum of the maximum dimension and
   the current dimension.
3. Resize the image with the LANCZOS resampling.
4. Save the image to a bytes buffer.
5. Return the bytes.

## 10. Dependencies

### 10.1 Runtime Dependencies

- Go 1.23
- image/jpeg, image/png, image/gif (stdlib)
- github.com/santhosh-tekuri/jsonschema/v5
- aios-core
- aios-tools
- aios-models

### 10.2 Module Dependencies

The aios-image module depends on the following
internal modules.

- aios-core
- aios-tools
- aios-models

The aios-image module is depended on by the
following internal modules.

- aios-runtime
- aios-planner
- aios-cli

## 11. Deployment

The module compiles into the monorepo under `internal/<module>/`. The
package name is `github.com/<org>/aios/internal/image`.

## 12. Risks and Mitigations

| Risk | Impact | Probability | Mitigation |
| --- | --- | --- | --- |
| The image is too large. | High | Medium | The tool enforces a maximum size. The attach API supports resize. |
| The structured output fails validation. | High | Medium | The function retries on failure. The function returns the validation error. |
| The provider changes the image format. | Medium | Low | The attach API supports the provider format. The attach API is versioned. |
