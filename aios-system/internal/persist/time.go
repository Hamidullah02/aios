package persist

import "time"

// nowRFC3339 returns the current UTC time formatted per
// RFC 3339 with nanosecond precision. The function is
// used by components that do not take a clock injection
// (e.g. Handoff, where the timestamp is informational).
func nowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}
