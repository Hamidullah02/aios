package cli

import (
	"bytes"
	"flag"
	"testing"
)

// The three defects in this file were found by DRIVING the
// product (probe1.ps1, probe2.ps1, probe5.ps1), not by reading
// the parser. Each test below pins the behaviour the operator
// typed.

// TestParse_GlobalFlagAfterSubcommand is the regression gate
// for the parser bug.
//
// The global flags were parsed from the PREFIX only, and
// everything after the subcommand was handed to the
// subcommand's flag set. That set only knows the flags that
// subcommand declares, so every global written after the
// subcommand was rejected:
//
//	aios approve <id> --socket=path
//	aios daemon start --socket=path --pidfile=path
//
// failed with "flag provided but not defined: -socket". The
// second one is the invocation the tool's own help text
// advertises, so the documented command did not work.
func TestParse_GlobalFlagAfterSubcommand(t *testing.T) {
	cases := []struct {
		name string
		argv []string
		want func(*testing.T, *Args)
	}{
		{
			name: "socket after approve",
			argv: []string{"aios", "approve", "someid", "--socket=/tmp/x.sock"},
			want: func(t *testing.T, a *Args) {
				if a.SocketPath != "/tmp/x.sock" {
					t.Errorf("SocketPath = %q, want /tmp/x.sock", a.SocketPath)
				}
				if !a.SocketExplicit {
					t.Error("SocketExplicit = false, want true: naming a socket means the daemon")
				}
				if len(a.Positional) != 1 || a.Positional[0] != "someid" {
					t.Errorf("Positional = %v, want [someid]", a.Positional)
				}
			},
		},
		{
			name: "socket after daemon start",
			argv: []string{"aios", "daemon", "start", "--socket=/tmp/x.sock", "--pidfile=/tmp/x.pid"},
			want: func(t *testing.T, a *Args) {
				if a.SocketPath != "/tmp/x.sock" {
					t.Errorf("SocketPath = %q, want /tmp/x.sock", a.SocketPath)
				}
				if a.DaemonPID != "/tmp/x.pid" {
					t.Errorf("DaemonPID = %q, want /tmp/x.pid", a.DaemonPID)
				}
				if a.Subcommand != DaemonStart {
					t.Errorf("Subcommand = %q, want %q", a.Subcommand, DaemonStart)
				}
			},
		},
		{
			name: "config after audit",
			argv: []string{"aios", "audit", "--config=./var/config.yaml"},
			want: func(t *testing.T, a *Args) {
				if a.ConfigPath != "./var/config.yaml" {
					t.Errorf("ConfigPath = %q, want ./var/config.yaml", a.ConfigPath)
				}
			},
		},
		{
			name: "debug after run",
			argv: []string{"aios", "run", "plan.yaml", "--debug"},
			want: func(t *testing.T, a *Args) {
				if !a.Debug {
					t.Error("Debug = false, want true")
				}
				if a.PlanPath != "plan.yaml" {
					t.Errorf("PlanPath = %q, want plan.yaml", a.PlanPath)
				}
			},
		},
		{
			name: "separated socket value after approve",
			argv: []string{"aios", "approve", "someid", "--socket", "/tmp/x.sock"},
			want: func(t *testing.T, a *Args) {
				if a.SocketPath != "/tmp/x.sock" {
					t.Errorf("SocketPath = %q, want /tmp/x.sock", a.SocketPath)
				}
				if !a.SocketExplicit {
					t.Error("SocketExplicit = false, want true")
				}
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a, err := Parse(tc.argv, &bytes.Buffer{})
			if err != nil {
				t.Fatalf("Parse(%v): %v", tc.argv, err)
			}
			tc.want(t, a)
		})
	}
}

