package sandbox

import "sync"

// defaultRunner is the runner used by any package that needs a
// subprocess runner but was not given one.
//
// The reason it exists is enforcement. The git tools, the search
// diagnostics, the shell tool and the test runner each fall back
// to their own runner when a caller does not supply one, and a
// policy applied only at the explicit construction sites would be
// silently absent from every one of those fallbacks. The one
// place a caller can therefore be sure is covered is the
// package-level default.
//
// Production wiring calls SetDefaultRunner once, at startup, with a
// runner carrying the network policy. After that, every subprocess
// in the process is governed by it, whichever package asked.
var (
	defaultRunnerMu sync.RWMutex
	defaultRunner   Runner
)

func init() {
	defaultRunner = NewSubprocessRunner()
}

// DefaultRunner returns the process-wide default runner.
//
// It is what the packages with a nil-Runner fallback use. A test
// that wants to substitute a runner for the whole process calls
// SetDefaultRunner and restores the previous value with defer.
func DefaultRunner() Runner {
	defaultRunnerMu.RLock()
	defer defaultRunnerMu.RUnlock()
	if defaultRunner == nil {
		return NewSubprocessRunner()
	}
	return defaultRunner
}

// SetDefaultRunner replaces the process-wide default runner.
//
// It returns the previous value so a caller can restore it, which
// is what a test does:
//
//	old := sandbox.SetDefaultRunner(fake)
//	defer sandbox.SetDefaultRunner(old)
//
// The change is process-wide on purpose. A per-caller default
// would leave the fallback sites unguarded, which is the exact
// gap this type exists to close.
func SetDefaultRunner(r Runner) Runner {
	defaultRunnerMu.Lock()
	defer defaultRunnerMu.Unlock()
	old := defaultRunner
	if r == nil {
		r = NewSubprocessRunner()
	}
	defaultRunner = r
	return old
}
