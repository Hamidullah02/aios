package cli

import (
	"fmt"
	"io"
	"time"

	"github.com/sadi/aios-system/internal/webui"
)

// DaemonApprovalResolver resolves UI decisions through a
// running approval daemon.
//
// It is the `aios web` case: a browser in its own process,
// with no orchestrator and no approvals service. The daemon
// owns the pending approvals, so this polls it for the queue
// and forwards decisions to it.
//
// Polling, not subscribing. The daemon protocol has no push
// for approvals -- `wait` blocks one caller on one id, which
// is the wrong shape for a dashboard that shows a queue. A
// short poll is honest about that: it costs one round trip
// every few seconds against a loopback socket, and it works
// with a daemon that was started before the browser opened.
type DaemonApprovalResolver struct {
	SocketPath string
	Out        io.Writer
}

// pollInterval is how often the dashboard re-reads the
// queue.
//
// Two seconds is a compromise: fast enough that an approval
// appears to arrive as the run raises it, slow enough that an
// idle dashboard is not making a socket call a hundred times
// a minute.
const pollInterval = 2 * time.Second

// NewDaemonApprovalResolver returns a resolver for a
// standalone UI backed by the daemon on socketPath.
func NewDaemonApprovalResolver(socketPath string, out io.Writer) *DaemonApprovalResolver {
	return &DaemonApprovalResolver{SocketPath: socketPath, Out: out}
}

// Resolve applies an operator's decision through the daemon.
func (d *DaemonApprovalResolver) Resolve(id string, approve bool) (string, error) {
	if d == nil {
		return "", webui.ErrNoResolver
	}
	c, err := dialDaemon(d.SocketPath, 5*time.Second)
	if err != nil {
		return "", err
	}
	defer c.Close()

	// The wire verb is "grant"; `deny` is its own verb. The
	// CLI verb "approve" is not a protocol message.
	verb := "deny"
	if approve {
		verb = "grant"
	}
	resp, err := c.send(map[string]any{
		"kind":  verb,
		"id":    id,
		"actor": "operator",
	})
	if err != nil {
		return "", err
	}
	state, _ := resp["state"].(string)
	if state == "" {
		if approve {
			state = "granted"
		} else {
			state = "denied"
		}
	}
	if d.Out != nil {
		fmt.Fprintf(d.Out, "web: approval %s: %s\n", id, state)
	}
	return state, nil
}

// Sync pulls the daemon's pending approvals into hub and
// returns how many are now waiting.
//
// It is called on a timer and replaces the whole list each
// time, because a decision made elsewhere (a second operator
// at a terminal, `aios deny` from a script) must disappear
// from this view. Merging deltas would keep a resolved item
// on screen until a reconnect.
func (d *DaemonApprovalResolver) Sync(hub *webui.Hub) (int, error) {
	if d == nil || hub == nil {
		return 0, webui.ErrNoResolver
	}
	c, err := dialDaemon(d.SocketPath, 5*time.Second)
	if err != nil {
		return 0, err
	}
	defer c.Close()

	resp, err := c.send(map[string]any{"kind": "list"})
	if err != nil {
		return 0, err
	}
	raw, _ := resp["approvals"].([]any)
	out := make([]webui.PendingApproval, 0, len(raw))
	for _, item := range raw {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		p := webui.PendingApproval{}
		p.ID, _ = m["id"].(string)
		p.TaskID, _ = m["task"].(string)
		p.Action, _ = m["action"].(string)
		p.Target, _ = m["target"].(string)
		p.Reason, _ = m["reason"].(string)
		p.Actor, _ = m["actor"].(string)
		if p.ID == "" {
			continue
		}
		out = append(out, p)
	}

	// Replace rather than merge, and remove anything the daemon
	// no longer reports so a decision taken elsewhere clears
	// the card.
	for _, want := range out {
		hub.AddApproval(want)
	}
	for _, have := range hub.PendingApprovals() {
		found := false
		for _, want := range out {
			if want.ID == have.ID {
				found = true
				break
			}
		}
		if !found {
			hub.RemoveApproval(have.ID)
		}
	}
	return len(out), nil
}

// pollDaemonApprovals keeps the hub's approval list in step
// with the daemon until ctx-like stop is closed.
//
// It returns immediately on the first successful sync so the
// caller does not wait pollInterval to render the queue. A
// daemon that is not running is not an error worth stopping
// for: the dashboard still renders its history, and the
// message on stderr says where decisions go instead.
func pollDaemonApprovals(res *DaemonApprovalResolver, hub *webui.Hub, stop <-chan struct{}, stderr io.Writer) {
	if res == nil || hub == nil {
		return
	}
	warned := false
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	for {
		if _, err := res.Sync(hub); err != nil {
			if !warned && stderr != nil {
				fmt.Fprintf(stderr, "note: approval daemon not reachable (%v); decide at the run terminal or with `aios approve <id>`\n", err)
				warned = true
			}
		} else {
			warned = false
		}
		select {
		case <-stop:
			return
		case <-ticker.C:
		}
	}
}