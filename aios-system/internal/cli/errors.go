// Package cli is the aios-cli module. The package is the
// single entry point for the operator. It parses argv,
// dispatches the subcommand, and prints the result.
//
// Substitutions vs the doc set:
//
//   - The doc set's Python package `aios.cli` becomes the Go
//     package `internal/cli`. The Python split into
//     `parser.py`, `run_command.py`, `daemon.py`, etc. is
//     collapsed into a single package with multiple files.
//   - `argparse` (Python) → `flag` (Go stdlib). The doc set
//     names argparse explicitly; flag is the Go equivalent and
//     the natural way to express the same surface.
//   - `PyYAML` (Python) → `gopkg.in/yaml.v3` (already in
//     go.mod from aios-egress). The plan files and the config
//     file are YAML.
//   - The doc set's "atexit" → `defer` and explicit cleanup.
//     Go does not have an atexit; the daemon's cleanup is in
//     the Stop() method.
//   - The doc set's "PyPI" console script → Go binary built
//     with `go build` and installed by the build system, not
//     by pip.
package cli

import (
	"errors"
	"fmt"
)

// ExitCode constants. Per REQ-CLI-ERR-003 the CLI returns:
//
//   - 0 on success
//   - 1 on user error (bad input, missing file, invalid plan)
//   - 2 on system error (panic, internal failure, dependency crash)
const (
	ExitOK          = 0
	ExitUserError   = 1
	ExitSystemError = 2
)

// CliError is the typed error the CLI raises for "I could not
// run because the operator gave me bad input or because the
// environment is wrong". Per REQ-CLI-ERR-100.
//
// The Cause field is the underlying error. The Kind is a short
// label the error display uses to pick a colour / category.
type CliError struct {
	Kind  string
	What  string
	Cause error
}

func (e *CliError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s: %v", e.Kind, e.What, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Kind, e.What)
}

func (e *CliError) Unwrap() error { return e.Cause }

// Is allows errors.Is to match the underlying cause.
func (e *CliError) Is(target error) bool {
	return errors.Is(e.Cause, target)
}

// ConfigError is the typed error for a bad config file. Per
// REQ-CLI-ERR-101.
type ConfigError struct {
	Path string
	Err  error
}

func (e *ConfigError) Error() string {
	return fmt.Sprintf("config error in %s: %v", e.Path, e.Err)
}

func (e *ConfigError) Unwrap() error { return e.Err }

// PlanLoadError is the typed error for a bad plan file. Per
// REQ-CLI-ERR-102.
type PlanLoadError struct {
	Path string
	Err  error
}

func (e *PlanLoadError) Error() string {
	return fmt.Sprintf("plan load error in %s: %v", e.Path, e.Err)
}

func (e *PlanLoadError) Unwrap() error { return e.Err }

// DisplayError formats an error for the operator. The output
// is one line per error, with a short prefix that names the
// category. When debug is true, the output includes the
// underlying error chain (errors.Unwrap traversal).
//
// Per REQ-CLI-ERR-001 the message is clear. Per REQ-CLI-ERR-002
// the traceback is printed when --debug is set.
func DisplayError(err error, debug bool) string {
	if err == nil {
		return ""
	}
	out := ""
	if cli, ok := err.(*CliError); ok {
		out = "error: " + cli.Error() + "\n"
	} else if cfg, ok := err.(*ConfigError); ok {
		out = "config error: " + cfg.Error() + "\n"
	} else if pl, ok := err.(*PlanLoadError); ok {
		out = "plan error: " + pl.Error() + "\n"
	} else {
		out = "error: " + err.Error() + "\n"
	}
	if debug {
		for cur := err; cur != nil; cur = errors.Unwrap(cur) {
			out += "  - " + cur.Error() + "\n"
		}
	}
	return out
}

// ExitCodeFor returns the exit code for an error. Per
// REQ-CLI-ERR-003. User errors map to 1; system errors
// (anything else) map to 2. A nil error maps to 0.
func ExitCodeFor(err error) int {
	if err == nil {
		return ExitOK
	}
	var cli *CliError
	if errors.As(err, &cli) {
		return ExitUserError
	}
	var cfg *ConfigError
	if errors.As(err, &cfg) {
		return ExitUserError
	}
	var pl *PlanLoadError
	if errors.As(err, &pl) {
		return ExitUserError
	}
	return ExitSystemError
}
