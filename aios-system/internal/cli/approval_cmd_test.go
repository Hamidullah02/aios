package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sadi/aios-system/internal/approvals"
	"github.com/sadi/aios-system/internal/core/schemas"
)

// encJSON / decJSON are the line-JSON codec the daemon uses.
func encJSON(w net.Conn, v any) error {
	return json.NewEncoder(w).Encode(v)
}

func decJSON(r net.Conn, v any) error {
	return json.NewDecoder(r).Decode(v)
}

// startTestDaemon brings up a daemon on a temp socket and
// returns the socket path plus a request id already pending.
func startTestDaemon(t *testing.T) (sock, id string) {
	t.Helper()
	dir := t.TempDir()
	sock = filepath.Join(dir, "a.sock")
	pid := filepath.Join(dir, "a.pid")
	log := newTestAudit(t)
	svc := approvals.New(approvals.Config{Audit: log})
	d := NewDaemon(DaemonConfig{SocketPath: sock, PidPath: pid, Approvals: svc})
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go func() { _ = d.Start(ctx) }()
	t.Cleanup(func() { _ = d.Stop() })

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

	// Leave one request pending for the command to resolve.
	conn, err := net.Dial("unix", sock)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	payload := schemas.ApprovalPayload{
		Action:      "shell",
		Target:      "echo hi",
		Reason:      "gated",
		RequesterID: "t1",
	}
	if err := encJSON(conn, map[string]any{
		"kind": "request", "actor": "t1", "payload": payload,
	}); err != nil {
		t.Fatal(err)
	}
	var resp map[string]any
	if err := decJSON(conn, &resp); err != nil {
		t.Fatal(err)
	}
	id, _ = resp["id"].(string)
	if id == "" {
		t.Fatal("daemon did not return an approval id")
	}
	return sock, id
}

// TestRunApprovalDecision_Approve pins the wire verb.
//
// The CLI verb is `approve`; the daemon protocol verb is
// `grant`. Sending "approve" gets `unknown kind: approve`
// back from the daemon, which is exactly what the first
// version did -- the command was unusable and the daemon was
// the only thing that could have told us.
func TestRunApprovalDecision_Approve(t *testing.T) {
	sock, id := startTestDaemon(t)

	var out, errOut bytes.Buffer
	code := runApprovalDecision(&Args{SocketPath: sock, Positional: []string{id}}, true, &out, &errOut)
	if code != ExitOK {
		t.Fatalf("exit = %d, stderr = %q", code, errOut.String())
	}
	if !strings.Contains(out.String(), id) {
		t.Errorf("output should name the approval; got %q", out.String())
	}
	if !strings.Contains(out.String(), "granted") {
		t.Errorf("output should report the granted state; got %q", out.String())
	}
}

// TestRunApprovalDecision_Deny covers the other verb.
func TestRunApprovalDecision_Deny(t *testing.T) {
	sock, id := startTestDaemon(t)

	var out, errOut bytes.Buffer
	code := runApprovalDecision(&Args{SocketPath: sock, Positional: []string{id}}, false, &out, &errOut)
	if code != ExitOK {
		t.Fatalf("exit = %d, stderr = %q", code, errOut.String())
	}
	if !strings.Contains(out.String(), "denied") {
		t.Errorf("output should report the denied state; got %q", out.String())
	}
}

// TestRunApprovalDecision_MissingID covers the usage error.
func TestRunApprovalDecision_MissingID(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := runApprovalDecision(&Args{SocketPath: "unused"}, true, &out, &errOut); code != ExitUserError {
		t.Errorf("exit = %d, want %d (user error)", code, ExitUserError)
	}
	if !strings.Contains(errOut.String(), "approval-id") {
		t.Errorf("usage error should show the argument name; got %q", errOut.String())
	}
}

// TestRunApprovalDecision_DaemonDown covers the unreachable
// case: the message must name the socket and the remedy, or
// the operator is left with a bare dial error.
func TestRunApprovalDecision_DaemonDown(t *testing.T) {
	var out, errOut bytes.Buffer
	sock := filepath.Join(t.TempDir(), "missing.sock")
	code := runApprovalDecision(&Args{SocketPath: sock, Positional: []string{"x"}}, true, &out, &errOut)
	if code != ExitSystemError {
		t.Errorf("exit = %d, want %d (system error)", code, ExitSystemError)
	}
	msg := errOut.String()
	if !strings.Contains(msg, "daemon start") {
		t.Errorf("error should name the remedy; got %q", msg)
	}
	if !strings.Contains(msg, "missing.sock") {
		t.Errorf("error should name the socket; got %q", msg)
	}
}

// TestRunApprovalDecision_WorksWithoutPolicy pins
// REQ-CLI-APV-004: resolving an approval must not require a
// local policy, because a broken policy is precisely when
// someone needs to clear what is pending.
func TestRunApprovalDecision_WorksWithoutPolicy(t *testing.T) {
	sock, id := startTestDaemon(t)
	// A workspace with no policy.yaml at all.
	args := &Args{
		SocketPath: sock,
		ConfigPath: filepath.Join(t.TempDir(), "var", "config.yaml"),
		Positional: []string{id},
	}
	if code := runApprovalDecision(args, true, &bytes.Buffer{}, &bytes.Buffer{}); code != ExitOK {
		t.Fatalf("approve should not need a policy; exit = %d", code)
	}
}