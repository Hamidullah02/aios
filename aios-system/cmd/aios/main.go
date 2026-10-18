// aios is the CLI binary for the aios-system project.
//
// The binary is a thin shell around the cli.Main function
// exposed by the internal/cli package. The shell hands
// os.Args, os.Stdout, and os.Stderr to cli.Main, then
// returns its exit code. All the real work (argv parsing,
// dependency wiring, subcommand dispatch) lives in the
// cli package; the binary exists so go build can produce
// an executable from cmd/aios.
//
// Layout:
//
//	cmd/aios/main.go   — this file (package main)
//	internal/cli/      — the library the binary calls
package main

import (
	"os"

	"github.com/sadi/aios-system/internal/cli"
)

func main() {
	os.Exit(cli.Main(os.Args, os.Stdout, os.Stderr))
}
