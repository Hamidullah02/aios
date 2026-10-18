package optimizer

import (
	"errors"
	"path/filepath"
	"testing"
)

// testRegistry returns a registry with cheap, mid, strong.
func testRegistry(t *testing.T) *ClassBasedModelRegistry {
	t.Helper()
	reg, err := NewClassBasedModelRegistry([]Class{
		{Name: "cheap", ModelID: "m-cheap", CostPer1kInput: 0.001, CostPer1kOutput: 0.002, ContextWindow: 1000},
		{Name: "mid", ModelID: "m-mid", CostPer1kInput: 0.005, CostPer1kOutput: 0.01, ContextWindow: 4000},
		{Name: "strong", ModelID: "m-strong", CostPer1kInput: 0.015, CostPer1kOutput: 0.075, ContextWindow: 16000},
	})
	if err != nil {
		t.Fatal(err)
	}
	return reg
}

func testTable(t *testing.T) *CostQualityTable {
	t.Helper()
	dir := t.TempDir()
	t1, err := LoadCostQualityTable(filepath.Join(dir, "t.json"))
	if err != nil {
		t.Fatal(err)
	}
	return t1
}

func TestRoute_OverrideOff(t *testing.T) {
	r := NewRouter(testRegistry(t), testTable(t))
	d, err := r.Route("fp", OverrideOff, 0)
	if err != nil {
		t.Fatal(err)
	}
	if d.Class != "strong" || d.Source != SourceOptimizerOff {
		t.Fatalf("got %+v", d)
	}
}

func TestRoute_OverrideStrong(t *testing.T) {
	r := NewRouter(testRegistry(t), testTable(t))
	d, err := r.Route("fp", OverrideStrong, 0)
	if err != nil {
		t.Fatal(err)
	}
	if d.Class != "strong" || d.Source != SourceOverride {
		t.Fatalf("got %+v", d)
	}
}

func TestRoute_OverrideMid(t *testing.T) {
	r := NewRouter(testRegistry(t), testTable(t))
	d, _ := r.Route("fp", OverrideMid, 0)
	if d.Class != "mid" || d.Source != SourceOverride {
		t.Fatalf("got %+v", d)
	}
}

func TestRoute_OverrideCheap(t *testing.T) {
	r := NewRouter(testRegistry(t), testTable(t))
	d, _ := r.Route("fp", OverrideCheap, 0)
	if d.Class != "cheap" || d.Source != SourceOverride {
		t.Fatalf("got %+v", d)
	}
}

func TestRoute_OverrideUnknown(t *testing.T) {
	r := NewRouter(testRegistry(t), testTable(t))
	_, err := r.Route("fp", Override("mystery"), 0)
	if !errors.Is(err, ErrUnknownClass) {
		t.Fatalf("err = %v", err)
	}
}

func TestRoute_OverrideUnknownClassInRegistry(t *testing.T) {
	// override "strong" but the registry has been
	// trimmed to only "cheap".
	reg, _ := NewClassBasedModelRegistry([]Class{
		{Name: "cheap", ModelID: "a", CostPer1kInput: 0.001, CostPer1kOutput: 0.002},
	})
	r := NewRouter(reg, testTable(t))
	_, err := r.Route("fp", OverrideStrong, 0)
	if !errors.Is(err, ErrUnknownClass) {
		t.Fatalf("err = %v", err)
	}
}

func TestRoute_DefaultEmpty(t *testing.T) {
	r := NewRouter(testRegistry(t), testTable(t))
	d, err := r.Route("fp", OverrideAuto, 0)
	if err != nil {
		t.Fatal(err)
	}
	// default rule is cost_first → cheapest.
	if d.Class != "cheap" || d.Source != SourceDefault {
		t.Fatalf("got %+v", d)
	}
}

func TestRoute_DefaultStrongFirst(t *testing.T) {
	r := NewRouter(testRegistry(t), testTable(t))
	r.SetDefaultRule("strong_first")
	d, _ := r.Route("fp", OverrideAuto, 0)
	if d.Class != "strong" {
		t.Fatalf("got %+v", d)
	}
}

func TestRoute_DefaultUnknownRuleFallsBack(t *testing.T) {
	r := NewRouter(testRegistry(t), testTable(t))
	r.SetDefaultRule("nonsense")
	d, _ := r.Route("fp", OverrideAuto, 0)
	if d.Class != "cheap" {
		t.Fatalf("got %+v", d)
	}
}

func TestRoute_TableSingle(t *testing.T) {
	tab := testTable(t)
	r := NewRouter(testRegistry(t), tab)
	fp := Fingerprint("fp1")
	_ = tab.Record(fp, "cheap", 0.001, true)
	d, _ := r.Route(fp, OverrideAuto, 0)
	if d.Class != "cheap" || d.Source != SourceTable {
		t.Fatalf("got %+v", d)
	}
}

