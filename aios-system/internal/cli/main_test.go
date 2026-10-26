package cli

import (
	"os"
	"testing"
)

// TestMain keeps the package's tests hermetic.
//
// The CLI loads a local .env file, which is what an operator
// expects. It also means that a developer running the tests on
// their own machine would have their real API keys loaded into
// the test process: the "no API key" tests would then make real
// network calls to a real provider, and the suite would depend
// on the developer's network and would cost them money.
//
// Turning the loader off for the whole package is the fix. The
// tests that exercise the loader call applyDotEnvFile directly,
// so they still cover the code -- they just do not have it
// switched on underneath them.
func TestMain(m *testing.M) {
	os.Setenv(EnvNoDotEnv, "1")
	os.Exit(m.Run())
}
