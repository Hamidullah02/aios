package sandbox

import (
	"os"
	"path/filepath"
	"runtime"
)

// The sandbox clears the child's environment (REQ-SND-ISO-002), and
// the point of that is to keep the operator's credentials out of a
// gated subprocess: a child that inherited the parent's environment
// saw OPENROUTER_API_KEY and could exfiltrate it.
//
// "Clear the environment" is not the same as "no environment", and
// the difference is a bug an operator hits on the first real task.
//
// A child with no PATH cannot start any program it did not name by
// absolute path. On Windows that failure is quiet and confusing:
// cmd.exe and PowerShell are built into their own shells, so
// `cmd /c echo hello` works, while `cmd /c git --version` fails with
// "'git' is not recognized as an internal or external command". The
// plan the model writes is correct and the product cannot run it.
//
// The same cliff exists on POSIX: `sh -c "ls"` fails for the same
// reason, even though `ls` is on every machine.
//
// The fix is to give the child the PATH, and nothing else that
// matters.
//
// Why PATH and not the whole environment? PATH is a list of
// directories. It carries no credential, and a program that is
// installed on the machine is there to be run. Every other variable
// stays out: the API keys, the tokens, the proxy settings, and
// everything else in the parent's environment are exactly what the
// isolation requirement is about.
//
// The caller's own RunOptions.Env still wins over these values, so
// a caller that needs a different PATH can set one.
const (
	// EnvPath is the search path. It is the one variable a
	// sandboxed child cannot do without.
	EnvPath = "PATH"

	// EnvSystemRoot is the parent of the Windows system
	// directory. The DLL loader needs it to find system DLLs.
	EnvSystemRoot = "SystemRoot"

	// EnvComSpec is the path to cmd.exe. A tool that shells out
	// to the command processor reads it.
	EnvComSpec = "ComSpec"

	// EnvPathext lists the extensions Windows treats as
	// executable. Without it `tool` does not resolve to
	// `tool.exe`.
	EnvPathext = "PATHEXT"

	// EnvTemp and EnvTmp are the temp directory, under the
	// Windows and POSIX spellings. A tool that writes a
	// temporary file needs one of them.
	EnvTemp = "TEMP"
	EnvTmp  = "TMP"

	// EnvTmpDir is the POSIX spelling of the temp directory.
	EnvTmpDir = "TMPDIR"

	// EnvHome and EnvUserProfile are the user's home
	// directory. They are locations, not secrets, and a tool
	// that reads a config file from there needs them.
	EnvHome        = "HOME"
	EnvUserProfile = "USERPROFILE"
)

// childBaseEnv returns the environment variables a sandboxed child
// needs beyond what the caller passed.
//
// The returned map is a starting point, not the whole environment.
// The caller's RunOptions.Env overrides any of these values, and the
// network policy is applied after this function so the proxy
// variables always win over everything.
//
// The set is deliberately small. It is the set of variables that
// decide whether a program can start at all, and every one of them
// names a location rather than a secret.
func childBaseEnv() map[string]string {
	out := make(map[string]string, 8)

	// The search path. This is the one that decides whether
	// `git`, `python`, or `ls` can be found by name.
	if p := os.Getenv(EnvPath); p != "" {
		out[EnvPath] = p
	}

	if runtime.GOOS == "windows" {
		// The system root and the command processor. Both are
		// needed for the DLL loader and for anything that shells
		// out to cmd.
		if root := os.Getenv(EnvSystemRoot); root != "" {
			out[EnvSystemRoot] = root
		}
		if cs := os.Getenv(EnvComSpec); cs != "" {
			out[EnvComSpec] = cs
		} else if root := os.Getenv(EnvSystemRoot); root != "" {
			out[EnvComSpec] = filepath.Join(root, "System32", "cmd.exe")
		}
		// The executable extensions. Without PATHEXT, `tool`
		// does not resolve to `tool.exe`.
		if ext := os.Getenv(EnvPathext); ext != "" {
			out[EnvPathext] = ext
		} else {
			out[EnvPathext] = ".COM;.EXE;.BAT;.CMD"
		}
		// The temp directory. The Windows spellings come first
		// because os.TempDir on Windows reads TEMP before TMP.
		if t := os.Getenv(EnvTemp); t != "" {
			out[EnvTemp] = t
		}
		if t := os.Getenv(EnvTmp); t != "" {
			out[EnvTmp] = t
		}
		if p := os.Getenv(EnvUserProfile); p != "" {
			out[EnvUserProfile] = p
		}
		return out
	}

	// POSIX: the home directory and the temp directory. Both are
	// locations, and a tool that reads a config file or writes a
	// scratch file needs them.
	if h := os.Getenv(EnvHome); h != "" {
		out[EnvHome] = h
	}
	if t := os.Getenv(EnvTmpDir); t != "" {
		out[EnvTmpDir] = t
	}
	return out
}
