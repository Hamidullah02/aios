package runtime

import (
	"sync"
	"time"
)

// Budget is the per-task limit envelope (TRD §7.2). A zero
// field means "no cap on this dimension". The four
// concurrency caps are: max_parallel_workers,
// max_tool_calls_per_minute, max_model_calls_per_minute,
// max_network_requests_per_minute. The other limits are
// totals: max_cost, max_wall_time, max_tool_calls,
// max_model_calls, max_network_requests.
type Budget struct {
	MaxCostUSD         float64 `json:"max_cost_usd,omitempty"`
	MaxWallTimeSec     int     `json:"max_wall_time_sec,omitempty"`
	MaxToolCalls       int     `json:"max_tool_calls,omitempty"`
	MaxModelCalls      int     `json:"max_model_calls,omitempty"`
	MaxNetworkRequests int     `json:"max_network_requests,omitempty"`

	MaxParallelWorkers          int `json:"max_parallel_workers,omitempty"`
	MaxToolCallsPerMinute       int `json:"max_tool_calls_per_minute,omitempty"`
	MaxModelCallsPerMinute      int `json:"max_model_calls_per_minute,omitempty"`
	MaxNetworkRequestsPerMinute int `json:"max_network_requests_per_minute,omitempty"`
}

// BudgetMeter is the live tracker for a Budget
// (REQ-RT-BGT-001 / 002 / 003 / 004 / 005).
//
// The meter records the totals and the per-minute rates. The
// per-minute caps use a sliding window of timestamps. The
// parallel-worker cap is a buffered channel acting as a
// semaphore. The cost cap is a separate field on the meter
// because the price table is consulted at call time.
type BudgetMeter struct {
	mu     sync.Mutex
	budget Budget

	// Totals
	startedAt    time.Time
	costUSD      float64
	toolCalls    int
	modelCalls   int
	netRequests  int
	wallDeadline time.Time // zero if MaxWallTimeSec is zero

	// Sliding windows: each call appends a timestamp; expired
	// timestamps (older than 60s) are pruned on every check.
	toolWindow    []time.Time
	modelWindow   []time.Time
	networkWindow []time.Time

	// Semaphore for parallel workers. Buffered to the cap;
	// each worker acquires one slot and releases on return.
	workerSlots chan struct{}
}

// NewBudgetMeter returns a meter initialized with b. The
// meter's clock is the wall clock; tests should swap the
// meter's `Now` function for determinism.
func NewBudgetMeter(b Budget) *BudgetMeter {
	m := &BudgetMeter{budget: b, startedAt: time.Now()}
	if b.MaxWallTimeSec > 0 {
		m.wallDeadline = m.startedAt.Add(time.Duration(b.MaxWallTimeSec) * time.Second)
	}
	if b.MaxParallelWorkers > 0 {
		m.workerSlots = make(chan struct{}, b.MaxParallelWorkers)
	}
	return m
}

