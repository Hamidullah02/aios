package version

import "testing"

// TestString_ReportsTheStampedVersion pins that a stamped build
// reports what the linker put there.
func TestString_ReportsTheStampedVersion(t *testing.T) {
	oldVersion, oldCommit := Version, Commit
	t.Cleanup(func() { Version, Commit = oldVersion, oldCommit })

	Version = "v1.2.3"
	Commit = ""
	if got := String(); got != "aios v1.2.3" {
		t.Errorf("String() = %q, want %q", got, "aios v1.2.3")
	}

	Version = "v1.2.3"
	Commit = "abc1234"
	got := String()
	if got != "aios v1.2.3 (abc1234)" {
		t.Errorf("String() = %q, want the version and the commit", got)
	}
}

// TestString_FallsBackToTheModuleVersion pins that a binary built
// without the linker stamp still says which version it is.
//
// This is the `go install` case. The Go toolchain records the
// module version in the binary's build info, but it does not apply
// the release workflow's -ldflags, so the stamp is absent. Before
// this, such a binary reported "aios dev" -- which is both
// unhelpful and wrong: it was a released build.
//
// The test binary carries "(devel)" for the main module, so the
// fallback cannot find a version and the honest answer is "dev".
// That is the right answer for a build from a working tree, and it
// is what the test pins.
func TestString_FallsBackToTheModuleVersion(t *testing.T) {
	oldVersion, oldCommit := Version, Commit
	t.Cleanup(func() { Version, Commit = oldVersion, oldCommit })

	Version = "dev"
	Commit = ""

	got := String()
	if got == "" {
		t.Fatal("String() returned nothing")
	}
	if got != "aios dev" {
		t.Errorf("String() = %q, want %q for a test binary with no module version", got, "aios dev")
	}
}

// TestModuleVersionFromBuildInfo_NeverReturnsDevel pins that the
// fallback does not report "(devel)".
//
// "(devel)" is what the Go toolchain records for a build from a
// working tree. Printing it would be noise in a bug report, and it
// is not a version a user could look up.
func TestModuleVersionFromBuildInfo_NeverReturnsDevel(t *testing.T) {
	if got := moduleVersionFromBuildInfo(); got == "(devel)" {
		t.Error("moduleVersionFromBuildInfo() = \"(devel)\", want a real version or \"\"")
	}
}

// TestVersion_StampedValueWinsOverTheModuleVersion pins that the
// stamp takes precedence.
//
// The release workflow stamps the tag, and the tag is the version
// the release page and the installer both name. A stamped build
// must not be overridden by whatever the module version says.
func TestVersion_StampedValueWinsOverTheModuleVersion(t *testing.T) {
	oldVersion := Version
	t.Cleanup(func() { Version = oldVersion })

	Version = "v9.9.9"
	if got := version(); got != "v9.9.9" {
		t.Errorf("version() = %q, want the stamped %q", got, "v9.9.9")
	}
}
