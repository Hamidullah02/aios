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

// LazyDefaultRunner is a Runner that resolves to the process-wide
// default runner at the moment it is used, rather than at the
// moment it is constructed.
//
// It exists because the order of startup is wrong for capturing
// the default: the tool set is built when the dependency graph is
// wired, and the egress guard that installs the network policy on
// the default runner is started afterwards. A tool that captured
// DefaultRunner() at construction time would hold the
// policy-less runner that existed before the guard started, and
// its subprocesses would go straight to the network.
//
// Resolving at call time means the tool always uses whichever
// runner is installed now. A tool that needs a fixed runner (a
// test, or a caller with a specific policy) still supplies one
// directly.
type LazyDefaultRunner struct{}

// Run resolves the default runner and delegates to it.
func (LazyDefaultRunner) Run(opts RunOptions) (SandboxResult, error) {
	return DefaultRunner().Run(opts)
}

// Compile-time check: the lazy runner satisfies the interface.
var _ Runner = LazyDefaultRunner{}
