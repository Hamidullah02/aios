package cli

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// dotenvFiles are the names searched for a local environment
// file, in order.
//
// The file is a convenience, not a requirement: an operator who
// exports OPENROUTER_API_KEY in their shell never has one. But a
// key pasted into a file is the most common way an operator
// hands a secret to a tool, and a tool that ignores that file
// sends them looking for a mechanism that does not exist.
var dotenvFiles = []string{".env", ".env.local"}

// EnvNoDotEnv turns the .env loader off.
//
// It exists for two reasons. An operator may have a .env in
// their project that holds a DIFFERENT project's keys and does
// not want it read. And the test suite needs the loader to be
// off, because a developer's own .env would otherwise put real
// credentials into the tests and send them over the network.
//
// Any non-empty value disables the loader.
const EnvNoDotEnv = "AIOS_NO_DOTENV"

// dotEnvDisabled reports whether the loader is switched off.
func dotEnvDisabled() bool {
	return os.Getenv(EnvNoDotEnv) != ""
}

// loadDotEnv reads a local .env file and puts its variables into
// the process environment.
//
// It is deliberately forgiving and deliberately narrow:
//
//   - A missing file is not an error. There is no .env in most
//     workspaces, and that is the normal case.
//   - A malformed line is skipped, not fatal. One bad line in a
//     secret file must not stop the tool from starting.
//   - An existing environment variable WINS over the file. The
//     shell is the more explicit signal: an operator who
//     exported a key for one command means that key, and a file
//     left lying around from another project must not override
//     it. An EMPTY environment variable does not count as
//     existing: an empty key is not a key, and treating it as
//     one would leave the operator with a tool that says
//     "OPENROUTER_API_KEY is not set" while their .env file
//     holds the key right there.
//
// The file is searched for in the directory of the --config
// file first (that is where an operator's project lives), then
// in the working directory and its ancestors.
//
// The function returns the paths it loaded, so the caller can
// mention them. It never returns an error: a file that cannot be
// read is a file that is not there.
func loadDotEnv(configPath string) []string {
	if dotEnvDisabled() {
		return nil
	}
	var loaded []string
	seen := make(map[string]bool)
	for _, dir := range dotenvSearchDirs(configPath) {
		if seen[dir] {
			continue
		}
		seen[dir] = true
		for _, name := range dotenvFiles {
			path := filepath.Join(dir, name)
			if applyDotEnvFile(path) {
				loaded = append(loaded, path)
			}
		}
	}
	return loaded
}

// configPathFromArgv pulls the --config value out of argv.
//
// The .env search order depends on where the config file is,
// and the argv has to be inspected before the parse because the
// parse is what happens after the .env is loaded.
//
// Both spellings are accepted: "--config=path" and
// "--config path". Anything else -- a subcommand named
// "config", say -- is ignored.
func configPathFromArgv(argv []string) string {
	for i, a := range argv {
		if v, ok := strings.CutPrefix(a, "--config="); ok {
			return v
		}
		if a == "--config" && i+1 < len(argv) {
			return argv[i+1]
		}
	}
	return ""
}

// dotenvSearchDirs returns the directories to search, nearest
// first.
//
// Two roots are walked: the config file's directory and the
// working directory. From each, the whole ancestor chain is
// searched, because the project's .env usually sits at the
// project root while the tool is run from a subdirectory
// (a workspace, a temp dir) or from somewhere else entirely.
//
// The config file's directory is the more reliable of the two.
// It names the project the operator pointed the tool at, and
// the .env for that project is somewhere above it. The working
// directory alone finds nothing when the operator runs the tool
// from outside the project.
//
// The walk stops at the filesystem root.
func dotenvSearchDirs(configPath string) []string {
	var roots []string
	if configPath != "" {
		if abs, err := filepath.Abs(configPath); err == nil {
			roots = append(roots, filepath.Dir(abs))
		}
	}
	if wd, err := os.Getwd(); err == nil {
		roots = append(roots, wd)
	}

	var dirs []string
	seen := make(map[string]bool)
	for _, root := range roots {
		for dir := root; ; {
			if !seen[dir] {
				seen[dir] = true
				dirs = append(dirs, dir)
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}
	return dirs
}

// applyDotEnvFile reads one file and applies its variables.
// It reports whether the file existed and was applied.
//
// The return value is false for a missing file, an unreadable
// file, and a file with no usable variables. The caller uses it
// only to decide what to mention, so collapsing those cases is
// fine.
func applyDotEnvFile(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()

	applied := false
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		key, value, ok := parseDotEnvLine(scanner.Text())
		if !ok {
			continue
		}
		// The shell wins. See the doc comment.
		if v, exists := os.LookupEnv(key); exists && v != "" {
			continue
		}
		if err := os.Setenv(key, value); err == nil {
			applied = true
		}
	}
	return applied
}

// parseDotEnvLine parses one line of a .env file.
//
// The format is the common one:
//
//	KEY=value
//	KEY="quoted value"
//	KEY='single quoted value'
//	# a comment
//
// It returns the key and the value, and whether the line was
// usable. Blank lines, comments, lines with no "=", and lines
// with an empty key are all unusable.
//
// The value's surrounding quotes are stripped, and a double-quoted
// value has its \n and \t escapes expanded. Single quotes are
// literal, which is what shell quoting means.
//
// A UTF-8 BOM at the start of the line is stripped. Windows
// tools write one: Notepad's "UTF-8" encoding, PowerShell's
// Set-Content -Encoding utf8, and the redirect operator in
// older shells all prepend it. Without this, the first key in
// the file is read as "\ufeffOPENROUTER_API_KEY" and set under
// that name -- a variable nothing will ever read. The file
// looks correct to the operator and the key is simply absent,
// so the tool reports that no key is configured while the key
// sits right there in the file.
func parseDotEnvLine(line string) (string, string, bool) {
	trimmed := strings.TrimPrefix(line, "\ufeff")
	trimmed = strings.TrimSpace(trimmed)
	if trimmed == "" || strings.HasPrefix(trimmed, "#") {
		return "", "", false
	}
	// An "export " prefix is common in files people hand-edit.
	if strings.HasPrefix(trimmed, "export ") {
		trimmed = strings.TrimSpace(strings.TrimPrefix(trimmed, "export "))
	}
	eq := strings.Index(trimmed, "=")
	if eq <= 0 {
		return "", "", false
	}
	key := strings.TrimSpace(trimmed[:eq])
	if key == "" {
		return "", "", false
	}
	value := strings.TrimSpace(trimmed[eq+1:])
	// A trailing comment is only stripped when the value is
	// unquoted: in a quoted value "#" is part of the value.
	if len(value) > 0 && value[0] != '"' && value[0] != '\'' {
		if hash := strings.Index(value, " #"); hash >= 0 {
			value = strings.TrimSpace(value[:hash])
		}
	}
	switch {
	case len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"':
		value = expandEscapes(value[1 : len(value)-1])
	case len(value) >= 2 && value[0] == '\'' && value[len(value)-1] == '\'':
		value = value[1 : len(value)-1]
	}
	return key, value, true
}

// expandEscapes expands the escapes a double-quoted value may
// carry.
//
// Only the two that matter in practice are handled. A \n in a
// secret is almost always a mistake, and expanding it silently
// would corrupt the key; leaving it literal would send a broken
// key to the provider. Expanding is the more useful reading.
func expandEscapes(s string) string {
	s = strings.ReplaceAll(s, `\n`, "\n")
	s = strings.ReplaceAll(s, `\t`, "\t")
	s = strings.ReplaceAll(s, `\"`, `"`)
	return s
}
