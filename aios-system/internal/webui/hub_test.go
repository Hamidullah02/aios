package webui

import (
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"
)

// --- nullable timestamps (v0.41.2) ---
//
// The defect: Agent.StartedAt / UpdatedAt / Message.At used
// `json:",omitzero"`, a Go 1.24 encoding/json option. This
// module targets Go 1.23, where an unknown format verb is
// parsed as a no-op rather than rejected, so a zero time
// serialised as "0001-01-01T00:00:00Z" instead of being
// omitted.
//
// The UI sorts its activity feed on updated_at, so an idle
// agent carried a placeholder timestamp that sorted ABOVE
// real activity instead of below it.

// TestTime_ZeroMarshalsAsNull is the direct regression gate.
func TestTime_ZeroMarshalsAsNull(t *testing.T) {
	b, err := json.Marshal(NewTime(time.Time{}))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(b) != "null" {
		t.Errorf("zero Time marshalled as %s, want null", b)
	}
	if strings.Contains(string(b), "0001-01-01") {
		t.Errorf("placeholder timestamp leaked: %s", b)
	}
}

// TestTime_SetMarshalsAsRFC3339 guards the other half.
func TestTime_SetMarshalsAsRFC3339(t *testing.T) {
	want := time.Date(2026, 3, 4, 5, 6, 7, 0, time.UTC)
	b, err := json.Marshal(NewTime(want))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(b) != `"2026-03-04T05:06:07Z"` {
		t.Errorf("set Time marshalled as %s", b)
	}
}

// TestTime_NewTimeTreatsZeroAsAbsent covers the constructor's
// whole reason for existing: the runtime sends a zero event
// time, and that must not become a year-one timestamp.
func TestTime_NewTimeTreatsZeroAsAbsent(t *testing.T) {
	got := NewTime(time.Time{})
	if !got.IsZero() {
		t.Error("NewTime(zero) should be absent")
	}
	b, _ := json.Marshal(got)
	if string(b) != "null" {
		t.Errorf("NewTime(zero) marshalled as %s, want null", b)
	}
}

// TestTime_RoundTrip covers the unmarshal side, so a client
// posting a layout or a message that carries a timestamp is
// not rejected.
func TestTime_RoundTrip(t *testing.T) {
	want := time.Date(2026, 3, 4, 5, 6, 7, 0, time.UTC)
	b, err := json.Marshal(NewTime(want))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var back Time
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !back.Or().Equal(want) {
		t.Errorf("round trip = %v, want %v", back.Or(), want)
	}
}

// TestTime_UnmarshalNull covers the absent case.
func TestTime_UnmarshalNull(t *testing.T) {
	var got Time
	if err := json.Unmarshal([]byte("null"), &got); err != nil {
		t.Fatalf("unmarshal null: %v", err)
	}
	if !got.IsZero() {
		t.Errorf("null unmarshalled to a set time: %v", got.Or())
	}
}

// TestSnapshot_NoPlaceholderTimestamps is the end-to-end gate:
// the bytes the browser actually receives must contain no
// year-one timestamps at all.
func TestSnapshot_NoPlaceholderTimestamps(t *testing.T) {
	hub := NewHub()
	// An idle agent: created but never started.
	hub.Upsert("idle-agent", func(a *Agent) {})
	hub.Post(Message{From: "idle-agent", Body: "hello"})

	b, err := json.Marshal(hub.Snapshot())
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(b), "0001-01-01") {
		t.Errorf("snapshot carries a placeholder timestamp:\n%s", b)
	}
}

// TestSnapshot_IdleAgentHasNullStartedAt is the specific
// assertion on the field the UI sorts by.
func TestSnapshot_IdleAgentHasNullStartedAt(t *testing.T) {
	hub := NewHub()
	hub.Upsert("idle-agent", func(a *Agent) {})
	agents := hub.Agents()
	if len(agents) != 1 {
		t.Fatalf("got %d agents, want 1", len(agents))
	}
	if !agents[0].StartedAt.IsZero() {
		t.Errorf("an agent that never started has StartedAt = %v", agents[0].StartedAt.Or())
	}
	b, _ := json.Marshal(agents[0])
	if !strings.Contains(string(b), `"started_at":null`) {
		t.Errorf("idle agent serialised without a null started_at:\n%s", b)
	}
}

// TestSetStatus_StampsStartedAt is the positive counterpart:
// starting a task must still produce a real timestamp, so the
// fix did not over-correct into always-null.
func TestSetStatus_StampsStartedAt(t *testing.T) {
	hub := NewHub()
	a := hub.SetStatus("t1", StatusRunning, "")
	if a == nil {
		t.Fatal("SetStatus returned nil")
	}
	if a.StartedAt.IsZero() {
		t.Error("SetStatus(running) did not stamp StartedAt")
	}
	if a.TaskCount != 1 {
		t.Errorf("TaskCount = %d, want 1", a.TaskCount)
	}
	b, _ := json.Marshal(a)
	if strings.Contains(string(b), `"started_at":null`) {
		t.Errorf("a running agent serialised with a null started_at:\n%s", b)
	}
}

// --- version counter (v0.41.2) ---
//
// The defect: Snapshot.Version carried nextSub + len(msgs),
// which is not a version of anything. It moved backwards
// whenever a browser disconnected and moved on a message post
// that changed no agent.

