package egress

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"

	"github.com/Hamidullah02/aios/aios-system/internal/core/schemas"
)

// hashListPair is shared between AllowList.Hash and similar future
// fingerprinting helpers. It hashes two ordered lists of strings via
// the canonical JSON of a {a, b} struct, so two lists with the same
// elements in the same order hash equally.
func hashListPair(a, b []string) string {
	type doc struct {
		A []string `json:"allow"`
		B []string `json:"deny"`
	}
	// Sort a copy for determinism; the caller already passes stable
	// order, but sorting makes Hash stable across reorders of the
	// YAML file too.
	ac := append([]string(nil), a...)
	bc := append([]string(nil), b...)
	sort.Strings(ac)
	sort.Strings(bc)
	raw, err := schemas.MarshalCanonical(doc{A: ac, B: bc})
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

// hashHeaderRules is the same shape for the header rule set. Both
// lists are sorted by Name so a re-arranged file still hashes the
// same.
func hashHeaderRules(remove, add []HeaderRule) string {
	type doc struct {
		Remove []HeaderRule `json:"remove"`
		Add    []HeaderRule `json:"add"`
	}
	rc := append([]HeaderRule(nil), remove...)
	ac := append([]HeaderRule(nil), add...)
	sortByName(rc)
	sortByName(ac)
	raw, err := schemas.MarshalCanonical(doc{Remove: rc, Add: ac})
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func sortByName(s []HeaderRule) {
	sort.Slice(s, func(i, j int) bool { return s[i].Name < s[j].Name })
}
