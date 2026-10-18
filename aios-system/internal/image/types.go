// Package image is the multimodal input and structured output
// layer. The package exports ReadImageTool, ImageAttachment,
// StructuredOutputValidator, and SchemaConstrainedCompletion.
//
// ReadImageTool reads an image file from disk, decodes the
// dimensions, and returns the image data.
//
// ImageAttachment converts an Image into the provider-specific
// block for Anthropic or OpenAI; it also resizes the image to a
// maximum dimension.
//
// StructuredOutputValidator validates a JSON response against
// a JSON Schema. The validator is a small, hand-rolled
// implementation of the JSON Schema 2020-12 subset the
// project needs (type, properties, required, items, enum,
// minimum, maximum).
//
// SchemaConstrainedCompletion calls a model through the router
// and validates the response through the validator; on
// validation failure it retries with the error message.
package image

// Image is the decoded image (REQ-IMG-DAT-001).
type Image struct {
	Path     string `json:"path"`
	MIMEType string `json:"mime_type"`
	Data     []byte `json:"data"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
}
