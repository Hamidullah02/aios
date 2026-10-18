package memory

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

// PinEntry is the per-hash metadata stored in the
// pin set. The struct is intentionally minimal: the
// memory stores a reason, a timestamp, and an
// optional actor. The content hash is the join key;
// the payload itself stays in the raw layer.
type PinEntry struct {
	Hash      string `json:"hash"`
	Reason    string `json:"reason,omitempty"`
	Timestamp string `json:"timestamp"`
	Actor     string `json:"actor,omitempty"`
}

// PinSet is the content-addressed pin set.
//
// The pin set is a JSON object. The keys are the
// content hashes; the values are the PinEntry
// records. The pin set id is the SHA-256 of the
// canonical JSON of the pin set; the id is what the
// retrieval and compaction algorithms stamp on
// the CompactedView so the developer can verify the
// pin set was honored.
//
// Per REQ-MEM-PIN-004 pinned artifacts are always
// included in the compacted view. Per REQ-MEM-PIN-005
// the pin set is editable; the developer can
// `aios memory pin` and `aios memory unpin`.
type PinSet struct {
	mu    sync.RWMutex
	path  string
	pins  map[string]PinEntry
	dirty bool
}

// PinSetConfig controls PinSet construction.
type PinSetConfig struct {
	// Path is the directory under which the pin set
	// file is stored. The file is <path>/pins.json.
	// Required.
	Path string
}

// NewPinSet constructs a PinSet and loads the on-disk
// file when one exists. A missing file is treated as
// an empty pin set, not as an error.
func NewPinSet(cfg PinSetConfig) (*PinSet, error) {
	if cfg.Path == "" {
		return nil, &MemoryError{Op: "init", Reason: "path is required"}
	}
	if err := os.MkdirAll(cfg.Path, 0o755); err != nil {
		return nil, &MemoryError{Op: "init", Reason: err.Error()}
	}
	ps := &PinSet{
		path: cfg.Path,
		pins: make(map[string]PinEntry),
	}
	if err := ps.loadFromDisk(); err != nil {
		return nil, err
	}
	return ps, nil
}

// Add pins the hash. The method is idempotent:
// re-adding the same hash keeps the original
// timestamp and overwrites the reason/actor. The
// method saves the pin set to disk on every change.
//
// REQ-MEM-PIN-001.
func (p *PinSet) Add(hash, reason, actor string) error {
	if hash == "" {
		return &MemoryError{Op: "pin", Reason: "hash is required"}
	}
	if !p.knownHash(hash) {
		return &MemoryError{Op: "pin", Reason: "hash is not a valid 64-char hex content hash"}
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	existing, ok := p.pins[hash]
	ts := nowRFC3339()
	if ok {
		ts = existing.Timestamp
	}
	p.pins[hash] = PinEntry{
		Hash:      hash,
		Reason:    reason,
		Timestamp: ts,
		Actor:     actor,
	}
	p.dirty = true
	return p.saveLocked()
}

// Remove unpins the hash. The method is a no-op
// when the hash is not in the pin set.
//
// REQ-MEM-PIN-001.
func (p *PinSet) Remove(hash string) error {
	if hash == "" {
		return &MemoryError{Op: "unpin", Reason: "hash is required"}
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, ok := p.pins[hash]; !ok {
		return nil
	}
	delete(p.pins, hash)
	p.dirty = true
	return p.saveLocked()
}

// Has reports whether the hash is pinned.
func (p *PinSet) Has(hash string) bool {
	if hash == "" {
		return false
	}
	p.mu.RLock()
	defer p.mu.RUnlock()
	_, ok := p.pins[hash]
	return ok
}

// Entry returns the pin entry for the hash. The
// function returns (PinEntry{}, false) when the
// hash is not pinned.
func (p *PinSet) Entry(hash string) (PinEntry, bool) {
	if hash == "" {
		return PinEntry{}, false
	}
	p.mu.RLock()
	defer p.mu.RUnlock()
	entry, ok := p.pins[hash]
	return entry, ok
}

// Size returns the number of pinned hashes.
func (p *PinSet) Size() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.pins)
}

