package runtime

import "crypto/sha256"

// NoProgressDetector tracks the worker's outputs and trips
// when the same output repeats more than the limit
// (REQ-RT-NPR-001 to 003).
//
// The detector is content-based: it hashes each output and
// compares hashes. The limit is the number of *consecutive*
// identical outputs that trip the detector.
type NoProgressDetector struct {
	limit int
	prev  string
	count int
}

// NewNoProgressDetector returns a detector with the given
// limit. limit <= 0 disables the detector (Observe always
// returns nil).
func NewNoProgressDetector(limit int) *NoProgressDetector {
	return &NoProgressDetector{limit: limit}
}

// Observe records the worker's latest output and returns a
// *NoProgressDetected when the same output has been seen
// `limit` times in a row. Different content resets the count.
func (d *NoProgressDetector) Observe(output string) error {
	if d.limit <= 0 {
		return nil
	}
	h := hashOutput(output)
	if h == d.prev {
		d.count++
	} else {
		d.prev = h
		d.count = 1
	}
	if d.count > d.limit {
		return &NoProgressDetected{Count: d.count - 1}
	}
	return nil
}

// Reset clears the detector. Useful when the orchestrator
// moves to a new task.
func (d *NoProgressDetector) Reset() {
	d.prev = ""
	d.count = 0
}

func hashOutput(s string) string {
	sum := sha256.Sum256([]byte(s))
	return string(sum[:])
}
