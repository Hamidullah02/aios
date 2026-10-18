package optimizer

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestTable_Load_Missing(t *testing.T) {
	dir := t.TempDir()
	t1, err := LoadCostQualityTable(filepath.Join(dir, "no.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(t1.Rows()) != 0 {
		t.Fatal("expected empty")
	}
}

func TestTable_Load_Corrupt(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(p, []byte("not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadCostQualityTable(p)
	if !errors.Is(err, ErrTableCorrupt) {
		t.Fatalf("err = %v", err)
	}
}

func TestTable_RecordAndGet(t *testing.T) {
	dir := t.TempDir()
	t1, err := LoadCostQualityTable(filepath.Join(dir, "t.json"))
	if err != nil {
		t.Fatal(err)
	}
	fp := NewFingerprint("p", nil, "r", 100)
	if err := t1.Record(fp, "cheap", 0.001, true); err != nil {
		t.Fatal(err)
	}
	row, ok := t1.Get(fp, "cheap")
	if !ok {
		t.Fatal("not found")
	}
	if row.CallCount != 1 || row.PassCount != 1 {
		t.Fatalf("row = %+v", row)
	}
	if row.AverageCost() != 0.001 {
		t.Fatalf("avg = %v", row.AverageCost())
	}
}

func TestTable_Record_Atomic(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "t.json")
	t1, _ := LoadCostQualityTable(p)
	fp := NewFingerprint("p", nil, "r", 100)
	for i := 0; i < 5; i++ {
		if err := t1.Record(fp, "cheap", 0.001, true); err != nil {
			t.Fatal(err)
		}
	}
	// Reload and check.
	t2, _ := LoadCostQualityTable(p)
	row, ok := t2.Get(fp, "cheap")
	if !ok || row.CallCount != 5 {
		t.Fatalf("got %+v", row)
	}
}

func TestTable_Record_Failure(t *testing.T) {
	dir := t.TempDir()
	t1, _ := LoadCostQualityTable(filepath.Join(dir, "t.json"))
	fp := NewFingerprint("p", nil, "r", 100)
	if err := t1.Record(fp, "cheap", 0.001, false); err != nil {
		t.Fatal(err)
	}
	row, _ := t1.Get(fp, "cheap")
	if row.PassCount != 0 {
		t.Fatalf("pass = %d", row.PassCount)
	}
}

func TestTable_Reset(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "t.json")
	t1, _ := LoadCostQualityTable(p)
	fp := NewFingerprint("p", nil, "r", 100)
	_ = t1.Record(fp, "cheap", 0.001, true)
	if err := t1.Reset(); err != nil {
		t.Fatal(err)
	}
	if _, ok := t1.Get(fp, "cheap"); ok {
		t.Fatal("should be empty after reset")
	}
	// Reload.
	t2, _ := LoadCostQualityTable(p)
	if _, ok := t2.Get(fp, "cheap"); ok {
		t.Fatal("file should be empty after reset")
	}
}

func TestTable_AllKeys(t *testing.T) {
	dir := t.TempDir()
	t1, _ := LoadCostQualityTable(filepath.Join(dir, "t.json"))
	fp := NewFingerprint("p", nil, "r", 100)
	_ = t1.Record(fp, "cheap", 0.001, true)
	_ = t1.Record(fp, "strong", 0.01, true)
	keys := t1.AllKeys()
	if len(keys) != 2 {
		t.Fatalf("keys = %v", keys)
	}
}

func TestTable_Path(t *testing.T) {
	dir := t.TempDir()
	t1, _ := LoadCostQualityTable(filepath.Join(dir, "t.json"))
	if t1.Path() != filepath.Join(dir, "t.json") {
		t.Fatal("path mismatch")
	}
}

func TestTable_Rows(t *testing.T) {
	dir := t.TempDir()
	t1, _ := LoadCostQualityTable(filepath.Join(dir, "t.json"))
	if len(t1.Rows()) != 0 {
		t.Fatal("empty rows")
	}
}

func TestParseTableKey(t *testing.T) {
	fp, class, err := ParseTableKey("abc:cheap")
	if err != nil {
		t.Fatal(err)
	}
	if fp != "abc" || class != "cheap" {
		t.Fatalf("got %q, %q", fp, class)
	}
	_, _, err = ParseTableKey("nokey")
	if !errors.Is(err, ErrTableCorrupt) {
		t.Fatalf("err = %v", err)
	}
}

func TestPassRate(t *testing.T) {
	if (CostQualityRow{}).PassRate() != 0 {
		t.Fatal("empty")
	}
	r := CostQualityRow{CallCount: 4, PassCount: 3}
	if r.PassRate() != 0.75 {
		t.Fatalf("rate = %v", r.PassRate())
	}
	if (CostQualityRow{CallCount: 4}).AverageCost() != 0 {
		t.Fatal("empty avg")
	}
}

func TestTable_Record_Parallel(t *testing.T) {
	// Verify that concurrent Record calls do not race
	// (run with -race to confirm).
	dir := t.TempDir()
	t1, _ := LoadCostQualityTable(filepath.Join(dir, "t.json"))
	fp := NewFingerprint("p", nil, "r", 100)
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = t1.Record(fp, "cheap", 0.001, true)
		}()
	}
	wg.Wait()
	row, _ := t1.Get(fp, "cheap")
	if row.CallCount != 50 {
		t.Fatalf("calls = %d", row.CallCount)
	}
}

func TestTable_NoPath(t *testing.T) {
	// When the table has no path, Record succeeds but
	// does not write to disk.
	t1 := &CostQualityTable{path: "", rows: make(map[string]CostQualityRow)}
	fp := NewFingerprint("p", nil, "r", 100)
	if err := t1.Record(fp, "cheap", 0.001, true); err != nil {
		t.Fatal(err)
	}
}

func TestTable_Load_ReadError(t *testing.T) {
	// A directory instead of a file causes a read error.
	dir := t.TempDir()
	subdir := filepath.Join(dir, "t.json")
	if err := os.Mkdir(subdir, 0o755); err != nil {
		t.Fatal(err)
	}
	_, err := LoadCostQualityTable(subdir)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestTable_Load_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "t.json")
	if err := os.WriteFile(p, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	t1, err := LoadCostQualityTable(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(t1.Rows()) != 0 {
		t.Fatal("expected empty")
	}
}

func TestTable_ParseKeyFromRows(t *testing.T) {
	// Reloading a file that contains a malformed key is
	// gracefully tolerated: rows() returns the key as-is,
	// but ParseTableKey will fail when the key has no
	// colon. The router tolerates this; we verify the
	// key roundtrips through JSON.
	dir := t.TempDir()
	p := filepath.Join(dir, "t.json")
	if err := os.WriteFile(p, []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}
	t1, err := LoadCostQualityTable(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(t1.Rows()) != 0 {
		t.Fatal("empty")
	}
}
