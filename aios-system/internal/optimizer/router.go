package optimizer

import (
	"fmt"
	"math"
	"sort"
)

// Router is the routing decision engine
// (REQ-OPT-RTE-001 to 009). The router is a pure
// function of the registry, the table, the override, and
// the budget. The router never calls the model and
// never calls the verify layer.
type Router struct {
	registry     *ClassBasedModelRegistry
	table        *CostQualityTable
	defaultClass string
	defaultRule  string
}

// NewRouter returns a router. The default class is "mid"
// and the default rule is "cost_first" (REQ-OPT-RTE-008).
func NewRouter(registry *ClassBasedModelRegistry, table *CostQualityTable) *Router {
	return &Router{
		registry:     registry,
		table:        table,
		defaultClass: "mid",
		defaultRule:  "cost_first",
	}
}

// SetDefaultClass changes the default class returned when
// the table is empty for a fingerprint.
func (r *Router) SetDefaultClass(name string) {
	r.defaultClass = name
}

// SetDefaultRule changes the default rule. The rule is
// "cost_first" (cheapest class) or "strong_first"
// (strongest class). Unknown rules fall back to
// "cost_first".
func (r *Router) SetDefaultRule(rule string) {
	switch rule {
	case "cost_first", "strong_first":
		r.defaultRule = rule
	default:
		r.defaultRule = "cost_first"
	}
}

// Route returns the routing decision for a call. The
// router is a pure function: two calls with the same
// fingerprint, the same override, the same budget, and
// the same table return the same decision.
func (r *Router) Route(fp Fingerprint, override Override, budgetUSD float64) (RoutingDecision, error) {
	if override == OverrideOff {
		return r.strongDecision("optimizer disabled", SourceOptimizerOff)
	}
	if override == OverrideStrong || override == OverrideMid || override == OverrideCheap {
		c, err := r.registry.Get(string(override))
		if err != nil {
			return RoutingDecision{}, err
		}
		return r.enforceBudget(RoutingDecision{
			Class:   c.Name,
			ModelID: c.ModelID,
			Reason:  "override: " + string(override),
			Source:  SourceOverride,
		}, budgetUSD)
	}
	if override != OverrideAuto {
		return RoutingDecision{}, fmt.Errorf("%w: %s", ErrUnknownClass, string(override))
	}
	rows := r.rowsForFingerprint(fp)
	if len(rows) >= 2 {
		return r.pickBestByScore(rows, budgetUSD)
	}
	if len(rows) == 1 {
		for class := range rows {
			c, err := r.registry.Get(class)
			if err != nil {
				return RoutingDecision{}, err
			}
			return r.enforceBudget(RoutingDecision{
				Class:   c.Name,
				ModelID: c.ModelID,
				Reason:  "table: only one class observed",
				Source:  SourceTable,
			}, budgetUSD)
		}
	}
	decision, err := r.defaultDecision()
	if err != nil {
		return RoutingDecision{}, err
	}
	return r.enforceBudget(decision, budgetUSD)
}

// rowsForFingerprint returns the rows for the
// fingerprint, keyed by class name. The function reads
// the table directly.
func (r *Router) rowsForFingerprint(fp Fingerprint) map[string]CostQualityRow {
	if r.table == nil {
		return nil
	}
	out := map[string]CostQualityRow{}
	for _, key := range r.table.AllKeys() {
		keyFp, class, err := ParseTableKey(key)
		if err != nil {
			continue
		}
		if keyFp != fp {
			continue
		}
		row, ok := r.table.Get(fp, class)
		if ok {
			out[class] = row
		}
	}
	return out
}

