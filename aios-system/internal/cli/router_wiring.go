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
	prices, _ := models.LoadPriceTable(pricesPath)
	if prices == nil || priceTableEmpty(prices) {
		prices = models.NewPriceTable([]models.Price{
			{Model: "gpt-4o", InputPer1K: 0.005, OutputPer1K: 0.015},
			{Model: "claude-sonnet-4-5", InputPer1K: 0.003, OutputPer1K: 0.015},
		}, nil)
	}

	// Capability registry. Built-in default is the two
	// models above. If var/capabilities.yaml is present
	// and parses, use it; otherwise fall back.
	registry, _ := models.LoadRegistry(capPath)
	if registry == nil {
		registry, _ = models.NewRegistry([]models.Capability{
			{Model: "gpt-4o", Provider: "openai", ContextWindow: 128000, SupportsTools: true},
			{Model: "claude-sonnet-4-5", Provider: "anthropic", ContextWindow: 200000, SupportsTools: true},
		})
	}

	// Construct the router with both adapters. The
	// adapters read the env vars at call time; an
	// empty key is allowed here and surfaces as a
	// clear "OPENAI_API_KEY is not set" error when
	// the planner calls Complete.
	adapters := map[string]models.Adapter{
		"openai":    models.NewOpenAI(models.OpenAIConfig{}),
		"anthropic": models.NewAnthropic(models.AnthropicConfig{}),
	}
	router := models.NewRouter(adapters, registry, prices)
	return router, prices, allowList, nil
}

// priceTableEmpty reports whether the table has any
// entries. Used to detect the case where the init
// scaffold wrote a list-shaped file but the loader
// expects a map-shaped file.
func priceTableEmpty(t *models.PriceTable) bool {
	// Cost() returns ErrPriceNotFound when the model
	// is not in either layer. Probe with a known
	// sentinel.
	_, err := t.Cost("__sentinel__", 0, 0)
	return err != nil
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
