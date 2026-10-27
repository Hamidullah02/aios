package image

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/Hamidullah02/aios/aios-system/internal/models"
)

// ModelCompleter is the call shape SchemaConstrainedCompletion
// uses to talk to a model. *models.Router satisfies this
// interface. Tests substitute a fake.
type ModelCompleter interface {
	Complete(ctx context.Context, req models.Request) (models.Response, error)
}

// SchemaConstrainedCompletionOptions tunes the call.
type SchemaConstrainedCompletionOptions struct {
	// MaxRetries is the number of retries after a
	// validation failure. Default 3.
	MaxRetries int
	// System is the system prompt. Optional.
	System string
}

// SchemaConstrainedCompletion calls the model with the
// schema as a constraint (REQ-IMG-SCG-001 to 005) and
// validates the response through the validator. The
// function retries with the validation error message
// when the response fails to validate.
//
// The function is the entry point for schema-constrained
// generation. The model is told to return JSON that
// matches the schema; the response is parsed and
// validated; failures are retried up to MaxRetries
// times. The function returns the parsed JSON object.
func SchemaConstrainedCompletion(
	ctx context.Context,
	router ModelCompleter,
	prompt string,
	model string,
	schema any,
	opts SchemaConstrainedCompletionOptions,
) (map[string]any, error) {
	validator, err := NewStructuredOutputValidator(schema)
	if err != nil {
		return nil, err
	}
	maxRetries := opts.MaxRetries
	if maxRetries < 0 {
		maxRetries = 0
	}
	schemaJSON, err := json.Marshal(schema)
	if err != nil {
		return nil, fmt.Errorf("image: constrain: marshal schema: %w", err)
	}
	current := buildConstrainedPrompt(prompt, string(schemaJSON), opts.System, "")
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		resp, err := router.Complete(ctx, models.Request{
			Model:  model,
			Prompt: current,
		})
		if err != nil {
			return nil, err
		}
		parsed, perr := parseJSON(resp.Text)
		if perr != nil {
			current = buildConstrainedPrompt(prompt, string(schemaJSON), opts.System, perr.Error())
			continue
		}
		if vErr := validator.Validate(parsed); vErr != nil {
			current = buildConstrainedPrompt(prompt, string(schemaJSON), opts.System, vErr.Error())
			continue
		}
		if obj, ok := parsed.(map[string]any); ok {
			return obj, nil
		}
		// Model returned a non-object. Wrap it so the
		// next retry sees a clear error.
		current = buildConstrainedPrompt(prompt, string(schemaJSON), opts.System, "response is not a JSON object")
	}
	return nil, &ValidationFailure{Reason: "retries exhausted"}
}

// buildConstrainedPrompt composes the prompt for one
// call. When prevErr is non-empty, it is appended so the
// model can correct itself on the retry.
func buildConstrainedPrompt(prompt, schemaJSON, system, prevErr string) string {
	var b strings.Builder
	if system != "" {
		b.WriteString("System: ")
		b.WriteString(system)
		b.WriteString("\n\n")
	}
	b.WriteString(prompt)
	b.WriteString("\n\n")
	b.WriteString("Respond with JSON that matches this schema:\n")
	b.WriteString(schemaJSON)
	if prevErr != "" {
		b.WriteString("\n\nPrevious response was invalid: ")
		b.WriteString(prevErr)
	}
	return b.String()
}

// parseJSON parses the model text as JSON. The function
// returns the parsed value and an error when the text
// is not valid JSON. The error message is suitable for
// appending to a retry prompt.
func parseJSON(text string) (any, error) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return nil, errors.New("empty response")
	}
	// Strip ```json ... ``` fences when the model wraps
	// the JSON in a markdown code block.
	if strings.HasPrefix(trimmed, "```") {
		trimmed = stripCodeFence(trimmed)
	}
	var v any
	dec := json.NewDecoder(strings.NewReader(trimmed))
	dec.UseNumber()
	if err := dec.Decode(&v); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}
	return normalizeNumbers(v), nil
}

// stripCodeFence removes a leading ``` and a trailing
// ``` from a markdown code block.
func stripCodeFence(s string) string {
	s = strings.TrimPrefix(s, "```")
	if i := strings.Index(s, "\n"); i >= 0 {
		s = s[i+1:]
	}
	if j := strings.LastIndex(s, "```"); j >= 0 {
		s = s[:j]
	}
	return strings.TrimSpace(s)
}

// normalizeNumbers converts json.Number values to
// float64 so the validator's type checks see numbers
// uniformly. Maps and slices are walked recursively.
func normalizeNumbers(v any) any {
	switch x := v.(type) {
	case map[string]any:
		for k, val := range x {
			x[k] = normalizeNumbers(val)
		}
		return x
	case []any:
		for i, val := range x {
			x[i] = normalizeNumbers(val)
		}
		return x
	case json.Number:
		if f, err := x.Float64(); err == nil {
			return f
		}
		return string(x)
	}
	return v
}
