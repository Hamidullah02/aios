package cli

import (
	"os"

	"github.com/sadi/aios-system/internal/models"
)

// EnvModel is the environment variable that selects the model
// the planner uses.
//
// It is the escape hatch, not the mechanism: a workspace that has
// no model configured gets the default below, and the operator
// never has to know this variable exists. It is for the operator
// who wants a different model for one run.
//
// The value must be a model id the capability registry knows.
const EnvModel = "AIOS_MODEL"

// DefaultModel is the model the planner uses when the operator
// has not chosen one.
//
// It is the OpenRouter free variant of NVIDIA's Nemotron 3 Ultra.
// The reasons for that choice, in order:
//
//   - It is free. A first run of this product must not cost
//     money to try, and a task that fails because the operator
//     has no credit card is a task that never gets tried.
//   - It is good at the one thing the planner asks for: emitting
//     a JSON object and nothing else. The planner's prompt says
//     "respond with JSON only", and a model that adds a sentence
//     of commentary costs a repair round every time.
//
// The ":free" suffix is the part that matters. Without it the
// same model bills per token, and a retry loop against a model
// that keeps failing its verifier is exactly the case where a
// bill you did not expect is the worst outcome.
const DefaultModel = "nvidia/nemotron-3-ultra-550b-a55b:free"

// defaultOpenRouterModel is the capability of the default model.
//
// The context window is the model's own; the value is not used
// for anything today, but the registry requires it and a wrong
// number here would be a lie in the table.
var defaultOpenRouterModel = models.Capability{
	Model:         DefaultModel,
	Provider:      "openrouter",
	ContextWindow: 262144,
	SupportsTools: true,
}

// defaultOpenRouterPrice is the price of the default model.
//
// The free variant is free, so the numbers are zero. They are
// here rather than omitted because the router treats a model
// with no price as a configuration error and refuses the call:
// a cost of zero is a fact, a missing price is a mistake.
var defaultOpenRouterPrice = models.Price{
	Model:       DefaultModel,
	InputPer1K:  0,
	OutputPer1K: 0,
}

// defaultOpenRouterModels are the models the OpenRouter provider
// contributes when the workspace does not name them itself.
//
// There is more than one because the free tier's model list
// changes, and an operator whose preferred model disappears
// should still be able to run something.
var defaultOpenRouterModels = []models.Capability{
	defaultOpenRouterModel,
	{
		Model:         "nvidia/nemotron-3-super-120b-a12b:free",
		Provider:      "openrouter",
		ContextWindow: 262144,
		SupportsTools: true,
	},
}

// defaultOpenRouterPrices are the prices of the models above.
var defaultOpenRouterPrices = []models.Price{
	defaultOpenRouterPrice,
	{Model: "nvidia/nemotron-3-super-120b-a12b:free", InputPer1K: 0, OutputPer1K: 0},
}

// resolveModel returns the model id the planner should use.
//
// The precedence is most-explicit first:
//
//  1. The --model flag. The operator named the model for this
//     command; nothing overrides that.
//  2. The AIOS_MODEL environment variable. The operator set it
//     for this shell; it is more specific than a default but
//     less specific than the flag.
//  3. The built-in default.
//
// The default is a free model on purpose. See DefaultModel.
//
// The override argument is the --model flag value; it is empty
// when the operator did not pass one.
func resolveModel(override string) string {
	if override != "" {
		return override
	}
	if m := os.Getenv(EnvModel); m != "" {
		return m
	}
	return DefaultModel
}
