// Package optimizer is the model router. The router picks
// the cheapest model that the verify layer will accept for
// a call fingerprint. The router is a pure function of
// the cost/quality table, the override, the budget, and
// the fingerprint. The router has no model dependency.
//
// The package exports the following items:
//
//	Router                    — the routing decision engine
//	ClassBasedModelRegistry   — class -> model mapping
//	CostQualityTable          — empirical pass rate per fingerprint
//	Fingerprint               — SHA-256 of (prompt, tools, role, bucket)
//	RoutingDecision           — the decision
//	Override                  — the per-call override
//	Class                     — a model class
//	NewFingerprint            — compute a fingerprint
package optimizer

// Class is a model class (REQ-OPT-DAT-001). The class
// holds the list cost, the model id, and the context
// window. The class is the unit the router selects.
type Class struct {
	Name            string  `json:"name"             yaml:"name"`
	ModelID         string  `json:"model_id"         yaml:"model_id"`
	CostPer1kInput  float64 `json:"cost_per_1k_input" yaml:"cost_per_1k_input"`
	CostPer1kOutput float64 `json:"cost_per_1k_output" yaml:"cost_per_1k_output"`
	ContextWindow   int     `json:"context_window"   yaml:"context_window"`
}

// Override is the per-call override (REQ-OPT-RTE-005/006).
type Override string

const (
	// OverrideAuto is the empty override. The router
	// uses the cost/quality table.
	OverrideAuto Override = ""
	// OverrideStrong forces the strong class.
	OverrideStrong Override = "strong"
	// OverrideMid forces the mid class.
	OverrideMid Override = "mid"
	// OverrideCheap forces the cheap class.
	OverrideCheap Override = "cheap"
	// OverrideOff disables the optimizer and returns
	// strong for every call.
	OverrideOff Override = "off"
)

// Source is the source of the routing decision
// (REQ-OPT-RTE-004).
type Source string

const (
	SourceOverride     Source = "override"
	SourceTable        Source = "table"
	SourceDefault      Source = "default"
	SourceBudget       Source = "budget"
	SourceOptimizerOff Source = "optimizer_off"
)

// RoutingDecision is the result of a Route call
// (REQ-OPT-DAT-003).
type RoutingDecision struct {
	Class   string `json:"class"`
	ModelID string `json:"model_id"`
	Reason  string `json:"reason"`
	Source  Source `json:"source"`
}

// CostQualityRow is one row in the cost/quality table
// (REQ-OPT-DAT-005). The row is keyed by
// (fingerprint, class) and holds the empirical pass
// rate and the average empirical cost.
type CostQualityRow struct {
	CallCount    int     `json:"call_count"`
	PassCount    int     `json:"pass_count"`
	TotalCostUSD float64 `json:"total_cost_usd"`
	FirstSeen    string  `json:"first_seen"`
	LastSeen     string  `json:"last_seen"`
}

// PassRate returns the empirical pass rate (REQ-OPT-EPR-002).
// The function returns 0.0 when CallCount is 0.
func (r CostQualityRow) PassRate() float64 {
	if r.CallCount == 0 {
		return 0.0
	}
	return float64(r.PassCount) / float64(r.CallCount)
}

// AverageCost returns the empirical average cost per call.
// The function returns 0.0 when CallCount is 0.
func (r CostQualityRow) AverageCost() float64 {
	if r.CallCount == 0 {
		return 0.0
	}
	return r.TotalCostUSD / float64(r.CallCount)
}
