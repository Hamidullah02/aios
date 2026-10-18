package audit

import "encoding/hex"

// mustHexDecode decodes a hex string. It panics on bad input; callers
// pass strings that were just hexOf'd, so a panic indicates a bug.
func mustHexDecode(s string) []byte {
	b, err := hex.DecodeString(s)
	if err != nil {
		panic("audit: internal: bad hex: " + err.Error())
	}
	return b
}
