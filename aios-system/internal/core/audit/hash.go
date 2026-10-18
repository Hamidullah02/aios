package audit

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
)

// redactedHash computes SHA256(payload).
//
// It is a separate hash so that a verifier with only the redacted
// payload can confirm the payload column equals the hashed value,
// without re-running the redactor.
func redactedHash(payload []byte) []byte {
	sum := sha256.Sum256(payload)
	return sum[:]
}

// contentHash computes SHA256(payload || actor || kind || ts ||
// approval_id || idempotency_key), per global PRD §11.2.
//
// `ts` is a Unix-nanosecond big-endian int64. We use the binary
// representation rather than the textual form to avoid locale
// surprises.
func contentHash(payload []byte, actor, kind string, tsNS int64, approvalID, idempotencyKey string) []byte {
	h := sha256.New()
	h.Write(payload)
	h.Write([]byte(actor))
	h.Write([]byte("\x00"))
	h.Write([]byte(kind))
	h.Write([]byte("\x00"))
	var tsBuf [8]byte
	binary.BigEndian.PutUint64(tsBuf[:], uint64(tsNS))
	h.Write(tsBuf[:])
	h.Write([]byte(approvalID))
	h.Write([]byte(idempotencyKey))
	sum := h.Sum(nil)
	return sum
}

// chainHash computes SHA256(content_hash || prev_chain_hash || seq),
// per global PRD §11.2.
//
// `seq` is an 8-byte big-endian int64. `prev` is the previous row's
// chain_hash (32 bytes) or, for the first row, 32 zero bytes.
func chainHash(contentHashBytes, prev []byte, seq int64) []byte {
	h := sha256.New()
	h.Write(contentHashBytes)
	h.Write(prev)
	var seqBuf [8]byte
	binary.BigEndian.PutUint64(seqBuf[:], uint64(seq))
	h.Write(seqBuf[:])
	sum := h.Sum(nil)
	return sum
}

// hexOf returns the lowercase hex of a hash.
func hexOf(b []byte) string { return hex.EncodeToString(b) }

// hashPair is the (content, chain) pair produced by hashAndSeal.
type hashPair struct {
	redacted  []byte
	content   []byte
	chain     []byte
	redactedH string
	contentH  string
	chainH    string
	idempKey  string
}

func (p hashPair) String() string {
	return fmt.Sprintf("redacted=%s content=%s chain=%s", p.redactedH, p.contentH, p.chainH)
}

// hashAndSeal computes all three hashes for a row and assigns the
// row's internal fields.
//
// The function is pure with respect to its inputs; it does not
// read the database.
func (r *Row) hashAndSeal(prevChain []byte, seq int64) error {
	// Idempotency key: if empty, default to the content_hash (in hex).
	// The content hash depends on the idempotency key, so we have a
	// chicken-and-egg. We resolve this by first hashing with an empty
	// idempotency key, taking the hex, and then re-hashing with that
	// hex as the idempotency key. This is exactly what global PRD
	// §11.14 means by "the idempotency key defaults to the content_hash
	// of the row."
	if r.IdempotencyKey == "" {
		// First pass with empty key.
		first := contentHash(
			r.payloadJSON,
			r.Actor,
			r.Kind,
			r.RecordedAt.UnixNano(),
			r.ApprovalID,
			"",
		)
		r.IdempotencyKey = hexOf(first)
	}
	// Final pass with the chosen idempotency key.
	ch := contentHash(
		r.payloadJSON,
		r.Actor,
		r.Kind,
		r.RecordedAt.UnixNano(),
		r.ApprovalID,
		r.IdempotencyKey,
	)
	kh := chainHash(ch, prevChain, seq)
	r.contentHash = ch
	r.chainHash = kh
	return nil
}
