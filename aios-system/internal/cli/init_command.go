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
		{"policy.yaml", defaultPolicyYAML},
		{"egress.yaml", defaultEgressYAML},
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
default: deny
rules:
  - tool: read_file
    action: allow
  - tool: search_files
    action: allow
  - tool: run_shell
    action: ask
    reason: shell access is gated by the approval daemon
  - tool: ask_user
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
prices:
  gpt-4o:
    input_per_1k: 0.005
    output_per_1k: 0.015
  claude-sonnet-4-5:
    input_per_1k: 0.003
    output_per_1k: 0.015
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
