package edit

import (
	"fmt"
	"strings"
)

// unifiedDiff returns a small unified diff between before
// and after. The function is line-based; it returns an
// empty string when the two inputs match. The format is
// `--- before\n+++ after\n@@ ... @@\n<context>\n-<old>\n+<new>`.
//
// We keep the format minimal: the doc set only requires the
// diff to be in unified-diff form for the audit row.
func unifiedDiff(before, after string) string {
	a := strings.Split(before, "\n")
	b := strings.Split(after, "\n")
	if equalLines(a, b) {
		return ""
	}
	// Compute the longest common prefix and the longest
	// common suffix. The middle is the hunk body.
	prefix := 0
	for prefix < len(a) && prefix < len(b) && a[prefix] == b[prefix] {
		prefix++
	}
	suffix := 0
	for suffix < len(a)-prefix && suffix < len(b)-prefix &&
		a[len(a)-1-suffix] == b[len(b)-1-suffix] {
		suffix++
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "--- before\n+++ after\n@@ -%d,%d +%d,%d @@\n",
		prefix+1, len(a)-prefix-suffix, prefix+1, len(b)-prefix-suffix)
	for _, line := range a[:prefix] {
		fmt.Fprintf(&sb, " %s\n", line)
	}
	for _, line := range a[prefix : len(a)-suffix] {
		fmt.Fprintf(&sb, "-%s\n", line)
	}
	for _, line := range b[prefix : len(b)-suffix] {
		fmt.Fprintf(&sb, "+%s\n", line)
	}
	for _, line := range a[len(a)-suffix:] {
		fmt.Fprintf(&sb, " %s\n", line)
	}
	return sb.String()
}

func equalLines(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
