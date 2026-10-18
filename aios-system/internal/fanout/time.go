package fanout

import "time"

// nowRFC3339 returns the current UTC time formatted
// per RFC 3339 with nanosecond precision. The
// function is the timestamp helper the audit log
// and the result types use.
func nowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}

// elapsedMs returns the wall-clock duration in
// milliseconds between start and now. The function
// is the helper the FanOutResult.DurationMs field
// uses.
func elapsedMs(start time.Time) int64 {
	return time.Since(start).Milliseconds()
}
