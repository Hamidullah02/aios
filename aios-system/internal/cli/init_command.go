package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Init scaffolds the workspace. Per REQ-CLI-INI-001 the
// command creates ./var/, the default policy file, the
// default egress file, the default price file, and the
// default config file. Existing files are left alone; only
// missing files are created. The function is idempotent.
//
// The function returns a list of paths that were created (so
// the caller can print them) and a list of paths that already
// existed. Errors from os.MkdirAll / os.WriteFile bubble up
// as *ConfigError or *CliError.
func Init(workspace string) (created []string, existed []string, err error) {
	if workspace == "" {
		workspace = "./var"
	}
	abs, err := filepath.Abs(workspace)
	if err != nil {
		return nil, nil, &CliError{Kind: "init", What: "resolve workspace", Cause: err}
	}

	// Create the directory.
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return nil, nil, &CliError{Kind: "init", What: "create " + abs, Cause: err}
	}

	// The four default files. We pass relative names so the
	// scaffolded paths are relative to the workspace; the
	// caller (or operator) can also pass absolute paths.
	files := []struct {
		rel  string
		body string
	}{
		// The policy and the egress list are the FIRST-RUN
		// ones, not the read-only defaults. `aios init` and the
		// implicit first run must produce the same workspace:
		// an operator who runs init and an operator who does not
		// should end up with a tool that behaves the same way.
		//
		// The read-only default was wrong for the first run of
		// a tool whose job is to do what was asked: the operator
		// said "write the word hello into a file called hi.txt"
		// and the run failed with "write denied by policy: no
		// rule matched; default deny".
		//
		// The policy is generated from the registered tool set,
		// so it cannot name a tool that does not exist or omit
		// one that does.
		{"policy.yaml", firstRunPolicyYAMLFor(productionToolSet(nil, nil))},
		{"egress.yaml", firstRunEgressYAML},
		{"prices.yaml", defaultPricesYAML},
		{"config.yaml", defaultConfigYAML},
	}
	for _, f := range files {
		path := filepath.Join(abs, f.rel)
		if _, statErr := os.Stat(path); statErr == nil {
			existed = append(existed, path)
			continue
		}
		if err := os.WriteFile(path, []byte(f.body), 0o644); err != nil {
			return created, existed, &ConfigError{Path: path, Err: err}
		}
		created = append(created, path)
	}
	return created, existed, nil
}

// PrintInitReport is a small helper that prints a one-line
// summary of the init run.
func PrintInitReport(w interface{ Write([]byte) (int, error) }, created, existed []string) {
	if w == nil {
		return
	}
	if len(created) > 0 {
		fmt.Fprintln(w, "Created:")
		for _, p := range created {
			fmt.Fprintln(w, "  "+p)
		}
	}
	if len(existed) > 0 {
		fmt.Fprintln(w, "Already present:")
		for _, p := range existed {
			fmt.Fprintln(w, "  "+p)
		}
	}
	if len(created) == 0 && len(existed) == 0 {
		fmt.Fprintln(w, "Nothing to do.")
		return
	}
	fmt.Fprintln(w, "Next: edit the policy + prices, then `aios run <plan.yaml>`.")
}

const defaultPolicyYAML = `# aios default policy. REQ-POL-DAT-001.
# Deny-by-default. Allow common read-only operations.
# Every rule has an "id" (required by the policy
# validator), a "match" block, and an "action". The
# "ask" rule carries an "approval" template so the
# approval daemon knows what to bind.
default: deny
rules:
  - id: allow-read
    match:
      tool: read_file
    action: allow
  - id: allow-search
    match:
      tool: search_files
    action: allow
  - id: ask-shell
    match:
      tool: run_shell
    action: ask
    reason: shell access is gated by the approval daemon
    approval:
      action: shell
      # Tool arguments are referenced as {arg.<name>}, not a
      # bare {<name>}. This shipped as {command} and the
      # template engine rejects it ("unknown placeholder"), so
      # the policy this command writes could not be loaded by
      # the very tool it gated: every run_shell call failed
      # with a config error instead of raising the approval
      # prompt this rule exists to gate. The reserved names
      # are task, tool, path, host, actor, tier.
      resource: "{arg.command}"
      reason: "run a shell command"
  - id: allow-ask-user
    match:
      tool: ask_user
    action: allow
`

const defaultEgressYAML = `# aios default egress policy. REQ-EGS-DAT-001.
# Empty allow list = deny everything. Add the hosts you need.
allow: []
deny: []
headers:
  remove: []
  add: {}
`

const defaultPricesYAML = `# aios default price table. REQ-MOD-PRC-001.
# Add your models here. The key is the model id; values are
# USD per 1k tokens. The map-shaped format is what
# models.LoadPriceTable reads; a list-shaped file is
# silently treated as empty (v0.33.0 fix: the previous
# scaffold wrote a list the loader could not read).
#
# The OpenRouter models are free, so their price is zero.
# A model with no price row is a configuration error: the
# router refuses the call rather than reporting a cost it
# cannot know.
prices:
  gpt-4o:
    input_per_1k: 0.005
    output_per_1k: 0.015
  claude-sonnet-4-5:
    input_per_1k: 0.003
    output_per_1k: 0.015
  nvidia/nemotron-3-ultra-550b-a55b:free:
    input_per_1k: 0
    output_per_1k: 0
  nvidia/nemotron-3-super-120b-a12b:free:
    input_per_1k: 0
    output_per_1k: 0
`

const defaultConfigYAML = `# aios default config. REQ-CLI-DAT-001.
defaults:
  workspace: ./var
  log_level: info
`

// joinLines is a small helper used by the init helper tests.
func joinLines(parts ...string) string {
	return strings.Join(parts, "\n")
}