// TestSnapshot_VersionIsMonotonic pins the contract that a
// client can actually use: it only ever increases, and only
// when state changed.
func TestSnapshot_VersionIsMonotonic(t *testing.T) {
	hub := NewHub()
	start := hub.Snapshot().Version

	hub.Upsert("t1", func(a *Agent) { a.Status = StatusRunning })
	afterUpsert := hub.Snapshot().Version
	if afterUpsert <= start {
		t.Errorf("version did not advance on a mutation: %d -> %d", start, afterUpsert)
	}

	hub.Post(Message{From: "t1", Body: "hi"})
	afterPost := hub.Snapshot().Version
	if afterPost <= afterUpsert {
		t.Errorf("version did not advance on a message post: %d -> %d", afterUpsert, afterPost)
	}

	// A subscriber connecting is NOT a state change, so the
	// version must hold still. The old implementation counted
	// subscribers, so it moved here.
	_, cancel := hub.subscribe()
	afterSub := hub.Snapshot().Version
	if afterSub != afterPost {
		t.Errorf("version moved on a subscriber change: %d -> %d", afterPost, afterSub)
	}
	cancel()
	afterCancel := hub.Snapshot().Version
	if afterCancel != afterPost {
		t.Errorf("version moved on unsubscribe: %d -> %d", afterPost, afterCancel)
	}
}

// TestSnapshot_VersionUnchangedWhenIdle asserts no drift
// without a mutation.
func TestSnapshot_VersionUnchangedWhenIdle(t *testing.T) {
	hub := NewHub()
	a := hub.Snapshot().Version
	b := hub.Snapshot().Version
	if a != b {
		t.Errorf("version drifted without a mutation: %d then %d", a, b)
	}
}

// --- subscriber lifecycle (v0.41.2) ---
//
// The defect: broadcast copied the subscriber channels,
// released h.mu, and then sent. A subscriber whose cancel()
// ran in that window closed its channel between the copy and
// the send, so the writer panicked with
// "send on closed channel".

// TestSubscribe_CancelIsIdempotent asserts cancel twice is
// safe, since a defer plus an explicit call is a normal shape.
func TestSubscribe_CancelIsIdempotent(t *testing.T) {
	hub := NewHub()
	_, cancel := hub.subscribe()
	cancel()
	cancel()
	// A broadcast after both cancels must not panic.
	hub.Upsert("t1", func(a *Agent) {})
}

// TestBroadcast_ConcurrentCancelIsSafe is the regression gate
// for the send-on-closed-channel panic. Subscribers come and
// go while the hub is being written to, which is exactly what
// a browser refreshing does. Before the fix this panicked;
// a panic in any goroutine fails the test binary.
//
// The shape matters. The first attempt at this test churned
// subscribe and cancel against the writers and the mutant
// survived: the vulnerable window is one mutex handoff wide,
// and keeping the subscriber table small keeps the window
// narrow. So instead there is a standing set of subscribers
// that are cancelled repeatedly, which keeps a channel in the
// hub's map at the moment a writer is most likely to be
// sending to it.
func TestBroadcast_ConcurrentCancelIsSafe(t *testing.T) {
	hub := NewHub()

	// A standing set of subscribers, torn down and rebuilt.
	// The set is partitioned across the cancellers so each one
	// owns its slots outright rather than sharing them.
	const cancellers = 6
	const slotsPerCanceller = 2
	live := cancellers * slotsPerCanceller
	channels := make([]func(), live)
	for i := range channels {
		_, cancel := hub.subscribe()
		channels[i] = cancel
	}

	var wg sync.WaitGroup
	stop := make(chan struct{})

	// Writers publish continuously. Every publish is a
	// broadcast, and every broadcast iterates the subscriber
	// table, so this is the side that would panic.
	for w := 0; w < 6; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				hub.SetStatus("t1", StatusRunning, "")
			}
		}()
	}

	// Cancellers resubscribe in a loop, so the table always
	// holds channels that are closing right now.
	//
	// Each goroutine owns its own slots. An earlier version
	// shared one `channels` slice across all six, which is a
	// data race in the TEST: -race flagged concurrent writes to
	// the slice elements. The hub was fine; the harness was
	// not, and a racy harness is a harness that reports
	// nonsense about the code under test.
	for c := 0; c < cancellers; c++ {
		wg.Add(1)
		go func(mine []func()) {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				for i := range mine {
					mine[i]() // close + delete
					_, mine[i] = hub.subscribe()
				}
			}
		}(channels[c*slotsPerCanceller : (c+1)*slotsPerCanceller])
	}

	time.Sleep(300 * time.Millisecond)
	close(stop)
	wg.Wait()
}

// TestSubscribe_ClosedChannelIsDrained asserts a cancelled
// subscriber's channel is closed, so a reader ranging over it
// terminates rather than blocking forever.
func TestSubscribe_ClosedChannelIsDrained(t *testing.T) {
	hub := NewHub()
	ch, cancel := hub.subscribe()
	cancel()
	select {
	case _, open := <-ch:
		if open {
			// A buffered event is fine; the channel is closed.
			if _, stillOpen := <-ch; stillOpen {
				t.Error("channel still open after cancel")
			}
		}
	case <-time.After(2 * time.Second):
		t.Fatal("cancelled subscriber's channel was never closed")
	}
}

// TestBroadcast_DropsRatherThanBlocks covers the slow-consumer
// contract: a subscriber that stops reading must not stall a
// writer.
func TestBroadcast_DropsRatherThanBlocks(t *testing.T) {
	hub := NewHub()
	_, cancel := hub.subscribe() // never drained
	defer cancel()

	done := make(chan struct{})
	go func() {
		// Far more than subBuffer, so this can only finish if
		// sends drop rather than block.
		for i := 0; i < subBuffer*10; i++ {
			hub.SetStatus("t1", StatusRunning, "")
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("a non-draining subscriber blocked the writer")
	}
}
