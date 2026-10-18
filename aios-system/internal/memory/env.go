package memory

import "os"

// getenv is the os.Getenv wrapper. The function is
// the one place the package reads the environment.
// The wrapper is split out so tests can swap the
// environment without touching the real one.
func getenv(name string) string {
	return os.Getenv(name)
}
