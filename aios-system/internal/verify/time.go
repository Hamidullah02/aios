package verify

import "time"

// now returns the current UTC time. The
// function is the clock helper the test-runner
// result uses for the duration measurement.
// Tests may inject a fake clock through
// RunTestsTool.Now.
func now() time.Time { return time.Now().UTC() }

// elapsedMs returns the duration between
// start and the current time in milliseconds.
// The function is the helper the RunTests
// tool uses to populate TestResult.DurationMs.
func elapsedMs(start time.Time) int64 {
	return time.Since(start).Milliseconds()
}
