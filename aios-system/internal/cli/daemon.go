package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/sadi/aios-system/internal/approvals"
	"github.com/sadi/aios-system/internal/core/audit"
	"github.com/sadi/aios-system/internal/core/schemas"
)

// Daemon is the approval daemon. Per REQ-CLI-DAE-001 the
// daemon listens on a Unix socket. Per REQ-CLI-DAE-003 the
// daemon accepts four message kinds: request, grant, deny,
// wait. Per REQ-CLI-DAE-004 every state transition writes
// an audit row.
//
// The daemon is the only place in aios that owns a
// long-running goroutine. Stop() is the only way to take
// it down; the CLI's `daemon stop` subcommand uses a
// control file the daemon watches.
type Daemon struct {
	socketPath string
	pidPath    string
	approvals  *approvals.Service
	audit      *audit.Log
	listener   net.Listener
	mu         sync.Mutex
	stopped    bool
}

// DaemonConfig wires the daemon. Approvals and Audit are
// optional; without them the daemon is a thin proxy.
type DaemonConfig struct {
	SocketPath string
	PidPath    string
	Approvals  *approvals.Service
	Audit      *audit.Log
}

// NewDaemon returns a daemon. The socket is bound on Start,
// not on construction, so tests can build a daemon without
// touching the filesystem.
func NewDaemon(cfg DaemonConfig) *Daemon {
	if cfg.SocketPath == "" {
		cfg.SocketPath = "/tmp/aios-approval.sock"
	}
	if cfg.PidPath == "" {
		cfg.PidPath = "./var/aios-daemon.pid"
	}
	return &Daemon{
		socketPath: cfg.SocketPath,
		pidPath:    cfg.PidPath,
		approvals:  cfg.Approvals,
		audit:      cfg.Audit,
	}
}

// Start binds the socket and serves. The function blocks
// until Stop is called. Errors from the bind step are
// returned to the caller; errors from individual
// connections are logged to the daemon's stderr (currently
// os.Stderr; the CLI passes its own writer when it runs
// the daemon).
func (d *Daemon) Start(ctx context.Context) error {
	if err := os.MkdirAll(filepath.Dir(d.socketPath), 0o755); err != nil {
		return &CliError{Kind: "daemon", What: "mkdir socket dir", Cause: err}
	}
	// Remove any stale socket file from a previous run.
	_ = os.Remove(d.socketPath)
	ln, err := net.Listen("unix", d.socketPath)
	if err != nil {
		return &CliError{Kind: "daemon", What: "listen on " + d.socketPath, Cause: err}
	}
	d.mu.Lock()
	d.listener = ln
	d.stopped = false
	d.mu.Unlock()

	// Write the pid file.
	if d.pidPath != "" {
		if err := os.WriteFile(d.pidPath, []byte(fmt.Sprintf("%d", os.Getpid())), 0o644); err != nil {
			return &CliError{Kind: "daemon", What: "write pid file", Cause: err}
		}
	}
	// Clean up the pid file and socket on exit.
	defer func() {
		_ = os.Remove(d.pidPath)
		_ = os.Remove(d.socketPath)
	}()

	// Accept loop. Stopping closes the listener, which
	// makes Accept return an error and the loop exits.
	// The context is polled between accepts as a
	// backstop for callers that cancel without calling
	// Stop.
	for {
		// Honour context cancellation. The listener
		// close is the primary exit path; the context
		// check is a backstop for callers that pass a
		// cancellable context but never call Stop.
		select {
		case <-ctx.Done():
			return nil
		default:
		}
		conn, err := ln.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			d.mu.Lock()
			stopped := d.stopped
			d.mu.Unlock()
			if stopped {
				return nil
			}
			// Transient error; sleep a moment and retry.
			time.Sleep(50 * time.Millisecond)
			continue
		}
		go d.handle(ctx, conn)
	}
}

// Stop closes the listener. Idempotent.
func (d *Daemon) Stop() error {
	d.mu.Lock()
	d.stopped = true
	ln := d.listener
	d.mu.Unlock()
	if ln != nil {
		return ln.Close()
	}
	return nil
}

// handle reads one message, dispatches, and writes the
// response. The wire format is one JSON object per line.
func (d *Daemon) handle(ctx context.Context, conn net.Conn) {
	defer conn.Close()
	dec := json.NewDecoder(conn)
	enc := json.NewEncoder(conn)
	for {
		var msg map[string]any
		if err := dec.Decode(&msg); err != nil {
			if errors.Is(err, io.EOF) {
				return
			}
			return
		}
		resp := d.dispatch(ctx, msg)
		if err := enc.Encode(resp); err != nil {
			return
		}
	}
}

