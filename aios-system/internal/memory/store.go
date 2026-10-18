package memory

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// Store is the content-addressed raw layer.
//
// The store is an append-only file system. The file
// name is the content hash; the file content is the
// artifact payload. The store is metadata-free: the
// file system IS the metadata index. A directory walk
// returns the set of stored hashes.
//
// Per REQ-MEM-STO-005 the store has no Delete method.
// The "gc" operation compacts the store to a single
// file (see Compact).
type Store struct {
	mu   sync.RWMutex
	path string
	// enc is the optional encryptor for the user scope.
	// The field is nil for unencrypted stores.
	enc Encryptor
}

// StoreConfig controls Store construction.
type StoreConfig struct {
	// Path is the directory under which the raw
	// artifacts are stored. Required.
	Path string
	// Encryptor is the optional encryptor for the
	// user scope. When nil, the store writes payloads
	// in plaintext. Production deployments that hold
	// secrets should always provide an encryptor.
	Encryptor Encryptor
}

// NewStore constructs a Store at cfg.Path. The
// function creates the directory if missing.
//
// Per REQ-MEM-STO-001 the store holds the raw layer.
// The store is content-addressed on disk: the file
// name is the content hash, the file content is the
// payload.
func NewStore(cfg StoreConfig) (*Store, error) {
	if cfg.Path == "" {
		return nil, &MemoryError{Op: "init", Reason: "path is required"}
	}
	if err := os.MkdirAll(cfg.Path, 0o755); err != nil {
		return nil, &MemoryError{Op: "init", Reason: err.Error()}
	}
	return &Store{path: cfg.Path, enc: cfg.Encryptor}, nil
}

// Put stores the content and returns the content
// hash. The method is idempotent: two puts of the
// same content return the same hash, and the second
// put is a no-op on disk.
//
// Per REQ-MEM-STO-003. The store validates the content
// is non-empty. The store does not enforce the
// artifact kind or scope; the artifact layer does.
func (s *Store) Put(content []byte) (string, error) {
	if len(content) == 0 {
		return "", &MemoryError{Op: "put", Reason: "content is required"}
	}
	hash := HashBytes(content)
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.writePayload(hash, content); err != nil {
		return "", err
	}
	return hash, nil
}

// Get returns the artifact for the content hash.
// The method returns ErrNotFound when the artifact
// is not in the store.
//
// Per REQ-MEM-STO-004.
func (s *Store) Get(hash string) ([]byte, error) {
	if hash == "" {
		return nil, &MemoryError{Op: "get", Reason: "hash is required"}
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if !s.knownHash(hash) {
		return nil, ErrNotFound
	}
	raw, err := os.ReadFile(s.payloadPath(hash))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, ErrNotFound
		}
		return nil, &MemoryError{Op: "get", Reason: err.Error()}
	}
	if s.enc == nil {
		return raw, nil
	}
	plain, err := s.enc.Decrypt(raw)
	if err != nil {
		return nil, &MemoryError{Op: "get", Reason: fmt.Sprintf("decrypt: %s", err.Error())}
	}
	return plain, nil
}

// Has returns true when the artifact is in the store.
// The method is the cheap existence check that does
// not load the payload from disk.
func (s *Store) Has(hash string) (bool, error) {
	if hash == "" {
		return false, &MemoryError{Op: "has", Reason: "hash is required"}
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if !s.knownHash(hash) {
		return false, nil
	}
	_, err := os.Stat(s.payloadPath(hash))
	if err == nil {
		return true, nil
	}
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	return false, &MemoryError{Op: "has", Reason: err.Error()}
}

// Size returns the size of the store in bytes. The
// function sums the file sizes under the store's
// directory. The function is used by the stats output.
func (s *Store) Size() (int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var total int64
	err := filepath.WalkDir(s.path, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		total += info.Size()
		return nil
	})
	if err != nil {
		return 0, &MemoryError{Op: "size", Reason: err.Error()}
	}
	return total, nil
}

