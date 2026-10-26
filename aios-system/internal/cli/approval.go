package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"github.com/sadi/aios-system/internal/runtime"
)

// Approval prompt answers an approval request from a human at
// the terminal.
//
// Before this existed the run path raised
// *runtime.WaitingForApproval and stopped with
// "runtime: waiting for approval ask:t1" and exit code 2.
// Nothing reachable from a terminal could resolve it:
// approvals.Service.Grant existed and the daemon handled
// grant/deny on its socket, but no CLI command spoke that
// protocol, and the daemon's in-memory approval store was in
// the daemon's process while the run was in the CLI's. So the
// gate the default policy puts on run_shell was a dead end --
// the one feature meant to keep an agent from touching a
// shell without a human was the one thing a human could not
// complete.
//
// The prompt is deliberately narrow: it approves THIS one
// call, bound to the task and tool the policy named. It does
// not offer "always allow", because a persistent grant is a
// different (and larger) decision than the policy asked for.
type ApprovalPrompt struct {
	In  io.Reader
	Out io.Writer

	// AssumeYes auto-approves without asking. Wired to
	// `aios run --yes`. It is loud on purpose: every
	// auto-approval is announced, so a log or a transcript
	// shows that a human was never asked.
	AssumeYes bool

	// Timeout bounds how long the prompt waits for input.
	// Without it, `aios run` in a non-interactive context
	// (a CI job, a piped stdin) would block forever on a
	// read that will never return.
	Timeout time.Duration
}

// approvalReadTimeout is the default wait for a keystroke.
//
// A human reading a prompt and answering is slow; a pipe
// with nobody at the other end is instant-EOF. Sixty seconds
// is long enough to read and decide, short enough that a
// forgotten run does not hang forever.
const approvalReadTimeout = 60 * time.Second

// Approve implements ApprovalHandler.
func (p *ApprovalPrompt) Approve(ctx context.Context, wfa *runtime.WaitingForApproval) (bool, error) {
	if p == nil || wfa == nil {
		return false, errors.New("approval: nil request")
	}
	payload, _ := approvalPayload(wfa)

	if p.AssumeYes {
		fmt.Fprintf(p.Out, "approval %s: AUTO-APPROVED (--yes) task=%s action=%s target=%s\n",
			wfa.ApprovalID, payload.RequesterID, payload.Action, payload.Target)
		return true, nil
	}

	fmt.Fprintf(p.Out, "\nAPPROVAL REQUIRED\n")
	fmt.Fprintf(p.Out, "  id     : %s\n", wfa.ApprovalID)
	if payload.Action != "" {
		fmt.Fprintf(p.Out, "  action : %s\n", payload.Action)
	}
	if payload.Target != "" {
		fmt.Fprintf(p.Out, "  target : %s\n", payload.Target)
	}
	if payload.Reason != "" {
		fmt.Fprintf(p.Out, "  reason : %s\n", payload.Reason)
	}
	fmt.Fprintf(p.Out, "Allow this one call? [y/N] ")

	ok, err := p.readYesNo(ctx)
	if err != nil {
		return false, err
	}
	if ok {
		fmt.Fprintf(p.Out, "approved: %s\n", wfa.ApprovalID)
		return true, nil
	}
	// A denial is a decision, not a failure: the plan stops
	// with stopped_reason=operator denied and the operator
	// can resume after editing the policy.
	fmt.Fprintf(p.Out, "denied: %s\n", wfa.ApprovalID)
	return false, nil
}

// readYesNo reads one answer from the prompt's input.
//
// It treats EOF as a denial rather than an error. A run
// whose stdin is closed or a pipe is not an operator saying
// "no"; it is a run that cannot ask, and the safe reading of
// "cannot ask" for a gated action is "do not do it".
func (p *ApprovalPrompt) readYesNo(ctx context.Context) (bool, error) {
	type result struct {
		line string
		err  error
	}
	ch := make(chan result, 1)
	go func() {
		r := bufio.NewReader(p.In)
		line, err := r.ReadString('\n')
		ch <- result{line: line, err: err}
	}()

	timeout := p.Timeout
	if timeout <= 0 {
		timeout = approvalReadTimeout
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case r := <-ch:
		if r.err != nil && strings.TrimSpace(r.line) == "" {
			if errors.Is(r.err, io.EOF) {
				return false, nil // no operator: deny
			}
			return false, fmt.Errorf("approval: read input: %w", r.err)
		}
		switch strings.ToLower(strings.TrimSpace(r.line)) {
		case "y", "yes":
			return true, nil
		default:
			return false, nil
		}
	case <-ctx.Done():
		return false, ctx.Err()
	case <-timer.C:
		fmt.Fprintf(p.Out, "\n  (no answer in %s; treating as denied)\n", timeout)
		return false, nil
	}
}

