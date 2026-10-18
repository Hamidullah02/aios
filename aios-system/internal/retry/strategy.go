package retry

// RetryStrategy computes the backoff in milliseconds for a
// given retry count (REQ-RTR-STR-001 to 003). The modes
// match the doc set:
//
//	none         — always 0
//	immediate    — always 0
//	linear       — baseDelay * retryCount
//	exponential  — baseDelay * 2^retryCount, capped at MaxBackoffMs
type RetryStrategy struct {
	Mode      StrategyMode
	BaseDelay int // ms; 0 → DefaultBaseDelayMs
}

// NewStrategy returns a strategy with the given mode.
func NewStrategy(mode StrategyMode) *RetryStrategy {
	return &RetryStrategy{Mode: mode, BaseDelay: DefaultBaseDelayMs}
}

// Backoff returns the delay in milliseconds for the given
// retry count. retryCount is 0 on the first call (no
// previous failure), 1 after the first failure, and so on.
func (s *RetryStrategy) Backoff(retryCount int) int {
	if s == nil {
		return 0
	}
	base := s.BaseDelay
	if base <= 0 {
		base = DefaultBaseDelayMs
	}
	if retryCount < 0 {
		retryCount = 0
	}
	var ms int
	switch s.Mode {
	case StrategyLinear:
		ms = base * retryCount
	case StrategyExponential:
		// 2^retryCount — clamp the exponent to avoid
		// overflow on absurd retry counts.
		if retryCount > 30 {
			ms = base * (1 << 30)
		} else {
			ms = base * (1 << retryCount)
		}
	default: // none, immediate, or unknown
		return 0
	}
	if ms > MaxBackoffMs {
		ms = MaxBackoffMs
	}
	return ms
}
