package policy

import (
	"path/filepath"
	"strconv"
	"strings"
)

// Call is the tool-call context passed to the gateway.
//
// The gateway matches each rule's Match against a Call. Only
// the fields relevant to the rule table are exposed; the
// gateway does not see the full Envelope.
type Call struct {
	Tool   string
	Args   map[string]any
	Actor  string
	TaskID string

	// Path is the resource path extracted from the call's args
	// under the canonical key "path". If the args do not carry
	// a "path", Path is "" and path-prefix/path-glob conditions
	// cannot match.
	Path string

	// Host is the URL host extracted from the call's args
	// under the canonical key "host".
	Host string

	// RiskTier is the action's tier (0-3) at the time of the
	// call. If zero, tier conditions cannot match.
	RiskTier int
}

// NewCall constructs a Call from a tool name, args, actor, and
// taskID. Path and Host are extracted from the args under the
// canonical keys "path" and "host" if present.
func NewCall(tool string, args map[string]any, actor, taskID string) Call {
	c := Call{Tool: tool, Args: args, Actor: actor, TaskID: taskID}
	if args != nil {
		if p, ok := args["path"].(string); ok {
			c.Path = p
		}
		if h, ok := args["host"].(string); ok {
			c.Host = h
		}
		if t, ok := args["tier"]; ok {
			c.RiskTier = toInt(t)
		}
	}
	return c
}

// toInt coerces a YAML/JSON-decoded value to int. YAML scalars
// can decode as int, int64, float64, or string; we accept all
// of those for robustness.
func toInt(v any) int {
	switch t := v.(type) {
	case int:
		return t
	case int32:
		return int(t)
	case int64:
		return int(t)
	case float64:
		return int(t)
	case string:
		n, _ := strconv.Atoi(t)
		return n
	}
	return 0
}

// Matches returns true if the rule's match conditions are all
// satisfied by the call. An empty match matches everything.
//
// Condition semantics:
//
//   - tool:       exact string equality.
//   - tool_glob:  filepath.Match against the tool name.
//   - path_prefix: the call's Path starts with the prefix.
//   - path_glob:  filepath.Match against the call's Path.
//   - host_glob:  filepath.Match against the call's Host.
//   - tier:       exact equality with the call's RiskTier.
//   - actor:      exact string equality.
//   - task:       exact string equality.
//
// Glob patterns are compiled (via filepath.Match) on every
// call. The TRD mentions caching compiled patterns; for the
// MVP we keep the implementation simple and rely on the
// gateway's 1ms latency budget. If profiling shows hot, the
// cache can be added without changing the API.
func (m Match) Matches(c Call) bool {
	if m.Tool != "" && m.Tool != c.Tool {
		return false
	}
	if m.ToolGlob != "" {
		ok, err := filepath.Match(m.ToolGlob, c.Tool)
		if err != nil || !ok {
			return false
		}
	}
	if m.PathPrefix != "" {
		if c.Path == "" || !strings.HasPrefix(c.Path, m.PathPrefix) {
			return false
		}
	}
	if m.PathGlob != "" {
		if c.Path == "" {
			return false
		}
		ok, err := filepath.Match(m.PathGlob, c.Path)
		if err != nil || !ok {
			return false
		}
	}
	if m.HostGlob != "" {
		if c.Host == "" {
			return false
		}
		ok, err := filepath.Match(m.HostGlob, c.Host)
		if err != nil || !ok {
			return false
		}
	}
	if m.Tier != nil && *m.Tier != c.RiskTier {
		return false
	}
	if m.Actor != "" && m.Actor != c.Actor {
		return false
	}
	if m.Task != "" && m.Task != c.TaskID {
		return false
	}
	return true
}