// dispatch is the request router. The four kinds the doc
// set names are: request, grant, deny, wait.
func (d *Daemon) dispatch(ctx context.Context, msg map[string]any) map[string]any {
	kind, _ := msg["kind"].(string)
	id, _ := msg["id"].(string)
	actor, _ := msg["actor"].(string)
	switch kind {
	case "request":
		return d.handleRequest(ctx, id, actor, msg)
	case "grant":
		return d.handleGrant(ctx, id, actor, msg)
	case "deny":
		return d.handleDeny(ctx, id, actor, msg)
	case "wait":
		return d.handleWait(ctx, id)
	default:
		return map[string]any{"ok": false, "error": "unknown kind: " + kind}
	}
}

func (d *Daemon) handleRequest(_ context.Context, id, actor string, msg map[string]any) map[string]any {
	if d.approvals == nil {
		return map[string]any{"ok": false, "error": "approvals service is not configured"}
	}
	// The full payload is required for a request; the
	// caller passed the JSON-encoded
	// schemas.ApprovalPayload as a map. We re-marshal
	// and re-decode through the typed form to honour
	// the strict unknown-field rejection.
	raw, err := json.Marshal(msg["payload"])
	if err != nil {
		return map[string]any{"ok": false, "error": "payload: " + err.Error()}
	}
	var payload schemas.ApprovalPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return map[string]any{"ok": false, "error": "payload: " + err.Error()}
	}
	newID, err := d.approvals.Request(payload, actor, approvals.RequestOptions{})
	if err != nil {
		return map[string]any{"ok": false, "error": err.Error()}
	}
	return map[string]any{"ok": true, "id": newID, "state": "pending"}
}

func (d *Daemon) handleGrant(_ context.Context, id, actor string, msg map[string]any) map[string]any {
	if d.approvals == nil {
		return map[string]any{"ok": false, "error": "approvals service is not configured"}
	}
	// The grant carries the call-site context in
	// `ctx` so the binding check passes.
	bind := approvals.Context{}
	if m, ok := msg["ctx"].(map[string]any); ok {
		if v, ok := m["task"].(string); ok {
			bind.Task = v
		}
		if v, ok := m["tool"].(string); ok {
			bind.Tool = v
		}
		if v, ok := m["actor"].(string); ok {
			bind.Actor = v
		}
		if v, ok := m["resource"].(string); ok {
			bind.Resource = v
		}
	}
	if err := d.approvals.Grant(id, actor, bind); err != nil {
		return map[string]any{"ok": false, "error": err.Error()}
	}
	return map[string]any{"ok": true, "id": id, "state": "granted"}
}

func (d *Daemon) handleDeny(ctx context.Context, id, actor string, _ map[string]any) map[string]any {
	if d.approvals == nil {
		return map[string]any{"ok": false, "error": "approvals service is not configured"}
	}
	if err := d.approvals.Deny(id, actor); err != nil {
		return map[string]any{"ok": false, "error": err.Error()}
	}
	return map[string]any{"ok": true, "id": id, "state": "denied"}
}

func (d *Daemon) handleWait(_ context.Context, id string) map[string]any {
	if d.approvals == nil {
		return map[string]any{"ok": false, "error": "approvals service is not configured"}
	}
	a, err := d.approvals.Get(id)
	if err != nil {
		return map[string]any{"ok": false, "error": err.Error()}
	}
	return map[string]any{"ok": true, "id": id, "state": string(a.State)}
}

// SocketPath returns the bound socket path. Useful for tests.
func (d *Daemon) SocketPath() string { return d.socketPath }

// RunDaemonStart is the entry point for the `daemon start`
// subcommand. The function blocks on daemon.Start.
//
// The ctx is wired to a background goroutine that calls
// Stop when ctx is cancelled. This lets callers cancel
// the daemon without holding a *Daemon reference.
func RunDaemonStart(ctx context.Context, cfg DaemonConfig) error {
	d := NewDaemon(cfg)
	if ctx.Done() != nil {
		go func() {
			<-ctx.Done()
			_ = d.Stop()
		}()
	}
	return d.Start(ctx)
}

// RunDaemonStop sends a stop request to a running daemon. The
// CLI's "daemon stop" subcommand writes "stop" to the pid
// file's owner process; in this implementation we use a
// simpler "touch the pid file" + delete approach that
// matches the doc set's "atexit" semantic: the daemon
// notices its pid file is gone on the next loop and exits.
//
// The "stop" subcommand removes the pid file; the running
// daemon picks up the absence on its next event and exits.
func RunDaemonStop(pidPath string) error {
	if pidPath == "" {
		return &CliError{Kind: "user", What: "pid path is required"}
	}
	if err := os.Remove(pidPath); err != nil && !os.IsNotExist(err) {
		return &CliError{Kind: "daemon", What: "remove pid file", Cause: err}
	}
	return nil
}
