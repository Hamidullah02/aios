package webui

import (
	"fmt"
	"strings"
)

// PanelID identifies a panel the operator can show, hide,
// move, or resize.
type PanelID string

const (
	// PanelAgents is the multi-agent grid: one card per
	// agent with its live status.
	PanelAgents PanelID = "agents"
	// PanelComms is the agent communication log.
	PanelComms PanelID = "comms"
	// PanelAudit is the audit-log tail.
	PanelAudit PanelID = "audit"
)

// AllPanels lists every panel the UI knows about, in the
// order they appear when an operator has saved no layout.
//
// This list is the authority for validation: a saved
// layout naming a panel not in here is rejected rather
// than silently ignored, so a typo surfaces immediately
// instead of producing a UI with a mysteriously missing
// panel.
var AllPanels = []PanelID{PanelAgents, PanelComms, PanelAudit}

// KnownPanel reports whether id names a real panel.
func KnownPanel(id PanelID) bool {
	for _, p := range AllPanels {
		if p == id {
			return true
		}
	}
	return false
}

// PanelState is one panel's placement and visibility.
type PanelState struct {
	// ID is the panel being placed.
	ID PanelID `json:"id"`
	// Visible hides the panel when false.
	Visible bool `json:"visible"`
	// Order is the sort key, ascending. The UI reads
	// Order rather than trusting array position, so a
	// client that appends a panel without renumbering
	// still renders predictably (the new panel sorts last
	// among equals by ID).
	Order int `json:"order"`
	// Width is a CSS grid column span, 1 to 12.
	//
	// It is stored as an int rather than free-form CSS so
	// a saved layout cannot inject arbitrary style. A
	// layout is operator input that ends up in a
	// stylesheet-adjacent position, and "12 columns of a
	// 12-column grid" covers every real arrangement.
	Width int `json:"width"`
}

// Layout is the operator's saved dashboard arrangement.
//
// The whole layout round-trips through JSON in the browser
// and on disk, so every field is exported and the shape is
// deliberately flat.
type Layout struct {
	// Panels is the full set, including hidden ones. A
	// panel is stored rather than omitted so "hidden" and
	// "never configured" are distinguishable.
	Panels []PanelState `json:"panels"`
	// Theme is "dark" or "light". An empty string means
	// the client should follow the OS preference.
	Theme string `json:"theme,omitempty"`
	// Compact tightens the grid for small screens.
	Compact bool `json:"compact,omitempty"`
}

// DefaultLayout returns the layout used before an operator
// has customised anything: every panel visible, agents
// spanning half the grid.
func DefaultLayout() Layout {
	return Layout{
		Panels: []PanelState{
			{ID: PanelAgents, Visible: true, Order: 0, Width: 12},
			{ID: PanelComms, Visible: true, Order: 1, Width: 6},
			{ID: PanelAudit, Visible: true, Order: 2, Width: 6},
		},
	}
}

// MaxWidth is the widest a panel may span. It matches the
// 12-column grid the UI renders.
const MaxWidth = 12

// MaxPanels caps how many panels a saved layout may carry,
// as a bound on operator input reaching a loop.
const MaxPanels = 32

// Normalize validates a layout and fills in anything the
// operator left out.
//
// It is deliberately forgiving in one direction and strict
// in the other:
//
//   - MISSING panels are added back with their defaults.
//     A layout saved before a panel existed should not
//     break; the new panel simply appears.
//   - UNKNOWN panels are an error. Silently dropping one
//     would hide an operator's mistake, and the UI has no
//     way to say "you configured a panel I do not have".
//   - Out-of-range values are clamped rather than
//     rejected. Width 99 is a mis-drag, not a broken
//     layout, and refusing to save would lose every other
//     correct setting alongside it.
//
// The function returns an error only for the unknown-panel
// and too-many-panels cases.
func (l Layout) Normalize() (Layout, error) {
	if len(l.Panels) > MaxPanels {
		return l, fmt.Errorf("webui: layout has %d panels, max is %d", len(l.Panels), MaxPanels)
	}
	seen := make(map[PanelID]bool, len(l.Panels))
	out := make([]PanelState, 0, len(AllPanels))

	for _, p := range l.Panels {
		if !KnownPanel(p.ID) {
			return l, fmt.Errorf("webui: unknown panel %q (known: %s)", p.ID, joinPanels(AllPanels))
		}
		if seen[p.ID] {
			// A duplicate would make the render order
			// ambiguous, so the later one is dropped
			// rather than erroring: the operator's intent
			// (this panel exists, these settings) is
			// unambiguous even if they pasted the block
			// twice.
			continue
		}
		seen[p.ID] = true
		out = append(out, clampPanel(p))
	}
	// Add any panel the layout did not mention.
	for _, id := range AllPanels {
		if seen[id] {
			continue
		}
		def := DefaultLayout()
		for _, p := range def.Panels {
			if p.ID == id {
				out = append(out, p)
			}
		}
	}
	// Sort by the caller's Order, breaking ties on ID so
	// the result is deterministic. This is an insertion
	// sort because panel counts are tiny (bounded by
	// MaxPanels) and it avoids pulling in "sort" for one
	// call site.
	sortPanels(out)
	return Layout{
		Panels:  out,
		Theme:   normalizeTheme(l.Theme),
		Compact: l.Compact,
	}, nil
}

// clampPanel brings one panel's numeric fields into range.
func clampPanel(p PanelState) PanelState {
	if p.Width < 1 {
		p.Width = 1
	}
	if p.Width > MaxWidth {
		p.Width = MaxWidth
	}
	if p.Order < 0 {
		p.Order = 0
	}
	return p
}

// normalizeTheme maps an unknown or empty theme to the
// empty string, which the client reads as "follow the OS".
func normalizeTheme(t string) string {
	switch strings.ToLower(strings.TrimSpace(t)) {
	case "dark":
		return "dark"
	case "light":
		return "light"
	default:
		return ""
	}
}

// sortPanels orders by Order, breaking ties on ID so the
// result is deterministic.
func sortPanels(ps []PanelState) {
	for i := 1; i < len(ps); i++ {
		for j := i; j > 0; j-- {
			a, b := ps[j-1], ps[j]
			if b.Order < a.Order || (b.Order == a.Order && b.ID < a.ID) {
				ps[j-1], ps[j] = b, a
				continue
			}
			break
		}
	}
}

func joinPanels(ids []PanelID) string {
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = string(id)
	}
	return strings.Join(parts, ", ")
}
