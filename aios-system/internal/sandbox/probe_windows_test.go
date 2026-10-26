//go:build windows

package sandbox

import (
	"os"
	"os/exec"
	"testing"
	"time"

	"golang.org/x/sys/windows"
)

// TestProcessRunning_ReportsTerminatedProcessAsDead pins the
// liveness probe against the case that broke it.
//
// A terminated process keeps its process object alive for as long
// as any handle to it is open, and OpenProcess(SYNCHRONIZE)
// succeeds for it. So "the open worked" is not evidence that the
// process is running.
//
// This matters for the sandbox specifically: the runner holds the
// child's handle, so a probe that stops at a successful open
// reports a killed grandchild as alive for as long as the killer
// holds a handle. The tree-kill test then fails for a reason that
// is not a containment failure -- and worse, a real containment
// failure would be indistinguishable from it.
//
// The probe must wait on the handle, not just open it.
func TestProcessRunning_ReportsTerminatedProcessAsDead(t *testing.T) {
	cmd := exec.Command(os.Args[0], "-test.run=^TestHelperLivenessChild$")
	cmd.Env = append(os.Environ(), "AIOS_LIVENESS_HELPER=1")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	pid := cmd.Process.Pid

	// The child must be up before it can be probed.
	deadline := time.Now().Add(5 * time.Second)
	for !processAlive(t, pid) && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if !processAlive(t, pid) {
		t.Fatal("child never reported as alive")
	}

	// Kill it but do NOT wait: the handle stays open, which is
	// the state that used to be misreported as "alive".
	if err := cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}

	// Give the kernel a moment to mark it terminated.
	deadline = time.Now().Add(5 * time.Second)
	for processAlive(t, pid) && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if processAlive(t, pid) {
		t.Error("a terminated process was reported as alive")
	}

	// Reap it so the test does not leave a process behind.
	_ = cmd.Wait()
}

// TestProcessRunning_ReportsLiveProcessAsAlive is the other
// direction: the fix must not make the probe claim everything is
// dead.
//
// A process that reports as dead when it is alive would turn the
// tree-kill test into a test that proves nothing, and would make
// the runner believe it has nothing left to kill.
func TestProcessRunning_ReportsLiveProcessAsAlive(t *testing.T) {
	cmd := exec.Command(os.Args[0], "-test.run=^TestHelperLivenessChild$")
	cmd.Env = append(os.Environ(), "AIOS_LIVENESS_HELPER=1")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	})

	deadline := time.Now().Add(5 * time.Second)
	for !processAlive(t, cmd.Process.Pid) && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if !processAlive(t, cmd.Process.Pid) {
		t.Error("a live process was reported as dead")
	}
}

// TestProcessExited_TerminatedHandleIsExited covers the helper
// directly.
//
// The distinction it pins is the whole point of the fix: an
// OpenProcess that succeeds says nothing, and only the wait tells
// you whether the process has exited.
func TestProcessExited_TerminatedHandleIsExited(t *testing.T) {
	cmd := exec.Command(os.Args[0], "-test.run=^TestHelperLivenessChild$")
	cmd.Env = append(os.Environ(), "AIOS_LIVENESS_HELPER=1")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	pid := uint32(cmd.Process.Pid)

	handle, err := windows.OpenProcess(windows.SYNCHRONIZE, false, pid)
	if err != nil {
		t.Fatal(err)
	}
	defer windows.CloseHandle(handle)

	// A live process has not exited.
	if processExited(handle) {
		t.Error("a live process was reported as exited")
	}

	if err := cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for !processExited(handle) && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if !processExited(handle) {
		t.Error("a terminated process was not reported as exited")
	}

	_ = cmd.Wait()
}

// TestHelperLivenessChild is the child process the liveness tests
// use. It does nothing but exist until it is killed.
func TestHelperLivenessChild(t *testing.T) {
	if os.Getenv("AIOS_LIVENESS_HELPER") == "" {
		return
	}
	time.Sleep(120 * time.Second)
}
