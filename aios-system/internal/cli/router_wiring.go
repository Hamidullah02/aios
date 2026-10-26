package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sadi/aios-system/internal/egress"
	"github.com/sadi/aios-system/internal/models"
)

// buildRouter constructs the production model router. The
// router is the seam between the planner and the model
// providers (OpenAI, Anthropic). The CLI ships adapters
// for both providers; the operator selects a model via
// the capability registry the init scaffold creates in
// var/egress.yaml (or var/capabilities.yaml if present).
//
// The function reads three files in order, falling back
// to safe defaults when a file is missing:
//
//  1. var/egress.yaml — egress allow list (deny by
//     default). The egress list is a future hook the
//     router will honour; the current implementation
//     does not enforce it but reads the file so the
//     scaffold's `aios init` layout is honored.
//
//  2. var/prices.yaml — per-model USD pricing. If the
//     file is missing or in the wrong format, the
//     function falls back to a built-in default for
//     gpt-4o and claude-sonnet-4-5. This is the
//     v0.31.0 review's fail-closed-but-recoverable
//     stance: the CLI does not refuse to start
//     because of a malformed scaffold, but the
//     fallback is the well-known public list.
//
//  3. var/capabilities.yaml — per-model provider
//     and context window. If the file is missing,
//     the function falls back to the built-in
//     default registry.
//
// The function returns the router, the price table
// (so the caller can compute model costs), the egress
// allow list (so the caller can present it in the
// audit log), and any error.
//
// API keys are NOT a prerequisite: the OpenAI and
// Anthropic adapters read OPENAI_API_KEY and
// ANTHROPIC_API_KEY at call time. A missing key
// produces a clear "OPENAI_API_KEY is not set" error
// when the planner calls Complete; the CLI does
// not pre-flight the key (that would couple the
// CLI to a network round-trip).
func buildRouter(workspace string) (*models.Router, *models.PriceTable, *egress.AllowList, error) {
	workspace, _ = filepath.Abs(workspace)
	egressPath := filepath.Join(workspace, "egress.yaml")
	pricesPath := filepath.Join(workspace, "prices.yaml")
	capPath := filepath.Join(workspace, "capabilities.yaml")

	// Egress allow list. The default is deny-by-default;
	// the loader returns an empty list for a missing file.
	allowList, _, err := egress.LoadFile(egressPath)
	if err != nil {
		return nil, nil, nil, &CliError{
			Kind:  "config",
			What:  fmt.Sprintf("egress load failed: %s", egressPath),
			Cause: err,
		}
	}

	// Price table. Built-in default is the v0.31.0 list:
	// gpt-4o and claude-sonnet-4-5 at the public OpenAI
	// and Anthropic prices. The loader is map-shaped; the
	// init scaffold historically wrote a list-shaped file.
	// If the loader returns an empty table, fall back to
	// the default.
	//
	// The OpenRouter models are ALWAYS added, not only on
	// the fallback path. A workspace with a prices.yaml
	// that names only gpt-4o would otherwise have no price
	// for the default model, and the router treats a model
	// with no price as a configuration error and refuses the
	// call: "price not found for model id". An operator
	// would have to edit a file to run the product.
	prices, _ := models.LoadPriceTable(pricesPath)
	if prices == nil || priceTableEmpty(prices) {
		prices = models.NewPriceTable([]models.Price{
			{Model: "gpt-4o", InputPer1K: 0.005, OutputPer1K: 0.015},
			{Model: "claude-sonnet-4-5", InputPer1K: 0.003, OutputPer1K: 0.015},
		}, nil)
	}
	prices = withDefaultPrices(prices, defaultOpenRouterPrices)

	// Capability registry. Built-in default is the two
	// models above. If var/capabilities.yaml is present
	// and parses, use it; otherwise fall back.
	//
	// The OpenRouter models are added here for the same
	// reason as the prices: the registry is what says
	// which adapter a model id routes to, and a model
	// that is not in it cannot be called at all.
	registry, _ := models.LoadRegistry(capPath)
	if registry == nil {
		registry, _ = models.NewRegistry([]models.Capability{
			{Model: "gpt-4o", Provider: "openai", ContextWindow: 128000, SupportsTools: true},
			{Model: "claude-sonnet-4-5", Provider: "anthropic", ContextWindow: 200000, SupportsTools: true},
		})
	}
	for _, cap := range defaultOpenRouterModels {
		// A capability the workspace already names wins: the
		// operator's file is more specific than ours.
		if _, err := registry.Get(cap.Model); err == nil {
			continue
		}
		_ = registry.Set(cap)
	}

	// Construct the router with the adapters. The
	// adapters read the env vars at call time; an
	// empty key is allowed here and surfaces as a
	// clear "OPENAI_API_KEY is not set" error when
	// the planner calls Complete.
	adapters := map[string]models.Adapter{
		"openai":     models.NewOpenAI(models.OpenAIConfig{}),
		"anthropic":  models.NewAnthropic(models.AnthropicConfig{}),
		"openrouter": models.NewOpenRouter(models.OpenRouterConfig{}),
	}
	router := models.NewRouter(adapters, registry, prices)
	return router, prices, allowList, nil
}

// withDefaultPrices adds the default prices to a table that
// does not already carry them.
//
// The defaults are added as BASE entries, not as overrides: an
// operator's workspace price for one of these models is more
// specific than ours, and it must win. The price table's Has
// method is what tells the two apart -- asking Cost would not,
// because a free model's zero cost and a missing model's zero
// cost are the same number.
//
// The table is returned unchanged when it already has every
// default, so the common case allocates nothing.
func withDefaultPrices(table *models.PriceTable, defaults []models.Price) *models.PriceTable {
	if table == nil {
		table = models.NewPriceTable(nil, nil)
	}
	missing := make([]models.Price, 0, len(defaults))
	for _, p := range defaults {
		if table.Has(p.Model) {
			continue
		}
		missing = append(missing, p)
	}
	if len(missing) == 0 {
		return table
	}
	return table.WithDefaults(missing)
}

// priceTableEmpty reports whether the table has any
// entries. Used to detect the case where the init
// scaffold wrote a list-shaped file but the loader
// expects a map-shaped file.
//
// The count is the table's own, not a probe call. A probe
// with a sentinel model asks whether the table prices
// THAT model, which is never true, so it reported every
// table as empty -- and the caller then replaced the
// operator's prices.yaml with the built-in list.
func priceTableEmpty(t *models.PriceTable) bool {
	if t == nil {
		return true
	}
	return t.Len() == 0
}

// ensureWorkspace is a small helper that returns the
// resolved workspace path or "./var" if the input is
// empty. Used by runFix and the other subcommands.
func ensureWorkspace(workspace string) string {
	if workspace == "" {
		// Default: the parent directory of the policy
		// file (matches the v0.32.0 wiring default).
		workspace = "./var"
	}
	if abs, err := filepath.Abs(workspace); err == nil {
		workspace = abs
	}
	return workspace
}

// envOrEmpty is a tiny indirection around os.Getenv
// so the planner wiring tests can stub the env.
var envOrEmpty = func(name string) string { return os.Getenv(name) }