// CanProceed returns nil when the budget allows the next step,
// or a *BudgetExhausted when it does not (REQ-RT-BGT-003).
// It prunes the rate-cap windows of expired entries as a
// side effect so the windows do not grow without bound.
func (m *BudgetMeter) CanProceed() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	m.toolWindow = pruneWindow(m.toolWindow, now)
	m.modelWindow = pruneWindow(m.modelWindow, now)
	m.networkWindow = pruneWindow(m.networkWindow, now)

	if m.budget.MaxCostUSD > 0 && m.costUSD >= m.budget.MaxCostUSD {
		return &BudgetExhausted{Kind: "max_cost_usd", Limit: m.budget.MaxCostUSD, Used: m.costUSD, Reason: "cost cap reached"}
	}
	if !m.wallDeadline.IsZero() && now.After(m.wallDeadline) {
		return &BudgetExhausted{Kind: "max_wall_time_sec", Limit: float64(m.budget.MaxWallTimeSec), Used: now.Sub(m.startedAt).Seconds(), Reason: "wall-time cap reached"}
	}
	if m.budget.MaxToolCalls > 0 && m.toolCalls >= m.budget.MaxToolCalls {
		return &BudgetExhausted{Kind: "max_tool_calls", Limit: float64(m.budget.MaxToolCalls), Used: float64(m.toolCalls), Reason: "tool-call cap reached"}
	}
	if m.budget.MaxModelCalls > 0 && m.modelCalls >= m.budget.MaxModelCalls {
		return &BudgetExhausted{Kind: "max_model_calls", Limit: float64(m.budget.MaxModelCalls), Used: float64(m.modelCalls), Reason: "model-call cap reached"}
	}
	if m.budget.MaxNetworkRequests > 0 && m.netRequests >= m.budget.MaxNetworkRequests {
		return &BudgetExhausted{Kind: "max_network_requests", Limit: float64(m.budget.MaxNetworkRequests), Used: float64(m.netRequests), Reason: "network-request cap reached"}
	}
	if m.budget.MaxToolCallsPerMinute > 0 && len(m.toolWindow) >= m.budget.MaxToolCallsPerMinute {
		return &BudgetExhausted{Kind: "max_tool_calls_per_minute", Limit: float64(m.budget.MaxToolCallsPerMinute), Used: float64(len(m.toolWindow)), Reason: "tool-call rate cap reached"}
	}
	if m.budget.MaxModelCallsPerMinute > 0 && len(m.modelWindow) >= m.budget.MaxModelCallsPerMinute {
		return &BudgetExhausted{Kind: "max_model_calls_per_minute", Limit: float64(m.budget.MaxModelCallsPerMinute), Used: float64(len(m.modelWindow)), Reason: "model-call rate cap reached"}
	}
	if m.budget.MaxNetworkRequestsPerMinute > 0 && len(m.networkWindow) >= m.budget.MaxNetworkRequestsPerMinute {
		return &BudgetExhausted{Kind: "max_network_requests_per_minute", Limit: float64(m.budget.MaxNetworkRequestsPerMinute), Used: float64(len(m.networkWindow)), Reason: "network-request rate cap reached"}
	}
	return nil
}

// RecordToolCall bumps the tool-call totals and adds a
// timestamp to the rate window. The caller is expected to
// have already checked CanProceed.
func (m *BudgetMeter) RecordToolCall() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.toolCalls++
	now := time.Now()
	m.toolWindow = append(m.toolWindow, now)
}

// RecordModelCall records one model call and adds the cost
// (USD) to the running total. The price computation is the
// caller's job (see models.PriceTable.Cost).
func (m *BudgetMeter) RecordModelCall(costUSD float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.modelCalls++
	m.costUSD += costUSD
	now := time.Now()
	m.modelWindow = append(m.modelWindow, now)
}

// RecordNetworkRequest bumps the network-request totals and
// adds a timestamp to the rate window.
func (m *BudgetMeter) RecordNetworkRequest() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.netRequests++
	now := time.Now()
	m.networkWindow = append(m.networkWindow, now)
}

// AcquireWorker takes a parallel-worker slot. It blocks until
// a slot is free, or returns immediately if the cap is zero
// (meaning "no cap"). The caller MUST call ReleaseWorker.
//
// Returns false when the cap is set to 0 (no cap = always
// "acquired" without consuming a slot).
func (m *BudgetMeter) AcquireWorker() bool {
	if m.budget.MaxParallelWorkers <= 0 {
		return true
	}
	m.workerSlots <- struct{}{}
	return true
}

// ReleaseWorker gives back a parallel-worker slot.
func (m *BudgetMeter) ReleaseWorker() {
	if m.budget.MaxParallelWorkers <= 0 {
		return
	}
	<-m.workerSlots
}

// Snapshot returns a copy of the current totals. Useful for
// reports and checkpoints.
func (m *BudgetMeter) Snapshot() BudgetSnapshot {
	m.mu.Lock()
	defer m.mu.Unlock()
	return BudgetSnapshot{
		CostUSD:         m.costUSD,
		ToolCalls:       m.toolCalls,
		ModelCalls:      m.modelCalls,
		NetworkRequests: m.netRequests,
		WallElapsed:     time.Since(m.startedAt),
	}
}

// BudgetSnapshot is the read-only view of the meter. It is
// what the report and the checkpoint store.
type BudgetSnapshot struct {
	CostUSD         float64
	ToolCalls       int
	ModelCalls      int
	NetworkRequests int
	WallElapsed     time.Duration
}

// pruneWindow returns the window with entries older than 60s
// removed. The 60s window is the rate-cap granularity
// (REQ-RT-BGT-004).
func pruneWindow(window []time.Time, now time.Time) []time.Time {
	cutoff := now.Add(-60 * time.Second)
	out := window[:0]
	for _, t := range window {
		if t.After(cutoff) {
			out = append(out, t)
		}
	}
	return out
}
