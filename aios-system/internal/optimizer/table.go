package optimizer

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// CostQualityTable is the empirical pass rate and cost
// per (fingerprint, class) pair (REQ-OPT-CQT-001 to 006).
//
// The table is an in-memory map. The on-disk file is a
// JSON object keyed by "<fingerprint>:<class>". The
// in-memory map is the source of truth; the file is
// rewritten on every Record call through an atomic
// rename. The file is also rewritten on Reset.
type CostQualityTable struct {
	mu   sync.Mutex
	path string
	rows map[string]CostQualityRow
}

// LoadCostQualityTable reads the table from the file at
// path. The function returns an empty table when the
// file does not exist. The function returns
// ErrTableCorrupt when the file exists but does not
// parse.
func LoadCostQualityTable(path string) (*CostQualityTable, error) {
	t := &CostQualityTable{path: path, rows: make(map[string]CostQualityRow)}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return t, nil
		}
		return nil, fmt.Errorf("optimizer: read %s: %w", path, err)
	}
	if len(data) == 0 {
		return t, nil
	}
	raw := map[string]CostQualityRow{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrTableCorrupt, err)
	}
	t.rows = raw
	return t, nil
}

// Path returns the file path of the table.
func (t *CostQualityTable) Path() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.path
}

// Get returns the row for the (fingerprint, class) pair.
// The function returns (CostQualityRow{}, false) when
// the pair is not in the table.
func (t *CostQualityTable) Get(fp Fingerprint, class string) (CostQualityRow, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	row, ok := t.rows[tableKey(fp, class)]
	return row, ok
}

// Record updates the table for the (fingerprint, class)
// pair (REQ-OPT-CQT-004). The function is not safe for
// concurrent use; callers must serialize Record calls
// when the table is shared.
//
// The function writes the table to the file atomically
// through a rename syscall.
func (t *CostQualityTable) Record(fp Fingerprint, class string, costUSD float64, verifyPassed bool) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	key := tableKey(fp, class)
	row, ok := t.rows[key]
	now := time.Now().UTC().Format(time.RFC3339)
	if !ok {
		row = CostQualityRow{FirstSeen: now}
	}
	row.CallCount++
	if verifyPassed {
		row.PassCount++
	}
	row.TotalCostUSD += costUSD
	row.LastSeen = now
	t.rows[key] = row
	return t.flushLocked()
}

// Reset clears the table (REQ-OPT-CQT-005). The
// function writes the empty table to the file.
func (t *CostQualityTable) Reset() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.rows = make(map[string]CostQualityRow)
	return t.flushLocked()
}

// Rows returns a copy of the in-memory rows. The map is
// keyed by "<fingerprint>:<class>". The function is
// used by the CLI to print the table.
func (t *CostQualityTable) Rows() map[string]CostQualityRow {
	t.mu.Lock()
	defer t.mu.Unlock()
	out := make(map[string]CostQualityRow, len(t.rows))
	for k, v := range t.rows {
		out[k] = v
	}
	return out
}

// AllKeys returns the (fingerprint, class) keys in a
// stable order. The CLI uses the result to print
// the table.
func (t *CostQualityTable) AllKeys() []string {
	t.mu.Lock()
	defer t.mu.Unlock()
	out := make([]string, 0, len(t.rows))
	for k := range t.rows {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// flushLocked writes the table to disk. The function is
// called under the lock.
func (t *CostQualityTable) flushLocked() error {
	if t.path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(t.path), 0o755); err != nil {
		return fmt.Errorf("optimizer: mkdir %s: %w", filepath.Dir(t.path), err)
	}
	data, err := json.MarshalIndent(t.rows, "", "  ")
	if err != nil {
		return fmt.Errorf("optimizer: marshal table: %w", err)
	}
	tmp := t.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("optimizer: write %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, t.path); err != nil {
		return fmt.Errorf("optimizer: rename %s -> %s: %w", tmp, t.path, err)
	}
	return nil
}

// tableKey is the "<fingerprint>:<class>" key. The
// fingerprint already includes a colon-separated hash;
// the second colon makes the key parseable.
func tableKey(fp Fingerprint, class string) string {
	return string(fp) + ":" + class
}

// ParseTableKey returns the fingerprint and class
// embedded in a key. The function returns
// ErrTableCorrupt when the key is malformed.
func ParseTableKey(key string) (Fingerprint, string, error) {
	idx := strings.LastIndex(key, ":")
	if idx < 0 {
		return "", "", fmt.Errorf("%w: bad key %q", ErrTableCorrupt, key)
	}
	return Fingerprint(key[:idx]), key[idx+1:], nil
}
