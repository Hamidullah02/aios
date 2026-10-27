package search

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/Hamidullah02/aios/aios-system/internal/tools"
)

// Match is the typed content match (REQ-SRH-DAT-001).
type Match struct {
	Path   string   `json:"path"`
	Line   int      `json:"line"`
	Text   string   `json:"text"`
	Before []string `json:"context_before,omitempty"`
	After  []string `json:"context_after,omitempty"`
}

// GrepTool searches file content with a regex
// (REQ-SRH-GRP-001 to 009).
//
// Args:
//
//	pattern     (string, required) — regex.
//	path        (string, optional) — directory or file to search
//	              (default: workspace root).
//	glob        (string, optional) — restrict to files matching
//	              this basename glob (e.g. "*.go").
//	output_mode (string, optional) — "content" (default),
//	              "files_with_matches", or "count".
//	show_line_numbers (bool, optional) — prepend ":lineno" to
//	              each match line. Always on for "content".
//	max_results (int,    optional) — cap on matches (default 1000).
//	timeout     (duration string, optional) — wall-clock budget,
//	              default "30s".
type GrepTool struct{}

func (GrepTool) Name() string { return "grep" }

func (GrepTool) Run(args map[string]any, ctx tools.Context) (tools.Result, error) {
	pattern, _ := args["pattern"].(string)
	if pattern == "" {
		return tools.Result{OK: false, Summary: "missing pattern"},
			fmt.Errorf("search: grep: pattern is required")
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return tools.Result{OK: false, Summary: "bad regex"}, err
	}
	mode, _ := args["output_mode"].(string)
	if mode == "" {
		mode = "content"
	}
	globPat, _ := args["glob"].(string)
	max := 1000
	if v, ok := args["max_results"].(int); ok && v > 0 {
		max = v
	}
	timeoutStr, _ := args["timeout"].(string)
	if timeoutStr == "" {
		timeoutStr = "30s"
	}
	timeout, err := time.ParseDuration(timeoutStr)
	if err != nil {
		return tools.Result{OK: false, Summary: "bad timeout"}, err
	}
	ws := ctx.Workspace
	if ws == "" {
		ws, _ = tools.Workspace()
	}
	wsAbs, _ := filepath.Abs(ws)
	ig := loadIgnore(wsAbs)

	root := wsAbs
	if p, _ := args["path"].(string); p != "" {
		if filepath.IsAbs(p) {
			root = filepath.Clean(p)
		} else {
			root = filepath.Clean(filepath.Join(wsAbs, p))
		}
	}
	deadline := time.Now().Add(timeout)

	var matches []Match
	filesHit := map[string]struct{}{}
	filesCount := map[string]int{}

	walkErr := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		// Stop on timeout.
		if time.Now().After(deadline) {
			return filepath.SkipAll
		}
		rel, _ := filepath.Rel(wsAbs, path)
		rel = filepath.ToSlash(rel)
		if rel != "." && ig.match(rel) {
			return nil
		}
		if globPat != "" {
			if ok, _ := filepath.Match(globPat, filepath.Base(path)); !ok {
				return nil
			}
		}
		count, contentMatches := scanFile(path, re, deadline)
		if count == 0 {
			return nil
		}
		filesHit[rel] = struct{}{}
		filesCount[rel] = count
		if mode == "content" {
			matches = append(matches, contentMatches...)
			if len(matches) >= max {
				return filepath.SkipAll
			}
		}
		return nil
	})
	if walkErr != nil {
		return tools.Result{OK: false, Summary: "walk error"}, walkErr
	}
	if time.Now().After(deadline) && len(matches) == 0 && mode == "content" {
		return tools.Result{OK: false, Summary: "timeout"},
			&GrepTimeoutError{Pattern: pattern, Timeout: timeoutStr}
	}

	switch mode {
	case "files_with_matches":
		out := make([]string, 0, len(filesHit))
		for f := range filesHit {
			out = append(out, f)
		}
		sortStrings(out)
		return tools.Result{
			OK:      true,
			Output:  map[string]any{"files": out, "count": len(out)},
			Summary: fmt.Sprintf("%d files", len(out)),
		}, nil
	case "count":
		out := make(map[string]int, len(filesCount))
		for k, v := range filesCount {
			out[k] = v
		}
		return tools.Result{
			OK:      true,
			Output:  map[string]any{"counts": out, "total": len(filesCount)},
			Summary: fmt.Sprintf("%d files matched", len(filesCount)),
		}, nil
	}
	// content mode
	if len(matches) > max {
		matches = matches[:max]
	}
	lines := make([]string, 0, len(matches))
	for _, m := range matches {
		lines = append(lines, fmt.Sprintf("%s:%d:%s", m.Path, m.Line, m.Text))
	}
	return tools.Result{
		OK:      true,
		Output:  map[string]any{"matches": matches, "lines": lines, "count": len(matches)},
		Summary: fmt.Sprintf("%d matches", len(matches)),
	}, nil
}

// scanFile reads the file line-by-line and returns the
// matching lines. The deadline caps the wall clock.
func scanFile(path string, re *regexp.Regexp, deadline time.Time) (int, []Match) {
	f, err := os.Open(path)
	if err != nil {
		return 0, nil
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	var out []Match
	count := 0
	lineNo := 0
	for sc.Scan() {
		lineNo++
		if time.Now().After(deadline) {
			break
		}
		text := sc.Text()
		if re.MatchString(text) {
			count++
			out = append(out, Match{Path: filepath.ToSlash(path), Line: lineNo, Text: text})
		}
	}
	return count, out
}
