package search

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/sadi/aios-system/internal/sandbox"
	"github.com/sadi/aios-system/internal/tools"
)

// Diagnostic is the typed lint/type-check result
// (REQ-SRH-DAT-002).
type Diagnostic struct {
	Path     string `json:"path"`
	Line     int    `json:"line"`
	Column   int    `json:"column,omitempty"`
	Severity string `json:"severity"`
	Code     string `json:"code,omitempty"`
	Message  string `json:"message"`
}

// DiagnosticsTool runs an external checker and parses the
// output into typed diagnostics (REQ-SRH-DGN-001 to 005).
//
// Args:
//
//	path    (string, optional) — directory or file to check
//	          (default: workspace).
//	checker (string, optional) — one of: "go vet", "gofmt",
//	          "pylint", "eslint". Default: "go vet".
//	extra_args ([]string, optional) — extra args to the checker.
//
// The tool checks for the checker binary on PATH before
// running. Missing binary raises CheckerNotFoundError.
type DiagnosticsTool struct {
	// Runner is the sandbox runner. nil means use the default.
	Runner sandbox.Runner
	// LookPath is the binary lookup. nil defaults to
	// exec.LookPath. Tests can override to bypass PATH.
	LookPath func(name string) (string, error)
}

func (DiagnosticsTool) Name() string { return "diagnostics" }

func (d DiagnosticsTool) Run(args map[string]any, ctx tools.Context) (tools.Result, error) {
	checker, _ := args["checker"].(string)
	if checker == "" {
		checker = "go vet"
	}
	ws := ctx.Workspace
	if ws == "" {
		ws, _ = tools.Workspace()
	}
	path := ws
	if p, _ := args["path"].(string); p != "" {
		path = p
	}
	extra, _ := args["extra_args"].([]string)

	cmd, err := checkerCommand(checker, path, extra)
	if err != nil {
		return tools.Result{OK: false, Summary: "bad checker"}, err
	}
	look := d.LookPath
	if look == nil {
		look = exec.LookPath
	}
	if _, lookErr := look(cmd[0]); lookErr != nil {
		return tools.Result{OK: false, Summary: "checker not found"},
			&CheckerNotFoundError{Name: cmd[0]}
	}

	runner := d.Runner
	if runner == nil {
		// Default to the real subprocess runner. Tests always
		// pass a fake runner, so this branch is exercised by
		// integration callers.
		runner = sandbox.NewSubprocessRunner()
	}
	res, err := runner.Run(sandbox.RunOptions{Command: cmd, Cwd: ws, Timeout: 120})
	diags := parseDiagnostics(checker, res.Stdout, res.Stderr, path)
	return tools.Result{
		OK: err == nil && res.ExitCode == 0,
		Output: map[string]any{
			"diagnostics": diags,
			"count":       len(diags),
			"exit_code":   res.ExitCode,
			"stderr":      res.Stderr,
		},
		Summary: fmt.Sprintf("%d diagnostics from %s", len(diags), checker),
	}, err
}

// checkerCommand returns the argv for a given checker name.
// Unknown checkers return an error.
func checkerCommand(checker, path string, extra []string) ([]string, error) {
	switch checker {
	case "go vet":
		return append([]string{"go", "vet", path}, extra...), nil
	case "gofmt":
		// gofmt -l lists files that need formatting.
		return append([]string{"gofmt", "-l", path}, extra...), nil
	case "pylint":
		return append([]string{"pylint", "--output-format=json", path}, extra...), nil
	case "eslint":
		return append([]string{"eslint", "--format=json", path}, extra...), nil
	}
	return nil, fmt.Errorf("search: diagnostics: unknown checker %q", checker)
}

// parseDiagnostics dispatches to the per-checker parser.
// The caller (Run) passes a name checkerCommand already
// validated, so the switch is exhaustive.
func parseDiagnostics(checker, stdout, stderr, base string) []Diagnostic {
	switch checker {
	case "go vet":
		return parseGoVet(stderr, base)
	case "gofmt":
		return parseGofmt(stdout, base)
	case "pylint":
		return parsePylint(stdout, base)
	case "eslint":
		return parseESLint(stdout, base)
	}
	return nil
}

// goVetLine matches the standard "file:line:col: message"
// format produced by go vet.
var goVetLine = regexp.MustCompile(`^(.+?):(\d+):(\d+):\s+(.*)$`)

func parseGoVet(stderr, base string) []Diagnostic {
	var out []Diagnostic
	for _, line := range strings.Split(stderr, "\n") {
		m := goVetLine.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		out = append(out, Diagnostic{
			Path:     relPath(m[1], base),
			Line:     atoi(m[2]),
			Column:   atoi(m[3]),
			Severity: "warning",
			Message:  m[4],
		})
	}
	return out
}

// parseGofmt treats each listed path as a "format" diagnostic
// on line 1.
func parseGofmt(stdout, base string) []Diagnostic {
	var out []Diagnostic
	for _, line := range strings.Split(strings.TrimSpace(stdout), "\n") {
		if line == "" {
			continue
		}
		out = append(out, Diagnostic{
			Path:     relPath(line, base),
			Line:     1,
			Severity: "info",
			Code:     "gofmt",
			Message:  "file needs formatting",
		})
	}
	return out
}

// pylintLine is one row of the pylint JSON array.
type pylintLine struct {
	Path    string `json:"path"`
	Line    int    `json:"line"`
	Column  int    `json:"column"`
	Type    string `json:"type"`
	Symbol  string `json:"symbol"`
	Message string `json:"message"`
}

func parsePylint(stdout, base string) []Diagnostic {
	var rows []pylintLine
	if err := json.Unmarshal([]byte(stdout), &rows); err != nil {
		return nil
	}
	out := make([]Diagnostic, 0, len(rows))
	for _, r := range rows {
		out = append(out, Diagnostic{
			Path:     relPath(r.Path, base),
			Line:     r.Line,
			Column:   r.Column,
			Severity: r.Type,
			Code:     r.Symbol,
			Message:  r.Message,
		})
	}
	return out
}

// eslintMsg is one entry of the ESLint JSON results array.
type eslintMsg struct {
	FilePath string `json:"filePath"`
	Line     int    `json:"line"`
	Column   int    `json:"column"`
	Severity int    `json:"severity"` // 1=warn, 2=error
	RuleID   string `json:"ruleId"`
	Message  string `json:"message"`
}

func parseESLint(stdout, base string) []Diagnostic {
	var wrap struct {
		Results []struct {
			FilePath string      `json:"filePath"`
			Messages []eslintMsg `json:"messages"`
		} `json:"results"`
	}
	if err := json.Unmarshal([]byte(stdout), &wrap); err != nil {
		return nil
	}
	var out []Diagnostic
	for _, r := range wrap.Results {
		for _, m := range r.Messages {
			sev := "warning"
			if m.Severity == 2 {
				sev = "error"
			}
			out = append(out, Diagnostic{
				Path:     relPath(r.FilePath, base),
				Line:     m.Line,
				Column:   m.Column,
				Severity: sev,
				Code:     m.RuleID,
				Message:  m.Message,
			})
		}
	}
	return out
}

func atoi(s string) int {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0
		}
		n = n*10 + int(c-'0')
	}
	return n
}

// relPath returns the path relative to base when possible;
// otherwise the original path.
func relPath(p, base string) string {
	if base == "" {
		return p
	}
	if r, err := filepath.Rel(base, p); err == nil {
		return r
	}
	return p
}
