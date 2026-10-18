package optimizer

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
)

// StatsFilter filters the table. The empty value is a
// pass-through. The CLI subcommand builds a filter
// from the --fingerprint and --class flags.
type StatsFilter struct {
	Fingerprint Fingerprint
	Class       string
}

// StatsSummary is the summary printed by `aios stats`
// (REQ-OPT-STA-006). The summary aggregates the table
// for the human reader.
type StatsSummary struct {
	TotalCallCount   int                `json:"total_call_count"`
	TotalCostUSD     float64            `json:"total_cost_usd"`
	OverrideCount    int                `json:"override_count"`
	AveragePassRate  float64            `json:"average_pass_rate"`
	AverageCostByCls map[string]float64 `json:"average_cost_by_class"`
	CallCountByCls   map[string]int     `json:"call_count_by_class"`
}

// PrintStats writes the table and the summary to w
// (REQ-OPT-STA-001 to 006). The filter is optional. The
// output is human-readable by default; a JSON option
// emits the raw map for tooling.
func PrintStats(w io.Writer, table *CostQualityTable, filter StatsFilter, asJSON bool) error {
	rows := filteredRows(table, filter)
	if asJSON {
		return json.NewEncoder(w).Encode(rows)
	}
	// Pretty-print the table.
	keys := make([]string, 0, len(rows))
	for k := range rows {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	fmt.Fprintln(w, "fingerprint:class  calls  pass  pass-rate  avg-cost-usd  last-seen")
	for _, k := range keys {
		r := rows[k]
		fmt.Fprintf(w, "%s  %d  %d  %.3f  %.6f  %s\n",
			k, r.CallCount, r.PassCount, r.PassRate(), r.AverageCost(), r.LastSeen)
	}
	// Summary.
	summary := summarize(filteredRowsAll(table, filter))
	fmt.Fprintln(w)
	fmt.Fprintln(w, "summary:")
	fmt.Fprintf(w, "  total call count: %d\n", summary.TotalCallCount)
	fmt.Fprintf(w, "  total cost:       %.6f\n", summary.TotalCostUSD)
	fmt.Fprintf(w, "  override count:   %d\n", summary.OverrideCount)
	fmt.Fprintf(w, "  avg pass rate:    %.3f\n", summary.AveragePassRate)
	for class, c := range summary.CallCountByCls {
		fmt.Fprintf(w, "  class %-7s:    calls=%d  avg-cost=%.6f\n",
			class, c, summary.AverageCostByCls[class])
	}
	return nil
}

// filteredRows returns the rows that pass the filter.
func filteredRows(table *CostQualityTable, filter StatsFilter) map[string]CostQualityRow {
	all := table.Rows()
	if filter.Fingerprint == "" && filter.Class == "" {
		return all
	}
	out := map[string]CostQualityRow{}
	for k, v := range all {
		fp, class, err := ParseTableKey(k)
		if err != nil {
			continue
		}
		if filter.Fingerprint != "" && fp != filter.Fingerprint {
			continue
		}
		if filter.Class != "" && class != filter.Class {
			continue
		}
		out[k] = v
	}
	return out
}

// filteredRowsAll is the same as filteredRows but
// returns the full map regardless of filter. The
// summary uses this when a filter is set so the
// numbers stay coherent.
func filteredRowsAll(table *CostQualityTable, filter StatsFilter) map[string]CostQualityRow {
	_ = filter
	return table.Rows()
}

// summarize aggregates the rows.
func summarize(rows map[string]CostQualityRow) StatsSummary {
	out := StatsSummary{
		AverageCostByCls: map[string]float64{},
		CallCountByCls:   map[string]int{},
	}
	classCosts := map[string]float64{}
	classCount := map[string]int{}
	for k, r := range rows {
		out.TotalCallCount += r.CallCount
		out.TotalCostUSD += r.TotalCostUSD
		_, class, err := ParseTableKey(k)
		if err != nil {
			continue
		}
		classCount[class] += r.CallCount
		classCosts[class] += r.TotalCostUSD
	}
	for class, n := range classCount {
		if n > 0 {
			out.AverageCostByCls[class] = classCosts[class] / float64(n)
		}
		out.CallCountByCls[class] = n
	}
	if out.TotalCallCount > 0 {
		// Pass rate is averaged across the rows.
		rate := 0.0
		for _, r := range rows {
			rate += r.PassRate()
		}
		out.AveragePassRate = rate / float64(len(rows))
	}
	return out
}

// FormatOverrideList returns the human-readable list of
// override values for the help text. The list is sorted
// for stable output.
func FormatOverrideList() string {
	overrides := []string{string(OverrideAuto), string(OverrideStrong), string(OverrideMid), string(OverrideCheap), string(OverrideOff)}
	return strings.Join(overrides, ", ")
}
