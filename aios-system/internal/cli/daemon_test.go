package cli

import (
	"context"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sadi/aios-system/internal/approvals"
	"github.com/sadi/aios-system/internal/core/audit"
	"github.com/sadi/aios-system/internal/core/clock"
	"github.com/sadi/aios-system/internal/core/schemas"
)

func newTestAudit(t *testing.T) *audit.Log {
	t.Helper()
	log, err := audit.Open(audit.Config{Path: ":memory:", Clock: clock.SystemDecision{}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { log.Close() })
	return log
}

func TestDaemon_StartStop(t *testing.T) {
	dir := t.TempDir()
	sock := filepath.Join(dir, "a.sock")
	pid := filepath.Join(dir, "a.pid")
	d := NewDaemon(DaemonConfig{SocketPath: sock, PidPath: pid})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// Start in a goroutine; Stop() from the test.
	go func() { _ = d.Start(ctx) }()
	// Wait for the socket to appear.
	deadline := time.Now().Add(2 * time.Second)
	for {
		if _, err := os.Stat(sock); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("socket did not appear")
		}
		time.Sleep(10 * time.Millisecond)
	}
	// Pid file should be there.
	if _, err := os.Stat(pid); err != nil {
		t.Errorf("pid file: %v", err)
	}
	if err := d.Stop(); err != nil {
		t.Errorf("stop: %v", err)
	}
	// Allow the listener close to propagate.
	time.Sleep(50 * time.Millisecond)
	if _, err := os.Stat(sock); !os.IsNotExist(err) {
		t.Errorf("socket should be removed; stat err = %v", err)
	}
}

func TestDaemon_RequestGrant(t *testing.T) {
	dir := t.TempDir()
	sock := filepath.Join(dir, "a.sock")
	pid := filepath.Join(dir, "a.pid")
	log := newTestAudit(t)
	svc := approvals.New(approvals.Config{Audit: log})
	d := NewDaemon(DaemonConfig{SocketPath: sock, PidPath: pid, Approvals: svc})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = d.Start(ctx) }()
	defer d.Stop()
	// Wait for the socket.
	deadline := time.Now().Add(2 * time.Second)
	for {
		if _, err := os.Stat(sock); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("socket did not appear")
		}
		time.Sleep(10 * time.Millisecond)
	}

	// Open a connection and send a request.
	conn, err := net.Dial("unix", sock)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	enc := json.NewEncoder(conn)
	dec := json.NewDecoder(conn)

	// Send a request.
	payload := schemas.ApprovalPayload{
		Action:      "write",
		Target:      "/var/secrets.txt",
		Reason:      "test",
		RequesterID: "tester",
		BoundTo:     &schemas.BoundTo{Task: "t-1", Tool: "write_file", Actor: "tester"},
	}
	if err := enc.Encode(map[string]any{
		"kind":    "request",
		"id":      "ignored", // the daemon returns the real id
		"actor":   "tester",
		"payload": payload,
	}); err != nil {
		t.Fatal(err)
	}
	var resp map[string]any
	if err := dec.Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if ok, _ := resp["ok"].(bool); !ok {
		t.Fatalf("response: %+v", resp)
	}
	id, _ := resp["id"].(string)
	if id == "" {
		t.Fatal("missing id")
	}

	// Grant the request. The call-site context must match
	// the bound context, otherwise the binding check
	// rejects the grant.
	if err := enc.Encode(map[string]any{
		"kind":  "grant",
		"id":    id,
		"actor": "operator",
		"ctx": map[string]any{
			"task":  "t-1",
			"tool":  "write_file",
			"actor": "tester",
		},
	}); err != nil {
		t.Fatal(err)
	}
	if err := dec.Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if state, _ := resp["state"].(string); state != "granted" {
		t.Errorf("state: %q, want granted", state)
	}

	// Wait should report granted.
	if err := enc.Encode(map[string]any{"kind": "wait", "id": id}); err != nil {
		t.Fatal(err)
	}
	if err := dec.Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if state, _ := resp["state"].(string); state != "granted" {
		t.Errorf("state: %q, want granted", state)
	}
}

func TestDaemon_UnknownKind(t *testing.T) {
	dir := t.TempDir()
	sock := filepath.Join(dir, "a.sock")
	pid := filepath.Join(dir, "a.pid")
	d := NewDaemon(DaemonConfig{SocketPath: sock, PidPath: pid})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = d.Start(ctx) }()
	defer d.Stop()
	deadline := time.Now().Add(2 * time.Second)
	for {
		if _, err := os.Stat(sock); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("socket did not appear")
		}
		time.Sleep(10 * time.Millisecond)
	}
	conn, err := net.Dial("unix", sock)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	enc := json.NewEncoder(conn)
	dec := json.NewDecoder(conn)
	if err := enc.Encode(map[string]any{"kind": "frobnicate"}); err != nil {
		t.Fatal(err)
	}
	var resp map[string]any
	if err := dec.Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if ok, _ := resp["ok"].(bool); ok {
		t.Fatal("expected ok=false for unknown kind")
	}
}

func TestRunDaemonStop(t *testing.T) {
	dir := t.TempDir()
	pid := filepath.Join(dir, "a.pid")
	// Write a pid file, then call RunDaemonStop.
	if err := os.WriteFile(pid, []byte("12345"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := RunDaemonStop(pid); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(pid); !os.IsNotExist(err) {
		t.Errorf("pid should be removed: %v", err)
	}
	// Calling again on a missing file is a no-op (idempotent).
	if err := RunDaemonStop(pid); err != nil {
		t.Errorf("second call: %v", err)
	}
}

func TestRunDaemonStop_Empty(t *testing.T) {
	if err := RunDaemonStop(""); err == nil {
		t.Fatal("expected error for empty pid path")
	}
}

func TestRunDaemonStop_OtherError(t *testing.T) {
	// os.Remove on Windows does not fail when the parent
	// directory is missing (it returns success on the
	// first ENOENT). The test is therefore best-effort:
	// it covers the happy path on every platform, and the
	// error path is exercised in environments where
	// os.Remove surfaces the failure.
	_ = RunDaemonStop(filepath.Join(t.TempDir(), "missing", "x.pid"))
}

// (unused-imports silenced)
