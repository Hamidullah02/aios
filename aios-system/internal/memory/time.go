package memory

import "time"

// nowRFC3339 returns the current UTC time formatted
// per RFC 3339 with nanosecond precision. The function
// is used by components that do not take a clock
// injection (e.g. the artifact's Timestamp default).
func nowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}

// parseTimestamp parses an ISO 8601 timestamp. The
// function tries RFC 3339 with and without nanoseconds.
// The function returns the zero time when the input
// does not match either layout.
func parseTimestamp(s string) (time.Time, bool) {
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}
