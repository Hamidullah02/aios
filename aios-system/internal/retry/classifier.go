package retry

import (
	"regexp"
	"strings"
)

// ErrorClassifier classifies an error into one of three
// classes: transient, permanent, fixable (REQ-RTR-CLS-001 to
// 005). The classifier matches the error's message against
// an ordered pattern list. The first match wins; when no
// pattern matches, the classifier returns permanent.
type ErrorClassifier struct {
	// Patterns is the ordered list of (class, pattern)
	// pairs. Built-in defaults are populated by New.
	Patterns []classifierRule
}

// classifierRule is one entry in the classifier's pattern list.
type classifierRule struct {
	Class   ErrorClass
	Pattern *regexp.Regexp
}

// NewClassifier returns a classifier populated with the
// built-in default patterns (REQ-RTR-CLS-003 to 005).
func NewClassifier() *ErrorClassifier {
	c := &ErrorClassifier{}
	for _, p := range defaultPatterns {
		c.Patterns = append(c.Patterns, classifierRule{
			Class:   p.class,
			Pattern: regexp.MustCompile(p.pattern),
		})
	}
	return c
}

// Add appends a custom rule. The pattern is compiled with
// MustCompile; pass a valid regex.
func (c *ErrorClassifier) Add(class ErrorClass, pattern string) {
	c.Patterns = append(c.Patterns, classifierRule{
		Class:   class,
		Pattern: regexp.MustCompile(pattern),
	})
}

// Classify returns the error class. The first matching
// pattern wins (REQ-RTR-CLS-002).
func (c *ErrorClassifier) Classify(err error) ErrorClass {
	if err == nil {
		return ClassPermanent
	}
	msg := strings.ToLower(err.Error())
	for _, p := range c.Patterns {
		if p.Pattern.MatchString(msg) {
			return p.Class
		}
	}
	return ClassPermanent
}

type patternSpec struct {
	class   ErrorClass
	pattern string
}

// defaultPatterns is the built-in rule list. The order
// matters: fixable rules come first (more specific), then
// transient, then permanent.
var defaultPatterns = []patternSpec{
	// Fixable — the model can rewrite the call (REQ-RTR-CLS-005).
	{ClassFixable, `not unique`},
	{ClassFixable, `not found`},
	{ClassFixable, `file conflict`},
	{ClassFixable, `merge conflict`},
	{ClassFixable, `syntax error`},

	// Transient — retry with backoff (REQ-RTR-CLS-003).
	{ClassTransient, `timeout`},
	{ClassTransient, `timed out`},
	{ClassTransient, `connection refused`},
	{ClassTransient, `connection reset`},
	{ClassTransient, `network is unreachable`},
	{ClassTransient, `rate limit`},
	{ClassTransient, `too many requests`},
	{ClassTransient, `server error`},
	{ClassTransient, `internal server error`},
	{ClassTransient, `bad gateway`},
	{ClassTransient, `service unavailable`},
	{ClassTransient, `gateway timeout`},
	{ClassTransient, `temporary failure`},

	// Permanent — do not retry (REQ-RTR-CLS-004).
	{ClassPermanent, `permission denied`},
	{ClassPermanent, `not authorized`},
	{ClassPermanent, `unauthorized`},
	{ClassPermanent, `invalid input`},
	{ClassPermanent, `invalid argument`},
	{ClassPermanent, `validation failed`},
	{ClassPermanent, `unsupported`},
	{ClassPermanent, `forbidden`},
}
