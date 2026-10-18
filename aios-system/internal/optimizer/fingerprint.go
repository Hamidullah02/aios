package optimizer

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"
)

// Fingerprint is a SHA-256 hash that identifies one kind
// of model call (REQ-OPT-FIN-001, REQ-OPT-DAT-002). The
// string is a lowercase hex encoding of 64 characters.
type Fingerprint string

// contextBuckets are the input-token buckets the
// fingerprint rounds to. The doc lists these explicitly.
var contextBuckets = []int{
	256, 512, 1024, 2048, 4096, 8192, 16384, 32768, 65536, 131072,
}

// NewFingerprint computes the fingerprint for a call
// (REQ-OPT-FIN-002, REQ-OPT-FIN-003). The function is a
// pure function: two calls with the same inputs return
// the same fingerprint.
//
// The inputs are:
//
//   - promptShape: the prompt with the variable parts
//     replaced by their type tags (see ShapePrompt).
//   - tools: the sorted list of tool names used by the
//     call.
//   - role: the role name.
//   - inputTokens: the input token count, which is
//     rounded down to the nearest bucket.
func NewFingerprint(promptShape string, tools []string, role string, inputTokens int) Fingerprint {
	bucket := ContextBucket(inputTokens)
	// Sort tool names for a stable fingerprint across
	// orderings.
	sorted := append([]string(nil), tools...)
	sort.Strings(sorted)
	// Build the input as a stable concatenation. The
	// separator is a NUL byte so it cannot appear in a
	// tool name or a role.
	parts := []string{
		promptShape,
		strings.Join(sorted, "\x00"),
		role,
		itoa(bucket),
	}
	joined := strings.Join(parts, "\x1f")
	sum := sha256.Sum256([]byte(joined))
	return Fingerprint(hex.EncodeToString(sum[:]))
}

// ShapePrompt replaces the variable parts of a prompt
// with their type tags (REQ-OPT-REG-TRD §6.2). The
// function is a small helper; production code should
// call it once per call and pass the shape to
// NewFingerprint.
//
// The replacements are:
//
//	str  -> a string literal in quotes
//	num  -> a numeric literal
//	path -> a path literal
//	code -> a code block
//
// The function is intentionally permissive: it does
// not parse the prompt; it does substring replacement
// on quoted strings, numeric runs, /-prefixed paths,
// and ```-fenced code blocks. The goal is to make
// fingerprints insensitive to variable content but
// sensitive to the shape of the call.
func ShapePrompt(prompt string) string {
	var b strings.Builder
	b.Grow(len(prompt))
	i := 0
	for i < len(prompt) {
		// Code fence ```...```
		if strings.HasPrefix(prompt[i:], "```") {
			j := strings.Index(prompt[i+3:], "```")
			if j < 0 {
				b.WriteString("code")
				return b.String()
			}
			b.WriteString("code")
			i += 3 + j + 3
			continue
		}
		// Quoted string. The function only treats a
		// quote character as a string delimiter when a
		// matching close appears later in the prompt.
		// A lone quote (e.g. an apostrophe in
		// "no_quote's_here") is written verbatim, so
		// the shape stays "no_quotes_here" rather than
		// collapsing to "no_quotestr". The lookahead
		// is O(n) per quote; the outer loop keeps the
		// total cost linear.
		if prompt[i] == '"' || prompt[i] == '\'' {
			quote := prompt[i]
			close := -1
			for k := i + 1; k < len(prompt); k++ {
				if prompt[k] == '\\' && k+1 < len(prompt) {
					k++
					continue
				}
				if prompt[k] == quote {
					close = k
					break
				}
			}
			if close < 0 {
				// No matching close. Write the quote
				// literally and advance by one byte so
				// the outer loop keeps going.
				b.WriteByte(prompt[i])
				i++
				continue
			}
			b.WriteString("str")
			i = close + 1
			continue
		}
		// Number
		if prompt[i] >= '0' && prompt[i] <= '9' {
			b.WriteString("num")
			i++
			for i < len(prompt) && (prompt[i] >= '0' && prompt[i] <= '9' || prompt[i] == '.') {
				i++
			}
			continue
		}
		// Path (starts with / or ./ or ../)
		if prompt[i] == '/' || (prompt[i] == '.' && i+1 < len(prompt) && (prompt[i+1] == '/' || prompt[i+1] == '.')) {
			b.WriteString("path")
			i++
			for i < len(prompt) && !isPathSep(prompt[i]) {
				i++
			}
			continue
		}
		b.WriteByte(prompt[i])
		i++
	}
	return b.String()
}

// ContextBucket rounds the input token count down to the
// nearest power-of-2 bucket (REQ-OPT-FIN-002). The
// function returns 256 when inputTokens is 0.
func ContextBucket(inputTokens int) int {
	if inputTokens <= 0 {
		return contextBuckets[0]
	}
	b := contextBuckets[0]
	for _, n := range contextBuckets {
		if n <= inputTokens {
			b = n
		} else {
			break
		}
	}
	return b
}

// isPathSep reports whether c is whitespace, a quote, or
// a backtick that ends a path literal.
func isPathSep(c byte) bool {
	switch c {
	case ' ', '\t', '\n', '\r', '"', '\'', '`', '<', '>':
		return true
	}
	return false
}

// itoa is a small int-to-string for the bucket index.
// Using fmt.Sprintf for a single int is wasteful; a
// minimal itoa keeps the hot path cheap.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
