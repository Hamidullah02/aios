package edit

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/sadi/aios-system/internal/tools"
)

// hunk is one @@ block in a unified diff. OldStart is 1-based
// and OldLen is the number of context+remove lines. The Adds
// and Removes hold the line bodies (no leading marker).
type hunk struct {
	OldStart int
	OldLen   int
	NewStart int
	NewLen   int
	Lines    []hunkLine
}

type hunkLine struct {
	Op   byte // ' ', '-', '+'
	Text string
}

// filePatch is one file's worth of diff (one ---/+++ pair).
type filePatch struct {
	OldPath string
	NewPath string
	Hunks   []hunk
}

// ApplyPatchTool applies a unified diff to one or more files
// (REQ-EDT-APP-001 to 007). The tool parses the diff, reads
// each target file, validates the hunk context, applies the
// hunks, and writes the result.
//
// Args:
//
//	patch (string, required) — the unified diff text.
type ApplyPatchTool struct{}

func (ApplyPatchTool) Name() string { return "apply_patch" }

func (ApplyPatchTool) Run(args map[string]any, ctx tools.Context) (tools.Result, error) {
	patch, _ := args["patch"].(string)
	if patch == "" {
		return tools.Result{OK: false, Summary: "missing patch"},
			fmt.Errorf("edit: patch is required")
	}
	ws, err := tools.Workspace()
	if ctx.Workspace != "" {
		ws = ctx.Workspace
	}
	if err != nil && ctx.Workspace == "" {
		return tools.Result{OK: false, Summary: "no workspace"}, err
	}
	files, err := parsePatch(patch)
	if err != nil {
		return tools.Result{OK: false, Summary: "patch parse"}, err
	}
	type applied struct {
		path string
		diff string
	}
	var out []applied
	for _, f := range files {
		// Strip the optional "a/" or "b/" prefix that git
		// diffs use, then resolve against the workspace.
		old := strings.TrimPrefix(f.OldPath, "a/")
		old = strings.TrimPrefix(old, "b/")
		old = strings.TrimPrefix(old, "/")
		resolved, err := tools.ResolvePath(ws, old)
		if err != nil {
			return tools.Result{OK: false, Summary: "path rejected"}, err
		}
		data, err := os.ReadFile(resolved)
		if err != nil {
			return tools.Result{OK: false, Summary: "read failed"}, err
		}
		before := string(data)
		after, err := applyHunks(before, f.Hunks, resolved)
		if err != nil {
			return tools.Result{OK: false, Summary: "hunk mismatch"}, err
		}
		if after == before {
			out = append(out, applied{path: resolved, diff: ""})
			continue
		}
		info, _ := os.Stat(resolved)
		mode := os.FileMode(0o644)
		if info != nil {
			mode = info.Mode().Perm()
		}
		if err := atomicWrite(resolved, []byte(after), mode); err != nil {
			return tools.Result{OK: false, Summary: "write failed"}, err
		}
		out = append(out, applied{path: resolved, diff: unifiedDiff(before, after)})
	}
	paths := make([]string, 0, len(out))
	for _, a := range out {
		paths = append(paths, a.path)
	}
	return tools.Result{
		OK:      true,
		Output:  map[string]any{"files": paths, "count": len(paths)},
		Summary: fmt.Sprintf("patched %d file(s)", len(paths)),
	}, nil
}

