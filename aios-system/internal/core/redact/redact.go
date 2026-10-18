// Package redact provides secret redaction for audit and log payloads.
//
// Per the doc set:
//
//   - REQ-CORE-RED-001: 5 secret patterns are recognized (API key, JWT,
//     PEM block, AWS access key, GitHub token). Each is replaced with a
//     placeholder of the form `<REDACTED:kind:sha256>`.
//   - REQ-CORE-RED-002: placeholders are stable across runs for the same
//     (kind, value) pair. Re-running the redactor on the same input
//     produces the same output bytes.
//   - REQ-CORE-RED-003: the approval_id field, when present, is
//     preserved verbatim and is not recursed into for redaction. This
//     lets downstream code join audit rows to approvals without a
//     separate copy.
//   - REQ-CORE-RED-004: redaction is depth-first over the value tree.
//     Strings are replaced. Maps and slices recurse.
package redact

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"regexp"
	"strings"
)

// Kind is the secret kind used in the placeholder.
type Kind string

const (
	KindAPIKey Kind = "api_key"
	KindJWT    Kind = "jwt"
	KindPEM    Kind = "pem"
	KindAWSKey Kind = "aws_key"
	KindGitHub Kind = "github_token"
)

// PlaceholderPrefix is the prefix used in every redacted placeholder.
const PlaceholderPrefix = "<REDACTED:"

// PlaceholderSuffix is the suffix used in every redacted placeholder.
const PlaceholderSuffix = ">"

// hashByteLen is the number of bytes (hex-doubled) included in the
// placeholder's stable hash. 8 bytes (16 hex chars) is enough to be
// stable for a single redacted string without bloating the audit row.
const hashByteLen = 8

// Match is a single detection event returned by FindAll.
type Match struct {
	Start  int
	End    int
	Kind   Kind
	Stable string // the <REDACTED:kind:sha256> placeholder
}

type pattern struct {
	kind Kind
	re   *regexp.Regexp
}

// patterns are ordered. The first match wins at any given offset.
//
// PEM blocks span multiple lines and a closing `-----END ...-----`
// marker, so we use [\s\S] with a non-greedy match.
//
// API-key and AWS/GitHub token patterns are designed to be high-precision;
// they require either a known prefix word ("api_key", "AKIA", "ghp_")
// or a contextual separator (":", "="). Random long hex strings are
// not matched.
var patterns = []pattern{
	{
		kind: KindAPIKey,
		re:   regexp.MustCompile(`(?i)(?:api[_-]?key|apikey|api[_-]?secret)\s*[:=]\s*["']?([A-Za-z0-9._\-]{16,})["']?`),
	},
	{
		kind: KindJWT,
		re:   regexp.MustCompile(`eyJ[A-Za-z0-9_-]+\.eyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+`),
	},
	{
		kind: KindPEM,
		re:   regexp.MustCompile(`-----BEGIN [A-Z ]+PRIVATE KEY-----[\s\S]+?-----END [A-Z ]+PRIVATE KEY-----`),
	},
	{
		kind: KindAWSKey,
		re:   regexp.MustCompile(`\bAKIA[0-9A-Z]{16}\b`),
	},
	{
		kind: KindGitHub,
		re:   regexp.MustCompile(`\bgh[pousr]_[A-Za-z0-9]{30,}\b`),
	},
}

// candidate is an internal match candidate before overlap resolution.
type candidate struct {
	start, end int
	kind       Kind
}

// StablePlaceholder returns the placeholder for a given kind and original
// value. The placeholder is stable for a given (kind, value) pair.
func StablePlaceholder(k Kind, original string) string {
	sum := sha256.Sum256([]byte(string(k) + "\x00" + original))
	return PlaceholderPrefix + string(k) + ":" + hex.EncodeToString(sum[:hashByteLen]) + PlaceholderSuffix
}

