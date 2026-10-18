package models

import (
	"context"
	"fmt"
	"sync"
)

// Router is the orchestrator for model calls
// (REQ-MOD-RTR-001 to 004).
//
// The router holds:
//   - a name -> adapter map (the actual HTTP clients)
//   - a capability registry (the model -> provider mapping)
//   - a price table (the cost computation)
//
// Route() is a one-time lookup that resolves a model id to its
// adapter. The router caches the lookup so repeated calls for
// the same model are constant time. Adapters themselves are
// stateless; the cache is the per-model adapter pointer.
type Router struct {
	adapters map[string]Adapter // provider -> adapter
	registry *Registry
	prices   *PriceTable
	mu       sync.RWMutex
	cache    map[string]Adapter // model -> adapter
}

// NewRouter returns a router. The adapters map is keyed by
// provider name ("openai", "anthropic"). The capabilities and
// price table are required.
func NewRouter(adapters map[string]Adapter, registry *Registry, prices *PriceTable) *Router {
	if adapters == nil {
		adapters = map[string]Adapter{}
	}
	return &Router{
		adapters: adapters,
		registry: registry,
		prices:   prices,
		cache:    make(map[string]Adapter),
	}
}

// AdapterFor resolves the model id to its adapter, using the
// capability registry to find the provider.
//
// ErrUnknownModel when the model is not in the registry.
// The returned error wraps the underlying ProviderError when
// the adapter is not registered for the provider.
func (r *Router) AdapterFor(model string) (Adapter, error) {
	// Fast path: cached. The read lock is taken once; the
	// returned adapter is safe to use after RUnlock because
	// adapters are stateless.
	r.mu.RLock()
	if a, ok := r.cache[model]; ok {
		r.mu.RUnlock()
		return a, nil
	}
	r.mu.RUnlock()

	// Slow path: resolve the provider, look up the adapter,
	// then populate the cache under a single write lock. The
	// write lock covers the whole slow-path lookup so two
	// concurrent goroutines cannot duplicate the work and
	// the cache always reflects the resolved adapter.
	r.mu.Lock()
	defer r.mu.Unlock()
	if a, ok := r.cache[model]; ok {
		return a, nil
	}
	cap, err := r.registry.Get(model)
	if err != nil {
		return nil, err
	}
	a, ok := r.adapters[cap.Provider]
	if !ok {
		return nil, fmt.Errorf("models: no adapter registered for provider %q (model %q)", cap.Provider, model)
	}
	r.cache[model] = a
	return a, nil
}

// Complete is the high-level call: route to the right adapter,
// call it, then attach the USD cost from the price table
// (REQ-MOD-RTR-003).
func (r *Router) Complete(ctx context.Context, req Request) (Response, error) {
	a, err := r.AdapterFor(req.Model)
	if err != nil {
		return Response{}, err
	}
	resp, err := a.Complete(ctx, req)
	if err != nil {
		return Response{}, err
	}
	cost, err := r.prices.Cost(req.Model, resp.Usage.InputTokens, resp.Usage.OutputTokens)
	if err != nil {
		// A model with no price is a configuration error, not a
		// call failure. We return the response with zero cost
		// and the error so the caller can decide what to do
		// (typically: log and proceed).
		return resp, fmt.Errorf("models: cost: %w", err)
	}
	resp.Usage.CostUSD = cost
	resp.Usage.Model = req.Model
	return resp, nil
}

// Stream is the high-level streaming call. Returns the
// provider's stream directly; the cost is available via
// Stream.Usage() after the stream is exhausted.
func (r *Router) Stream(ctx context.Context, req Request) (Stream, error) {
	a, err := r.AdapterFor(req.Model)
	if err != nil {
		return nil, err
	}
	return a.Stream(ctx, req)
}