// parsePatch parses a unified diff into file blocks. The
// parser supports the minimal format:
//
//	--- old/path
//	+++ new/path
//	@@ -a,b +c,d @@
//	 context
//	-removed
//	+added
//
// "diff --git" and "index ..." lines are ignored.
func parsePatch(patch string) ([]filePatch, error) {
	lines := strings.Split(patch, "\n")
	var files []filePatch
	var cur *filePatch
	var h *hunk
	flushHunk := func() {
		if h != nil && cur != nil {
			cur.Hunks = append(cur.Hunks, *h)
			h = nil
		}
	}
	flushFile := func() {
		flushHunk()
		if cur != nil {
			files = append(files, *cur)
			cur = nil
		}
	}
	for i, line := range lines {
		lineno := i + 1
		switch {
		case strings.HasPrefix(line, "diff --git"),
			strings.HasPrefix(line, "index "),
			line == "" && cur == nil:
			// skip preamble / blank lines between files
		case strings.HasPrefix(line, "--- "):
			flushFile()
			cur = &filePatch{OldPath: strings.TrimPrefix(line, "--- ")}
		case strings.HasPrefix(line, "+++ "):
			if cur == nil {
				return nil, &PatchParseError{Line: lineno, Msg: "+++ without ---"}
			}
			cur.NewPath = strings.TrimPrefix(line, "+++ ")
		case strings.HasPrefix(line, "@@"):
			flushHunk()
			hk, err := parseHunkHeader(line)
			if err != nil {
				return nil, &PatchParseError{Line: lineno, Msg: err.Error()}
			}
			h = hk
		case h != nil:
			op := byte(' ')
			switch {
			case strings.HasPrefix(line, "-"):
				op = '-'
			case strings.HasPrefix(line, "+"):
				op = '+'
			case strings.HasPrefix(line, " "):
				op = ' '
			default:
				// Not a recognised hunk line — end the hunk.
				flushHunk()
				continue
			}
			h.Lines = append(h.Lines, hunkLine{Op: op, Text: line[1:]})
		}
	}
	flushFile()
	if len(files) == 0 {
		return nil, &PatchParseError{Msg: "no file blocks found"}
	}
	return files, nil
}

// parseHunkHeader parses "@@ -a,b +c,d @@" or "@@ -a +c @@".
func parseHunkHeader(line string) (*hunk, error) {
	line = strings.TrimPrefix(line, "@@")
	line = strings.SplitN(line, "@@", 2)[0]
	parts := strings.Fields(strings.TrimSpace(line))
	if len(parts) != 2 {
		return nil, fmt.Errorf("malformed hunk header")
	}
	old, err := parseRange(parts[0])
	if err != nil {
		return nil, fmt.Errorf("old range: %w", err)
	}
	new, err := parseRange(parts[1])
	if err != nil {
		return nil, fmt.Errorf("new range: %w", err)
	}
	return &hunk{
		OldStart: old.start, OldLen: old.length,
		NewStart: new.start, NewLen: new.length,
	}, nil
}

type rng struct {
	start, length int
}

func parseRange(s string) (rng, error) {
	s = strings.TrimPrefix(s, "-")
	s = strings.TrimPrefix(s, "+")
	parts := strings.SplitN(s, ",", 2)
	start, err := strconv.Atoi(parts[0])
	if err != nil {
		return rng{}, err
	}
	r := rng{start: start, length: 1}
	if len(parts) == 2 {
		n, err := strconv.Atoi(parts[1])
		if err != nil {
			return rng{}, err
		}
		r.length = n
	}
	return r, nil
}

// applyHunks applies every hunk to content. The hunk
// context is matched line-for-line; a mismatch raises
// PatchContextMismatch.
func applyHunks(content string, hunks []hunk, path string) (string, error) {
	lines := strings.Split(content, "\n")
	out := make([]string, 0, len(lines))
	idx := 0 // current line in `lines` (0-based)
	for hi, h := range hunks {
		// Position the cursor at the hunk's start (1-based).
		target := h.OldStart - 1
		if target < idx {
			// Out-of-order hunks are not supported.
			return "", &PatchContextMismatch{Path: path, Hunk: hi, Line: idx + 1,
				Expect: h.Lines[0].Text, Got: lineAt(lines, idx)}
		}
		// Copy unchanged lines between hunks.
		for idx < target {
			out = append(out, lines[idx])
			idx++
		}
		// Match every hunk line.
		for _, hl := range h.Lines {
			if idx >= len(lines) {
				return "", &PatchContextMismatch{Path: path, Hunk: hi, Line: idx + 1,
					Expect: hl.Text, Got: ""}
			}
			if hl.Op == '+' {
				out = append(out, hl.Text)
				continue
			}
			if lines[idx] != hl.Text {
				return "", &PatchContextMismatch{Path: path, Hunk: hi, Line: idx + 1,
					Expect: hl.Text, Got: lines[idx]}
			}
			if hl.Op == ' ' {
				out = append(out, lines[idx])
			}
			// '-' lines are dropped (not added to out).
			idx++
		}
	}
	// Copy the trailing lines.
	for idx < len(lines) {
		out = append(out, lines[idx])
		idx++
	}
	return strings.Join(out, "\n"), nil
}

func lineAt(lines []string, i int) string {
	if i < 0 || i >= len(lines) {
		return ""
	}
	return lines[i]
}
