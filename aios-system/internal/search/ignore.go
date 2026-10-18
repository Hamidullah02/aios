package search

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// ignorePatterns holds gitignore-style patterns loaded from
// .aiosignore. Each pattern is matched against the slash-
// separated relative path.
type ignorePatterns struct {
	patterns []string
}

// loadIgnore reads .aiosignore from the workspace root. A
// missing file is not an error — the returned patterns are
// empty. Lines that are empty or start with '#' are
// skipped. Trailing whitespace is trimmed.
func loadIgnore(workspace string) ignorePatterns {
	var p ignorePatterns
	f, err := os.Open(filepath.Join(workspace, ".aiosignore"))
	if err != nil {
		return p
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		p.patterns = append(p.patterns, line)
	}
	return p
}

// match reports whether rel (a slash-separated path relative
// to the workspace) matches any ignore pattern. The match
// uses filepath.Match; a pattern that matches any path
// segment is treated as a hit.
func (p ignorePatterns) match(rel string) bool {
	if len(p.patterns) == 0 {
		return false
	}
	rel = filepath.ToSlash(rel)
	for _, pat := range p.patterns {
		if ok, _ := filepath.Match(pat, rel); ok {
			return true
		}
		if ok, _ := filepath.Match(pat, filepath.Base(rel)); ok {
			return true
		}
		// Directory-style patterns: "node_modules/" matches
		// any path with that segment.
		seg := strings.TrimSuffix(pat, "/")
		if seg != pat {
			for _, part := range strings.Split(rel, "/") {
				if part == seg {
					return true
				}
			}
		}
	}
	return false
}
