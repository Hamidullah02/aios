package fanout

import (
	"context"
	"fmt"
)

// BudgetCap is the typed budget cap the
// Scheduler enforces (REQ-FAN-SCH-003).
//
// The cap is checked after every task: when the
// running total exceeds the cap, the scheduler
// stops. The cap is the cost cap; the wall-time
// cap is enforced by the context.
type BudgetCap struct {
	// MaxCostUSD is the maximum total cost in
	// USD. Zero means no cap.
	MaxCostUSD float64
	// MaxTasks is the maximum number of tasks
	// the scheduler runs. Zero means no cap.
	MaxTasks int
}

// Scheduler is the budgeted batch runner
// (REQ-FAN-SCH-001).
//
// The scheduler runs the tasks in batches of
// MaxParallel (REQ-FAN-SCH-002). The scheduler
// respects the budget: when the cost exceeds
// MaxCostUSD or the task count exceeds MaxTasks,
// the scheduler stops (REQ-FAN-SCH-003).
type Scheduler struct {
	tool *FanOutTool
	cap  BudgetCap
}

// SchedulerConfig controls Scheduler construction.
type SchedulerConfig struct {
	// Tool is the FanOutTool the scheduler uses.
	// Required.
	Tool *FanOutTool
	// Cap is the budget cap. The zero value
	// means "no cap".
	Cap BudgetCap
}

// NewScheduler constructs a Scheduler. The
// function returns an error when the tool is nil.
func NewScheduler(cfg SchedulerConfig) (*Scheduler, error) {
	if cfg.Tool == nil {
		return nil, &FanOutError{Op: "init", Reason: "tool is required"}
	}
	return &Scheduler{tool: cfg.Tool, cap: cfg.Cap}, nil
}

// Schedule runs the tasks in budgeted batches.
// The function is the public entry point the
// runtime layer calls.
//
// The scheduler splits the task list into
// batches of MaxParallel and runs each batch
// through the FanOutTool. The scheduler stops
// early when the budget is exhausted.
//
// The function returns the merged result of the
// final batch. When the budget is exhausted the
// function returns the partial merged result
// and a *BudgetError.
func (s *Scheduler) Schedule(ctx context.Context, tasks []FanOutTask) (Merged, error) {
	if s == nil {
		return Merged{}, &FanOutError{Op: "schedule", Reason: "scheduler is nil"}
	}
	if len(tasks) == 0 {
		return Merged{Results: []FanOutResult{}, Strategy: s.tool.Merger().Strategy}, nil
	}
	// Validate every task up front.
	for i := range tasks {
		if err := tasks[i].Validate(); err != nil {
			return Merged{}, err
		}
	}
	allResults := make([]FanOutResult, 0, len(tasks))
	totalCost := 0.0
	batchSize := s.tool.MaxParallel()
	if batchSize <= 0 {
		batchSize = DefaultMaxParallel
	}
	for start := 0; start < len(tasks); start += batchSize {
		// Pre-batch budget check. The check
		// slices the batch to the remaining
		// task cap so we never overshoot
		// MaxTasks. The cost cap is checked
		// after the batch (cost is per-task).
		end := start + batchSize
		if s.cap.MaxTasks > 0 {
			remaining := s.cap.MaxTasks - len(allResults)
			if remaining <= 0 {
				return s.partialMerge(allResults), &BudgetError{
					Kind:   "max_tasks",
					Limit:  float64(s.cap.MaxTasks),
					Used:   float64(len(allResults)),
					Reason: "task cap reached during schedule",
				}
			}
			if remaining < end-start {
				end = start + remaining
			}
		}
		if end > len(tasks) {
			end = len(tasks)
		}
		batch := tasks[start:end]
		merged, err := s.tool.Run(ctx, batch)
		if err != nil {
			return Merged{}, err
		}
		allResults = append(allResults, merged.Results...)
		// Tally the cost.
		for _, r := range merged.Results {
			totalCost += r.CostUSD
		}
		// Check the cost budget.
		if s.cap.MaxCostUSD > 0 && totalCost > s.cap.MaxCostUSD {
			return s.partialMerge(allResults), &BudgetError{
				Kind:   "max_cost_usd",
				Limit:  s.cap.MaxCostUSD,
				Used:   totalCost,
				Reason: "cost cap reached during schedule",
			}
		}
		// Check the task budget (post-batch
		// guard for the cap-was-zero branch).
		if s.cap.MaxTasks > 0 && len(allResults) >= s.cap.MaxTasks {
			return s.partialMerge(allResults), &BudgetError{
				Kind:   "max_tasks",
				Limit:  float64(s.cap.MaxTasks),
				Used:   float64(len(allResults)),
				Reason: "task cap reached during schedule",
			}
		}
	}
	return s.partialMerge(allResults), nil
}

// partialMerge merges the per-task results
// without invoking the Summarizer. The function
// is the helper the budget-error path uses:
// when the budget is exhausted, the scheduler
// has not finished, so the Summary strategy is
// not appropriate.
func (s *Scheduler) partialMerge(results []FanOutResult) Merged {
	merger := NewResultMerger(MergeConcat)
	merger.Separator = s.tool.Merger().Separator
	merger.HeaderFmt = s.tool.Merger().HeaderFmt
	merged, _ := merger.Merge(context.Background(), results)
	merged.Strategy = s.tool.Merger().Strategy
	return merged
}

// Tool returns the FanOutTool the scheduler
// uses. The function is the getter the tests
// use.
func (s *Scheduler) Tool() *FanOutTool { return s.tool }

// Cap returns the budget cap. The function is
// the getter the tests use.
func (s *Scheduler) Cap() BudgetCap { return s.cap }

// formatBudgetError is the helper the scheduler
// uses to format the budget-exhausted message.
// The function is split out so the message is
// consistent across both caps.
func formatBudgetError(kind string, limit, used float64) string {
	return fmt.Sprintf("fanout: budget exhausted: %s (used %v of %v)", kind, used, limit)
}