// FindAll returns every match in s, ordered by start offset. Overlapping
// matches are resolved by picking the earliest-starting match; if two
// matches start at the same offset, the first pattern (in `patterns`
// order) wins.
func FindAll(s string) []Match {
	if s == "" {
		return nil
	}
	var cands []candidate
	for _, p := range patterns {
		for _, loc := range p.re.FindAllStringIndex(s, -1) {
			cands = append(cands, candidate{start: loc[0], end: loc[1], kind: p.kind})
		}
	}
	if len(cands) == 0 {
		return nil
	}
	sortCandidatesByStart(cands)

	// Greedy sweep: keep a match if its start is >= the last kept match's
	// end. Empty-length matches (start == end) are dropped.
	var kept []candidate
	lastEnd := 0
	for _, c := range cands {
		if c.start == c.end {
			continue
		}
		if len(kept) > 0 && c.start < lastEnd {
			continue
		}
		kept = append(kept, c)
		lastEnd = c.end
	}

	out := make([]Match, len(kept))
	for i, c := range kept {
		out[i] = Match{
			Start:  c.start,
			End:    c.end,
			Kind:   c.kind,
			Stable: StablePlaceholder(c.kind, s[c.start:c.end]),
		}
	}
	return out
}

// sortCandidatesByStart sorts candidates by (start asc, end asc). For
// candidates with the same start, the original order is preserved, so
// earlier patterns in `patterns` win. n is typically 0-3; insertion sort
// is fine.
func sortCandidatesByStart(cands []candidate) {
	for i := 1; i < len(cands); i++ {
		j := i
		for j > 0 && (cands[j].start < cands[j-1].start ||
			(cands[j].start == cands[j-1].start && cands[j].end < cands[j-1].end)) {
			cands[j-1], cands[j] = cands[j], cands[j-1]
			j--
		}
	}
}

// RedactString returns s with every detected secret replaced by its
// stable placeholder. Field-level preservation (e.g. the approval_id)
// is handled by RedactValue, not here.
func RedactString(s string) string {
	if s == "" {
		return s
	}
	matches := FindAll(s)
	if len(matches) == 0 {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	prev := 0
	for _, m := range matches {
		b.WriteString(s[prev:m.Start])
		b.WriteString(m.Stable)
		prev = m.End
	}
	b.WriteString(s[prev:])
	return b.String()
}

// RedactValue is the recursive redactor over a value tree.
//
// Strings are passed through RedactString. Maps and slices recurse.
// Other types (numbers, booleans, nil) are returned unchanged.
//
// `approvalIDKey` is the map key whose value is an approval ID; that
// value is preserved verbatim and is not recursed into for redaction
// (REQ-CORE-RED-003). The approval ID is a UUID, not a secret, but the
// field is excluded from redaction so that downstream code can join
// audit rows to approvals without a separate copy.
func RedactValue(v any, approvalIDKey string) any {
	switch t := v.(type) {
	case string:
		return RedactString(t)
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, e := range t {
			if k == approvalIDKey {
				out[k] = e
				continue
			}
			out[k] = RedactValue(e, approvalIDKey)
		}
		return out
	case []any:
		out := make([]any, len(t))
		for i, e := range t {
			out[i] = RedactValue(e, approvalIDKey)
		}
		return out
	default:
		return v
	}
}

// RedactJSONBytes applies RedactValue to a JSON-encoded
// payload and returns the redacted, canonical JSON bytes.
//
// The function performs a single decode + recurse + encode
// cycle. Callers that already hold canonical JSON (e.g. the
// audit row writer) should use this helper instead of
// decoding to a generic value, redacting, and re-encoding
// themselves; the helper centralizes the round-trip and
// keeps the redactor's input stable.
//
// `approvalIDKey` is forwarded to RedactValue and has the
// same semantics as there.
func RedactJSONBytes(raw []byte, approvalIDKey string) ([]byte, error) {
	if len(raw) == 0 {
		return raw, nil
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, err
	}
	redacted := RedactValue(v, approvalIDKey)
	return json.Marshal(redacted)
}
