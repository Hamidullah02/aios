package search

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Hamidullah02/aios/aios-system/internal/tools"
)

// CodeSearchTool finds symbols (function, class, method,
// variable) in source files (REQ-SRH-CDS-001 to 005).
//
// The tool uses per-language regex patterns — no tree-
// sitter dependency. Supported languages: go, javascript,
// typescript, python, rust, cpp, java.
//
// Args:
//
//	symbol (string, required) — name to look for (substring match
//	           on the captured name).
//	kind   (string, optional) — one of: function, class, method,
//	           variable. Empty matches anything.
//	lang   (string, optional) — restrict to one language. Empty
//	           means any supported language (ext inferred from
//	           the file name).
//	path   (string, optional) — search root (default: workspace).
//	max    (int,    optional) — cap on results (default 200).
type CodeSearchTool struct{}

func (CodeSearchTool) Name() string { return "code_search" }

func (CodeSearchTool) Run(args map[string]any, ctx tools.Context) (tools.Result, error) {
	symbol, _ := args["symbol"].(string)
	if symbol == "" {
		return tools.Result{OK: false, Summary: "missing symbol"},
			fmt.Errorf("search: code_search: symbol is required")
	}
	kind, _ := args["kind"].(string)
	langFilter, _ := args["lang"].(string)
	max := 200
	if v, ok := args["max"].(int); ok && v > 0 {
		max = v
	}
	ws := ctx.Workspace
	if ws == "" {
		ws, _ = tools.Workspace()
	}
	wsAbs, _ := filepath.Abs(ws)
	root := wsAbs
	if p, _ := args["path"].(string); p != "" {
		if filepath.IsAbs(p) {
			root = filepath.Clean(p)
		} else {
			root = filepath.Clean(filepath.Join(wsAbs, p))
		}
	}
	ig := loadIgnore(wsAbs)

	pat := buildPattern(kind, langFilter)
	if pat == "" {
		return tools.Result{OK: false, Summary: "no languages"},
			&UnsupportedLanguageError{Path: "<filter>"}
	}
	re, err := regexp.Compile(pat)
	if err != nil {
		return tools.Result{OK: false, Summary: "bad pattern"}, err
	}
	nameRe, err := regexp.Compile(regexp.QuoteMeta(symbol))
	if err != nil {
		return tools.Result{OK: false, Summary: "bad symbol"}, err
	}

	var hits []Match
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(wsAbs, path)
		rel = filepath.ToSlash(rel)
		if ig.match(rel) {
			return nil
		}
		ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(path)), ".")
		if langFor(ext) == "" {
			return nil
		}
		if langFilter != "" && !strings.EqualFold(langFilter, langFor(ext)) {
			return nil
		}
		hits = append(hits, scanSymbols(path, re, nameRe)...)
		if len(hits) >= max {
			return filepath.SkipAll
		}
		return nil
	})
	if len(hits) > max {
		hits = hits[:max]
	}
	return tools.Result{
		OK:      true,
		Output:  map[string]any{"matches": hits, "count": len(hits)},
		Summary: fmt.Sprintf("%d matches", len(hits)),
	}, nil
}

// scanSymbols reads path and returns the matching symbol
// declarations. The first regex selects candidate lines
// (declarations); the second matches the symbol name.
func scanSymbols(path string, decl, name *regexp.Regexp) []Match {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	var out []Match
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	lineNo := 0
	for sc.Scan() {
		lineNo++
		text := sc.Text()
		if !decl.MatchString(text) {
			continue
		}
		loc := decl.FindStringIndex(text)
		if loc == nil {
			continue
		}
		nameStart := loc[1]
		// Skip whitespace between decl keyword and the name.
		for nameStart < len(text) && (text[nameStart] == ' ' || text[nameStart] == '\t' || text[nameStart] == '*') {
			nameStart++
		}
		tail := text[nameStart:]
		nl := name.FindStringIndex(tail)
		if nl == nil {
			continue
		}
		out = append(out, Match{Path: filepath.ToSlash(path), Line: lineNo, Text: text})
	}
	return out
}

// langFor returns the language name for a file extension, or
// "" for unknown.
func langFor(ext string) string {
	switch ext {
	case "go":
		return "go"
	case "js", "jsx", "mjs", "cjs":
		return "javascript"
	case "ts", "tsx":
		return "typescript"
	case "py":
		return "python"
	case "rs":
		return "rust"
	case "c", "h", "cc", "cpp", "cxx", "hpp":
		return "cpp"
	case "java":
		return "java"
	}
	return ""
}

// buildPattern returns a regex that matches declarations of
// the given kind across supported languages. An empty kind
// matches any declaration.
func buildPattern(kind, lang string) string {
	langs := allLangs
	if lang != "" {
		langs = []string{strings.ToLower(lang)}
	}
	var alts []string
	for _, l := range langs {
		if p, ok := declPatterns[l]; ok {
			if kind == "" || kind == "any" {
				alts = append(alts, p.all)
			} else {
				if kp, ok := p.byKind[kind]; ok {
					alts = append(alts, kp)
				}
			}
		}
	}
	if len(alts) == 0 {
		return ""
	}
	return "(?:" + strings.Join(alts, "|") + ")"
}

type langDecls struct {
	all    string
	byKind map[string]string
}

var declPatterns = map[string]langDecls{
	"go": {
		all: `\b(?:func|var|const|type)\s+`,
		byKind: map[string]string{
			"function": `\bfunc\s+`,
			"method":   `\bfunc\s+\([^)]*\)\s+`,
			"class":    `\btype\s+\w+\s+struct\b`,
			"variable": `\b(?:var|const)\s+`,
		},
	},
	"javascript": {
		all: `\b(?:function|class|const|let|var)\s+`,
		byKind: map[string]string{
			"function": `\bfunction\s+`,
			"class":    `\bclass\s+`,
			"variable": `\b(?:const|let|var)\s+`,
		},
	},
	"typescript": {
		all: `\b(?:function|class|interface|type|const|let|var)\s+`,
		byKind: map[string]string{
			"function": `\bfunction\s+`,
			"class":    `\bclass\s+`,
			"variable": `\b(?:const|let|var)\s+`,
		},
	},
	"python": {
		all: `^\s*(?:def|class)\s+`,
		byKind: map[string]string{
			"function": `^\s*def\s+`,
			"class":    `^\s*class\s+`,
		},
	},
	"rust": {
		all: `\b(?:fn|struct|enum|let|const|static|impl|trait)\s+`,
		byKind: map[string]string{
			"function": `\b(?:fn|impl)\s+`,
			"class":    `\b(?:struct|enum|trait)\s+`,
			"variable": `\b(?:let|const|static)\s+`,
		},
	},
	"cpp": {
		all: `\b(?:class|struct|void|int|bool|auto|template)\s+`,
		byKind: map[string]string{
			"function": `\b(?:void|int|bool|auto)\s+`,
			"class":    `\b(?:class|struct)\s+`,
		},
	},
	"java": {
		all: `\b(?:class|interface|void|int|boolean|public|private|static)\s+`,
		byKind: map[string]string{
			"function": `\b(?:void|int|boolean)\s+`,
			"class":    `\b(?:class|interface)\s+`,
		},
	},
}

var allLangs = []string{"go", "javascript", "typescript", "python", "rust", "cpp", "java"}
