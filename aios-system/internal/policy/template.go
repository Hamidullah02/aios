package policy

import (
	"fmt"
	"strings"
)

// Substitute fills in `{name}` placeholders in s using values
// from the call. Supported names:
//
//   - {task}    -> call.TaskID
//   - {tool}    -> call.Tool
//   - {path}    -> call.Path
//   - {host}    -> call.Host
//   - {actor}   -> call.Actor
//   - {tier}    -> call.RiskTier (decimal)
//   - {arg.X}   -> call.Args[X] (string-coerced)
//
// Unknown placeholders produce an error so a typo in the
// template is caught at decision time, not silently dropped.
func Substitute(s string, c Call) (string, error) {
	var b strings.Builder
	b.Grow(len(s))
	i := 0
	for i < len(s) {
		if s[i] != '{' {
			b.WriteByte(s[i])
			i++
			continue
		}
		end := strings.IndexByte(s[i:], '}')
		if end < 0 {
			return "", fmt.Errorf("template: unterminated placeholder at offset %d", i)
		}
		// end is relative to i, including the closing brace.
		name := s[i+1 : i+end]
		val, err := lookupPlaceholder(name, c)
		if err != nil {
			return "", err
		}
		b.WriteString(val)
		i += end + 1
	}
	return b.String(), nil
}

func lookupPlaceholder(name string, c Call) (string, error) {
	switch name {
	case "task":
		return c.TaskID, nil
	case "tool":
		return c.Tool, nil
	case "path":
		return c.Path, nil
	case "host":
		return c.Host, nil
	case "actor":
		return c.Actor, nil
	case "tier":
		return fmt.Sprintf("%d", c.RiskTier), nil
	}
	if strings.HasPrefix(name, "arg.") {
		key := name[4:]
		if c.Args == nil {
			return "", fmt.Errorf("template: arg.%s: no args", key)
		}
		v, ok := c.Args[key]
		if !ok {
			return "", fmt.Errorf("template: arg.%s: not in args", key)
		}
		return fmt.Sprintf("%v", v), nil
	}
	return "", fmt.Errorf("template: unknown placeholder {%s}", name)
}

// SubstituteMap applies Substitute to every value in m.
func SubstituteMap(m map[string]string, c Call) (map[string]string, error) {
	out := make(map[string]string, len(m))
	for k, v := range m {
		s, err := Substitute(v, c)
		if err != nil {
			return nil, err
		}
		out[k] = s
	}
	return out, nil
}
