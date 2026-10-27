// Package version reports which build of aios this is.
//
// The version is stamped at build time with -ldflags
// "-X github.com/Hamidullah02/aios/aios-system/internal/version.Version=v0.56.0".
// Without that stamp the binary reports "dev", which is the
// honest answer for a build made with plain `go build`.
//
// Why this exists: a user reports a bug and says "I'm on the
// latest". The latest is only meaningful if the binary can say
// what it is. Without a version command, that conversation
// starts with "can you run aios --version", and the answer is an
// error.
package version

// Version is the released version of this build.
//
// It is a var rather than a const so the linker can overwrite it.
// The release workflow stamps it; a local build leaves it at the
// value below, which is "dev" -- the truth about a binary nobody
// stamped.
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
	s := "aios " + Version
	if Commit != "" {
		s += " (" + Commit + ")"
	}
	return s
}
