package models

import "context"

// Adapter is the protocol every model provider implements
// (REQ-MOD-ADP-001 / REQ-MOD-ADP-002).
//
// Complete returns the model's text completion and the token
// usage. Stream returns a Stream iterator of token chunks; the
// caller consumes chunks until io.EOF.
type Adapter interface {
	// Provider is the name of the upstream service ("openai",
	// "anthropic", ...). Used for error messages and routing.
	Provider() string

	// Complete runs a single non-streaming call. The returned
	// usage is what the provider reports; the router multiplies
	// it through the price table to get the USD cost.
	Complete(ctx context.Context, req Request) (Response, error)

	// Stream opens a streaming call. The returned Stream yields
	// chunks one at a time. Close releases the underlying HTTP
	// body. The final chunk has Usage != nil.
	Stream(ctx context.Context, req Request) (Stream, error)
}

// Request is the call-site input. Model is the model id; Prompt
// is the user text; Options carry temperature/max_tokens/system.
type Request struct {
	Model   string
	Prompt  string
	Options Options
}

// Response is the result of a Complete call.
type Response struct {
	Text  string
	Usage Usage
}

// Stream is the streaming counterpart of Response. The caller
// loops on Next() until io.EOF, accumulating the deltas into
// the final text. After io.EOF, Usage() returns the final
// token accounting.
type Stream interface {
	// Next returns the next chunk. io.EOF when the stream is
	// exhausted; the chunk returned alongside io.EOF carries the
	// final Usage.
	Next() (Chunk, error)
	// Usage returns the final usage; valid only after Next has
	// returned io.EOF.
	Usage() Usage
	// Close releases the underlying resources.
	Close() error
}

// Usage is the token accounting for one call.
type Usage struct {
	Model        string
	InputTokens  int
	OutputTokens int
	// CostUSD is filled in by the router after the call returns.
	// Adapters leave this zero; the router looks up the price
	// and writes the computed value.
	CostUSD float64
}

// Chunk is one streaming chunk. Most chunks have only Delta
// set. The final chunk (returned alongside io.EOF) has Usage
// set.
type Chunk struct {
	Delta string // the new text in this chunk
	// Usage is non-nil only on the final chunk.
	Usage *Usage
}

// Options are the call-site knobs. Zero values are the defaults:
// temperature=0 (per aios-models PRD §5 "Bounded by
// temperature=0"), max_tokens=0 means "use the model's default".
type Options struct {
	Temperature float64
	MaxTokens   int
	System      string
}