// List returns a copy of every pin entry, sorted
// by hash. The function is the bulk export the
// retrieval algorithm uses.
func (p *PinSet) List() []PinEntry {
	p.mu.RLock()
	defer p.mu.RUnlock()
	out := make([]PinEntry, 0, len(p.pins))
	for _, e := range p.pins {
		out = append(out, e)
	}
	sort.Slice(out, func(a, b int) bool {
		return out[a].Hash < out[b].Hash
	})
	return out
}

// Hashes returns the sorted list of pinned hashes.
func (p *PinSet) Hashes() []string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	out := make([]string, 0, len(p.pins))
	for h := range p.pins {
		out = append(out, h)
	}
	sort.Strings(out)
	return out
}

// ID returns the content-addressed id of the current
// pin set. The id is the SHA-256 of the canonical
// JSON of the pin set. The id changes when a pin is
// added or removed.
//
// Per REQ-MEM-PIN-002. Per REQ-MEM-DAT-003 the
// CompactedView's PinSetID field is this id.
func (p *PinSet) ID() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	raw, err := json.Marshal(p.pins)
	if err != nil {
		return ""
	}
	return HashBytes(raw)
}

// Path returns the pin set's directory. The
// function is the path getter for tests and the
// stats output.
func (p *PinSet) Path() string { return p.path }

// knownHash reports whether the hash is a valid
// content hash (64 lowercase hex chars). The
// function defends against path traversal.
func (p *PinSet) knownHash(hash string) bool {
	if len(hash) != 64 {
		return false
	}
	for _, c := range hash {
		switch {
		case c >= '0' && c <= '9':
		case c >= 'a' && c <= 'f':
		default:
			return false
		}
	}
	return true
}

// loadFromDisk reads the pin set file. A missing
// file is treated as an empty pin set. An
// unparseable file returns ErrPinSetCorrupt.
func (p *PinSet) loadFromDisk() error {
	raw, err := os.ReadFile(p.filePath())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return &MemoryError{Op: "load pins", Reason: err.Error()}
	}
	if len(raw) == 0 {
		return nil
	}
	pins := make(map[string]PinEntry)
	if err := json.Unmarshal(raw, &pins); err != nil {
		return fmt.Errorf("%w: %v", ErrPinSetCorrupt, err)
	}
	for hash, e := range pins {
		if hash == "" {
			return fmt.Errorf("%w: empty hash", ErrPinSetCorrupt)
		}
		if e.Hash == "" {
			e.Hash = hash
			pins[hash] = e
		}
	}
	p.pins = pins
	p.dirty = false
	return nil
}

// saveLocked writes the pin set to disk. The
// function is called with the pin set's mutex
// held. The save is atomic through a temp file
// in the same directory and a rename.
func (p *PinSet) saveLocked() error {
	if !p.dirty {
		return nil
	}
	raw, err := json.MarshalIndent(p.pins, "", "  ")
	if err != nil {
		return &MemoryError{Op: "save pins", Reason: err.Error()}
	}
	final := p.filePath()
	tmp, err := os.CreateTemp(p.path, "pins.json.tmp-*")
	if err != nil {
		return &MemoryError{Op: "save pins", Reason: err.Error()}
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op if rename succeeded
	if _, err := tmp.Write(raw); err != nil {
		tmp.Close()
		return &MemoryError{Op: "save pins", Reason: err.Error()}
	}
	if err := tmp.Close(); err != nil {
		return &MemoryError{Op: "save pins", Reason: err.Error()}
	}
	if err := os.Rename(tmpName, final); err != nil {
		return &MemoryError{Op: "save pins", Reason: err.Error()}
	}
	p.dirty = false
	return nil
}

// filePath returns the pin set file path.
func (p *PinSet) filePath() string {
	return filepath.Join(p.path, "pins.json")
}
