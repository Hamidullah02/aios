// Package version reports which build of aios this is.
//
// The version is stamped at build time with -ldflags
// "-X github.com/Hamidullah02/aios/aios-system/internal/version.Version=v0.57.0".
// Without that stamp the binary falls back to the module version the
// Go toolchain embedded in it, and reports "dev" only when there is
// no module version either -- a plain `go build` in a working tree.
//
// Why this exists: a user reports a bug and says "I'm on the
// latest". The latest is only meaningful if the binary can say
// what it is. Without a version command, that conversation
// starts with "can you run aios --version", and the answer is an
// error.
package version

import (
	"runtime/debug"
	"strings"
)

// Version is the released version of this build.
//
// It is a var rather than a const so the linker can overwrite it.
// The release workflow stamps it; a build that was not stamped
// leaves it at the value below and String falls back to the module
// version the toolchain embedded.
var Version = "dev"

// Commit is the git commit the build came from, when the build
// system stamped it. Empty otherwise.
var Commit = ""

// String returns the one-line version a user sees.
//
// The commit is included when it is known, because two builds
// from the same tag can differ: a release built twice, or a
// release built from a branch that was tagged after the fact.
func String() string {
	s := "aios " + version()
	if Commit != "" {
		s += " (" + Commit + ")"
	}
	return s
}

// version returns the version this build should report.
//
// The stamped value wins. When nothing was stamped -- which is the
// case for every binary installed with `go install`, because the Go
// toolchain does not apply the release workflow's -ldflags -- the
// module version is read out of the build info instead.
//
// This matters because `go install` is a documented install method.
// A binary installed that way used to report "aios dev", so the
// first question in a bug report -- which build is this? -- had no
// answer, and the answer it gave was wrong: it was not a dev build,
// it was a released one.
//
// A module version of "(devel)" means the build came from a working
// tree rather than a tagged module, so there is nothing honest to
// report but "dev".
func version() string {
	if Version != "" && Version != "dev" {
		return Version
	}
	if moduleVersion := moduleVersionFromBuildInfo(); moduleVersion != "" {
		return moduleVersion
	}
	return "dev"
}

// moduleVersionFromBuildInfo returns the version of the main module
// as the Go toolchain recorded it, or "" when it is not a tagged
// build.
func moduleVersionFromBuildInfo() string {
	info, ok := debug.ReadBuildInfo()
	if !ok || info.Main.Version == "" {
		return ""
	}
	v := strings.TrimSpace(info.Main.Version)
	if v == "" || v == "(devel)" {
		return ""
	}
	return v
}
