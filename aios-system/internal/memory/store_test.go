package memory

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// newTestStore constructs a Store pointed at a
// fresh temp directory. The function returns the
// store and a cleanup the caller must defer.
func newTestStore(t *testing.T) *Store {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "store")
	s, err := NewStore(StoreConfig{Path: dir})
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	return s
}

// TestStorePutIdempotent covers REQ-MEM-STO-003.
// Two puts of the same content return the same
// hash.
func TestStorePutIdempotent(t *testing.T) {
	s := newTestStore(t)
	h1, err := s.Put([]byte("hello"))
	if err != nil {
		t.Fatalf("Put: %v", err)
	}
	h2, err := s.Put([]byte("hello"))
	if err != nil {
		t.Fatalf("Put: %v", err)
	}
	if h1 != h2 {
		t.Fatalf("hashes differ: %s vs %s", h1, h2)
	}
	if h1 != HashBytes([]byte("hello")) {
		t.Fatalf("hash does not match: got %s", h1)
	}
}

// TestStorePutRejectsEmpty covers the empty-content
// guard.
func TestStorePutRejectsEmpty(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.Put(nil); err == nil {
		t.Fatal("expected error for nil content")
	}
	if _, err := s.Put([]byte{}); err == nil {
		t.Fatal("expected error for empty content")
	}
}

// TestStoreGetRoundTrip covers REQ-MEM-STO-004.
func TestStoreGetRoundTrip(t *testing.T) {
	s := newTestStore(t)
	want := []byte("round trip payload")
	h, err := s.Put(want)
	if err != nil {
		t.Fatalf("Put: %v", err)
	}
	got, err := s.Get(h)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("payload mismatch: got %q want %q", got, want)
	}
}

// TestStoreGetNotFound covers the ErrNotFound path.
func TestStoreGetNotFound(t *testing.T) {
	s := newTestStore(t)
	_, err := s.Get(HashBytes([]byte("missing")))
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

// TestStoreGetEmptyHash covers the empty-hash guard.
func TestStoreGetEmptyHash(t *testing.T) {
	s := newTestStore(t)
	_, err := s.Get("")
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestStoreHas covers the existence check.
func TestStoreHas(t *testing.T) {
	s := newTestStore(t)
	h, _ := s.Put([]byte("present"))
	ok, err := s.Has(h)
	if err != nil || !ok {
		t.Fatalf("Has(present) = %v, %v", ok, err)
	}
	ok, err = s.Has(HashBytes([]byte("absent")))
	if err != nil || ok {
		t.Fatalf("Has(absent) = %v, %v", ok, err)
	}
}

// TestStoreHasRejectsBadHash covers the path
// traversal defense.
func TestStoreHasRejectsBadHash(t *testing.T) {
	s := newTestStore(t)
	for _, bad := range []string{"", "../escape", "short", strings.ToUpper(HashBytes([]byte("x")))} {
		ok, err := s.Has(bad)
		if err != nil {
			t.Fatalf("Has(%q) error: %v", bad, err)
		}
		if ok {
			t.Fatalf("Has(%q) returned true", bad)
		}
	}
}

// TestStoreCountAndList cover the metadata-index
// helpers.
func TestStoreCountAndList(t *testing.T) {
	s := newTestStore(t)
	payloads := [][]byte{[]byte("a"), []byte("b"), []byte("c")}
	want := make([]string, len(payloads))
	for i, p := range payloads {
		h, err := s.Put(p)
		if err != nil {
			t.Fatalf("Put: %v", err)
		}
		want[i] = h
	}
	sort.Strings(want)
	n, err := s.Count()
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	if n != len(payloads) {
		t.Fatalf("Count = %d, want %d", n, len(payloads))
	}
	got, err := s.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if !equalStrings(got, want) {
		t.Fatalf("List = %v, want %v", got, want)
	}
}

// TestStoreSize covers the byte-size total.
func TestStoreSize(t *testing.T) {
	s := newTestStore(t)
	_, _ = s.Put([]byte("hello"))
	_, _ = s.Put([]byte("world!"))
	n, err := s.Size()
	if err != nil {
		t.Fatalf("Size: %v", err)
	}
	if n != int64(len("hello")+len("world!")) {
		t.Fatalf("Size = %d, want %d", n, len("hello")+len("world!"))
	}
}

// TestStoreCompactNoOp covers the gc no-op. The
// store is content-addressed by construction.
func TestStoreCompactNoOp(t *testing.T) {
	s := newTestStore(t)
	_, _ = s.Put([]byte("x"))
	r, err := s.Compact()
	if err != nil {
		t.Fatalf("Compact: %v", err)
	}
	if r.Before != 1 || r.After != 1 {
		t.Fatalf("report = %+v, want Before=1 After=1", r)
	}
}

// TestStoreNewRequiresPath covers the empty-path
// guard when the in-memory store is not used.
// (NewStore with empty path now returns an
// in-memory store; we only test that Path() is
// empty for the in-memory store.)
func TestStoreNewInMemory(t *testing.T) {
	s, err := NewStore(StoreConfig{})
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	if s.Path() != "" {
		t.Fatalf("Path = %q, want empty", s.Path())
	}
	h, err := s.Put([]byte("mem"))
	if err != nil {
		t.Fatalf("Put: %v", err)
	}
	got, err := s.Get(h)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if string(got) != "mem" {
		t.Fatalf("got %q, want mem", got)
	}
}

// TestStoreEncrypted covers the AES-GCM encryptor
// path. The on-disk file is not the plaintext.
func TestStoreEncrypted(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "enc-store")
	key := bytes.Repeat([]byte{0x42}, 32)
	enc, err := NewAESGCMEncryptor(key)
	if err != nil {
		t.Fatalf("NewAESGCMEncryptor: %v", err)
	}
	s, err := NewStore(StoreConfig{Path: dir, Encryptor: enc})
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	payload := []byte("secret payload")
	h, err := s.Put(payload)
	if err != nil {
		t.Fatalf("Put: %v", err)
	}
	// The on-disk file must not contain the
	// plaintext.
	raw, err := os.ReadFile(filepath.Join(dir, h))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if bytes.Contains(raw, payload) {
		t.Fatalf("ciphertext contains plaintext: %q", raw)
	}
	got, err := s.Get(h)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("decrypted mismatch")
	}
}

