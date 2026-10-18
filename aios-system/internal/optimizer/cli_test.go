package optimizer

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestPrintStats_Default(t *testing.T) {
	tab := testTable(t)
	fp := Fingerprint("fp1")
	_ = tab.Record(fp, "cheap", 0.001, true)
	_ = tab.Record(fp, "cheap", 0.001, false)
	var buf bytes.Buffer
	if err := PrintStats(&buf, tab, StatsFilter{}, false); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "cheap") {
		t.Fatalf("missing class: %s", out)
	}
	if !strings.Contains(out, "summary:") {
		t.Fatalf("missing summary: %s", out)
	}
}

func TestPrintStats_JSON(t *testing.T) {
	tab := testTable(t)
	fp := Fingerprint("fp1")
	_ = tab.Record(fp, "cheap", 0.001, true)
	var buf bytes.Buffer
	if err := PrintStats(&buf, tab, StatsFilter{}, true); err != nil {
		t.Fatal(err)
	}
	var decoded map[string]CostQualityRow
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("json: %v", err)
	}
	if len(decoded) != 1 {
		t.Fatalf("rows = %d", len(decoded))
	}
}

func TestPrintStats_FilterClass(t *testing.T) {
	tab := testTable(t)
	fp := Fingerprint("fp1")
	_ = tab.Record(fp, "cheap", 0.001, true)
	_ = tab.Record(fp, "strong", 0.01, true)
	var buf bytes.Buffer
	_ = PrintStats(&buf, tab, StatsFilter{Class: "cheap"}, false)
	if !strings.Contains(buf.String(), "cheap") {
		t.Fatalf("missing cheap: %s", buf.String())
	}
}

func TestPrintStats_FilterFingerprint(t *testing.T) {
	tab := testTable(t)
	_ = tab.Record("fp1", "cheap", 0.001, true)
	_ = tab.Record("fp2", "cheap", 0.001, true)
	var buf bytes.Buffer
	_ = PrintStats(&buf, tab, StatsFilter{Fingerprint: "fp1"}, false)
	if !strings.Contains(buf.String(), "fp1") {
		t.Fatalf("missing fp1: %s", buf.String())
	}
}

func TestFormatOverrideList(t *testing.T) {
	s := FormatOverrideList()
	if !strings.Contains(s, "strong") {
		t.Fatal("missing strong")
	}
}

func TestSummarize_Empty(t *testing.T) {
	s := summarize(map[string]CostQualityRow{})
	if s.TotalCallCount != 0 || s.TotalCostUSD != 0 {
		t.Fatal("expected zero")
	}
}

func TestSummarize_NonEmpty(t *testing.T) {
	s := summarize(map[string]CostQualityRow{
		"fp:cheap":  {CallCount: 3, PassCount: 2, TotalCostUSD: 0.003},
		"fp:strong": {CallCount: 5, PassCount: 4, TotalCostUSD: 0.05},
	})
	if s.TotalCallCount != 8 {
		t.Fatalf("calls = %d", s.TotalCallCount)
	}
	if s.CallCountByCls["cheap"] != 3 {
		t.Fatalf("cheap = %d", s.CallCountByCls["cheap"])
	}
}
