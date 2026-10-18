package retry

import "strings"

// RetryPolicy ties the classifier, strategy, and budget
// together and produces a RetryDecision for a given
// (tool, error, retryCount) triple (REQ-RTR-POL-001 to
// 007).
//
// The decision algorithm:
//
//  1. Classify the error.
//  2. If the error is permanent, return stop.
//  3. If the budget is exhausted for the (task, tool,
//     class) triple, return stop.
//  4. If the error is fixable, return retry_with_fix with
//     a fix prompt.
//  5. Otherwise (transient), return retry with the
//     strategy's backoff.
type RetryPolicy struct {
	Classifier *ErrorClassifier
	Strategy   *RetryStrategy
	Budget     *RetryBudget
}

// NewPolicy returns a policy wired to the given
// classifier, strategy, and budget. A nil classifier or
// budget is replaced with a sensible default.
func NewPolicy(c *ErrorClassifier, s *RetryStrategy, b *RetryBudget) *RetryPolicy {
	if c == nil {
		c = NewClassifier()
	}
	if b == nil {
		b = NewBudget(DefaultMaxRetries)
	}
	return &RetryPolicy{Classifier: c, Strategy: s, Budget: b}
}

// Decide is the policy's main entry point
// (REQ-RTR-POL-002 to 007).
func (p *RetryPolicy) Decide(taskID, tool string, err error, retryCount int) RetryDecision {
	class := p.Classifier.Classify(err)

	// Permanent: stop, no charge, no fix prompt.
	if class == ClassPermanent {
		return RetryDecision{
			Action: ActionStop,
			Reason: "permanent error: " + errMsg(err),
		}
	}

	// Budget: stop when the limit is reached.
	if p.Budget.Remaining(taskID, tool, class) <= 0 {
		return RetryDecision{
			Action: ActionStop,
			Reason: "retry budget exhausted",
		}
	}

	// Charge the budget for the upcoming retry.
	_ = p.Budget.MustAllow(taskID, tool, class)

	switch class {
	case ClassFixable:
		return RetryDecision{
			Action:    ActionRetryWithFix,
			FixPrompt: fixPromptFor(err),
			Reason:    "fixable error: " + errMsg(err),
		}
	default: // transient
		backoff := 0
		if p.Strategy != nil {
			backoff = p.Strategy.Backoff(retryCount)
		}
		return RetryDecision{
			Action:    ActionRetry,
			BackoffMs: backoff,
			Reason:    "transient error: " + errMsg(err),
		}
	}
}

// fixPromptFor returns a fix prompt tailored to the error.
// The function is best-effort: when no specific prompt
// matches, a generic prompt is returned.
func fixPromptFor(err error) string {
	msg := errMsg(err)
	switch {
	case strings.Contains(msg, "not unique"):
		return "The old_string matched more than one location. Add more context to make the old_string unique, then retry."
	case strings.Contains(msg, "not found"):
		return "The old_string was not found. Read the file again to find the current text, then retry with the correct old_string."
	case strings.Contains(msg, "file conflict"), strings.Contains(msg, "merge conflict"):
		return "The file has a conflict. Resolve the conflict and retry."
	case strings.Contains(msg, "syntax error"):
		return "The generated text had a syntax error. Rewrite the snippet to be valid, then retry."
	default:
		return "The previous call failed. Inspect the error and retry with a corrected call."
	}
}

func errMsg(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