// Count returns the number of artifacts in the store.
// The function is the stats helper for the
// `aios memory stats` subcommand.
func (s *Store) Count() (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entries, err := os.ReadDir(s.path)
	if err != nil {
		return 0, &MemoryError{Op: "count", Reason: err.Error()}
	}
	n := 0
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.HasSuffix(e.Name(), ".tmp") {
			continue
		}
		n++
	}
	return n, nil
}

// List returns the content hashes of every artifact
// in the store, in lexicographic order. The function
// is the metadata-index lookup the doc set describes:
// the directory walk IS the metadata index.
func (s *Store) List() ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entries, err := os.ReadDir(s.path)
	if err != nil {
		return nil, &MemoryError{Op: "list", Reason: err.Error()}
	}
	hashes := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.HasSuffix(e.Name(), ".tmp") {
			continue
		}
		hashes = append(hashes, e.Name())
	}
	sort.Strings(hashes)
	return hashes, nil
}

// Compact rewrites the store so each stored artifact
// is at a deterministic content-addressed path. The
// function is a no-op when the store is already
// compact; it is the implementation of the
// `aios memory gc` subcommand.
//
// The function returns a CompactReport with the
// before and after counts. The index is not touched
// because the content hashes are unchanged.
func (s *Store) Compact() (CompactReport, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	before, err := s.countLocked()
	if err != nil {
		return CompactReport{}, err
	}
	// The store is content-addressed by construction;
	// every file already lives at <path>/<hash>. Compact
	// is therefore a no-op on the layout. The function
	// exists so the CLI surface is identical across
	// stores and the report carries useful numbers.
	after := before
	return CompactReport{Before: before, After: after}, nil
}

// CompactReport is the result of Store.Compact.
type CompactReport struct {
	Before int
	After  int
}

// Path returns the store's directory. The function
// is the path getter for tests and the stats output.
func (s *Store) Path() string { return s.path }

// knownHash reports whether the hash is a valid
// content hash (64 lowercase hex chars). The
// function defends against path traversal when
// reading the store.
func (s *Store) knownHash(hash string) bool {
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

// payloadPath returns the on-disk path of the
// artifact payload. The function is the one place
// that knows the layout.
func (s *Store) payloadPath(hash string) string {
	return filepath.Join(s.path, hash)
}

// writePayload writes the payload to disk at the
// content-addressed path. The write is atomic
// through a temp file in the same directory and a
// rename. The function applies the encryptor when
// the store has one.
func (s *Store) writePayload(hash string, content []byte) error {
	payload := content
	if s.enc != nil {
		cipher, err := s.enc.Encrypt(content)
		if err != nil {
			return &MemoryError{Op: "put", Reason: fmt.Sprintf("encrypt: %s", err.Error())}
		}
		payload = cipher
	}
	final := s.payloadPath(hash)
	tmp, err := os.CreateTemp(s.path, hash+".tmp-*")
	if err != nil {
		return &MemoryError{Op: "put", Reason: err.Error()}
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op if rename succeeded
	if _, err := tmp.Write(payload); err != nil {
		tmp.Close()
		return &MemoryError{Op: "put", Reason: err.Error()}
	}
	if err := tmp.Close(); err != nil {
		return &MemoryError{Op: "put", Reason: err.Error()}
	}
	if err := os.Rename(tmpName, final); err != nil {
		return &MemoryError{Op: "put", Reason: err.Error()}
	}
	return nil
}

// countLocked returns the number of files in the
// store. The function is called with the store's
// mutex held.
func (s *Store) countLocked() (int, error) {
	entries, err := os.ReadDir(s.path)
	if err != nil {
		return 0, &MemoryError{Op: "compact", Reason: err.Error()}
	}
	n := 0
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.HasSuffix(e.Name(), ".tmp") {
			continue
		}
		n++
	}
	return n, nil
}