// pickBestByScore returns the class with the highest
// pass rate per unit cost (REQ-OPT-RTE-007).
func (r *Router) pickBestByScore(rows map[string]CostQualityRow, budgetUSD float64) (RoutingDecision, error) {
	type scored struct {
		class Class
		score float64
		cost  float64
		rate  float64
	}
	all := make([]scored, 0, len(rows))
	for class, row := range rows {
		c, err := r.registry.Get(class)
		if err != nil {
			return RoutingDecision{}, err
		}
		rate := row.PassRate()
		cost := row.AverageCost()
		if cost == 0 {
			cost = (c.CostPer1kInput + c.CostPer1kOutput) / 2.0
		}
		score := rate / cost
		all = append(all, scored{class: c, score: score, cost: cost, rate: rate})
	}
	sort.Slice(all, func(i, j int) bool {
		return all[i].score > all[j].score
	})
	best := all[0]
	return r.enforceBudget(RoutingDecision{
		Class:   best.class.Name,
		ModelID: best.class.ModelID,
		Reason: fmt.Sprintf(
			"table: best by score (rate=%.3f, cost=%.6f, score=%.3f)",
			best.rate, best.cost, best.score,
		),
		Source: SourceTable,
	}, budgetUSD)
}

// strongDecision returns the strong class with the
// given reason and source.
func (r *Router) strongDecision(reason string, source Source) (RoutingDecision, error) {
	c, err := r.registry.Get("strong")
	if err != nil {
		return RoutingDecision{}, err
	}
	return RoutingDecision{
		Class:   c.Name,
		ModelID: c.ModelID,
		Reason:  reason,
		Source:  source,
	}, nil
}

// defaultDecision returns the decision for the default
// rule (REQ-OPT-RTE-008).
func (r *Router) defaultDecision() (RoutingDecision, error) {
	if r.defaultRule == "strong_first" {
		return r.strongDecision("default rule: strong_first", SourceDefault)
	}
	names := r.registry.Classes()
	if len(names) == 0 {
		return RoutingDecision{}, fmt.Errorf("%w: registry is empty", ErrUnknownClass)
	}
	cheapest, err := r.cheapest(names)
	if err != nil {
		return RoutingDecision{}, err
	}
	return RoutingDecision{
		Class:   cheapest.Name,
		ModelID: cheapest.ModelID,
		Reason:  "default rule: cost_first",
		Source:  SourceDefault,
	}, nil
}

// cheapest returns the class with the lowest list cost
// (input + output average).
func (r *Router) cheapest(names []string) (Class, error) {
	if len(names) == 0 {
		return Class{}, fmt.Errorf("%w: no classes", ErrUnknownClass)
	}
	var best Class
	bestCost := math.Inf(1)
	for _, n := range names {
		c, err := r.registry.Get(n)
		if err != nil {
			return Class{}, err
		}
		cost := c.CostPer1kInput + c.CostPer1kOutput
		if cost < bestCost {
			bestCost = cost
			best = c
		}
	}
	if best.Name == "" {
		return Class{}, fmt.Errorf("%w: no classes", ErrUnknownClass)
	}
	return best, nil
}

// enforceBudget caps the decision to the budget. The
// function returns the cheapest class that fits when
// the decision's estimated cost exceeds the budget.
// When no class fits, the function returns
// ErrBudgetExhausted.
func (r *Router) enforceBudget(d RoutingDecision, budgetUSD float64) (RoutingDecision, error) {
	if budgetUSD <= 0 {
		return d, nil
	}
	estimate := r.estimateCost(d)
	if estimate <= budgetUSD {
		return d, nil
	}
	names := r.registry.Classes()
	sort.Slice(names, func(i, j int) bool {
		a, _ := r.registry.Get(names[i])
		b, _ := r.registry.Get(names[j])
		return a.CostPer1kInput+a.CostPer1kOutput < b.CostPer1kInput+b.CostPer1kOutput
	})
	for _, n := range names {
		c, err := r.registry.Get(n)
		if err != nil {
			return RoutingDecision{}, err
		}
		cost := (c.CostPer1kInput + c.CostPer1kOutput) / 2.0
		if cost <= budgetUSD {
			return RoutingDecision{
				Class:   c.Name,
				ModelID: c.ModelID,
				Reason:  fmt.Sprintf("budget: %v fits, %v does not", cost, estimate),
				Source:  SourceBudget,
			}, nil
		}
	}
	return RoutingDecision{}, ErrBudgetExhausted
}

// estimateCost returns the expected cost of the
// decision. The function uses the list cost per 1k
// averaged.
func (r *Router) estimateCost(d RoutingDecision) float64 {
	c, err := r.registry.Get(d.Class)
	if err != nil {
		return 0
	}
	return (c.CostPer1kInput + c.CostPer1kOutput) / 2.0
}
