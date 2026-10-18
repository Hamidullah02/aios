package memory

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
)

// Encryptor is the symmetric-encryption interface
// the user-scope Store uses. The interface lets
// tests substitute a no-op encryptor and lets
// production deployments plug in any cipher
// (e.g. AES-GCM with a passphrase-derived key).
//
// Encrypt turns a plaintext payload into a
// ciphertext. Decrypt is the inverse. Both methods
// must round-trip a payload through the same
// encryptor instance.
type Encryptor interface {
	Encrypt(plaintext []byte) ([]byte, error)
	Decrypt(ciphertext []byte) ([]byte, error)
}

// AESGCMEncryptor is the production Encryptor.
// The encryptor uses AES-256-GCM. The nonce is
// prepended to the ciphertext so a single byte slice
// round-trips through Encrypt/Decrypt.
//
// The 32-byte key is supplied by the caller. The
// recommended pattern is to derive the key from a
// passphrase through argon2 (handled by the CLI),
// or to load the key from the OS keychain.
type AESGCMEncryptor struct {
	gcm cipher.AEAD
}

// NewAESGCMEncryptor constructs an AESGCMEncryptor
// from a 32-byte key. The function returns an
// error when the key length is wrong.
//
// The key is the AES-256 key. The caller is
// responsible for key derivation, rotation, and
// storage. The encryptor does not copy the key; the
// caller must keep the slice alive.
func NewAESGCMEncryptor(key []byte) (*AESGCMEncryptor, error) {
	if len(key) != 32 {
		return nil, &MemoryError{Op: "encryptor", Reason: fmt.Sprintf("key must be 32 bytes, got %d", len(key))}
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, &MemoryError{Op: "encryptor", Reason: err.Error()}
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, &MemoryError{Op: "encryptor", Reason: err.Error()}
	}
	return &AESGCMEncryptor{gcm: gcm}, nil
}

// Encrypt seals the plaintext with a fresh nonce
// and returns nonce||ciphertext. The function
// returns an error when the entropy source fails.
func (e *AESGCMEncryptor) Encrypt(plaintext []byte) ([]byte, error) {
	nonce := make([]byte, e.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	ct := e.gcm.Seal(nil, nonce, plaintext, nil)
	out := make([]byte, 0, len(nonce)+len(ct))
	out = append(out, nonce...)
	out = append(out, ct...)
	return out, nil
}

// Decrypt opens nonce||ciphertext. The function
// returns an error when the ciphertext is too
// short or the authentication tag fails to verify.
func (e *AESGCMEncryptor) Decrypt(ciphertext []byte) ([]byte, error) {
	ns := e.gcm.NonceSize()
	if len(ciphertext) < ns {
		return nil, errors.New("ciphertext too short")
	}
	nonce, body := ciphertext[:ns], ciphertext[ns:]
	pt, err := e.gcm.Open(nil, nonce, body, nil)
	if err != nil {
		return nil, err
	}
	return pt, nil
}

// NoopEncryptor is the test encryptor. The
// encryptor returns the plaintext verbatim. The
// encryptor exists so tests can exercise the
// encryption code path without depending on a
// real cipher.
type NoopEncryptor struct{}

// Encrypt is a no-op.
func (NoopEncryptor) Encrypt(plaintext []byte) ([]byte, error) {
	out := make([]byte, len(plaintext))
	copy(out, plaintext)
	return out, nil
}

// Decrypt is a no-op.
func (NoopEncryptor) Decrypt(ciphertext []byte) ([]byte, error) {
	out := make([]byte, len(ciphertext))
	copy(out, ciphertext)
	return out, nil
}
