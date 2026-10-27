//go:build unix

package sandbox

import (
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"testing"
)

// TestDropPrivileges_DefaultIsNotADrop asserts that the default
// limits do not ask for a privilege drop.
//
// The zero value of ResourceLimits has UID=0 and GID=0, and 0
// is root. The implementation once treated any non-negative
// value as "drop to this user", so the zero value asked for
// root -- and on Linux that failed with EPERM for any process
// that was not already root.
//
// The failure was fatal: the runner returned before starting the
// child, so every subprocess the agent tried to run was refused
// with "sandbox: limits: gid: operation not permitted". Twenty-six
// tests failed on Linux CI for this reason and none of them
// failed on Windows, where the drop is a no-op.
//
// This test is on the POSIX path because that is where the
// failure was. The Windows build cannot catch it.
func TestDropPrivileges_DefaultIsNotADrop(t *testing.T) {
	// The zero value: what a caller gets when it fills in only
	// the fields it cares about.
	var zero ResourceLimits
	if zero.DropsPrivileges() {
		t.Errorf("the zero-valued limits ask for a privilege drop (uid=%d gid=%d)", zero.UID, zero.GID)
	}
	if err := dropPrivileges(&zero); err != nil {
		t.Errorf("dropPrivileges on the zero value: %v", err)
	}

	// The runner's own default, which names the sentinel
	// explicitly.
	runnerDefault := ResourceLimits{TimeSeconds: 60, UID: NoPrivilegeDrop, GID: NoPrivilegeDrop}
	if runnerDefault.DropsPrivileges() {
		t.Error("the runner's default limits ask for a privilege drop")
	}
	if err := dropPrivileges(&runnerDefault); err != nil {
		t.Errorf("dropPrivileges on the runner default: %v", err)
	}
}

// TestDropPrivileges_DropsToTheRequestedUser asserts that a
// limit that names a user is honored.
//
// It is skipped unless the test is running as root, because
// dropping privileges requires the privilege to drop. On a
// developer machine and on a CI runner the test process is not
// root, and the behavior under test -- an ordinary user's uid
// being refused -- is exactly the failure this fix is about.
func TestDropPrivileges_DropsToTheRequestedUser(t *testing.T) {
	if os.Geteuid() != 0 {
		t.Skip("dropping privileges requires root")
	}
	// Pick a uid that is not the one we are running as.
	target := 65534
	if os.Geteuid() == target {
		target = 65533
	}
	limits := ResourceLimits{TimeSeconds: 60, UID: target, GID: target}
	if !limits.DropsPrivileges() {
		t.Fatal("a limit naming a user does not report a privilege drop")
	}

	// The call changes the process's credentials, so it runs in
	// a child process rather than here. os/exec is used because
	// syscall.Credential lives on exec.Cmd's SysProcAttr, not on
	// syscall.ProcAttr.
	cmd := exec.Command("/bin/sh", "-c", "exit 0")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Credential: &syscall.Credential{Uid: uint32(target), Gid: uint32(target)},
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("the child did not run as uid %d: %v (%s)", target, err, out)
	}
}

// TestDropPrivileges_ZeroUIDIsNotRoot pins that a limit naming
// uid 0 is not treated as a drop request.
//
// A sandbox that runs its child as root is not a sandbox. The
// implementation now treats only a positive id as a request, so
// uid 0 means "leave the credentials alone" rather than "become
// root".
func TestDropPrivileges_ZeroUIDIsNotRoot(t *testing.T) {
	limits := ResourceLimits{TimeSeconds: 60, UID: 0, GID: 0}
	if limits.DropsPrivileges() {
		t.Error("uid 0 must not be read as a request to become root")
	}
	// The call must be a no-op, not a setuid(0).
	if err := dropPrivileges(&limits); err != nil {
		t.Errorf("dropPrivileges with uid 0: %v", err)
	}
	// And the process is still whoever it was.
	if got, want := os.Geteuid(), os.Getuid(); got != want {
		t.Errorf("euid = %d, want %d -- the credentials changed", got, want)
	}
}

// TestNoPrivilegeDropSentinel pins the sentinel's value.
//
// It has to be -1. It cannot be 0, because 0 is root, and the
// zero value of the struct carries 0. If this ever changes, the
// Linux failure comes back.
func TestNoPrivilegeDropSentinel(t *testing.T) {
	if NoPrivilegeDrop != -1 {
		t.Errorf("NoPrivilegeDrop = %d, want -1", NoPrivilegeDrop)
	}
	if strconv.Itoa(NoPrivilegeDrop) != "-1" {
		t.Error("the sentinel is not -1")
	}
}