// ---------------------------------------------------------------------------
// Daemon client
// ---------------------------------------------------------------------------

// daemonClient speaks the approval daemon's line-JSON
// protocol: one request object per line in, one response
// object per line out. It is the same wire format
// daemon.handle reads.
type daemonClient struct {
	conn net.Conn
	dec  *json.Decoder
	enc  *json.Encoder
}

// dialDaemon opens a connection to the approval daemon.
func dialDaemon(socketPath string, timeout time.Duration) (*daemonClient, error) {
	if socketPath == "" {
		return nil, errors.New("approval: socket path is empty")
	}
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	conn, err := net.DialTimeout("unix", socketPath, timeout)
	if err != nil {
		return nil, fmt.Errorf("approval: connect to daemon at %s (is `aios daemon start` running?): %w", socketPath, err)
	}
	return &daemonClient{conn: conn, dec: json.NewDecoder(conn), enc: json.NewEncoder(conn)}, nil
}

func (c *daemonClient) Close() error { return c.conn.Close() }

func (c *daemonClient) send(req map[string]any) (map[string]any, error) {
	if err := c.enc.Encode(req); err != nil {
		return nil, fmt.Errorf("approval: send: %w", err)
	}
	var resp map[string]any
	if err := c.dec.Decode(&resp); err != nil {
		return nil, fmt.Errorf("approval: receive: %w", err)
	}
	if ok, _ := resp["ok"].(bool); !ok {
		msg, _ := resp["error"].(string)
		if msg == "" {
			msg = "daemon refused the request"
		}
		return nil, fmt.Errorf("approval: %s", msg)
	}
	return resp, nil
}

// DaemonApprovalHandler resolves approvals through a running
// daemon.
//
// This is the multi-writer path: a `run` in one process
// parks its approval, and `aios approve <id>` in another
// resolves it. It is used when the caller passes
// --socket explicitly and is not a terminal, so an
// operator can drive approvals from a second window or a
// separate operator tool.
type DaemonApprovalHandler struct {
	SocketPath string
	Timeout    time.Duration
	Out        io.Writer
}

// Approve implements ApprovalHandler. It forwards the
// runtime request to the daemon, which owns the canonical
// approval record.
func (d *DaemonApprovalHandler) Approve(ctx context.Context, wfa *runtime.WaitingForApproval) (bool, error) {
	if d == nil || wfa == nil {
		return false, errors.New("approval: nil request")
	}
	payload, _ := approvalPayload(wfa)
	if payload.ApprovalID == "" {
		payload.ApprovalID = wfa.ApprovalID
	}

	c, err := dialDaemon(d.SocketPath, d.Timeout)
	if err != nil {
		return false, err
	}
	defer c.Close()

	resp, err := c.send(map[string]any{
		"kind":    "request",
		"actor":   payload.RequesterID,
		"payload": payload,
	})
	if err != nil {
		return false, err
	}
	id, _ := resp["id"].(string)
	if id == "" {
		id = wfa.ApprovalID
	}
	// Announce the id BEFORE waiting. The operator has to know
	// which id to act on, and the wait below blocks until
	// somebody decides -- so if the id were only printed after
	// the decision, the operator would have no way to find it.
	// This is the line that names the approval a second
	// terminal can approve.
	if d.Out != nil {
		fmt.Fprintf(d.Out, "approval %s: waiting for a decision (approve it with `aios approve %s`)\n", id, id)
	}

	// The daemon's "wait" BLOCKS until a human grants or
	// denies, and the response that comes back is the
	// terminal state -- so this is the response that matters.
	// Reading the state off the request reply instead would
	// report "pending" and treat every approval as a denial.
	resp, err = c.send(map[string]any{"kind": "wait", "id": id})
	if err != nil {
		return false, err
	}
	state, _ := resp["state"].(string)
	if d.Out != nil {
		fmt.Fprintf(d.Out, "approval %s: %s\n", id, state)
	}
	return state == approvalsStateGranted, nil
}

// approvalsStateGranted mirrors approvals.StateGranted so
// this file does not need the approvals import for one
// constant. Kept as a string comparison deliberately: the
// daemon speaks the state as a wire string.
const approvalsStateGranted = "granted"

// Compile-time checks that both handlers satisfy the
// interface the run loop calls.
var (
	_ ApprovalHandler = (*ApprovalPrompt)(nil)
	_ ApprovalHandler = (*DaemonApprovalHandler)(nil)
)