// TestAESGCMWrongKey covers the auth-failure path
// when the wrong key is used to decrypt.
func TestAESGCMWrongKey(t *testing.T) {
	enc, _ := NewAESGCMEncryptor(bytes.Repeat([]byte{0x11}, 32))
	ct, err := enc.Encrypt([]byte("hello"))
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	wrong, _ := NewAESGCMEncryptor(bytes.Repeat([]byte{0x22}, 32))
	if _, err := wrong.Decrypt(ct); err == nil {
		t.Fatal("expected auth failure")
	}
}

// TestAESGCMShortCiphertext covers the
// length-check guard.
func TestAESGCMShortCiphertext(t *testing.T) {
	enc, _ := NewAESGCMEncryptor(bytes.Repeat([]byte{0x33}, 32))
	if _, err := enc.Decrypt([]byte{1, 2, 3}); err == nil {
		t.Fatal("expected error for short ciphertext")
	}
}

// TestAESGCMBadKeyLength covers the key-length
// guard.
func TestAESGCMBadKeyLength(t *testing.T) {
	if _, err := NewAESGCMEncryptor([]byte("short")); err == nil {
		t.Fatal("expected error for short key")
	}
}

// TestNoopEncryptor covers the no-op encryptor
// used by tests that want the encryption code
// path without a real cipher.
func TestNoopEncryptor(t *testing.T) {
	var n NoopEncryptor
	ct, err := n.Encrypt([]byte("x"))
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if string(ct) != "x" {
		t.Fatalf("got %q, want x", ct)
	}
	pt, err := n.Decrypt(ct)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if string(pt) != "x" {
		t.Fatalf("got %q, want x", pt)
	}
}

// equalStrings is the test helper.
func equalStrings(a, b []string) bool {
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
