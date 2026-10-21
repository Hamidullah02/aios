package cli

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// policyPath resolves the policy file path from the
// user's --config flag. The resolution rules are:
//
//  1. If --config is empty, return an error.
//  2. If --config points at a file whose basename is
//     "policy.yaml" or "policy.yml", return --config
//     unchanged. This is the "operator passed a
//     policy file directly" path; tests rely on it.
//  3. Otherwise, look for a sibling file named
//     "policy.yaml" in the directory of --config.
//     If it exists, return its absolute path.
//  4. If no sibling exists, fall back to --config
//     itself. The pre-v0.36.0 behaviour read
//     --config as a policy file; the fallback keeps
//     that working so an operator who put the rules
//     directly in a single YAML file is not broken.
//
// This is the v0.36.0 fix for a v0.32.0 wiring bug:
// the CLI used to call policy.Load(args.ConfigPath)
// directly, which read var/config.yaml as a policy
// file. config.yaml holds workspace defaults, not
// rules; the loader found no `rules:` block and
// returned empty, so the bus default-denied every
// tool call.
//
// The canonical layout `aios init` scaffolds is:
//
//	var/
//	  config.yaml     # workspace + log defaults
//	  policy.yaml     # the policy rules
//	  prices.yaml     # model price table
//	  egress.yaml     # egress allow list
//
// policyPath makes that layout the source of truth
// without breaking callers that pass a single file.
func policyPath(configPath string) (string, error) {
	if configPath == "" {
		return "", errors.New("policy path: --config is empty")
	}
	abs, err := filepath.Abs(configPath)
	if err != nil {
		return "", err
	}
	// Rule 2: --config already names a policy file.
	base := strings.ToLower(filepath.Base(abs))
	if base == "policy.yaml" || base == "policy.yml" {
		return abs, nil
	}
	// Rule 3: --config points at config.yaml (or
	// anything else); look for a sibling policy.yaml.
	candidate := filepath.Join(filepath.Dir(abs), "policy.yaml")
	if _, err := os.Stat(candidate); err == nil {
		return candidate, nil
	}
	// Rule 4: no sibling; fall back to --config itself.
	return abs, nil
}