func TestRoute_TableBestByScore(t *testing.T) {
	tab := testTable(t)
	r := NewRouter(testRegistry(t), tab)
	fp := Fingerprint("fp1")
	// cheap: low cost, low rate
	_ = tab.Record(fp, "cheap", 0.0001, true)
	_ = tab.Record(fp, "cheap", 0.0001, false)
	// strong: high cost, high rate
	_ = tab.Record(fp, "strong", 0.01, true)
	_ = tab.Record(fp, "strong", 0.01, true)
	d, _ := r.Route(fp, OverrideAuto, 0)
	// cheap rate=0.5, cost=0.0001, score=5000
	// strong rate=1.0, cost=0.01, score=100
	// cheap wins.
	if d.Class != "cheap" {
		t.Fatalf("got %+v", d)
	}
}

func TestRoute_TableBestByScore_Equal(t *testing.T) {
	tab := testTable(t)
	r := NewRouter(testRegistry(t), tab)
	fp := Fingerprint("fp1")
	_ = tab.Record(fp, "cheap", 0.0001, true)
	_ = tab.Record(fp, "strong", 0.0001, true)
	d, _ := r.Route(fp, OverrideAuto, 0)
	// equal score: first wins after sort.Slice with
	// strict > comparator (so the original order is
	// stable in practice; the result is implementation
	// defined).
	if d.Class == "" {
		t.Fatal("no class")
	}
}

func TestRoute_BudgetFits(t *testing.T) {
	r := NewRouter(testRegistry(t), testTable(t))
	d, _ := r.Route("fp", OverrideAuto, 1.0)
	if d.Class != "cheap" {
		t.Fatalf("got %+v", d)
	}
}

func TestRoute_BudgetExhausted(t *testing.T) {
	r := NewRouter(testRegistry(t), testTable(t))
	_, err := r.Route("fp", OverrideAuto, 0.0001)
	if !errors.Is(err, ErrBudgetExhausted) {
		t.Fatalf("err = %v", err)
	}
}

func TestRoute_BudgetForcesCheaper(t *testing.T) {
	// Budget between mid and strong.
	r := NewRouter(testRegistry(t), testTable(t))
	d, err := r.Route("fp", OverrideStrong, 0.01) // strong = 0.045/2 = 0.0225
	if err != nil {
		t.Fatal(err)
	}
	if d.Class != "cheap" || d.Source != SourceBudget {
		t.Fatalf("got %+v", d)
	}
}

func TestRoute_BudgetOverride(t *testing.T) {
	// Override mid with a budget that doesn't cover mid
	// (mid = (0.005+0.01)/2 = 0.0075; budget 0.005).
	r := NewRouter(testRegistry(t), testTable(t))
	d, err := r.Route("fp", OverrideMid, 0.005)
	if err != nil {
		t.Fatal(err)
	}
	if d.Source != SourceBudget {
		t.Fatalf("got %+v", d)
	}
}

func TestRoute_BudgetZero(t *testing.T) {
	// budget 0 means no cap.
	r := NewRouter(testRegistry(t), testTable(t))
	d, _ := r.Route("fp", OverrideStrong, 0)
	if d.Class != "strong" {
		t.Fatalf("got %+v", d)
	}
}

func TestRoute_Purity(t *testing.T) {
	r := NewRouter(testRegistry(t), testTable(t))
	a, _ := r.Route("fp", OverrideAuto, 0)
	b, _ := r.Route("fp", OverrideAuto, 0)
	if a != b {
		t.Fatalf("not pure: %+v vs %+v", a, b)
	}
}

func TestRoute_EmptyRegistry(t *testing.T) {
	tab := testTable(t)
	reg, _ := NewClassBasedModelRegistry(nil)
	r := NewRouter(reg, tab)
	_, err := r.Route("fp", OverrideAuto, 0)
	if !errors.Is(err, ErrUnknownClass) {
		t.Fatalf("err = %v", err)
	}
}

func TestRoute_NilTable(t *testing.T) {
	r := NewRouter(testRegistry(t), nil)
	d, err := r.Route("fp", OverrideAuto, 0)
	if err != nil {
		t.Fatal(err)
	}
	if d.Source != SourceDefault {
		t.Fatalf("got %+v", d)
	}
}

func TestRoute_UnknownStrong(t *testing.T) {
	// Strong not in registry → ErrUnknownClass from
	// the optimizer_off path.
	reg, _ := NewClassBasedModelRegistry([]Class{
		{Name: "cheap", ModelID: "a"},
	})
	r := NewRouter(reg, testTable(t))
	_, err := r.Route("fp", OverrideOff, 0)
	if !errors.Is(err, ErrUnknownClass) {
		t.Fatalf("err = %v", err)
	}
}

func TestSetDefaultClass(t *testing.T) {
	r := NewRouter(testRegistry(t), testTable(t))
	r.SetDefaultClass("strong")
	d, _ := r.Route("fp", OverrideAuto, 0)
	if d.Class != "cheap" {
		t.Fatalf("got %+v", d)
	}
}

func TestCheapest(t *testing.T) {
	r := NewRouter(testRegistry(t), testTable(t))
	names := []string{"strong", "mid", "cheap"}
	c, err := r.cheapest(names)
	if err != nil {
		t.Fatal(err)
	}
	if c.Name != "cheap" {
		t.Fatalf("got %q", c.Name)
	}
}

func TestCheapest_Empty(t *testing.T) {
	r := NewRouter(testRegistry(t), testTable(t))
	_, err := r.cheapest(nil)
	if !errors.Is(err, ErrUnknownClass) {
		t.Fatalf("err = %v", err)
	}
}
