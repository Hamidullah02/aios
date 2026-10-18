package retry

import "testing"

func TestStrategy_None(t *testing.T) {
	s := NewStrategy(StrategyNone)
	if s.Backoff(0) != 0 || s.Backoff(5) != 0 {
		t.Fatal("none should be 0")
	}
}

func TestStrategy_Immediate(t *testing.T) {
	s := NewStrategy(StrategyImmediate)
	if s.Backoff(3) != 0 {
		t.Fatal("immediate should be 0")
	}
}

func TestStrategy_Linear(t *testing.T) {
	s := NewStrategy(StrategyLinear)
	if got := s.Backoff(1); got != DefaultBaseDelayMs {
		t.Fatalf("linear 1 = %d", got)
	}
	if got := s.Backoff(3); got != 3*DefaultBaseDelayMs {
		t.Fatalf("linear 3 = %d", got)
	}
}

func TestStrategy_Exponential(t *testing.T) {
	s := NewStrategy(StrategyExponential)
	if got := s.Backoff(0); got != DefaultBaseDelayMs {
		t.Fatalf("exp 0 = %d", got)
	}
	if got := s.Backoff(3); got != 8*DefaultBaseDelayMs {
		t.Fatalf("exp 3 = %d", got)
	}
}

func TestStrategy_ExponentialCapped(t *testing.T) {
	s := NewStrategy(StrategyExponential)
	if got := s.Backoff(50); got != MaxBackoffMs {
		t.Fatalf("exp 50 = %d, want capped at %d", got, MaxBackoffMs)
	}
}

func TestStrategy_ClampsNegative(t *testing.T) {
	s := NewStrategy(StrategyLinear)
	if got := s.Backoff(-1); got != 0 {
		t.Fatalf("linear -1 = %d", got)
	}
}

func TestStrategy_NilReceiver(t *testing.T) {
	var s *RetryStrategy
	if s.Backoff(5) != 0 {
		t.Fatal("nil receiver should be 0")
	}
}

func TestStrategy_DefaultBaseDelay(t *testing.T) {
	s := &RetryStrategy{Mode: StrategyLinear} // BaseDelay == 0
	if got := s.Backoff(1); got != DefaultBaseDelayMs {
		t.Fatalf("default base = %d", got)
	}
}

func TestStrategy_UnknownMode(t *testing.T) {
	s := &RetryStrategy{Mode: "weird"}
	if got := s.Backoff(5); got != 0 {
		t.Fatalf("unknown mode = %d", got)
	}
}
