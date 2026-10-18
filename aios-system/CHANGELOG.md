# Changelog

v0.30.0 — aios-image

Add the aios-image module. The module is the
image attachment layer: it decodes PNG / JPEG
bytes, validates the dimensions against the
model's cap, and renders the model-ready
blocks (Anthropic image block, OpenAI image
URL) for both providers.

The package is a single Go package at
`internal/image/` and depends on
`golang.org/x/image/draw` (already a direct
dep of the monorepo from v0.18.x). No new
third-party dependencies.

Components:

- `Image` — the typed image (bytes, format,
  width, height, color model). The image is
  immutable after `Decode`.
- `ImageAttachment` — the model-ready wrapper.
  Holds the decoded `Image` plus the
  resize-to-max-dimension policy.
- `ToAnthropicBlock` — the Anthropic image
  block (REQ-IMG-ATT-002). The block is a
  map the caller passes as a content block
  to the Anthropic messages API.
- `ToOpenAIURL` — the OpenAI image URL
  (REQ-IMG-ATT-003). The URL is a data-URL
  for the inline case and a https URL for
  the remote case.
- `Constrain` — the dimension-and-format
  validator (REQ-IMG-CON-001..005). The
  validator enforces a max width, max height,
  and a format allow-list.
- `Read` — the file-system reader. The
  reader decodes the bytes through
  `image.Decode` and validates the format
  through `Constrain`.
- `Validator` — the per-model validator
  (REQ-IMG-VAL-001..003). The validator
  reads the model's capability table and
  refuses images the model cannot handle.
- `AttachError`, `ConstrainError`,
  `DecodeError`, `ValidationError` — typed
  errors.

Substitutions vs the doc set:

- **Three sub-packages (`attach` +
  `constrain` + `read`) → single
  `internal/image/` Go package.** Same
  reasoning as `aios-roles` (