// TestParse_GlobalFlagsBeforeAndAfterSubcommand pins that both
// positions work at the same time, and that the FIRST spelling
// wins when the same global appears twice.
func TestParse_GlobalFlagsBeforeAndAfterSubcommand(t *testing.T) {
	a, err := Parse([]string{
		"aios", "--config=./first.yaml", "audit", "--log=./second.db",
	}, &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	if a.ConfigPath != "./first.yaml" {
		t.Errorf("ConfigPath = %q, want ./first.yaml", a.ConfigPath)
	}
	if a.LogPath != "./second.db" {
		t.Errorf("LogPath = %q, want ./second.db", a.LogPath)
	}
}

// TestParse_GlobalYesIsNotClobbered is the regression gate for
// the safety-adjacent flag bug.
//
// `--yes` was registered on BOTH the global flag set and the
// run/fix/resume flag sets, and the flag package's BoolVar
// writes its default into the target when it registers. The
// subcommand's registration therefore reset a.AssumeYes back
// to false AFTER the global prefix had set it to true:
//
//	aios --yes run plan.yaml
//
// still prompted, then waited a minute for an answer, and
// finally treated the silence as a denial. The flag that says
// "no human will decide each request" did not stop the human
// prompt.
func TestParse_GlobalYesIsNotClobbered(t *testing.T) {
	cases := []struct {
		name string
		argv []string
	}{
		{"global position", []string{"aios", "--yes", "run", "plan.yaml"}},
		{"subcommand position", []string{"aios", "run", "plan.yaml", "--yes"}},
		{"subcommand position, flag before positional", []string{"aios", "run", "--yes", "plan.yaml"}},
		{"both positions", []string{"aios", "--yes", "run", "--yes", "plan.yaml"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a, err := Parse(tc.argv, &bytes.Buffer{})
			if err != nil {
				t.Fatalf("Parse(%v): %v", tc.argv, err)
			}
			if !a.AssumeYes {
				t.Error("AssumeYes = false, want true: --yes was ignored")
			}
		})
	}
}

// TestParse_YesDefaultsToOff pins the other half: without the
// flag, nothing auto-approves.
func TestParse_YesDefaultsToOff(t *testing.T) {
	a, err := Parse([]string{"aios", "run", "plan.yaml"}, &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	if a.AssumeYes {
		t.Error("AssumeYes = true, want false: nothing auto-approves without the flag")
	}
}

// TestParse_SocketExplicitOnlyWhenNamed pins the distinction
// the approval strategy depends on: a socket the operator
// named means the daemon, a default socket means the terminal.
func TestParse_SocketExplicitOnlyWhenNamed(t *testing.T) {
	a, err := Parse([]string{"aios", "run", "plan.yaml"}, &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	if a.SocketExplicit {
		t.Error("SocketExplicit = true, want false: the default socket is not an explicit choice")
	}
}

// TestSplitGlobalFlags covers the helper that decides which
// side of the subcommand a flag belongs to.
func TestSplitGlobalFlags(t *testing.T) {
	globals := globalFlagSetForTest(t)
	cases := []struct {
		name       string
		args       []string
		wantGlobal []string
		wantSub    []string
	}{
		{
			name:       "global flag is pulled out",
			args:       []string{"--socket=/tmp/x.sock"},
			wantGlobal: []string{"--socket=/tmp/x.sock"},
			wantSub:    []string{},
		},
		{
			name:       "separated value stays with the flag",
			args:       []string{"--socket", "/tmp/x.sock"},
			wantGlobal: []string{"--socket", "/tmp/x.sock"},
			wantSub:    []string{},
		},
		{
			name:       "subcommand flag stays with the subcommand",
			args:       []string{"--plan=p.yaml"},
			wantGlobal: []string{},
			wantSub:    []string{"--plan=p.yaml"},
		},
		{
			name:       "positional stays with the subcommand",
			args:       []string{"someid", "--socket=/tmp/x.sock"},
			wantGlobal: []string{"--socket=/tmp/x.sock"},
			wantSub:    []string{"someid"},
		},
		{
			name:       "a boolean global does not swallow the next argument",
			args:       []string{"--yes", "plan.yaml"},
			wantGlobal: []string{"--yes"},
			wantSub:    []string{"plan.yaml"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			global, sub := splitGlobalFlags(tc.args, globals)
			assertArgs(t, "global", global, tc.wantGlobal)
			assertArgs(t, "sub", sub, tc.wantSub)
		})
	}
}

// globalFlagSetForTest builds a flag set with the same global
// flags the parser registers, so the helper is exercised
// against the real arity rules.
func globalFlagSetForTest(t *testing.T) *flag.FlagSet {
	t.Helper()
	fs := flag.NewFlagSet("aios", flag.ContinueOnError)
	fs.String("socket", "", "")
	fs.Bool("yes", false, "")
	fs.String("config", "", "")
	return fs
}

func assertArgs(t *testing.T, label string, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s = %v, want %v", label, got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("%s = %v, want %v", label, got, want)
		}
	}
}
