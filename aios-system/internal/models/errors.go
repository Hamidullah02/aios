// Package models is the model adapter layer for aios.
//
// The package exports four things:
//
//   - Adapter: a typed component that calls one model provider.
//   - Router: a typed component that picks the right adapter for
//     a model id and exposes Complete/Stream.
//   - PriceTable: a typed component that computes the USD cost
//     of a call.
//   - Registry: a typed component that tracks model capabilities.
//
// Plus two concrete adapters: OpenAIAdapter and AnthropicAdapter.
// Both use net/http (stdlib) and accept a custom http.Client
// for testability. The default client reads the standard
// environment variables (OPENAI_API_KEY, ANTHROPIC_API_KEY).
package models

import "errors"

// Sentinel errors. Per aios-models SRS §9.
var (
	// ErrUnknownModel is returned when a model id is not in the
	// capability registry (SRS §9.1).
	ErrUnknownModel = errors.New("models: unknown model id")

	// ErrPriceNotFound is returned when a model id is not in the
	// price table (SRS §9.2).
	ErrPriceNotFound = errors.New("models: price not found for model id")

	// ErrProvider is returned by an adapter when the provider
	// returns an error. The wrap carries the provider code and
	// the message; use errors.As to recover the ProviderError
	// type (SRS §9.3).
	ErrProvider = errors.New("models: provider error")
)

// ProviderError is the typed error returned by an adapter when
// the upstream provider rejects the call. The Code is the
// provider's status code or symbolic error name.
type ProviderError struct {
	Provider string
	Code     string
	Message  string
}

func (e *ProviderError) Error() string {
	return e.Provider + ": " + e.Code + ": " + e.Message
}

func (e *ProviderError) Is(target error) bool { return target == ErrProvider }
func (e *ProviderError) Unwrap() error        { return ErrProvider }
