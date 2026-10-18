package retry

import "sync"

// RetryBudget tracks the retry count for every task and
// decides whether the next retry is allowed (REQ-RTR-BGT-001
// to 003). The budget supports a global limit, a per-tool
// limit, and a per-error-class limit. The effective limit
// is the minimum of the three.
type RetryBudget struct {
	mu sync.Mutex

	// Max is the global retry limit. 0 → DefaultMaxRetries.
	Max int

	// PerTool is the per-tool retry limit. 0 → Max.
	PerTool map[string]int

	// PerClass is the per-error-class retry limit. 0 → Max.
	PerClass map[ErrorClass]int

	// usage tracks the retries used per (task, scope) pair.
	// scope is one of "global", "tool:<name>", "class:<class>".
	usage map[string]map[string]int
}

// NewBudget returns a budget with the default global limit.
func NewBudget(maxRetries int) *RetryBudget {
	if maxRetries <= 0 {
		maxRetries = DefaultMaxRetries
	}
	return &RetryBudget{
		Max:      maxRetries,
		PerTool:  map[string]int{},
		PerClass: map[ErrorClass]int{},
		usage:    map[string]map[string]int{},
	}
}

// SetToolLimit sets the per-tool retry limit. A value <= 0
// removes the override.
func (b *RetryBudget) SetToolLimit(tool string, limit int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if limit <= 0 {
		delete(b.PerTool, tool)
		return
	}
	b.PerTool[tool] = limit
}

// SetClassLimit sets the per-error-class retry limit. A
// value <= 0 removes the override.
func (b *RetryBudget) SetClassLimit(class ErrorClass, limit int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if limit <= 0 {
		delete(b.PerClass, class)
		return
	}
	b.PerClass[class] = limit
}

// Remaining returns the number of retries left for the
// given task, tool, and error class. Each scope (global,
// tool, class) tracks its own count; the result is the
// minimum of the three (max - used) pairs.
func (b *RetryBudget) Remaining(taskID, tool string, class ErrorClass) int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.RemainingLocked(taskID, tool, class)
}

// Allow charges one retry to the budget and returns true
// when the charge is within the limit. When false, the
// charge is NOT applied (so the next call gets the same
// answer).
func (b *RetryBudget) Allow(taskID, tool string, class ErrorClass) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.RemainingLocked(taskID, tool, class) <= 0 {
		return false
	}
	b.chargeLocked(taskID, tool, class)
	return true
}

// MustAllow is like Allow but returns RetryBudgetExceeded
// on rejection (REQ-RTR-ERR-001).
func (b *RetryBudget) MustAllow(taskID, tool string, class ErrorClass) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.RemainingLocked(taskID, tool, class) <= 0 {
		// Report the tightest scope for the operator.
		used := b.usedFor(taskID, "global")
		limit := b.Max
		if v := b.limitForTool(tool); v-b.usedFor(taskID, "tool:"+tool) < limit-used {
			used = b.usedFor(taskID, "tool:"+tool)
			limit = v
		}
		if v := b.limitForClass(class); v-b.usedFor(taskID, "class:"+string(class)) < limit-used {
			used = b.usedFor(taskID, "class:"+string(class))
			limit = v
		}
		return &RetryBudgetExceeded{
			TaskID: taskID, Tool: tool,
			Used:  used,
			Limit: limit,
		}
	}
	b.chargeLocked(taskID, tool, class)
	return nil
}

// Used returns the number of retries consumed for the
// given task. Each Allow call charges the global counter
// once, so Used returns the global count (the number of
// successful Allow calls for the task).
func (b *RetryBudget) Used(taskID, tool string, class ErrorClass) int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.usedFor(taskID, "global")
}

// RemainingLocked is the lock-free helper used inside Allow
// while the lock is already held. Each scope (global,
// tool, class) has its own count; the result is the
// minimum of (limit - used) across the three scopes.
//
// The result is always >= 0 because Allow only charges the
// budget when the previous result was strictly positive.
func (b *RetryBudget) RemainingLocked(taskID, tool string, class ErrorClass) int {
	min := b.Max - b.usedFor(taskID, "global")
	if v := b.limitForTool(tool) - b.usedFor(taskID, "tool:"+tool); v < min {
		min = v
	}
	if v := b.limitForClass(class) - b.usedFor(taskID, "class:"+string(class)); v < min {
		min = v
	}
	return min
}

func (b *RetryBudget) limitForTool(tool string) int {
	if v, ok := b.PerTool[tool]; ok && v < b.Max {
		return v
	}
	return b.Max
}

func (b *RetryBudget) limitForClass(class ErrorClass) int {
	if v, ok := b.PerClass[class]; ok && v < b.Max {
		return v
	}
	return b.Max
}

func (b *RetryBudget) chargeLocked(taskID, tool string, class ErrorClass) {
	b.bump(taskID, "global")
	b.bump(taskID, "tool:"+tool)
	b.bump(taskID, "class:"+string(class))
}

func (b *RetryBudget) usedFor(taskID, scope string) int {
	t, ok := b.usage[taskID]
	if !ok {
		return 0
	}
	return t[scope]
}

func (b *RetryBudget) bump(taskID, scope string) {
	t, ok := b.usage[taskID]
	if !ok {
		t = map[string]int{}
		b.usage[taskID] = t
	}
	t[scope]++
}
