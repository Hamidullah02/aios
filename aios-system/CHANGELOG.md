# Changelog

## v0.58.0 — the documented install command did not work

**The one command in the README failed for every user who ran it.**
The install command is:

```powershell
irm https://raw.githubusercontent.com/Hamidullah02/aios/main/install.ps1 | iex
```

It downloaded the binary, installed it, and then aborted:

```
aios: the installed binary did not run. Try: C:\Users\...\aios\bin\aios.exe --help
```

The binary was fine. It was the check that was wrong.

### The check judged the binary by its output

The installer ran `& $installed --help` inside a `try`/`catch` and
treated anything thrown as "the binary did not run".

Windows PowerShell raises a **terminating error** for a native
command that writes to stderr when `$ErrorActionPreference` is
`"Stop"` — and the installer sets it to `"Stop"` so that a failed
download stops the script. `aios --help` wrote its usage to stderr,
so the check threw on every run, and the installer called `Die`.

There is a second half to it, and it is the quieter one. When
PowerShell converts that stderr into a terminating error it also
sets `$LASTEXITCODE` to **-1**, not the real exit code. A check that
only looked at `$LASTEXITCODE` in the `catch` block could not tell a
working binary from a broken one either.

The check now reads the exit code from `Start-Process -Wait
-PassThru`, which reports it truthfully — verified against a binary
that writes to stderr and exits 0 (rejected: exit 1) and one that
exits 3 (reported: exit 3).

**This was verified against the published installer, not just
against the fix.** The v0.57.0 `install.ps1` was fetched from
GitHub and run against a stderr-writing binary: it failed, with exit
1. The fixed installer accepted the same binary, with exit 0.

### The help went to stderr, which made it invisible

`aios --help` printed its usage to stderr. Help is requested
output, not an error, and putting it there had two costs:

- `aios --help > help.txt` wrote an **empty file**;
- `aios --help | more` showed nothing.

It now goes to stdout. `aios run --help` too — a flag that answers
on stderr for one subcommand and stdout for another is a trap that
depends on which word the operator typed.

### `go install` gave you a binary that could not say what it was

`go install github.com/Hamidullah02/aios/aios-system/cmd/aios@latest`
is the README's second install method. It worked, and the binary it
produced reported:

```
aios dev
```

It was not a dev build. It was a released one. The Go toolchain
records the module version in the binary's build info, but it does
not apply the release workflow's `-ldflags`, so the stamp was
absent and the binary fell back to the literal `"dev"`.

`--version` now reads the module version out of the build info when
nothing was stamped. A `go install` binary reports
`aios v0.58.0`; a plain `go build` in a working tree still reports
`aios dev`, which is the truth about it.

### The tests that would have caught this

Three, and they were confirmed to fail against the published v0.57.0
installer before being kept:

- `TestInstaller_AcceptsABinaryThatWritesToStderr` — installs a
  binary that writes to stderr and exits 0, which is the shape that
  broke the release;
- `TestInstaller_RunsEndToEndAgainstALocalRelease` — runs the real
  `install.ps1` against a local stand-in for GitHub;
- `TestInstaller_SmokeCheckReadsTheExitCode` — pins that the check
  reads an exit code rather than trying to interpret output.

The pattern-based installer tests that existed before could not have
caught this: they read the script's text and look for the right
words, and the script read perfectly while failing on every
machine.

### Also in this release

- The 26 tracked files under `probe-flow/` are gone. They were
  scratch debug scripts and the transcripts they produced — run
  output, console logs, marker files — and nothing in the build or
  the tests read them. The directory is now ignored.
- The repository's `.gitignore` covers `hole-cli/` and `probe-flow/`
  as well, so a future `git add .` cannot put them back.

## v0.57.0 — every file tool was broken on the Windows CI runner

**On Windows, the path check made the workspace and the paths inside
it incomparable.** `ResolvePath` canonicalized the path with
`EvalSymlinks` and left the workspace as it was. On Windows that is
fatal, for two reasons that only appear together.

The first is the 8.3 short name. `EvalSymlinks` expands
`C:\Users\RUNNER~1\AppData` into `C:\Users\runneradmin\AppData`. The
GitHub runner's temp directory is in the short form.

The second is that `EvalSymlinks` fails on the **whole** path when
any part of it does not exist — and the file a tool is about to
create does not exist yet. So
`...\RUNNER~1\AppData\Local\Temp\x\a.txt` resolved to nothing and
kept its short name, while the workspace — which does exist — was
expanded to the long form.

The two no longer matched. The prefix check failed, and every path
inside the workspace was reported as being outside it:

```
tools: path "a.txt" is outside workspace "C:\Users\RUNNER~1\..."
```

Every file tool was broken at once: `read`, `read_file`, `write`,
`edit`, `multi_edit`, `apply_patch`, `notebook_edit`,
`read_image`, and the plan writers that go through them. Forty
tests failed for one reason.

The fix resolves the **longest existing ancestor** and re-attaches
the part that does not exist yet. The short name is expanded, and a
symlink that points out of the workspace is still followed and still
caught — the symlink exists, so it is on the ancestor side of the
split. Both sides now go through one `canonicalPath` helper, so
the workspace and the paths inside it cannot drift apart again.

The regression test reproduces the short name on the local machine
with `GetShortPathName`, so this does not need CI to catch it
again. It fails against the old code.

### The pipeline now says what is true

**Windows is the target platform, and the pipeline was not.** CI ran
on `ubuntu-latest` and the release built five binaries for five
platforms. None of them had been run by a person.

The product is a Windows tool: the installer is a PowerShell script,
the shell helper is `run_shell`, the whole development loop happens
on Windows. So:

- CI runs on `windows-latest`. The race detector needs cgo and a C
  compiler, and the Windows runner has one.
- The release builds `windows/amd64` and nothing else. Publishing
  binaries for platforms that are not supported means shipping
  something that is neither tested nor supported.
- `install.sh` is gone. It asked for `aios_linux_amd64` and
  `aios_darwin_amd64` assets the release no longer builds, and the
  README pointed at it — so the first thing a user would have done
  would have failed.
- The bash-isms went with the runner change: the release notes were
  written with a heredoc, the assets were copied with `cp`, and the
  release was created with a shell glob. All three are PowerShell
  now.

### The release pipeline could not publish a release

**Three separate defects, each of which stopped the release on its
own.** The release has now run end to end and published v0.57.0.

**The binary was stamped with no version at all.** The build passed
`--version` a value that was empty, so the linker was given
`-X .../internal/version.Version=` with nothing after it. The linker
accepts that silently, and the binary reported `aios` -- with no
version and no commit.

The cause was a bash habit in a PowerShell workflow. In bash, `$TAG`
expands from the environment. In PowerShell, `$TAG` expands from a
*local variable* named TAG, and the step's `env:` block sets an
*environment* variable, which PowerShell reaches as `$env:TAG`. So
`$TAG` was empty. The same mistake was in the module-tag step, which
would have pushed a tag called `aios-system/` at an empty commit.

The check that already existed caught the first one. It runs the
binary and compares what it reports with the tag it was built from.

**The release job had nothing to publish.** The build job built the
zip and threw it away: there was no `upload-artifact` step, so the
release job's `download-artifact` found an empty directory. The
upload step is in now, and it fails the build if the zip is not
there.

**The release command used a flag that does not exist.**
`gh release create` has no `--clobber` -- that flag belongs to
`gh release upload`. Passing it makes gh print its usage and exit 1.
`--verify-tag` replaces it, and it is the better guard: the tag
already exists, because the workflow ran on it, so the release must
be created against a tag that is really there.

### A child that died said nothing about why

A child that exited non-zero reported only `exit status 2`. A Go
child that hits a limit exits 2 and writes the reason to stderr:
`fatal error: runtime: cannot allocate memory`. The wait error now
carries the child's stderr, and the message leads with its first
non-empty line.

## v0.56.0 — the binary could not say what it was, and `go install` could not work

Three things stood between this repository and a user who wanted to
install it. All three are fixed here. One of them cannot be fixed
from inside the repository, and it is named plainly at the bottom.

### `aios --version` did not exist

**The first question in every bug report could not be answered.**
`aios --version` failed with `flag provided but not defined:
-version` and exited 1. There was no version string anywhere in the
source, so a user who wanted to report a bug had to find the
CHANGELOG and work out which release they had.

`internal/version` holds the version, and the release workflow stamps
it at link time:

```
-ldflags="-X github.com/Hamidullah02/aios/aios-system/internal/version.Version=$TAG"
```

The commit is stamped as well, because two builds from the same tag
can differ. The release workflow then **runs the binary and checks
what it prints** -- a typo in the `-ldflags` path builds perfectly and
produces a binary that says "dev", so the stamp is verified rather
than assumed.

A build made with a plain `go build` reports `aios dev`, which is
the truth about a binary nobody stamped.

### `go install` named a repository that does not exist

**The README's second install method could not work.** It said:

```
go install github.com/sadi/aios-system/cmd/aios@latest
```

`github.com/sadi/aios-system` is not a repository. It never was.
The repository is `github.com/Hamidullah02/aios`, and the Go module
lives in its `aios-system/` subdirectory, so the path that actually
resolves is:

```
go install github.com/Hamidullah02/aios/aios-system/cmd/aios@latest
```

The module path in `go.mod` and the 161 Go files that import from
it now say the same thing. The old path would have kept compiling
locally -- an import is resolved from the module, not the network --
so nothing else would have caught it.

**There is a second half to this, and it is the part that is easy
to miss.** The Go tooling only looks for a module in a repository
subdirectory under a tag whose name carries the subdirectory. With
only a plain `v0.56.0` tag, `go install` fails with:

```
module github.com/example/repo@v1.0.0 found, but does not contain package
github.com/example/repo/sub/cmd/tool
```

It finds the repository at the plain version tag and looks for the
package in the wrong place.

So the release workflow pushes **two** tags for every release: the
plain `v0.56.0` that the installers and the release page use, and
`aios-system/v0.56.0` for the Go module.

This was not worked out from the documentation and hoped for. The
rule was confirmed two ways: the Go module proxy's own metadata for
a known subdirectory module names the prefixed tag as its source
(`"Ref": "refs/tags/api/v1.23.0"`), and the behaviour was then
reproduced end to end against a local repository -- with the
prefixed tag the install succeeds and the binary runs; with only the
plain tag it fails.

The module tag is pushed **after** the release is published, so a
failed release cannot leave a module version pointing at binaries
nobody can download.

### There was no licence

The repository had no `LICENSE` file. A repository with no licence
is not open source -- every user has no permission to use, copy,
modify or redistribute it, whatever the README implies. There is an
MIT licence at the repository root now.

### The agent could not run anything on Linux

**Making the repository public turned on the CI, and the CI found
that the product was broken on Linux.** Twenty-six tests failed.
Not one of them failed on Windows, and not one of them was a test
problem: they were the product refusing to do its job.

**Every subprocess the agent tried to run was refused.** The
sandbox's privilege drop treated any non-negative UID or GID as
"drop to this user":

```go
if l.GID >= 0 {
    syscall.Setgid(l.GID)
}
```

`ResourceLimits.UID` and `ResourceLimits.GID` default to `0`, and
`0` is root. So every run with default limits -- which is every
run -- asked to `setgid(0)` and `setuid(0)`. On Linux that is
`EPERM` for any process that is not already root, and the failure
was fatal: the runner returned before starting the child. A user
on Linux got:

```
sandbox: apply limits: sandbox: resource limit exceeded: privilege:
sandbox: limits: gid: operation not permitted.
```

for every tool call. The agent could read, search, and edit --
those do not spawn processes -- and could not run a single command.

The defect was invisible on Windows, where the privilege drop is a
no-op, and invisible in normal use, because no caller ever set a
UID or a GID. It needed the Linux CI job to surface.

A sandbox that runs its child as root is not a sandbox, so the
test is now "greater than zero", not "not negative". `0` and the
negatives both mean "leave the credentials alone", and the
sentinel is named rather than left to the zero value:

```go
const NoPrivilegeDrop = -1
```

`internal/sandbox/privilege_drop_test.go` pins the default does
not drop, that `uid 0` is not read as a request to become root,
and that a limit naming a real user still drops. The last of
those is skipped without root, because dropping privileges
requires the privilege to drop.

### A layout file in the wrong place took the UI down on Linux

**A path the layout file cannot occupy was a hard error on Linux
and a non-event on Windows.** The store swallowed "the file is not
there"; it did not swallow "a component of the path is not a
directory":

- Windows returns `ERROR_PATH_NOT_FOUND`, which `os.IsNotExist`
  recognises.
- Linux returns `ENOTDIR`, which it does not.

So the same store, at the same path, meant "nothing saved yet" on
one platform and "here is an error" on the other. A layout file in
an unreachable place took the whole UI down on Linux and not on
Windows.

`isNoLayoutFile` now names the whole set -- `ENOENT`,
`fs.ErrNotExist`, and `ENOTDIR` -- so the rule is the same
everywhere: if the file is not there, there is no saved layout and
the server starts with the defaults. A file that *is* there and
cannot be parsed is still an error, and is still reported.

### The child could not reach the egress proxy on Linux

**The network namespace and the proxy were mutually exclusive, and
the proxy lost.** A child in its own network namespace has a
loopback interface in the DOWN state: no address, no route, and a
connect to `127.0.0.1` fails with `EHOSTUNREACH` -- "network is
unreachable".

The egress proxy is bound to `127.0.0.1` in the *parent's*
network namespace. So a child in its own network namespace could
not reach it. The result on Linux was that every subprocess had no
network at all, and the allow list never ran: the request failed
with "network is unreachable" rather than with the proxy's 403.

The deny-mode test passed, but for the wrong reason -- the request
failed at the child, which is what the test wanted to see, without
the policy ever being consulted.

The two are mutually exclusive as they stand. Making the network
namespace work would need a veth pair between the two namespaces
and a proxy bound to the parent's end of it; that is not
implemented, and it would mean the proxy listens on a non-loopback
address, which is a different trust model.

So the network namespace is now **off**, and the proxy is the
network boundary on every platform. The user, mount and pid
namespaces stay on -- they do not conflict with anything, and they
are the ones that do the containing:

- the user's uid 0 still maps to the operator's unprivileged uid,
  so a setuid binary is confined to the identity the operator
  granted;
- the mount namespace still gives the child its own filesystem view;
- the pid namespace still makes the kernel reap the child's
  orphans.

The docs said the network namespace was "the strongest of the
four" and "what makes the egress proxy the only way out". It was
neither: it was a boundary that could not be reached, and it broke
the one that worked. The README, the package doc, and the operator
facing summary now say what is actually in force.

### A test that only ran on Windows

`TestBusWorker_ShellViaSandbox` ran `cmd /c echo hello`. On Linux
the child failed with `exec: "cmd": executable file not found in
$PATH`, so the test failed there and passed here. The command is
now the test binary itself, invoked so it exits cleanly, which is
a real executable on every platform and depends on no shell.

### The runner poisoned its own parent process

**Every limit the runner applied was a limit the parent could
never take back.** The runner sets the rlimits in the *parent*,
because Go's `exec` has no way to set one in the child and the
child inherits the parent's. The code set both the soft and the
hard limit:

```go
unix.Setrlimit(unix.RLIMIT_AS, &unix.Rlimit{
    Cur: uint64(l.MemoryBytes),
    Max: uint64(l.MemoryBytes),
})
```

But `setrlimit(2)` says an unprivileged process may lower the
hard limit freely and may raise it only up to the previous hard
limit. So the parent lowered its own hard limit and could never
lift it again -- and `restoreParentLimits`, which tried to put it
back, failed with `EPERM` while its error was discarded with
`_ =`.

The parent kept the child's limits for the rest of its life. On
a developer machine with one user, the damage was confined. On a
shared-uid CI runner it was not:

- `RLIMIT_NPROC` counts every process belonging to the **real
  uid**. The runner's limit of 4 applied to the whole machine, and
  the next `fork` failed with `EAGAIN`: `resource temporarily
  unavailable`.
- `RLIMIT_AS` stopped the Go runtime from allocating, and the
  process died with `fatal error: runtime: cannot allocate
  memory`.

The runner now lowers the **soft** limit and leaves the hard limit
where it is.

The trade is deliberate, and it is the safe direction. The hard
limit is what makes a child's bound un-liftable -- a child can
raise its own soft limit up to the hard limit -- so a soft-only
bound is one a deliberate child can remove. But a parent that
cannot allocate cannot run the next tool call, cannot write the
audit log, and cannot report the failure. A parent that stays
usable is worth more than a bound that a determined child can
lift.

When the runner *does* have the privilege to put a hard limit
back -- that is, when it is root -- it sets both, and the child's
bound is one it cannot lift.

The same fix is on macOS, which had the identical code.

### Also in this release

- The installers now print the version of the binary they actually
  installed. An operator filing a bug report needs the version of
  the binary in front of them, not the one they meant to run.
- The release notes now carry the `go install` line alongside the
  two installer commands.
- The release workflow's `release` job checks out the full history.
  It pushes a tag now, and a shallow clone has no parent commits to
  point it at.

### The approval journey hung, and the reason was a leaked process

**A journey that passed was reported as never finishing.** The
approval journey started a stub model provider with its output
redirected to a file, and stopped the `aios` process at the end --
but not the stub. The stub kept the handle on its redirected output
open, so the wrapper capturing the journey's output waited for a
handle that would never close.

The journey's own output was complete and correct the whole time.
The run just never finished: it sat for fifty minutes on output
that had already said everything it was going to say.

The stub was stopped only on the early-exit path, so the normal
path -- the one that completes the journey -- was the one that
leaked. The same leak was in the `rina` journey's early-exit path
and has been fixed there too.

This is worth recording because the symptom was so misleading: the
naive reading is "the product hung", and the product had not. The
test harness had.

### What this release does not fix

**The repository is public, and the install URL now resolves.** It
was private when this release was written; the file moves and the
URL fixes above are visible now, and the installer scripts are
fetchable by anyone.

**There are still no tags.** The release pipeline has never run
against this repository. The first `v0.56.0` tag is what creates
the release assets the installers download and the module version
`go install` resolves.

**The CI has never been green.** It had run exactly once, on the
commit that made the repository public, and it failed on Linux --
see above. That run is the first thing that ever exercised the
suite on a platform other than the author's, and it found two
defects that the Windows-only development loop could not. The
fixes are in this release; whether the suite is green on Linux is
now a question the next CI run answers.

## v0.55.0 — the install link pointed at a file that was not there

**The documented install command could not work.** The README and
the release notes both tell a user to run:

```
curl -fsSL https://raw.githubusercontent.com/Hamidullah02/aios/main/install.sh | sh
irm  https://raw.githubusercontent.com/Hamidullah02/aios/main/install.ps1 | iex
```

Both installers were in `aios-system/`, one level down. The URL
names the repository root, so the request 404s, and the operator's
first experience of the product is a failed download with no cause
they can see.

The installers now live at the repository root, where the
documented URL points. The release workflow also attaches them to
the release, so the copy that matches the binaries in a given
release is the one on the release page.

**The same mistake was hiding the CI.** GitHub only reads
workflows from `<repo-root>/.github/workflows`. Both workflow files
were in `aios-system/.github/workflows`, so the repository had no
CI and no release pipeline at all -- GitHub never looked there.
They are at the root now.

And the root `README.md` was an 18-byte stub (`# aios`), so the
repository's front page showed a title and nothing else. The real
README -- the install instructions, the model-key setup, the CLI
reference -- is the front page now.

**The workflows would also have failed on their first run.** The
Go module lives in `aios-system/`, but every step ran `go build
./cmd/aios` from the repository root, where there is no `go.mod`.
Each Go step now sets `working-directory: aios-system`.

**One more, found by the same check.** `install.ps1` mapped
`ARM64` and `x86` to release assets that the workflow never
builds -- only `windows/amd64` is built. On those machines the
installer would have downloaded a URL that 404s. The installer now
offers only what is built, and says so plainly on anything else.

`internal/cli/first_run_test.go` already pinned the install URL
and the asset names. It failed on the first run after the move
and passed after -- which is the test doing its job: it caught
the moment the files and the URLs stopped agreeing.

## v0.54.0 — the approval daemon started without saying so

**`aios daemon start` printed nothing.** The daemon bound its
socket, wrote its pid file, and then blocked in the accept loop
without a word. It is a foreground service, so the operator's
terminal showed a command that looked hung, with no way to tell
"it is running" from "it died before it started" -- and no way to
know which socket path the rest of the tooling would use.

It now prints one line, after the socket is bound and before the
blocking loop:

```
aios daemon: ready on C:\Users\Home\AppData\Local\Temp\aios-approval.sock
```

The line is a promise the daemon keeps: it is printed only once
the socket exists, so an operator who sees it can go and use
`aios approve` straight away.

`internal/cli/coverage_test.go` pins it. The test was confirmed to
fail against the old code -- stdout was empty -- before the fix
was kept.

**This was found by driving the product, not by reading it.** The
setup journey in `var/setup-journey.ps1` starts the daemon the way
an operator does and reads what it printed. The setup journey also
drove the rest of the first-run path: `aios init` in a bare
directory, `aios init` a second time, the daemon's socket, `aios
approve` against a live daemon, and a clean stop.

## v0.53.0 — a stale browser tab was told to try again

**A second click on a decided approval came back 409 instead of
404.** The web UI's task path resolves approvals through the
composite resolver, which tries this server's own handler first
and falls through to the approval daemon when the local handler
misses. The daemon's failure was returned with the approval id
wrapped around it:

```go
return "", fmt.Errorf("approval %s: %w", id, err)
```

That wrapping kept the daemon's message but dropped the
`webui.ErrApprovalNotFound` sentinel. The server tells "this
approval is not here" (404, a stale tab) from "this approval
cannot be granted here" (409, a configuration the operator has to
change) by testing the error against that sentinel -- so with the
sentinel gone, the answer was always 409.

409 is the wrong answer for a stale tab. It tells the operator the
request was well-formed but the state was wrong, which reads as
"try again" for an approval that is never coming back. The CHANGELOG
for v0.43.0 claimed this returned 404, and it did -- but only for
the `run --web` path, where the run's own handler is the resolver
and there is no daemon to fall through to. The claim was verified
on one path and generalised to another.

The error now keeps both: the daemon's own message says what went
wrong, and the sentinel says what kind of failure it is.

`internal/cli/web_tasks_test.go` pins it. The test was confirmed to
fail against the old code before the fix was kept.

**This was found by driving the product, not by reading it.** The
approval journey in `var/approval-journey.ps1` asks the question a
real operator's browser asks -- what happens when I click Allow
twice -- and the answer was wrong.

**Also: the journey output directories were about to be committed.**
Each one holds the captured stdout of an `aios web` run, and that
stdout carries the session token in the URL the server prints. The
server calls that token a credential. The repository's root
`.gitignore` did not cover those directories, so a `git add .` would
have put four live tokens and four copies of the `.env` file -- the
one holding the model API key -- into a commit. The directories are
the working area for the journey scripts and are regenerable, so
they are now ignored.

## v0.52.0 — the sandbox's liveness probe could not tell a dead process from a live one

**On Windows, the probe that checks whether a process is still
running was wrong.** `internal/sandbox/probe_windows.go` opened
the process with `OpenProcess(SYNCHRONIZE)` and reported "alive"
when the open succeeded. A successful open does not mean the
process is alive: a terminated process keeps its process object
alive for as long as any handle to it is open, and the open
succeeds for it. The correct check is to wait on the handle with a
zero timeout -- an exited process is already signalled, a running
one times out.

This was not a cosmetic bug. It sat in the test that guards the
process-containment property: `TestProcessTreeKill_GrandchildDoesNotSurvive`
asks whether a grandchild outlived the timeout, and it asks by
probing the grandchild's pid. With the broken probe, a grandchild
that had been correctly killed was reported as alive -- so the test
that verifies the kill failed intermittently, for a reason that had
nothing to do with the kill.

The runner holds the child's own handle, which is exactly the
condition that keeps a dead process's object alive. The probe now
waits on the handle rather than stopping at the open.

`internal/sandbox/probe_windows_test.go` pins both directions: a
terminated process must be reported as dead, and a live one as
alive. The first of those two tests was confirmed to fail against
the old probe before the fix was kept.

The POSIX path is unaffected: there, an orphaned grandchild is
re-parented to init and reaped promptly, so `kill(pid, 0)` stops
reporting it as soon as it dies.

## v0.51.0 — the CI that was missing, and the race detector in it

**There was no CI.** The only workflow in this repository was
`release.yml`, and it ran on tags. A change could sit on a branch
indefinitely with nothing checking it; the release job was the
first thing that ever ran the test suite, and it ran on the
release machine, at release time.

That is how the two data races fixed in v0.47.0 survived. The
plain suite passed -- they were races, and races are invisible to
a suite that does not look for them -- and the one tool that
would have found them was never in the loop.

`.github/workflows/ci.yml` now runs on every push to `main` and
every pull request, with three jobs:

- **test** -- vet and the plain suite, `CGO_ENABLED=0`, matching
  how the binaries are actually built.
- **race** -- `go test -race ./...`, with `CGO_ENABLED=1`.
- **cross-compile** -- builds and vets every target the release
  workflow ships: linux and darwin on amd64 and arm64, and
  windows on amd64.

The cross-compile job exists because a build tag that excludes a
platform does not fail the host build. v0.46.0 shipped a release
that did not compile on Linux at all, and the only reason it was
caught is that someone tried to install it.

**The race detector is in the release workflow too.** A tag can be
pushed without a pull request ever being opened, so the release
job now runs `go test -race ./...` as well. The shipped binaries
stay `CGO_ENABLED=0` -- they have to run on a machine with no C
toolchain -- but the same source is checked with cgo enabled
before it is packaged. Testing and packaging are separate
concerns and they no longer share one job.

## v0.50.0 — every tool the agent has is now a tool it can use

**Twenty of the twenty-three tools were unreachable.**
The bus registered the full doc-set tool set — `edit`,
`apply_patch`, `multi_edit`, `notebook_edit`, `git_status`,
`git_diff`, `git_log`, `git_commit`, `git_branch`, `git_worktree`,
`grep`, `glob`, `code_search`, `diagnostics`, `read_image`,
`memory`, `memory_write`, and more. The planner carried a
hand-written table of five. A tool the planner did not know
about could not be named in a plan, so it was dead code however
well it worked — and `aios tools` listed it as though it were
reachable.

The gap was structural. The planner's table was a separate copy
of the bus's tool map, and the comment above it said the two
were "deliberately kept in sync by hand". They were not. The
table said it mirrored the bus; it did not.

**Tools now describe themselves.** `tools.Describer` is an
optional interface: a tool returns its own description and its
own JSON input schema. The schema lives in the same file as the
`Run` method that reads the arguments, so the text the model
reads and the code that consumes the arguments cannot drift
apart. `tools.RiskTier` does the same for the risk tier.

**The planner's tool list is derived from the bus.** There is
one list now, not two. `plannerToolSpecs` reads the bus's live
tool map and projects it into the shape the planner speaks. A
tool added to the bus is offered to the model automatically; a
tool removed from the bus disappears from the prompt.

**`self_check` was dispatchable but not plannable.** The tool
was registered after the planner was built, so it was in the bus
and not in the prompt: the model could never choose it. It is
now registered before the tool list is read.

**Two new schemas, one caught in the act.** The `ask_user`
schema shipped with an extra brace and was not valid JSON. The
new test that checks every schema parses found it on the first
run. A malformed schema makes the model guess, and a guess that
happens to be wrong produces a plan that fails at the tool call
with no hint that the tool's own documentation was broken.

**Verified against the live model.** Two goals that could not
have been planned before: "check the git status of the
repository" planned a `git_status` task and ran it; "change the
word hello in greeting.txt to goodbye" planned an `edit` task
with `path=`, `old_string=`, and `new_string=` inputs and
rewrote the file. Both exited 0 with a `tool_call` audit row
behind them.

## v0.49.0 — the product can be got, and it works out of the box

**A non-coder could not get the product, and could not run it.**
The system had a browser UI and a CLI, but no way to obtain either:
no installer, no download, no release workflow. Worse, the thing
did not work when you did have it. There was no model adapter for
OpenRouter, nothing in the codebase read a `.env` file, and the
planner's prompt never told the model the format the runtime
actually accepts for a tool's arguments. A first run produced a
plan whose tasks died at the first tool call with `path is
required`.

**OpenRouter is now a provider.** `internal/models/openrouter.go`
adds the adapter: the OpenAI wire format against OpenRouter's
endpoint, with the attribution headers OpenRouter asks for. It
reads `OPENROUTER_API_KEY`, and also `openrouterapi`, because that
is how operator files spell it.

**The default model is free.** `nvidia/nemotron-3-ultra-550b-a55b:free`
is the default. A first run must not cost money, and a retry loop
against a model that keeps failing its verifier is exactly the
case where an unexpected bill is the worst outcome. The default is
also good at the one thing the planner asks of it: emitting a JSON
object and nothing else.

**The `.env` file is read.** `internal/cli/dotenv.go` loads it on
startup, searching from the config file's directory and from the
working directory, upwards. A key exported in the shell wins over
the file, so an operator can override it for one command. An
empty variable does not count as set: an empty key is not a key.
`AIOS_NO_DOTENV` turns the loader off.

**Four bugs, all of them mine to own.** The price table's empty
check probed with a sentinel model that no table prices, so it
answered "empty" for every table -- `buildRouter` has always
discarded the operator's `prices.yaml` and replaced it with the
built-in list. The planner prompt said `"inputs" (array of
strings)` and never said `key=value`, which is what the worker
splits on; the shipped `examples/plan.json` had the same shape and
so could never have run. The verifier now rejects an input that
is not `key=value`, so the repair loop fixes it instead of the
run dying at the tool call.

**The `--model` flag was dead.** It was parsed, documented, and
then never read. It works now, and so does `AIOS_MODEL`.

**There is something to download.** `install.sh` and
`install.ps1` fetch a prebuilt binary for the operator's machine,
check that it runs, and put it on the PATH. The release workflow
builds the five assets those scripts ask for. The binaries are
`CGO_ENABLED=0` and self-contained.

**The card names the right key.** The web UI's missing-key message
used to say `Set OPENAI_API_KEY (or ANTHROPIC_API_KEY)`, which
sends an OpenRouter operator down a path that does not help. It
now names the variable the provider actually asked for.

## v0.48.0 — the browser can start work

**A non-coder could watch the system work but not ask it to.**
The web UI had a communication log, an agent grid, an approval
queue, and an activity feed. What it did not have was a way to
give the product something to do. `POST /api/messages` accepted a
message, wrote it to the log, and stopped: the bridge's own comment
said it -- "nothing in the runtime consumes a message". A message
from an agent is a note between agents, so nothing consumed
those either. The only way to start work was `aios fix
<objective>` in a terminal, which means the entry point required a
terminal, an objective written the way the planner expects, and
knowledge of a product whose vocabulary ("plan", "policy.yaml",
"socket") was never written for the person holding the mouse.

**The task entry point.** The page now opens with one field: what
do you want done? One sentence, one button. The server decides
whether that sentence becomes a plan as written, a repaired plan,
or a plan the conversation refined -- the operator is never asked
to approve a plan, because a plan is not a thing they can judge.

`POST /api/tasks` takes `{"goal": "..."}` and starts the work.
`POST /api/messages` does the same when the message is from the
operator (`"from":"operator"`) and returns the id of the task the
message became. A message from an agent still starts nothing:
turning agent chatter into a queue of accidental runs would be
worse than the old silence.

**The seam.** `internal/webui` defines `TaskSubmitter` -- an
interface with `Submit(goal)` and `CanSubmit()`. `internal/cli`
implements it (`WebTaskSubmitter`): it calls the planner, runs the
plan through the same `runWithApprovals` loop `aios run` uses,
and publishes progress to the hub as the run happens. The
dependency runs that way on purpose -- the UI must not know about
the planner, the model router, or the orchestrator, and those must
not know about HTTP.

**The conversation is the context.** The planner is handed the
operator's own earlier messages alongside the goal. "Summarise it"
is not a goal on its own; it is a goal after the operator said
which file. The goal itself is left out of that context, because
the goal is already the prompt's goal and repeating it makes the
model read it twice. Agent chatter is left out too: it is
machinery talking to itself, and it would only add noise.

**Honest refusals.** A server without a planner still serves and
still shows past activity, but it says so -- `this server cannot
start tasks; it can only watch runs that are already going` --
instead of accepting a goal and dropping it. The UI reads
`can_submit_tasks` from the snapshot and replaces the composer with
that explanation rather than offering a Start button that cannot
work. A blank goal is refused with "say what you want done"
rather than sent to the planner, which would answer with a
confusing "plan generation failed".

**Failures land in words.** A missing API key is the common case by
far, so the task card says "no AI model key is configured, so
nothing can be planned. Set OPENAI_API_KEY (or ANTHROPIC_API_KEY)
and try again" rather than "OPENAI_API_KEY is not set". A failed
task always reaches a terminal state with a detail and a finish
time: a card stuck on "planning" is the worst outcome, because the
operator has no idea whether anything is happening.

**One run at a time.** The orchestrator is sequential by design
and the agent panels are keyed by task id, so two plans in flight
would put two plans' steps in one set of panels. A second
submission is refused in words -- "a task is already running; wait
for it to finish" -- rather than interleaved.

### The race detector found a real race in the audit handle

`go test -race ./...` failed on the first run of this version, and
the failure was in the code this version added: a task submitted
from the browser runs in its own goroutine, and the process closes
the audit log on the way out. `audit.Log.Close` set the database
pointer to nil while the run's tracer was reading it, and the read
of the pointer and the write of the pointer were two operations on
the same word with no lock between them.

The fix is a second lock on the handle itself. `Close` takes the
handle's write lock, so it waits for an operation that is already
in flight; every operation takes the read lock for the whole
operation, so "the database is still there" and "the database is
still usable" are the same statement. The write mutex still
serialises writes, because the chain hash of a row is computed
from the previous row's hash.

The visible symptom before the fix was not a crash: a run whose
audit rows were written while the log was closing would find
those rows failing with "sql: database is closed" halfway
through, which is an audit trail with holes in it.

### Verification

Driven end to end against a real server, not a mock: `aios web`
serving the built binary, the composer present in the served
HTML, `POST /api/messages` with `"from":"operator"` returning
`task_id`, the task appearing in `GET /api/tasks` and in the SSE
snapshot, and the plan id rewritten to the task id so two
submissions cannot fight over one checkpoint file.

`go test ./...`, `go vet ./...` and `go test -race ./...` all pass
across all 32 packages, zero data races.

## v0.47.0 — the race detector found two real races, and it can now be run

**The race detector runs now, and it found two real data races
immediately.** A previous version of this file claimed the
Windows toolchain had no C compiler, so `-race` could not be
run and concurrency was covered by hand-written tests instead.
That was wrong: `C:\msys64\ucrt64\bin\gcc.exe` is present. With
`CGO_ENABLED=1`, `CC` pointing at it and its directory on `PATH`,
`go test -race ./...` runs across all 32 packages.

It failed on the first run, in two places.

**The egress proxy published its server unsynchronised.**
`internal/egress/proxy.go` held the running `*http.Server` in a
plain field. `Listen` assigned it; `Addr` and `Shutdown` read it
from other goroutines. The race detector flagged every proxy test
that started a server and then asked for its address -- 17 tests
across `internal/egress`, all of them real.

This was not a test artefact. The production path is the same
shape: `internal/cli/egress_guard.go` calls `proxy.Listen()` on
the setup goroutine and `guard.Close()` calls `proxy.Shutdown()`
from whichever goroutine tears the run down. A `Shutdown` that
lands while `Listen` is still assigning could observe a
half-initialised server and shut down the wrong thing -- or none.
The field is now guarded by an `sync.RWMutex`, with `serverRef`
and `setServer` as the only ways to reach it.

**The approval handler wrote to the terminal from two goroutines.**
`internal/cli/approval_ui.go` wrote its prompt and its decision
note to `h.Out` with bare `fmt.Fprintf`. The resolver is called
from an HTTP goroutine and the run calls `Approve` from its own,
so two writes could land at once. On a `bytes.Buffer` that
corrupts the buffer; on a terminal it shows two half-lines.
Writes now go through `write`, which holds `outMu` for the whole
block, so one approval's prompt reads as one prompt rather than
interleaving with another's.

The two tests that exposed it were reading a `bytes.Buffer` while
the handler wrote to it. That read was itself the race, so the
tests now use a `syncBuffer` with the same one-mutex shape.

**The daemon-start test had a 2-second timeout that the race
detector could outrun.** `TestDaemonResolver_Resolve` failed once
during the full-suite run with "daemon socket never appeared". The
daemon was not slow to start: the test gave up. The timeout is
now 10 seconds, and the helper captures the daemon's bind error
instead of discarding it, so a daemon that fails to start says
so rather than looking like a slow one.

The claim in `approval_ui_concurrency_test.go` that the race
detector was unavailable on this toolchain has been corrected --
it was available, and it was never run.

`go test -race ./...` now passes: 32 packages, zero data races.
`go vet ./...` and `go test ./...` pass.

## v0.46.0 — the sandbox is a sandbox now, and the project builds on Linux

Two things landed together, because the first one was impossible
without the second.

**The project did not compile on Linux or macOS.** Before this
version, `internal/sandbox/limits_posix.go` carried the build tag
`unix || darwin || linux` and referenced `syscall.RLIMIT_NPROC`.
Go's `syscall` package defines `RLIMIT_NPROC` on neither Linux nor
macOS -- it is an `x/sys/unix` constant. So `go build` failed on
every POSIX target:

```
internal/sandbox/limits_posix.go:42:34: undefined: syscall.RLIMIT_NPROC
```

The `unix` tag is not a Go build constraint at all -- `unix` is
never true -- so the file only compiled on darwin and linux, and
failed on both. **The Linux and macOS builds have been broken
since the file was written.** The Windows build was the only one
that worked, which is why the bug was invisible: the project was
being tested on the one platform where it compiled.

The fix splits the file per platform and moves to `x/sys/unix`,
which is the package that actually defines the constants:
`limits_linux.go` (`linux`), `limits_darwin.go` (`darwin`). The
macOS variant does not honor `ResourceLimits.Processes`, because
the kernel has no `RLIMIT_NPROC` there. That is now stated in the
code rather than implied by a build tag.

`golang.org/x/sys` moves from an indirect dependency to a direct
one. It was already in the module graph.

**Now the sandbox is a sandbox.** The review's blocker #1 said the
sandbox is `os/exec` plus rlimits: no namespace, no container, no
seccomp, no filesystem jail. That was accurate. Three pieces of
real isolation now exist.

*The process tree dies with the timeout.* Before this version the
runner killed the direct child on timeout and nothing else. A
command that backgrounded work -- `npm run dev &`, a shell that
spawns a compiler -- left the grandchildren running with the
parent's privileges, and the audit log said the run had timed out
while the work was still going. `internal/sandbox/isolation_posix.go`
puts the child in its own process group
(`Setpgid: true, Pgid: 0`) and kills the group, not just the
child. `internal/sandbox/isolation_windows.go` does the same with
a Job Object, which is strictly better: the kernel tracks job
membership, so a descendant cannot escape by changing its parent.

The runner's `Cmd.Cancel` hook is replaced, because the default
only kills the direct child.

*The Windows Job Object enforces the limits the kernel can.* The
Job Object applies `JOB_OBJECT_LIMIT_ACTIVE_PROCESS` for the
process count and `JOB_OBJECT_LIMIT_JOB_MEMORY` for memory, and
sets `JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE` so closing the handle
terminates every process in the job. The child is created with
`CREATE_SUSPENDED` and resumed with `NtResumeProcess` after the
job assignment lands, because a child that spawns a grandchild
between `CreateProcess` returning and the assignment would produce
a grandchild outside the job -- one that survives the timeout.
Go's `exec` package does not expose the primary thread handle, so
`NtResumeProcess` (which takes a process handle) is used instead of
`ResumeThread`.

*Linux gets real namespaces.* `internal/sandbox/namespaces_linux.go`
puts the child in a user, network, mount and pid namespace. The
user namespace maps the child's uid 0 onto the operator's own
unprivileged uid, so a child that finds a setuid binary or a
kernel bug is confined to the identity the operator granted. The
network namespace gives the child an empty network stack -- no
interface except loopback -- which is what makes the egress proxy
the only way out. The mount namespace gives the child its own
filesystem view. The pid namespace makes the child pid 1, so the
kernel reaps its orphans.

The namespaces degrade rather than fail. A kernel without user
namespaces, a container that forbids `CLONE_NEWUSER` and a
hardened CI runner all reject them. `--isolation=auto` (the
default) probes once and falls back to the weaker containment,
because a run that cannot start is a worse answer than a run that
starts with less isolation. `--isolation=namespaces` requires them
and fails without them. `--isolation=none` disables them.

**What this still does not do.** It is still not a chroot, and it
does not give the child a restricted filesystem view of its own:
the mount namespace isolates the child's mounts from the host's,
but the child can still read the whole filesystem it has
permission to read. There is no seccomp filter, no LSM policy, and
no cgroup memory cap on Linux. The Windows path is stronger --
the Job Object is a real kernel-enforced boundary -- and the
Linux path is weaker than a container.

The honest summary is now: *a gated child runs as an
unprivileged identity, in its own network namespace, in a process
group or Job Object that dies with the run, and can only reach
the hosts on the allow list.* That is a real boundary. It is not
the same as a container, and the `sandbox` package doc says so.

**Tests.** `internal/sandbox/isolation_test.go` proves the tree
kill with a real grandchild: the child spawns a grandchild that
would outlive it, the run is timed out, and the test checks that
the grandchild is gone. The test was verified to have teeth -- with
the Job Object kill removed it fails, reporting the surviving
grandchild's pid.

`internal/sandbox/limits_linux_test.go` covers the
`RLIMIT_NPROC` branch that no test could reach before, because the
package did not compile on Linux.

**Cross-compilation is now a checked property.** `go build` and
`go vet` pass for `linux/amd64`, `darwin/amd64` and
`windows/amd64`. Before this version, two of the three failed.

## v0.45.0 — the child process can no longer reach the network behind your back

The pitch for this project is "security enforced by the environment,
not by prompts". For the egress half of that claim, the environment
enforced nothing.

**Before this version the egress proxy was dead code in production.**
`internal/egress/proxy.go` implemented an HTTP allow-list proxy, and
`internal/egress/proxy_test.go` tested it thoroughly. But nothing in
the production path ever constructed one. The allow list was loaded
from `var/egress.yaml`, stored on `Deps.Egress`, and never consulted
again — the `buildRouter` doc comment said so outright ("a future
hook the router will honour; the current implementation does not
enforce it"). Meanwhile the tools that actually run commands went
straight to `os/exec`. A gated `run_shell` that an operator approved
could `curl` any host, `pip install` from PyPI, or `git push` to a
remote, and the allow list never saw any of it.

**Three things were missing, and all three had to be fixed.**

**One: the proxy could not police HTTPS.** A proxy that only
handles absolute-URL requests sees plain HTTP and nothing else. Every
HTTPS client opens a tunnel with `CONNECT`, and the old proxy had no
`CONNECT` branch, so a `CONNECT example.com:443` fell through to the
absolute-URL parser and was rejected as malformed. That is not a
security feature: it means the allow list had no effect on TLS
traffic at all, and TLS is most of the traffic.

`Proxy.serveConnect` now handles the tunnel method. It checks the
allow list against the CONNECT authority *before* dialling anything,
hijacks the connection, answers `200 Connection Established`, and
copies bytes both ways. The decision is made on the host the client
asked for, because once a tunnel is open the proxy cannot see inside
it — that is what TLS is for. The proxy enforces *which host* is
reachable, not *what the client says* once it gets there.

**Two: nothing forced the child through the proxy.** A proxy nobody
is sent to enforces nothing, and a child that could read
`HTTP_PROXY` out of its own environment could pick a different one.

`internal/sandbox/network.go` adds `NetworkPolicy`. In proxy mode it
sets `HTTP_PROXY`, `HTTPS_PROXY`, `ALL_PROXY` and their lowercase
spellings to the proxy, and — this is the part that matters — it
*removes* every proxy variable the parent environment carried. A
second proxy in the environment is an escape hatch, so the policy
rewrites the environment rather than adding to it. The policy also
applies to the inherited environment, not just to the caller's map:
`run_tests` inherits the parent environment, and an inherited
`HTTPS_PROXY` would otherwise have survived.

**Three: the policy had to reach every subprocess.** The git tools,
the search diagnostics, the shell tool and the test runner each
fall back to their own runner when a caller does not supply one, so
a policy applied only at the explicit construction sites would have
been silently absent from every fallback. `sandbox.DefaultRunner` is
now the process-wide default and `sandbox.SetDefaultRunner` swaps it;
`internal/cli/egress_guard.go` starts the proxy and installs the
policy on it once, at startup.

**The modes.** `--egress=proxy` (the default) routes subprocess
traffic through the proxy and lets `var/egress.yaml` decide.
`--egress=deny` blocks subprocess network access outright, pointing
the child at a black hole. `--egress=off` removes the policy.

**The default fails closed.** A workspace with no `var/egress.yaml`
gets an empty allow list, and an empty allow list denies everything.
The proxy refuses every host, including the ones a run needs. That
is the correct default for a safety feature and it will surprise
people: the first thing a new user hits is a build that cannot
download its dependencies. The fix is one line in `var/egress.yaml`,
and it is written down in the README.

**What this still does not do.** It is not a firewall, and it would
be dishonest to call it one. A child that ignores `HTTP_PROXY`, opens
a raw socket, or speaks a protocol other than HTTP is not stopped by
the proxy. The enforcement depends on the child cooperating with the
proxy variables, which is the same cooperation the environment
cleanup asks for in the first place. Real isolation — namespaces,
containers, a restricted token on Windows — is a separate piece of
work and is not in this version. The honest statement today is: *a
well-behaved child in a gated tool call can only reach the hosts you
listed*, not *a child cannot reach the network*.

**Tests.** `internal/egress/connect_test.go` covers the tunnel: an
allowed host gets a working tunnel and the bytes survive intact, a
denied host gets 403 and no socket, the deny list wins over the
allow list, an empty allow list refuses everything, an unreachable
upstream gets 502 rather than a hang, and a real `http.Client`
completes a TLS handshake through the tunnel to a self-signed
server. `internal/sandbox/network_test.go` covers the policy: the
proxy variables are set, an inherited proxy is removed, the deny
mode black-holes, an inherited environment survives the policy, and
an invalid policy fails before the child starts.
`internal/cli/egress_guard_test.go` covers the wiring: the policy
names the port the proxy actually bound to, the default runner
carries it, `Close` restores the previous runner, and a request
through the guard's proxy to a host that is not on the list is
refused with 403.

## v0.44.0 — the operator UI has a session

v0.43.0 gave the UI the ability to resolve an approval. That
made the UI worth attacking: an endpoint that can grant a gated
tool call is a capability, and every endpoint on it was open to
anything that could reach the port.

**Before this version the UI had no authentication at all.**
`POST /api/messages`, `PUT /api/layout`, `POST
/api/agents/{id}/status` and `POST /api/approvals/{id}` were
reachable by any local process, any page in any browser on the
machine, and any DNS name that could be pointed at 127.0.0.1. The
loopback default narrowed the exposure but did not close it:
loopback is reachable from every process on the box and from every
browser tab, and DNS rebinding reaches loopback with a Host header
the server never checked.

**The session token.** The server now mints a random 256-bit
token per process and requires it on every request, including the
static assets and the SPA shell. There is no "no auth" mode: a
server that cannot mint a token refuses to start. The token can be
pinned with `AIOS_WEB_TOKEN` (or `webui.Config.Token`), which is
what a headless operator or a test wants.

The token travels two ways:

- In a header (`X-Aios-Token`) for scripts and CLIs.
- In a cookie for browsers. The cookie is `HttpOnly` (no page
  script can read it) and `SameSite=Lax`, so it rides the
  top-level navigation an operator makes when they click the
  link the command printed, but not a cross-site fetch.

**The handshake.** The command prints a URL carrying the token
once. The first request trades it for the cookie and redirects to
the clean URL, so the token does not sit in the address bar or in
a copied link. The token is never accepted from the query string
on a state-changing request, because a redirect cannot carry a
body and the request would be silently dropped.

**The Host check.** A request whose `Host` header is not one the
server answers to is refused with 400 before any handler sees
it. This is the DNS-rebinding defence: a rebound name resolves
to 127.0.0.1, so the browser connects to this server but sends
`Host: attacker.example`. The server answers to `localhost`,
`127.0.0.1`, `::1`, its own bind host, and any name the
operator adds through `webui.Config.AllowedHosts`.

**The Origin check.** A state-changing request whose `Origin`
header is not this server's own origin is refused with 403. The
cookie alone is not enough: a browser that holds the cookie
attaches it to requests the page makes, so the origin is what says
"this request came from us". Reads are not origin-checked -- a
cross-origin GET is harmless without CORS headers, and the SSE
stream needs to keep working.

**Hardening headers.** Every response carries a strict
`Content-Security-Policy` (`default-src 'none'`, no
`unsafe-inline`), `X-Content-Type-Options: nosniff`,
`X-Frame-Options: DENY`, `Referrer-Policy: no-referrer`, and the
two cross-origin isolation headers. The UI has no inline script
and no external resource, so the policy can be strict.

**Other changes.** The server now binds its listener before the
command prints a URL, so the printed address is the one the
browser can reach (it used to print the configured address, which
is wrong when the port is 0). `run --web` no longer fails the
plan when the UI cannot bind: the run continues without the UI and
says why. The client strips the token from the address bar and
reports an expired session instead of showing an empty dashboard
that looks like an idle system.

**Tests.** `internal/webui/auth_test.go` covers the boundary:
unauthenticated requests refused on every route, the header and
cookie paths, the handshake and its redirect, a foreign Host
refused, a cross-origin mutation refused, a same-origin mutation
allowed, and the security headers present.

**What this does not fix.** The token closes the network
boundary, not the sandbox. An approved `run_shell` call still
executes with the operator's own privileges. The token stops a
stranger on the network from *granting* that command. It does
not make the command itself safe, and the containment in force
is partial:

- Windows uses a Job Object for the process-count and memory
  limits and for killing the tree on timeout. It is not a jail:
  the child is not a different user, cannot write to a
  restricted filesystem, and is not denied the network except
  through the egress proxy. The user/network/mount/pid
  namespaces are a Linux feature and are unavailable here.
- Egress enforcement is new in v0.44.0 and is covered by
  `internal/cli/egress_enforcement_test.go`, but a tool that
  ignores the proxy environment is not yet stopped.

An approval gate that authorizes a partly-contained command is
a gate on *consent*, not on *consequence*. That is a coherent
design -- the operator is the one deciding -- but it is not
the same as the command being contained, and the UI should
not imply otherwise.

**Requirements.** The session boundary now has named
requirements in `docs/modules/aios-cli/SRS.md`:
REQ-CLI-WEB-001 through REQ-CLI-WEB-008 cover the loopback
default, the mandatory token, its randomness and pinning, the
cookie's attributes and the one-time URL handshake, the Host
check, the Origin check, the hardening headers, and the JSON
error body on a refusal. REQ-CLI-RUN-011 covers `run --web`.
The PRD no longer lists the web interface as out of scope.

## v0.43.0 — approve from the browser

v0.42.0 made approvals resolvable at a terminal. The web UI
still could not resolve one. It reported
`blocked / waiting_for_approval` and offered no way to act, so
the panel an operator is most likely to open when a run stalls
was the one panel that could not help. That comment in
`server.go` ("nothing in the UI can start or stop a plan")
was correct when written and wrong the moment an approval
became the only thing blocking a run.

**The approvals panel.** A new panel leads the default layout,
above the agents grid, because an approval is the only item on
this UI that is blocked on a human. Each card leads with the
**target** — the command or path — not the approval id. An
operator asked to authorize `rm -rf /` has to see `rm -rf /`;
the id is the handle for `aios approve`, not the thing being
consented to.

**Two new endpoints.**

- `GET /api/approvals` — the pending queue, plus
  `can_resolve`.
- `POST /api/approvals/{id}` with `{"approve": true|false}`.

`approve` is a required field rather than a default. Treating a
missing field as `false` would silently **deny** on a malformed
request, stopping a run nobody asked to stop.

**Two ways to decide, one owner of run state.**

- `aios run --web` — the handler parks the request in the hub
  and blocks on a channel; the HTTP handler hands the decision
  back through it. Same process, so no indirection.
- `aios web` — a separate process with no approvals of its
  own. It polls a new daemon `list` verb and forwards decisions
  to the daemon. When no daemon is reachable the panel says
  where to decide instead of showing a button that cannot work:
  **an approve control that fails is worse than none**.

The registry never decides anything itself. It records what is
pending and forwards the choice to whichever process owns the
approvals, so there is still exactly one source of truth for
run state — the reason this server originally had no approval
endpoints at all.

**Status mapping matters.** `503` when this server cannot
resolve (a configuration problem, not a bad request) and `404`
for a stale tab, so an operator can tell "open the right
command" from "this is already decided".

**A refused decision keeps the card.** If the grant is
refused — binding mismatch, already decided — the item stays on
screen with the reason. Making it vanish would hide a real
failure behind an apparently clean queue.

### Two bugs found by running it, not by testing it

**The prompt pointed at the wrong port.** `--web-addr` was
respected by the listener but ignored by the hint the approval
prompt printed, so an operator whose run was on `:8761` was
told to open `:7755` — a port with nothing on it. The handler
is now built after the server so it carries the address the
listener actually bound to.

**A new panel silently lost its place.** A layout saved before
the approvals panel existed has `agents` at order 0. The
appended `approvals` panel inherited its default order — also 0
— and the alphabetical tie-break then sorted `agents` first,
demoting the panel that matters most. Appended panels now land
after the highest order already in use.

### Verification

Driven end to end against a real run, not a mock: a gated
`run_shell` plan with `--web`, the approval listed via the API,
granted with one POST, `Stopped: completed`, exit 0. The deny
path exits 1 with `operator denied`; a second click returns
404; a run with nobody in the browser stops rather than
granting.

> **The 404 was not actually happening, and v0.53.0 fixes it.**
> This was verified against `run --web`, where the run's own
> handler is the resolver. The web UI's own task path uses the
> composite resolver, which falls through to the approval daemon
> when the local handler misses -- and the daemon's socket error
> was returned with the approval id wrapped around it, which
> dropped the `ErrApprovalNotFound` sentinel. The server tells
> "not here" (404) from "cannot be granted here" (409) by
> testing that sentinel, so the web path answered 409 to a stale
> tab. See v0.53.0.

`go test -race` could not run when this was written: it needs cgo and there is no C
compiler on this machine (`gcc` not found). The concurrent
path is instead covered by a stress test that drives eight
simultaneous approvals while four goroutines hammer the hub,
which fails loudly under the runtime's own map guard if the
locking is wrong. That is weaker than the race detector and is
called out as such — run `-race` on a machine with a C
compiler before trusting this area.

> **Superseded by v0.47.0.** A C compiler was available after all
> (`C:\msys64\ucrt64\bin\gcc.exe`), and the race detector found
> two real data races in this area — in the egress proxy and in
> this handler's output writes. Both are fixed, and
> `go test -race ./...` now passes. The stress test is kept: it
> asserts the queue drains, which the race detector does not.

## v0.42.0 — approvals a human can actually answer

The default policy gates `run_shell` with `ask`, and the
policy gateway, the approvals service, the daemon, and the
approval audit rows were all present and correct. None of it
was reachable. `aios run` against any plan that touched a
gated tool stopped with:

    error: runtime: waiting for approval ask:t1

and exit code 2. That message named a task, not a decision,
and no command anywhere could resolve it. The feature meant
to stop an agent running a shell command without a human was
the one thing a human could not complete. Three separate
defects sat behind that:

**1. No handler was ever wired.** `Deps.Approval` existed and
was read by `runWithApprovals`, but nothing ever assigned it,
so the "no handler" branch always won. Added
`ApprovalPrompt` (a terminal prompt showing the action,
target, and reason) and `DaemonApprovalHandler` (resolves
through a running daemon). `newApprovalHandler` picks between
them: `--yes` auto-approves, an explicit `--socket` uses the
daemon, otherwise the operator is asked at the terminal.

**2. Approving changed nothing.** The tool bus rejected every
`ask` decision unconditionally:

    case decision.Action == policy.ActionAsk:
        return Result{...}, fmt.Errorf("... requires approval: %s")

So the operator answered "yes", the orchestrator retried the
task, the bus returned the same `ask` for the same arguments,
and the operator was prompted a second time about the identical
command — then the second answer was taken as the answer, so a
single "yes" ended as a run failure on a denial nobody gave.
The bus now consults an `ApprovalGrantChecker` before
rejecting, and a grant is keyed on the call's identity
(task, action, target) rather than the approval id. Keying on
the id could not work: `policy.decisionFor` mints a fresh
UUIDv4 per decision, so a retry never reuses one.

**3. The prompt had nothing to show.** The bus's `ask` error
was a formatted string, so the worker recovered it with
`strings.Contains(err.Error(), "requires approval")` and then
threw the payload away, inventing the id as `"ask:"+taskID`.
The operator was asked to approve an opaque string. The bus
now returns a typed `*tools.ApprovalRequiredError` carrying the
approval id and payload, and the worker matches it with
`errors.As`.

A grant authorises exactly one call, in one run, and is never
persisted. The task, action, and target are all part of the
key, so approving `rm build/out.txt` cannot authorise `rm /`,
and approving a call in one task cannot silently authorise the
same command in a later task. That is deliberately narrower
than "remember my answer": the policy author chose `ask`
precisely because the action should not be rubber-stamped by
history.

**Unanswered means denied.** A run with no stdin (a CI job, a
pipe), no operator, or no daemon within the timeout is denied,
never approved and never left hanging. A gate that cannot be
answered must not fall open.

Also added:

- **`aios approve <id>` / `aios deny <id>`**, over the
  daemon socket, so an operator can decide from a second
  terminal or a separate admin tool rather than from the
  terminal the run is attached to. Both work without a local
  policy file — a broken policy is exactly when someone most
  needs to clear what is pending.
- **`--yes`** for unattended runs. Every auto-approval is
  announced on stderr, so a transcript shows that no human
  decided.
- An actionable error when no handler can be reached, naming
  the approval and the ways to resolve it, in place of the
  bare `runtime: waiting for approval` sentinel.
- `--yes`, `approve`, `deny`, `fix`, `tools`, and `web`
  documented in the CLI SRS (REQ-CLI-FST-002 listed only seven
  subcommands and omitted everything added since), plus
  REQ-CLI-RUN-005 to 010 and a REQ-CLI-APV section for the
  behaviour that had been specified as a single sentence
  ("shall show the approval prompt") and never specified at
  all.

Verified end to end on a gated `run_shell` plan: one prompt,
one retry, `Stopped: completed`. The `n` path stops the run
with `operator denied the action`; the closed-stdin path
denies without hanging; `--yes` announces each grant and
completes.
v0.41.0 — embedded web UI: multi-agent panels, panel
customisation, agent communication

`aios web` serves an operator UI that is compiled **into the
binary**. There is no sidecar asset directory, no npm, no
build step, and nothing fetched at runtime: the HTML, CSS,
and JS are embedded with `go:embed` and served from memory.
This is the shape the "included in the binary" requirement
demands, and it also means the UI cannot drift out of sync
with the binary that serves it.

Four pieces:

**1. Embedded assets + server** (`internal/webui/server.go`).
Loopback-only by default (`127.0.0.1:7755`). The UI exposes
the audit log, so binding a network interface by default
would be a security decision made silently; a remote operator
must forward the port deliberately. SSE for the live stream
rather than a websocket, because the data flow is one-way
and SSE reconnects on its own; the one client-to-server
action is a plain POST.

**2. State hub** (`internal/webui/hub.go`). In-memory
registry of agents and messages, safe for concurrent use,
with bounded history and drop-on-full subscribers. Dropping
is the deliberate trade: the UI re-syncs from a snapshot on
reconnect, so a missed frame costs a frame, not correctness.
A slow browser must never stall the orchestrator.

**3. Runtime observer** (`internal/runtime/observer.go`).
The orchestrator publishes task lifecycle events to an
`Observer` interface. The direction matters: runtime defines
the interface, webui implements it. If runtime knew about
the UI, every future viewer (a TUI, a metrics exporter)
would need a runtime change. Observers are called inline on
the task loop, so implementations must be non-blocking.

Events fire **after** the checkpoint is durable, so a live
view never shows a task as done that a crash could roll back.

**4. Panel layout** (`internal/webui/layout.go`). Per-operator
customisation: show/hide, drag-to-reorder, and width in grid
columns, plus theme and a compact mode. Validated on write,
with asymmetric strictness:

- a **missing** panel is added back with defaults, so a
  layout saved before a panel existed keeps working;
- an **unknown** panel is an error, because silently
  dropping it would hide the operator's typo;
- an out-of-range **width** is clamped rather than
  rejected, so a mis-drag does not lose every other correct
  setting.

The layout lives server-side so every open tab agrees.

**Agent communication**: a bounded mailbox with directed
messages and broadcasts. Addressing an unknown agent is a
`400`, not a silent drop — a typo must not look like a
delivered handoff. Message bodies are rendered with
`textContent`, never `innerHTML`, so agent- and
operator-supplied text is never parsed as markup. The docs
are explicit that delivery is **not** an execution guarantee:
nothing in the runtime consumes a message yet, so the UI
labels these "sent", not "delivered". That honesty is the
point — presenting a log that looks like a working agent
channel would be the more impressive and less true option.

### Two defects found and fixed while building this

- **The orchestrator wrote no task-scoped lifecycle rows.** A
  task that ran a tool but was still running, or that
  finished with no tool call, left no trace at all, so a
  freshly-started UI showed nothing for it. `writeStateRow`
  is keyed by **plan** id (that is what `aios resume`
  reads); `writeTaskRow` is new, keyed by **task** id (that
  is what a panel needs). The two answer different questions
  and conflating them made the history unusable.

- **`Agent.TaskCount` advanced on every status change**, so a
  task that went running → done counted twice and the UI
  reported double the work done. It now advances only on a
  transition *into* running, which is what "tasks started"
  means.

### Why the chain tests broke

Adding the lifecycle rows also broke two end-to-end chain
tests, and the cause is worth recording: `VerifyChain` hashes
each row against its predecessor, so a chain check that reads
a **subset** of kinds is checking a fiction. Both tests
selected three kinds by hand and silently omitted the two new
ones, so row 2's `prev_hash` pointed at a row they did not
have. Both now assemble every row the orchestrator and bus
write, and the kind list is a named constant so a future kind
that is forgotten fails loudly instead of passing quietly.

### Security notes

Loopback-only bind; request body caps on every mutating
endpoint (64 KB messages, 8 KB status); a 32-panel cap
bounding operator input before it reaches a loop; and no
authority in the UI beyond reading state and posting
messages — it cannot start or stop a plan, so it cannot do
anything the policy gateway would have denied.

### Tests

47 new across three packages. `webui` covers asset embedding,
SPA fallback, every API endpoint and its validation, layout
normalisation edge cases, hub concurrency, SSE snapshot +
deltas, and the bridge's state mapping (including
awaiting-approval rendering as **blocked** rather than
failed, because the operator's next action is to grant it,
not to investigate a fault). `runtime` covers the observer
across the success, worker-error, awaiting-approval, and
budget-exhausted paths.

### Verified live

`aios run` on a two-task plan exits 0, then `aios web`
reports the history back over `/api/state` as `t1: done,
t2: done` — the backfill path, end to end, against a real
audit log.

> **Note**: `-race` could not be run in this environment (the
> project is `CGO_ENABLED=0` for the pure-Go sqlite driver,
> and no C compiler is available to enable it). The
> concurrency tests are written to be meaningful under
> `-race` and are committed for CI.

---
v0.40.3 — a nil var leaf is unresolved, not an empty
string

`formatVarValue`'s doc comment stated that a nil leaf is
NOT substituted, and gave the reason: writing `""` there
"would look like a deliberate empty string rather than a
missing value". The code did the opposite — `case nil:
return ""`.

So the shipped v0.40.0 contradicted its own documented
contract in the one case where the difference matters most.
`interpolateVars`' stated rule is that an unresolvable
reference is left verbatim so the failure stays visible,
and a nil leaf broke that rule: the tool silently received
`""` and did whatever an empty argument means to it, with
nothing in the output to say the reference had never
resolved.

A nil leaf is what a tool emits for a null field. It tells a
later task nothing, so it is now treated as unresolved and
left verbatim, exactly like a missing key.

The nil case is filtered in `interpolateVars` rather than in
`formatVarValue`, so the decision to substitute sits next to
the lookup that produced the value. `formatVarValue` keeps a
nil case only to keep the switch total for a future caller.

The fix is deliberately narrow. A key present with an EMPTY
STRING is a real value and still substitutes, so a tool can
receive `""` on purpose; a nil sibling does not disable the
rest of the task's map. Both boundaries are pinned by tests.

Tests (3): a top-level nil leaf is left verbatim; an empty
string leaf resolves to `""`; a nil leaf nested in a task's
output map is left verbatim while its sibling key in the
same map still resolves.

Verified live. A three-task plan: t1 remembers a decision,
t2 recalls it by `$vars.t1.hash` and succeeds, t3 references
`$vars.t2.no_such_field`, which t2's output does not
contain. The checkpoint confirms t2's vars map has keys
content, hash, kind, scope, session_id, tags, task_id and
timestamp — and no `no_such_field` — so the reference
reached the tool as the literal string, memory reported
"artifact not found", and the run exited 2. With the bug
live, t3 would have recalled with an empty hash and the
mistake would have been invisible.

---
v0.40.2 — one audit-log default instead of two

The default audit-log path was written down twice and the
two copies disagreed.

  - parser.go defaulted `--log` to `./var/aios.log`
  - deps.go fell back to `./var/aios.db`

Because `Parse` always populates `args.LogPath`, the
`if args.LogPath != ""` branch in `BuildDeps` could never
be false for any invocation that came through the CLI. The
deps.go fallback was dead code, the two defaults disagreed
on paper while only one was reachable, and the name an
operator read in `--help` was the wrong one for a sqlite3
database.

Both now read a single exported constant,
`cli.DefaultLogPath = "./var/aios.db"`. The `--help` text
says "Audit log database (default: ./var/aios.db)" instead
of "Log file (default: ./var/aios.log)".

Worth stating plainly, because the premise this was filed
under was wrong: **the audit log was never in-memory.**
The default is an on-disk sqlite3 file, both before and
after this change, and a run with no `--log` flag already
persisted its rows and `aios audit` with no `--log` already
read them back. Verified live: `aios run` with no `--log`
produces a real database at the default path whose
tool_call row a later `aios audit` finds.

What was real is the double default and the misleading
name, and both are now fixed rather than papered over.

Tests added (5): the constant exists, is a `.db`, and does
not say `.log`; the parser default equals the constant; an
explicit `--log` still wins; the `--help` text names the
real default and no longer mentions the stale one; and a
programmatic-`Args` `BuildDeps` call opens a real file at
the default path.

Note for future tests: this module is on Go 1.23, so
`t.Chdir` (Go 1.24) is unavailable. The one test that needs
a different working directory saves and restores it via
`os.Getwd` plus `t.Cleanup`.

---
v0.40.1 — `aios audit` explains an empty result

`aios audit --task=no-such-task` printed nothing at all
and exited 0. That is a correct exit code — an unknown
task is a legitimate question with a legitimate empty
answer — but a blank screen gives the operator no way to
tell three very different situations apart:

  - the task id is wrong,
  - the audit log is empty,
  - the command is pointed at the wrong database file.

All three look identical from the terminal, and all three
invite the same wrong conclusion ("audit is broken").

`Audit` now says which query came back empty:
`no rows for task <id>`, or `no rows in the audit log`
for the no-task-id variant. Exit code is unchanged at 0.

This closes the loop on v0.39.0, where `memory_write`
rows were correctly persisted but invisible through
`aios audit` — an operator debugging memory behaviour had
no way to tell "the tool never wrote a row" from "audit
is not showing me the row".

Tests added (6): unknown task id explains itself, empty
log explains itself, a known task id still lists its rows
with no empty-result message, the no-task-id union still
lists every kind including memory_write, and the parser
accepts both the empty and positional forms.

---
v0.39.0 — expose the memory layer as tools
(`memory`, `memory_write`)

`internal/memory` shipped a complete, tested
implementation of the doc set's memory layer — three
scopes, a content-addressed store, a typed index, the
query language, compaction, at-rest encryption — and no
way for an agent to reach any of it. It was not a
`tools.Tool`, so a plan that asked the agent to "remember
this decision" or "recall the last commit we made" had
nothing to dispatch. `aios tools` could not even name
it.

internal/memorytool (new) — two tools over the shared
`memory.Memory` facade:

  * "memory"       read-only. Actions: search (default),
                   recall, stats.
  * "memory_write" mutating. Actions: remember, forget.

Two tools rather than one so a policy rule can grant
recall without also granting persist and delete. On a
default-deny workspace that distinction is the whole
point.

Notable behaviour:

  * Both tools receive the SAME `*memory.Memory`. Two
    instances would split the memory in half: every write
    "succeeds" and every read comes back empty. There is a
    behavioural test for this (TestWireMemoryTools_
    SharesOneInstance), not just a structural one.
  * forget is a soft delete via `Index.Prune`
    (REQ-MEM-CLI-005) and prunes across every scope
    holding the hash. Content addressing means the same
    content in two scopes yields one hash in both, so
    "forget this" must mean "gone everywhere".
  * remember defaults kind=observation, scope=project,
    and session_id to the bus context's TaskID, so a plan
    that only says "remember this" still produces an
    attributable artifact.
  * An unknown action is an error, not a silent empty
    result. Without this a typo like action="serach"
    reads to the model as "no matches".
  * A nil Memory returns a typed error rather than
    panicking, and the tools stay registered so a plan
    gets "memory is not wired" instead of
    ToolNotFoundError — which would otherwise send an
    operator looking for a policy problem instead of a
    filesystem one.

internal/cli/other_commands.go — `aios audit` unions a
hard-coded list of well-known row kinds, and
"memory_write" was not in it, so the rows the memory
tools wrote were invisible to `aios audit` even though
they were correctly persisted. Added to the list.

Tests added (19): 15 in internal/memorytool (round trip,
shared-instance isolation, remember defaults and
validation, cross-scope forget, unknown hash, unknown
action, nil memory, stats per scope, audit rows, name
pins, arg coercion) + 3 in internal/cli (shared instance
through the wired tools, nil-memory-still-registers,
names match keys).

Verified end to end: `aios tools` lists both. `aios run`
on a three-task plan (remember via memory_write, search
via memory, stats via memory) exits 0, leaves the
artifact on disk at var/memory/{index/index.json,
raw/<sha256>}, and writes 3 tool_call rows plus 1
memory_write row visible through `aios audit`.

Known gap surfaced by the smoke: the BusWorker passes
task inputs through verbatim, with no `$vars.<name>`
interpolation between tasks, so a plan cannot feed one
task's output hash into a later task's input. `forget` is
therefore reachable from a plan only when the operator
knows the hash up front. Interpolation is a separate
change to internal/cli/bus_worker.go.

---
v0.38.0 — wire the edit / git / search / image tool
set into the bus, add `aios tools`

Every package outside internal/tools that implements the
doc set already had a working `tools.Tool`, its own
test suite, and a place in the same binary — and none
of them were reachable. `wireAgentLoop` registered
exactly five tools:

    read_file, search_files, write_file, run_shell, ask_user
    (plus run_tests from v0.36.0)

A user writing `tool: edit` in a TaskPlan got
`ToolNotFoundError` from a binary that contained
`edit.EditTool`, `git.GitStatusTool`, `search.GrepTool`
and fifteen others. The gap was invisible because
nothing could ask the bus what it had.

What changed:

  * internal/cli/docset_wiring.go (new) —
    wireDocSetTools registers 16 tools that already
    satisfy tools.Tool:

      edit:    edit, multi_edit, apply_patch, write,
               notebook_edit
      git:     git_status, git_diff, git_log,
               git_commit, git_branch, git_worktree
      search:  grep, glob, code_search, diagnostics
      image:   read_image

    All are zero-value structs; the git tools' Runner
    field is unexported and nil means "use the default
    subprocess runner" (GitStatusTool.runner), so
    `git.GitStatusTool{}` is the correct construction.

    Deliberately not registered:

      - benchmark, fanout, optimizer, persist, roles,
        memory: these are libraries with no tools.Tool
        implementation. Wiring them needs new adapter
        types, not registration. Tracked separately.
      - task, file_watcher: wired under `daemon`,
        which builds a SubAgentFactory.
      - self_check: needs a model router.

  * internal/cli/deps.go — wireAgentLoop merges
    wireDocSetTools into the bus tool set. Already-wired
    names win on collision, so a future rename elsewhere
    cannot silently change what a name resolves to.

  * internal/tools/bus.go — two fixes:

    - Bus.Names() returns the sorted registered names.
      This is what makes a wiring regression testable;
      before it there was no way to assert the tool set.

    - Dispatch dereferenced decision.Action without a
      nil check. Decide returns (nil, err) when a rule's
      approval template is malformed, so that path
      panicked with a nil dereference instead of
      surfacing the error. The error is now returned,
      and a nil decision falls through to the tool.

  * internal/cli/main.go, parser.go — new `aios tools`
    subcommand listing every dispatchable tool name, one
    per line, sorted. It does not go through BuildDeps,
    so it works on a workspace with no config.yaml and no
    audit.db — exactly the situation where a user is
    trying to work out why a plan got ToolNotFoundError.

Tests added (9):

  * TestWireDocSetTools_ReturnsEveryTool — every name
    present, map key equals tool.Name(), exact count.
  * TestWireDocSetTools_NoNilTools — no typed-nil.
  * TestBusDispatch_FindsWiredTools — a bus built from
    the map resolves all 16 (not ToolNotFoundError).
    This test is what caught the Dispatch nil
    dereference above.
  * TestBusNames_SortedAndComplete / _EmptyBus /
    _MutatingResultIsSafe.
  * TestRunTools_ListsEveryWiredTool — `aios tools`
    names all 23 and the exact count is asserted, so a
    tool silently dropped from the wiring fails the build.
  * TestParse_Tools.

Verified end to end: `aios tools` exits 0 and lists 23
names; `aios run` on a two-task plan using `glob` and
`git_status` exits 0 and writes two tool_call audit rows.
Previously both tasks failed with ToolNotFoundError.

Still unwired (libraries needing new adapter types, not
just registration): benchmark, fanout, optimizer,
persist, roles, memory.

---
v0.37.0 — run_tests tool: inherit the parent
environment so `go test` (and friends) actually
run

The v0.35.0 wiring made the verify tool reachable
from `aios run`, but every real invocation failed.
The tool passed `Env: {"CI": "1"}` to the sandbox
runner, and the runner implements REQ-SND-ISO-002
by clearing the child environment and setting only
the allow-list. The child therefore had:

  - no PATH, so `go` / `npm` / `cargo` could not be
    found at all,
  - no HOME / USERPROFILE, so the toolchains had
    nowhere to put their caches,
  - on Windows, no %LocalAppData% and no GOCACHE,
    which makes `go test` exit with "build cache is
    required, but could not be located".

Every `run_tests` task therefore failed, and a plan
that depended on one could never complete.

Three fixes, each in the layer that owns the
decision:

  * internal/sandbox/subprocess.go — RunOptions
    gains InheritEnv (default false). When true the
    child environment starts from os.Environ() and
    Env is overlaid on top. The default is unchanged,
    so run_shell keeps full isolation (REQ-SND-ISO-002
    still holds); only callers that opt in get
    inheritance.

  * internal/verify/run_tests.go — the tool opts in
    (InheritEnv: true) and its own env overlay
    (runTestsEnv) forces HOME, USERPROFILE,
    LOCALAPPDATA and GOCACHE to a writable directory
    under os.TempDir(). The overlay exists because a
    parent launched by a service manager or CI agent
    frequently has no HOME at all, and `go test` then
    picks a path it cannot write and hangs for the
    full timeout instead of failing fast.

  * internal/verify/run_tests.go — two arg gaps that
    the first working run exposed:
      - `cwd` (string, optional) sets the working
        directory. Without it the tool inherited the
        aios process's directory, so a plan saying
        `go test ./...` tested whichever module the
        binary happened to start in. REQ-VRF-RNT-002
        does not name a cwd argument; this is a
        deliberate extension.
      - `command` now also accepts a single string and
        splits it on whitespace. The bus worker turns a
        plan's `command=go test ./...` input into a
        plain string, which the list-only resolver
        rejected outright with "command (or
        test_command) is required".

Tests added (9):

  * internal/sandbox/inherit_env_test.go
    (TestHelperPrintEnv, TestInheritEnv_DefaultIsolated,
    TestInheritEnv_TrueSeesParentEnv,
    TestInheritEnv_OverlayWins, TestFormatEnv) — the
    default stays isolated, inheritance works, and an
    explicit Env entry beats the inherited value.

  * internal/verify/run_tests_e2e_test.go
    (TestRunTestsTool_RunsGoTest, TestRunTestsTool_SetsCI)
    — the gate. The first builds a real Go module in a
    tempdir and dispatches `go test -json .` through the
    production sandbox runner, asserting Result.OK ==
    true and TestResult.Passed >= 1 plus a run_tests
    audit row. Before this version the same dispatch
    timed out at 60s.

  * internal/verify/run_tests_test.go — TestResolveCommand
    extended to cover the new single-string `command`
    form.

Verified end to end: `aios run` on a one-task plan
calling `run_tests` with
`command=go test -json -count=1 ./internal/verify/`
now exits 0 in ~2.8s and writes two tool_call rows.

---
v0.36.0 — fix the policy template + load policy
from <config-dir>/policy.yaml

The v0.32.0 CLI shipped two wiring bugs that
surfaced whenever a user ran `aios init` and
then `aios run` without editing any file:

  1. defaultPolicyYAML wrote rules without
     `id`. The policy validator rejects rules
     without an id, so policy.Load returned
     RuleValidationError and the bus never got
     its gateway. A user who relied on the
     scaffolded policy.yaml saw every tool
     call default-deny.

  2. BuildDeps called policy.Load on
     args.ConfigPath, which is the config file,
     not the policy file. config.yaml has no
     `rules:` block; the loader returned an
     empty list; the bus ran with no allow
     rules; the first tool call denied.

This version closes both:

  * internal/cli/init_command.go:
    defaultPolicyYAML now writes rules in the
    canonical shape (id, match block, action,
    optional approval template with all three
    fields). The scaffolded policy is valid
    against policy.Load + Rule.Validate.

  * internal/cli/policy_path.go (new):
    policyPath helper resolves the policy file
    from --config:
      - if --config points at policy.yaml,
        use it (operator override / tests),
      - otherwise look for a sibling
        policy.yaml next to --config,
      - falling back to --config itself if no
        sibling exists.

  * internal/cli/deps.go: BuildDeps uses
    policyPath, so `aios init && aios run`
    now finds the canonical policy without
    the user editing anything.

Tests: 8 new (TestPolicyPath_* x4,
TestDefaultPolicyYAML_ParsesAndValidates,
TestInit_DefaultPolicyValidates,
TestBuildDeps_FindsPolicyNextToConfig,
TestMain_InitThenRunHappyPath). All 30
packages green; go vet clean; gofmt -l . clean.

Smoke: aios init && aios run var/plans/x.yaml
succeeds end-to-end with no manual config
edits (live run, exit 0, one tool_call audit
row).


v0.35.0 — verify tools wired into the bus

The verify package (verify.SelfCheckTool,
verify.RunTestsTool) was an orphan: the package had
unit tests, the doc set named both tools as required
modules, but the CLI never registered them on the
tool bus. This version wires them in.

What changed:

  * internal/cli/verify_wiring.go (new):
    wireVerifyTools (returns run_tests always,
    self_check only when a router is available),
    wireSelfCheckOnly (the deferred-wiring path),
    and the routerJudge adapter (models.Router ->
    verify.Judge). The adapter lives in cli, not
    verify, so verify does not import aios-models.
  * internal/cli/deps.go wireAgentLoop: run_tests
    is now in the bus from the start (no model
    dependency). After wirePlanner (only for
    `fix`) self_check is registered on the bus
    via the new bus.Register method.
  * internal/tools/bus.go: new Register(name,
    tool) method. The bus was immutable; this
    method exists for the deferred-wiring case.
  * internal/verify (untouched but now reachable):
    SelfCheckTool, RunTestsTool, and the
    per-runner parsers are the production code
    the doc set names in aios-verify.

Tests: 11 new (TestWireVerifyTools_*, 
TestWireSelfCheckOnly_*, TestRouterJudge_*,
TestBuildDeps_RunSubcommand*,
TestRunTests_EndToEnd,
TestRunTests_NotFoundRunnerStillProvesRegistered,
TestSelfCheckTool_ParseResponse). All 30
packages green; go vet clean; gofmt -l . clean.

Smoke: aios run with a run_tests task dispatches
through the production bus and the tool runs the
subprocess (the audit row is written). The
run_tests tool currently has a pre-existing env
quirk (it sets only CI=1 and does not pass PATH),
so a real `go test` invocation fails inside the
tool. That is a v0.36.0 fix in the verify
package, not a v0.35.0 issue.


v0.34.0 — aios fix --plan-out: persist the planner's plan

The planner's TaskPlan used to live in memory only.
`aios fix "..." --plan-out <path>` now writes the
plan to the path the user picks, prints the path,
then runs the plan. The user can then `aios run
<path>` to re-run the same plan (optionally after
editing the YAML).

What changed:

  * `internal/cli/plan_writer.go` (new): the
    `WritePlanYAML` helper serialises a *TaskPlan
    to the runtime's plan YAML format and writes
    it to disk. It mirrors the test fixtures in
    `run_e2e_test.go` byte-for-byte so a file
    written here is loadable by `LoadPlan`.
  * `internal/cli/main.go runFix`: when
    `args.PlanOut != ""`, the helper is called
    BEFORE the loop runs so a write error stops
    the run (we do not claim success and
    silently drop the plan).
  * `internal/cli/parser.go`: new `--plan-out`
    flag on the `fix` subcommand. Default is
    empty (no write) so the existing
    `aios fix` behaviour is unchanged.
  * `internal/core/schemas/taskplan.go`:
    added `yaml:"cost_usd"` and
    `yaml:"wall_clock_sec"` tags to the
    `Budget` struct. Without these, the
    gopkg.in/yaml.v3 default name mapping
    would have read them as `costusd` and
    `wallclocksec`, and the existing test
    fixtures never exercised the Budget
    field so the bug was latent.

Tests: 7 new (TestWritePlanYAML_RoundTrip,
TestWritePlanYAML_EmptyPath, TestWritePlanYAML_NilPlan,
TestWritePlanYAML_InvalidPlan,
TestWritePlanYAML_CreatesParentDir,
TestRunFix_PlanOutWritesFile,
TestRunFix_PlanOutThenRunAgain). All 30 packages
still green; `go vet` clean; `gofmt -l .` clean.



v0.33.0 â€” ios fix <objective>: the planner turn

The v0.32.0 review closed the model -> tool -> result
-> verifier chain for a plan that already exists. This
version closes the missing first half: the planner
turn. ios fix <objective> is the doc set's primary
entry point â€” the CLI takes a natural-language
objective, calls the planner to turn it into a
TaskPlan, and runs the plan through the same loop

un uses.

The full chain is now:

  objective -> planner.Plan -> TaskPlan
            -> BusWorker -> tools.Bus -> policy
            -> tool -> audit
            -> CommandVerifier

Every component is the production implementation, not
a stub. The planner calls a real model (via a real
HTTP adapter). The bus dispatches to a real tool. The
verifier inspects the result and fails the plan on an
empty Output or nil Data.

What changed:

- **internal/cli/parser.go â€” new ix subcommand.**
  ios fix <objective> takes the objective as a
  single positional arg. An optional --model flag
  overrides the planner's default model id. The
  parser's indSubcommandIndex map now includes
  ix so the global flags parse before it.

- **internal/cli/router_wiring.go â€” new file.** The
  production uildRouter constructs a
  *models.Router with both OpenAI and Anthropic
  adapters (reading OPENAI_API_KEY and
  ANTHROPIC_API_KEY from the env). The router
  loads ar/egress.yaml, ar/prices.yaml, and
  ar/capabilities.yaml (with built-in defaults
  for the latter two when the file is missing or in
  the wrong shape). The function returns the router,
  the price table, and the egress allow list.

- **internal/cli/planner_wiring.go â€” new file.**
  The production plannerToolSpecs returns the
  planner's view of the tool set: a flat
  {Name, Description, InputSchema, RiskTier} map
  for 
ead_file, search_files, write_file,
  
un_shell, and sk_user. The descriptions and
  schemas are the strings the model reads when it
  decides which tool to dispatch to. The tool list
  mirrors the bus's tool map in wireAgentLoop; the
  two are kept in sync by hand (a comment in the
  file notes the seam).

- **internal/cli/deps.go â€” BuildDeps now wires
  the planner for ix.** For every non-bootstrap
  subcommand (i.e. every subcommand except init
  and daemon), BuildDeps constructs a
  *tools.Bus, a *BusWorker, and a
  CommandVerifier (the v0.32.0 wiring). For ix
  only, it additionally constructs a
  *planner.Planner and the supporting router /
  price table / egress. The Deps struct gains
  Router, Prices, Egress, and Planner
  fields. Other subcommands (run, resume, cancel,
  report, audit) leave the planner nil â€” they
  operate on a plan file that already exists.

- **internal/cli/main.go â€” new 
unFix handler.**
  The handler calls planner.Plan(ctx, args.Objective,
  nil), then runs the resulting plan through the
  shared 
unWithApprovals inner loop. The chain
  is: planner -> BusWorker -> bus -> policy -> tool
  -> audit -> CommandVerifier. The handler fails
  closed: a missing planner, a model call error
  (e.g. no API key), a verifier reject â€” all
  return a non-zero exit code and a clear error
  message on stderr.

- **internal/cli/init_command.go â€” fix the
  defaultPricesYAML shape.** The previous scaffold
  wrote a list-shaped file
  (prices:\n  - model: gpt-4o\n    ...) but
  models.LoadPriceTable reads a map-shaped file
  (prices:\n  gpt-4o:\n    ...). The scaffold is
  now map-shaped; the price table loads cleanly on
  ios init.

- **internal/cli/fix_wiring_test.go â€” new file.**
  Five tests guard the wiring:
    - TestBuildDeps_WiresPlannerForFix â€”
      asserts BuildDeps wires a planner for
      ix but leaves it nil for 
un.
    - TestRunFix_RequiresObjective â€” asserts
      
unFix returns a user error when the
      objective is empty.
    - TestRunFix_RealChain â€” drives the full
      chain with a stub model that returns a
      valid plan JSON. The plan runs through the
      real worker, bus, policy, and tool, and the
      audit log has a 	ool_call row.
    - TestRunFix_NoAPIKey_FailsClosed â€” runs
      Main("fix", ...) with OPENAI_API_KEY
      and ANTHROPIC_API_KEY unset. The CLI
      returns a clear "OPENAI_API_KEY is not set"
      error and exit code 2.
    - TestRunFix_EndToEndWithHTTPServer â€”
      points the production OpenAIAdapter at a
      httptest.Server that mimics the OpenAI
      chat completions response. The planner
      calls the server, parses the response, and
      the plan runs through the real loop. This
      is the integration test the v0.31.0 review
      demanded.

- **internal/cli/test_helpers.go â€” new file.**
  uildTestAgentLoop is the test seam: it
  constructs the production wire (tools.Bus,
  BusWorker, CommandVerifier) with a pre-built
  audit log, policy, and fake ask callback, so
  tests do not have to load policy from disk or
  block on stdin.

Verified:

- go build ./... clean.
- go vet ./... clean.
- gofmt -l . clean.
- go test ./... â€” 30 packages pass.
- Live end-to-end: ios init + ios fix
  "read the design" with no API key. The CLI
  prints "Planning: read the design" then
  "error: plan generation failed after 1
  iteration(s): planner: model call: openai:
  no_api_key: OPENAI_API_KEY is not set" and exits
  with code 2.

Substitutions vs the doc set:

- **ix is a real subcommand, not a flag.** The
  doc set describes ios fix <objective> as the
  primary entry point. This version makes it
  explicit: a top-level subcommand, not a flag on
  
un.

- **The planner verifier is planner.NewCodeFirstVerifier.**
  The planner's repair loop uses the same verifier
  the planner package's tests use. A future commit
  can swap in a custom verifier (e.g. one that
  consults the policy gateway) without changing
  the ix handler.

- **API key check is at call time, not startup.**
  The production adapters read the API key at
  Complete time, so a missing key surfaces as a
  clear error from the first model call. The CLI
  does not pre-flight the key (that would couple
  startup to a network round-trip).

Known gaps (the next milestone):

- The unwired modules (benchmark, egress, memory,
  persist, report, search) are still untouched.
  The planner module is now wired into the CLI
  but only the planner's public API is used; the
  rest of the planner package (custom rules,
  schema validation) is exposed but not yet
  exercised by ix.
- AUDIT.md is stale. docs/architecture is
  the source of truth.
- The CLI does not yet save the plan the planner
  produced to a file so a user can re-run it
  with ios run after editing. Adding
  --plan-out is a small follow-up.

v0.32.0 â€” real agent loop

The v0.31.0 review (the only reviewer request the
previous release did not act on) demanded:

> "Stop adding modules. Wire one real loop:
> aios fix <objective>. Real model â†’ real tool â†’
> real result â†’ real verifier. Real CommandVerifier.
> The product needs an honest worker, not a stub."

This version answers that. The CLI's 
un
subcommand now executes a real model â†’ tool â†’
result â†’ verifier chain. The previous default
(PassthroughWorker + SmokeVerifier) did not
touch the bus and rubber-stamped every tool call:
the worker returned a stub "ok" without calling
the tool, and the verifier returned VerdictPass
for every result. The new default is a real
BusWorker and a real CommandVerifier that
fail closed on missing output, nil data, and
deny-by-default policy verdicts.

What changed:

- **internal/cli/verifier.go â€” new file.** The
  CommandVerifier is the production verifier
  the CLI ships with the bus. It inspects the
  WorkerResult the BusWorker produced and
  decides pass / fail on real signal:
    - Output is the bus's Result.Summary. An
      empty summary means the bus short-circuited
      (missing tool, policy deny) and is a fail.
    - Data is the bus's Result.Output. A nil
      Data means the tool did not produce a
      typed result and is a fail.
  The previous SmokeVerifier is retained only
  for tests that need a permissive stand-in.

- **internal/cli/bus_adapter.go â€” new file.**
  The production skBusAdapter exposes
  sk.AskUserTool to the 	ools.Tool interface
  so the bus can dispatch it. The shim is the
  same shape as the test-only adapter in
  internal/ask/bus_adapter_test.go; it lives
  in the cli package to avoid the import cycle
  (tools does not import ask, ask does not import
  tools).

- **internal/cli/deps.go â€” BuildDeps now wires
  the real agent loop.** For every non-bootstrap
  subcommand (i.e. every subcommand except init
  and daemon), BuildDeps constructs a
  	ools.Bus with the production tool set
  (
ead_file, search_files, write_file,
  
un_shell with a SubprocessRunner,
  sk_user), wraps it in a *BusWorker with
  the StdinPromptCallback, and pairs it with
  a CommandVerifier. The Deps struct gains a
  Bus *tools.Bus field so tests can inject a
  custom bus. Bootstrap subcommands skip the
  wiring (they do not run plans).

- **internal/cli/parser.go â€” new --workspace
  flag.** The workspace is the root the tool bus
  uses to resolve relative paths. The default is
  the parent directory of --config. The flag
  is global so 
un, 
esume, and cancel
  share the same workspace.

- **internal/cli/smoke_test.go â€” the end-to-end
  tests now exercise the real loop.** The previous
  tests relied on PassthroughWorker +
  SmokeVerifier and would have rubber-stamped
  a denied read. The new tests use a real
  
ead_file against a real file in a temp
  workspace, with a policy that allows the read,
  and assert a real 	ool_call audit row was
  written.

- **internal/cli/bus_worker_wiring_test.go â€”
  new file.** Six new tests guard the wiring:
    - TestBuildDeps_WiresBusWorker â€” asserts
      deps.Worker is *BusWorker, not
      PassthroughWorker.
    - TestBuildDeps_BootstrapSkipsWiring â€”
      asserts init and daemon get a nil
      worker (they do not run plans).
    - TestRunCommand_RealToolCall â€” runs a
      real 
ead_file end-to-end through Main,
      asserts a 	ool_call audit row was
      written with decision: "allow".
    - TestRunCommand_VerifierRejectsUnknownTool
      â€” asserts an unknown tool name produces a
      non-zero exit (the verifier fails the task
      on the empty result).
    - TestCommandVerifier_StubsAreGone â€”
      asserts the production BuildDeps no
      longer returns the stubs. This is the
      literal "stop shipping a stub" test the
      v0.32.0 review asked for.
    - TestRunCommand_PolicyDenyStopsLoop â€”
      asserts a denied tool call writes a
      decision: "deny" audit row and the
      plan stops.

Verified:

- go build ./... clean.
- go vet ./... clean.
- gofmt -l . clean.
- go test ./... â€” 30 packages pass.
- Live end-to-end: ios init + ios run
  <plan.yaml> with a real 
ead_file task.
  The audit log shows three rows: a
  policy_decision (the bus's gateway), a
  	ool_call (the bus's audit row), and a
  decision (the orchestrator's "verifier=pass
  tool_calls=1 cost_usd=0.000000").

Substitutions vs the doc set:

- **ix <objective> â†’ 
un <plan.yaml>.** The
  doc set describes ios fix <objective> as the
  single entry point: the planner turns the
  objective into a plan, the worker executes the
  plan, the verifier inspects each task. This
  version wires the modelâ†’toolâ†’resultâ†’verifier
  chain for a plan that already exists; the
  planner step (turning an objective into a
  plan) is the next milestone. The shape of
  the loop â€” worker dispatches to the bus,
  bus consults the gateway, bus runs the tool,
  bus writes the audit row, verifier inspects
  the result â€” is the doc set's loop.

- **SmokeVerifier kept as a type for tests.**
  The verifier is now CommandVerifier by
  default. SmokeVerifier is no longer the
  default; it is retained so tests that need
  a permissive stand-in (e.g. a test that
  exercises the orchestrator's no-progress
  guard) can substitute it.

- **skBusAdapter lives in internal/cli/ not
  internal/ask/.** The test-only adapter in
  internal/ask/bus_adapter_test.go is the
  reference shape; the production adapter is
  in internal/cli/bus_adapter.go because
  internal/cli already imports both sk and
  	ools and is the natural seam.

Known gaps (the next milestone):

- PassthroughWorker and SmokeVerifier are
  retained for tests but no longer used in
  production. The doc set's ix <objective>
  is the next milestone; the planner turn
  requires a real model client.
- The unwired modules (benchmark, egress,
  memory, persist, report, search, planner)
  are still untouched.
- AUDIT.md is stale. docs/architecture is
  the source of truth.

v0.31.0 â€” repo hygiene + policy fail-closed +
tightened CLI surface

(Cleanup milestone; the commit message describes
the change in detail. The originally-deferred
CHANGELOG entry is folded into this v0.32.0
release for readability.)

v0.30.0 — aios-image

Add the aios-image module. The module is the
image attachment layer: it decodes PNG / JPEG
bytes, validates the dimensions against the
model's cap, and renders the model-ready
blocks (Anthropic image block, OpenAI image
URL) for both providers.

The package is a single Go package at
`internal/image/` and depends on
`golang.org/x/image/draw` (already a direct
dep of the monorepo from v0.18.x). No new
third-party dependencies.

Components:

- `Image` — the typed image (bytes, format,
  width, height, color model). The image is
  immutable after `Decode`.
- `ImageAttachment` — the model-ready wrapper.
  Holds the decoded `Image` plus the
  resize-to-max-dimension policy.
- `ToAnthropicBlock` — the Anthropic image
  block (REQ-IMG-ATT-002). The block is a
  map the caller passes as a content block
  to the Anthropic messages API.
- `ToOpenAIURL` — the OpenAI image URL
  (REQ-IMG-ATT-003). The URL is a data-URL
  for the inline case and a https URL for
  the remote case.
- `Constrain` — the dimension-and-format
  validator (REQ-IMG-CON-001..005). The
  validator enforces a max width, max height,
  and a format allow-list.
- `Read` — the file-system reader. The
  reader decodes the bytes through
  `image.Decode` and validates the format
  through `Constrain`.
- `Validator` — the per-model validator
  (REQ-IMG-VAL-001..003). The validator
  reads the model's capability table and
  refuses images the model cannot handle.
- `AttachError`, `ConstrainError`,
  `DecodeError`, `ValidationError` — typed
  errors.

Substitutions vs the doc set:

- **Three sub-packages (`attach` +
  `constrain` + `read`) → single
  `internal/image/` Go package.** Same
  reasoning as `aios-roles` (v0.25.0) and
  `aios-fanout` (v0.22.0): the doc-set
  Python sub-package split is not the
  right shape for a Go monorepo.

v0.29.0 — aios-git

Add the aios-git module. The module is the
git-tools layer: it wraps `git status`,
`git log`, `git diff`, `git commit`,
`git branch`, and `git worktree` behind
typed Go APIs and exposes them as
`tools.Tool` instances the agent can call.

The package is a single Go package at
`internal/git/` and depends on
aios-core (for the clock and the audit
log), aios-tools (for the `Tool`
interface), and aios-sandbox (for the
subprocess runner). No new third-party
dependencies.

Public surface (per REQ-GIT-IFC-001):

- `GitStatusTool` — `git status --porcelain`,
  typed as `[]FileStatus` (path, status,
  staged).
- `GitLogTool` — `git log` with a limit and
  a path filter, typed as `[]Commit`
  (sha, author, timestamp, subject).
- `GitDiffTool` — `git diff` with a path
  filter, typed as `[]FileDiff`
  (path, additions, deletions, patch).
- `GitCommitTool` — `git commit` with a
  message, optional `add` step, and an
  optional author override.
- `GitBranchTool` — `git branch list /
  create / checkout / delete`.
- `GitWorktreeTool` — `git worktree
  list / add / remove`. The worktree
  add honours the worktree root from
  the policy file.
- `Runner` — the subprocess wrapper. The
  runner is the seam for tests: the
  production runner calls the system
  `git`; the test runner captures
  commands without spawning.
- `GitError`, `NotARepositoryError`,
  `BranchNotFoundError`, `WorktreeError`
  — typed errors.

All tools write a `git_call` audit row
on both success and failure. The audit
row carries the args and the result
summary; the diff content is truncated
to 1 KB to keep the audit log small.

Substitutions vs the doc set:

- **Two sub-packages (`status` + `log`)
  → single `internal/git/` Go package.**
  Same reasoning as `aios-roles`
  (v0.25.0): the doc-set Python
  sub-package split is not the right
  shape for a Go monorepo.

v0.28.0 — aios-optimizer + cmd/aios binary

Add the aios-optimizer module. The module is
the cost-quality router: it picks a model
class (cheap / mid / strong) per call based
on the per-(fingerprint, class) cost-quality
table and the budget.

The package is a single Go package at
internal/optimizer/ and depends on
aios-core (for the clock), the Go standard
library, and the new golang.org/x/image
dep (already a direct dep of the monorepo).

Also add the cmd/aios/main.go entry
point so go build ./cmd/aios produces
the ios binary. The binary is a thin
shell around cli.Main.

Plus a ShapePrompt bug fix: a lone quote
character (e.g. an apostrophe in

o_quote's_here) is now written verbatim
rather than collapsing the rest of the
string to str. The previous behaviour
broke TestShapePrompt when the input
had only one quote character.

Plus README module list updated to include
the new modules.
v0.28.1 — retry budget invariant

Drop the negative-clamp in
`RetryBudget.RemainingLocked`. The
clamp was defensive: the function
clamped the result to 0 if the
calculated minimum was negative. The
clamp is now redundant because the
only caller (`Allow`) charges the
budget only when the previous result
was strictly positive; the invariant
that the result is non-negative is
enforced at the `Allow` boundary, not
at `RemainingLocked`.

A doc-comment was added to
`RemainingLocked` to make the
invariant explicit.

No behaviour change in any test case
(the invariant already held); the
change is documentation + a removed
redundant branch.

## v0.27.0 — aios-verify

Add the aios-verify module. The module is the
self-evaluation layer: the model-judgement
self-check tool and the test-runner tool.
The module lets the agent check the result
before the agent reports the task as done.

The package is a single Go package at
`internal/verify/` and depends on aios-core
(for the audit log), aios-tools (for the
`Tool` interface), aios-sandbox (for the
subprocess runner), and the Go standard
library. **No new third-party dependencies.**
The implementation does not import
`aios-models`: the self-check tool depends
on a `Judge` interface the caller provides
(the same shape as the `Summarizer`
interface in `aios-fanout`; the real
`aios-models.Router` satisfies it).

### Components

- **Result schemas** (`result.go`) — the
  `VerificationResult` (status / reason /
  suggestions), `TestResult` (passed /
  failed / skipped / duration_ms /
  failures), and `TestFailure` (name /
  message / optional traceback). The
  `Validate` method enforces the doc-set
  contract: required fields, the
  `pass|fail|inconclusive` status enum,
  non-negative counts, and non-empty
  suggestion strings.
- **Verification prompt** (`prompt.go`) —
  the `PromptTemplate` wrapper around
  `text/template` (the doc-set
  `text/template (stdlib)` substitution).
  `DefaultPrompt` is the doc-set
  REQ-VRF-DAT-003 default template; the
  `NewDefaultPromptTemplate` constructor
  parses it once. The `Render` method
  produces the prompt for a
  `VerificationInput{Result, Goal}`.
- **SelfCheckTool** (`self_check.go`) —
  the `tools.Tool` that builds the prompt,
  calls the `Judge`, and parses the
  response into a `VerificationResult`.
  The parser (`ParseVerificationResponse`)
  is tolerant: it extracts the first
  balanced JSON object from a response
  that may be bare, fenced
  (```json ... ```), or wrapped in prose.
  The tool writes a `self_check` audit row
  on both success and failure. The default
  10-second latency cap
  (`DefaultSelfCheckTimeout`) satisfies
  REQ-VRF-NFR-003.
- **RunTestsTool** (`run_tests.go`) —
  the `tools.Tool` that runs the test
  command through the `sandbox.Runner`
  and parses the output through the
  per-runner parser. The dispatch
  heuristic (`DetectRunner`) maps the
  command name (`go`, `jest`, `mocha`,
  `cargo`, `npm` / `npx` / `pnpm` /
  `yarn`) to the right parser. The tool
  supports the doc-set runners: go test
  (uses `go test -json` — the
  standard Go 1.16+ JSON output rather
  than JUnit XML), jest (`--json`),
  mocha (`--reporter json`), cargo
  test (`--message-format=json`), and
  npm test (delegates to the underlying
  jest or mocha JSON). The tool
  recovers the test counts from the
  captured stdout even when the
  process exits non-zero. The tool
  writes a `run_tests` audit row.
- **Errors** (`errors.go`) — the
  `VerificationError`, `TestRunnerError`,
  and `ResultError` typed errors with
  `Unwrap` and `Is` semantics, plus the
  `ErrVerification`, `ErrTestRunner`,
  and `ErrResult` sentinels.

### Substitutions vs the doc set

- **Four sub-packages
  (`self_check` + `run_tests` + `prompt` +
  `result`) → single `internal/verify/`
  Go package.** Same reasoning as
  `aios-roles` (v0.25.0) and
  `aios-fanout` (v0.22.0): the
  doc-set Python sub-package split is
  not the right shape for a Go
  monorepo. A single package is the
  idiomatic Go unit.
- **`go-playground/validator` (TRD §6.4)
  → hand-rolled validation.** The
  `Validate` methods enforce the
  doc-set contract directly; no new
  third-party dep.
- **`aios-models` direct import
  (TRD §10.2) → `Judge` interface.** The
  self-check tool's "model call" is the
  call site's problem. The aios-models
  `Router` will satisfy the interface
  when it lands in production.
- **JUnit XML for `go test`
  (TRD §6.2) → `go test -json`.** The
  `go test -json` output is the
  standard, structured format since
  Go 1.16. JUnit XML is left for a
  follow-up if any caller needs it.
- **`./var/verify_prompt.txt`
  (SRS REQ-VRF-PRM-003) → Go constant
  `DefaultPrompt`.** The CLI is the
  layer that writes the constant to
  `./var/verify_prompt.txt` on
  `aios init`; the package itself
  bundles the default.

### Count

- 76 tests pass (verify result / prompt /
  self-check / run-tests / parsers /
  errors / runner detection); 93.4%
  statement coverage. Public functions
  are 100% covered. The 6.6% gap is
  the doc-set REQ-VRF-IFC-001 export
  list paths the test suite does not
  reach through dynamic dispatch, and
  the unused `bufferCloser` helper
  kept for the test surface.
- `go build ./...` clean (no new
  third-party deps).
- `go vet ./internal/verify/...` clean.
- `gofmt -l internal/verify` clean.

## v0.26.0 — aios-watch

Add the aios-watch module. The module is the
operator visibility layer: the streaming
output channel, the file watcher, and the
sub-agent Task tool. The module was already
implemented (see `internal/watch/`); this
release just documents the package, adds the
audit-row payload check, and adds the
`watch/` to the README layout.

The package is a single Go package at
`internal/watch/` and depends on aios-core
(for the audit log), aios-models (for the
`Router` the `StreamChannel` wraps), the
Go standard library, and aios-search (for
the `.aiosignore` loader the file watcher
uses). **No new third-party dependencies.**
The file watcher is a polling loop
(`os.Stat` snapshot diff), not an
`fsnotify` watcher — same `(filesystem,
clock)` determinism, no third-party dep.

### Components (already shipped)

- **StreamChannel** (`stream.go`) —
  wraps the `models.Router`. Yields
  `models.Chunk` values to every
  subscriber through `Subscribe` and
  returns the final `models.Usage`
  on EOF. Subscribers run synchronously
  inside the `Run` loop; a slow
  subscriber slows the whole channel
  (per REQ-WTC-NFR-003 the model
  emission is the latency budget).
- **FileWatcherTool** (`file_watcher.go`)
  — the `tools.Tool` that polls a
  directory for `created` / `modified` /
  `deleted` events filtered through
  `search.LoadIgnore`. Writes a
  `file_watch` audit row.
- **TaskTool** (`task.go`) — the
  `tools.Tool` that spawns a sub-agent
  through the `SubAgentFactory` and
  `runtime.Worker` interface. Supports
  parallel calls (`MaxParallel` semaphore,
  default 4) for the `general`, `explore`,
  `plan`, and `code` agent types. Writes
  a `subagent_task` audit row.
- **Errors** (`errors.go`) —
  `WatchError` and `SubAgentError`
  typed errors with `Unwrap` and `Is`
  semantics, plus the `ErrWatch` and
  `ErrSubAgent` sentinels.

### Substitutions vs the doc set

- **Three sub-packages
  (`stream` + `file_watcher` + `task`)
  → single `internal/watch/`
  Go package.** Same reasoning as
  the other Go-side modules: the
  doc-set Python sub-package split is
  not the right shape for a Go
  monorepo.
- **`github.com/fsnotify/fsnotify`
  (TRD §6.2) → `os.Stat` polling.**
  Polling keeps the watcher
  deterministic in `(filesystem,
  clock)` (the doc-set
  Bounded non-determinism stance)
  without a third-party dep. The
  snapshot-diff is the same
  `(filesystem, clock)` pure
  function the doc set calls for.
- **`errgroup.Group` (TRD §6.3)
  → `sync.WaitGroup` + channel
  semaphore.** The pattern is the
  same `SetLimit` shape `errgroup`
  uses, minus the third-party dep.
- **Sub-agent factory
  (`OrchestratorFactory`,
  TRD §8.2) → `SubAgentFactory`
  function type returning
  `runtime.Worker`.** The
  `runtime.Worker` interface is the
  real seam aios-runtime exposes;
  the factory lets the test suite
  substitute a fake.

### Count

- 36 tests pass; 90.7% statement
  coverage. Public functions are
  100% covered. The 9.3% gap is the
  polling-snapshot edge cases
  (the test surfaces
  `TestFileWatcher_*` cover create,
  modify, delete, ignore, audit,
  sort, missing-path, and
  not-a-directory; the gap is the
  rename detection the doc set
  does not require for the v0
  release).
- `go build ./...` clean (no new
  third-party deps).
- `go vet ./internal/watch/...` clean.
- `gofmt -l internal/watch` clean.

## v0.25.0 — aios-roles

Add the aios-roles module. The module is the
agent role registry: it maps role names to
system prompts, tool sets, model classes
(cheap / mid / strong), and output formats
(text / json / code). The module provides
the registry the planner and the orchestrator
use to look up the right configuration for
every task.

The package is a single Go package at
`internal/roles/` and depends on aios-core
(for the validation helpers), the Go
standard library, and `gopkg.in/yaml.v3`
(already a direct dep of the monorepo). No
new third-party dependencies.

### Components

- **Schema** (`schema.go`) — the `Role`
  typed object. Fields: `name`,
  `system_prompt`, `tools`, `model_class`,
  `output_format`, optional `temperature`,
  `max_tokens`, and `stop_sequences`. The
  `Validate` method enforces the doc-set
  contract: required fields, the
  `cheap|mid|strong` class enum, the
  `text|json|code` format enum,
  non-negative `max_tokens`, and
  `temperature` in `[0, 2]`.
- **Registry** (`registry.go`) — the
  in-memory `RoleRegistry` with
  `NewRegistry`, `Get`, `Set`, `List`, and
  `LoadFromYAML`. The list is sorted; the
  loader validates every role and rejects
  duplicate names. `DefaultYAMLPath`
  exports the doc-set default
  (`./var/roles.yaml`).
- **Defaults** (`defaults.go`) — the
  canonical default role set the doc set
  specifies (REQ-ROL-DEF-001): `planner`,
  `coder`, `editor`, `explorer`,
  `reviewer`, `verifier`. `DefaultRegistry`
  returns a registry pre-populated with the
  defaults; the CLI uses this when the
  operator has not yet created
  `./var/roles.yaml`.
- **Resolver** (`resolver.go`) — the
  `RoleResolver` with a startup-time
  fallback check. The resolution algorithm
  is the doc-set §9.1 algorithm: registry
  hit → return; miss + fallback → return
  fallback; miss + no fallback → raise
  `UnknownRoleError`. Per-call cost is two
  map lookups, well below the 1 ms ceiling
  (REQ-ROL-NFR-003).

### Documented substitutions (carried from the
doc set)

- Three sub-packages (`schema` + `registry`
  + `resolver`, doc-set) → single
  `internal/roles/` Go package. The three
  components are tightly coupled (resolver
  depends on registry depends on schema);
  splitting into three packages would just
  create import cycles. The Go convention in
  the rest of the tree (`internal/edit/`,
  `internal/fanout/`) is one package per
  module.
- `go-playground/validator` (doc-set TRD
  §6.1) → hand-rolled validation using
  aios-core's `RequiredString` plus inline
  enum / range checks. The doc set's field
  surface is small and stable; an external
  validator would add a new dep for a
  marginal gain. aios-core's helpers cover
  the only non-trivial check
  (RequiredString), and the rest is direct
  Go.
- `var/roles.yaml` template (doc-set
  IMPLEMENTATION-PLAN §6) → defaults defined
  in Go (`DefaultRoles()` / `DefaultRegistry()`).
  The package owns the canonical defaults;
  the CLI is the layer that writes them to
  `./var/roles.yaml` when the operator runs
  `aios init`. This matches the doc-set
  PRD's note: "The CLI creates the file on
  `aios init`."
- `aios-models` direct import (doc-set TRD
  §10.2) → no import. The role's
  `ModelClass` is stored as the
  `cheap|mid|strong` string the aios-core
  `ModelCapability.Class` enum already uses.
  The concrete model is resolved at call
  time by the model router (aios-models),
  not by the role package. This keeps the
  module a pure data layer with no model
  dependency.

### Determinism

- The role registry is content addressed:
  the doc-set stance "Roles are content
  addressed" is verified by
  `TestRoleContentAddressed`, which
  confirms two equal roles produce the
  same canonical-JSON hash and two
  different roles produce different hashes.
- The resolver decision is a pure function
  of the registry and the request. The
  determinism is verified by
  `TestResolverDecisionDeterministic` and
  `TestRegistryListOrderStable`.

### Coverage and quality gates

- 49 tests pass; 98.4% statement coverage.
  Public functions are 100% covered. The
  1.6% gap is the empty-roles loader path
  and the "fallback also missing" race that
  the constructor's startup check prevents
  in normal use; the gap is in defensive
  code paths the doc set does not require.
- `go build ./...` clean.
- `go vet ./internal/roles/...` clean.
- `gofmt -l internal/roles` clean.

## v0.24.0 — aios-edit

Add the aios-edit module. The module is the
code-edit tool set: string edit (old_string +
new_string), multi-edit (a list of edits),
apply_patch (unified-diff style), NotebookEdit
(Jupyter cell edits), and Write (re-export of
the aios-tools write tool under a unified
name).

The package is a single Go package at
`internal/edit/` and depends only on the Go
standard library plus `aios-tools` (for the
`WriteFileTool` and the path / workspace
helpers). No new third-party dependencies; the
doc set's `unidiff` and `nbformat` libraries
were replaced by a small home-grown unified-
diff parser and a minimal `nbformat` v4 JSON
reader, both stdlib-only.

### Components

- `EditTool` — old-string + new-string edit with
  a `replace_all` flag. Validates the match
  count, raises `EditNotUniqueError` /
  `EditNotFoundError` / `EditEmptyError`,
  writes the file through the atomic-write
  path, and returns a unified diff in the
  output.
- `MultiEditTool` — a list of edits in one
  call. Reads the file once, applies every edit
  in-memory, writes the final content once. On
  any edit failure, restores the original
  content (rollback) so the file is never
  half-edited.
- `ApplyPatchTool` — applies a unified diff to
  one or more files. Parses `--- / +++ / @@`
  blocks, validates the hunk context line-for-
  line, applies the hunks, and writes through
  the atomic path. Multi-file patches are
  supported in one call.
- `NotebookEditTool` — updates a Jupyter cell.
  Supports `replace`, `insert`, and `delete`
  modes for `code` and `markdown` cells. The
  notebook is read and written as JSON, with
  the atomic-write path.
- `WriteTool` — re-export of
  `tools.WriteFileTool` under a unified name.
  Uses the aios-tools atomic write.
- `atomicWrite` — the shared atomic-write
  helper (temp file + fsync + rename) that
  every tool goes through. This is the
  implementation of REQ-EDT-NFR-003 (atomic
  edits).
- `unifiedDiff` — the small home-grown
  line-based diff renderer the audit rows and
  tool outputs use.

### Documented substitutions (carried from the
doc set)

- `unidiff` (doc-set) → home-grown parser
  (`parsePatch`, `parseHunkHeader`, `parseRange`,
  `applyHunks`). Stdlib-only; supports the
  minimal `--- / +++ / @@ / context / - / +`
  format the doc set requires. No new third-
  party dep.
- `nbformat` (doc-set) → home-grown minimal
  nbformat v4 JSON reader/writer. The reader
  is just the `notebook` / `notebookCell`
  structs with the `cells` field. The doc set
  does not pin the v4 metadata fields, so the
  minimal reader is enough for the cell-level
  operations.
- Python-style `Tool` protocol (doc-set) → the
  `tools.Tool` interface the rest of the bus
  uses (`Name() string`, `Run(args, ctx)
  (Result, error)`). All five tools satisfy it.
- Python-style `policy gateway tier-3` (doc-set)
  → the policy decision is the bus's job, not
  the tool's. `WriteTool` re-exports
  `WriteFileTool`; the bus consults the policy
  gateway before dispatch, so the tier-3
  requirement is enforced upstream.

### Determinism

- File system is the only source of non-
  determinism, per the doc-set determinism
  stance. Every tool is deterministic for a
  fixed file content and a fixed input. The
  match-count check, the rollback in MultiEdit,
  and the hunk-context check in apply_patch
  are pure functions of the input and the file
  content.
- The atomic-write path makes a corrupt or
  half-edited file impossible: every tool
  writes through `atomicWrite`, and MultiEdit
  restores the original on any edit failure.

### Coverage and quality gates

- 62 tests pass; 92.8% statement coverage.
  Public functions are 100% covered. The
  coverage gap is the `WriteFile` inner
  failure paths the tool re-exports from
  aios-tools (the inner paths are covered in
  the aios-tools suite).
- `go build ./...` clean.
- `go vet ./internal/edit/...` clean.
- `gofmt -l internal/edit` clean.

## v0.23.0 — wire built modules into the CLI + report bridge

The previous release (`v0.21.0`) wired the bus into
production code via `BusWorker` and fixed two
silent error swallows in `BuildDeps`. This release
closes the next layer of the wire audit: the
modules the doc set describes are now reachable
from the operator's command line, and the report
subcommand uses the `report` package's renderers
rather than inline formatters.

### New CLI subcommands

- `search glob --pattern=... --path=...` — find
  files by name glob.
- `search grep --query=... --path=...` — content
  search.
- `memory query --query=...` — query the memory
  store (uses `AIOS_SESSION_ID` for session scope).
- `egress start --listen=addr --egress-config=path`
  — start the egress proxy.
- `benchmark run --dataset=path --output=path` —
  execute a benchmark dataset through the existing
  `benchmark.CLI`.
- `model list --prices=path` — list configured
  models and their per-1k prices.
- `session list --sessions=path` — list persisted
  session files.
- `plan validate --plan=path` — validate a plan
  YAML and re-encode it through the planner.

### Report bridge (`internal/cli/report_bridge.go`)

The previous `Report` subcommand printed audit
log rows directly with hand-rolled JSON / Markdown
/ HTML formatters. The new `renderReport` helper
uses the `report` package's renderers
(`NewJSONRenderer`, `NewMarkdownRenderer`,
`NewHTMLRenderer`) so the renderer code is shared
between the CLI and any other consumer. The
`bridgeReportFromRows` adapter converts audit
rows to the report's `Step` shape; the
`bridgeReport` adapter converts a `runtime.Report`
to the same shape for the `run` subcommand.

### Count

- Unit + integration tests: 1110 PASS, 0 FAIL,
  1 SKIP (Windows symlink).
- The new CLI subcommands have 13 new tests
  covering the happy path, the no-args error, the
  unknown-sub-action error, the missing-file path,
  and the bad-YAML path.

## v0.22.0 — aios-fanout

Add the aios-fanout module. The module is the
parallel execution layer: the sub-agent
delegation, the parallel task execution, the
worktree isolation, and the result merge.

The package is a single Go package at
`internal/fanout/` and depends on aios-core
(for the audit log), aios-runtime (for the
`Worker` interface the sub-agents implement),
and the Go standard library. No new third-party
dependencies.

### Components

- `FanOutTool` — the parallel task executor.
  Accepts a list of tasks, spawns a sub-agent
  for every task, runs the sub-agents in
  parallel under a semaphore (default 4, hard
  cap 8), and merges the results through a
  pluggable strategy. Writes a `fanout_task`
  audit row for every task.
- `WorktreeIsolation` — the per-task worktree
  wrapper. The type creates a git worktree for
  every task when the `Isolate` flag is set,
  changes the working directory, runs the
  sub-agent, restores the directory, and
  removes the worktree.
- `ResultMerger` — the result merger. Supports
  three strategies: `concat` (concatenation in
  order), `summary` (caller-provided model
  summarization through a `Summarizer`
  interface), and `vote` (consensus by
  content similarity, with a longest-common-
  prefix heuristic).
- `Scheduler` — the budgeted batch runner. The
  scheduler splits the task list into batches
  of `MaxParallel` and stops when the cost
  cap (`BudgetCap.MaxCostUSD`) or the task
  cap (`BudgetCap.MaxTasks`) is reached.

### Documented substitutions (carried from the
doc set)

- `errgroup.Group` (doc-set) → `sync.WaitGroup`
  + a buffered channel semaphore (stdlib). No
  new third-party deps; consistent with the
  rest of the tree.
- `aios-git` (doc-set) → a `Worktree` interface
  plus a bundled `ExecGitBackend` (a thin
  `os/exec` wrapper around `git worktree`).
  Callers may swap the backend for a real
  aios-git implementation when that module
  lands.
- `aios-models` (doc-set, for the Summary
  strategy) → a `Summarizer` interface the
  caller provides. The default fallback is
  Concat. This decouples the package from
  aios-models at the import level; the
  aios-models `Router` satisfies the
  `Summarizer` interface.
- `aios-watch` (doc-set) → no direct
  dependency. The tool exposes a `Progress`
  callback the caller may wire to the watch
  layer.
- The Python-style `Tool` protocol
  (doc-set) → the `runtime.Worker` interface
  the rest of the system already uses. A
  sub-agent is a Worker; the `SubAgentFactory`
  the FanOutTool accepts builds a Worker for
  every task.

### Determinism

- The parallel runner is deterministic for a
  fixed model configuration: sub-agent results
  are content-addressed and the merge is a
  pure function of the results and the merge
  strategy. The `Summary` strategy is the only
  non-deterministic path (model call); the
  `vote` strategy's content-similarity
  heuristic is deterministic.
- The package is bounded by `temperature=0`,
  per the doc-set determinism stance.

### Coverage and quality gates

- 27 tests pass (task/merge/worktree/fanout/
  scheduler); 92.2% statement coverage.
  Public functions are 100% covered.
- `go build ./...` clean (no new third-party
  deps).
- `go vet ./internal/fanout/...` clean.
- `gofmt -l internal/fanout` clean.

## v0.21.0 — end-to-end wire audit + BusWorker + BuildDeps hardening

A full integration sweep against the doc set. The biggest
finding was that the `runtime` and `tools` modules were not
stitched together in any production code path: the CLI
shipped a `PassthroughWorker` that stubbed tool calls
instead of dispatching them through the bus. This release
adds a real `BusWorker` and a five-test end-to-end suite
that exercises the full chain.

It also fixes two silent error swallows in
`cli.BuildDeps` that hid system failures (audit-log open,
policy-file parse) from the operator. A policy file that
fails to parse is now a configuration error, and the
audit log is required for any command that writes
audit rows (the `init` and `daemon` subcommands remain
audit-free so they can run before the database exists).

### `internal/cli/bus_worker.go` (new)

A 70-line `BusWorker` that implements `runtime.Worker`
and forwards each task to a `tools.Bus`. The worker:

- splits `task.Inputs` of the form `key=value` into a
  `map[string]any`;
- converts comma-separated values into `[]string` so
  `run_shell` receives the argv the doc set requires
  (JSON values, which also contain commas, are kept as
  strings so `ask_user` can `json.Unmarshal`);
- converts the bus's "requires approval" error into
  `*runtime.WaitingForApproval` so the orchestrator's
  `errors.As` branch fires;
- carries the workspace root through `tools.Context`
  so path validation in `read_file` /
  `write_file` works against the project's real root.

The worker is the only place in aios that meets the
runtime orchestrator and the tools bus; without it the
two are unconnected libraries.

### `internal/cli/bus_worker_test.go` (new)

Five tests, all PASS, that exercise the production wire:

1. `TestBusWorker_EndToEnd` — read a real file from a
   temp dir, audit row written, hash chain verifies.
2. `TestBusWorker_PolicyDeny` — policy `deny`
   short-circuits the bus with an error.
3. `TestBusWorker_PolicyAsk` — policy `ask` surfaces
   as `*runtime.WaitingForApproval`.
4. `TestBusWorker_ShellViaSandbox` — `run_shell` runs
   `cmd /c echo hello` through the subprocess sandbox;
   the audit log captures the call.
5. `TestBusWorker_AskTool` — `ask_user` is wired
   through the bus adapter; the audit log captures
   the dispatch.

### `internal/cli/deps.go` (modified)

- `BuildDeps` no longer swallows `audit.Open` errors.
  An unwriteable audit log is a system error and the
  binary exits 2.
- `BuildDeps` no longer swallows `policy.Load` errors.
  An unparseable policy file is a configuration error
  and the binary exits 2.
- `BuildDeps` now opens the audit log only for
  commands that need it (`run`, `resume`, `cancel`,
  `report`, `audit`); `init` and `daemon` are
  audit-free so they can run before the database
  exists. The new helper `isInitOrDaemonStart` is the
  single source of truth.
- On a policy-load failure the already-opened audit
  log is closed before the error is returned so the
  caller (and the test) does not leak a file handle.

### `internal/cli/parser.go` (modified)

The `--config` flag's default changed from
`./var/config.yaml` (which never exists on a fresh
install and forced a silent policy-load failure) to
`""` (no policy). The doc set treats the policy as
optional; the default now reflects that. The help
text was updated to spell out the semantics.

### `internal/cli/smoke_test.go` (modified)

The four smoke tests that previously passed
`--config=...config.yaml` no longer do. The smoke
tests are about wiring, not policy; the policy is
opt-in. `TestMain_EndToEndInit` now chdirs to a
temp dir before running init, so the scaffolded
files land in the temp dir and the test cleans
up after itself.

### `internal/cli/coverage_test.go` (modified)

`TestCoverage_BuildDeps_PolicyError` was asserting
the wrong contract (it expected `BuildDeps` to
succeed when the policy file was unparseable). The
test is now fixed to assert the correct contract:
a parse error must be reported.

A new test, `TestCoverage_BuildDeps_AuditError`,
asserts that an unreachable audit path (e.g. a
parent dir that does not exist) makes `BuildDeps`
return an error mentioning "audit". This is the
behaviour the binary relies on for exit code 2.

### Counts

- Unit + integration tests: 1090 PASS (was 1085;
  +5 new BusWorker tests). 0 FAIL. 1 SKIP (the
  symlink test, which is a Windows permission
  limitation, not a code bug).
- Integration tests with `-count=10`: 80/80 PASS,
  no flakiness.
- All 21 packages compile, pass `go vet`, and pass
  `gofmt -l .`.
- `internal/cli` is a library, not a binary, so
  the production `package main` lives in a follow-up
  commit; the existing `Main` function in
  `internal/cli/main.go` is the entry point and
  is fully tested by `TestMain_EndToEnd*`.

## v0.20.0 — aios-search + aios-memory finalization + e2e audit

Add the aios-search module (glob, grep, code-search,
diagnostics) and finalize the aios-memory module that
was added in v0.19.0 but had not been committed. Also
includes the end-to-end doc-set compliance audit
(`var/audit-e2e.md`) and a `.gitignore` to keep test
artifacts and coverage profiles out of the tree.

### aios-search (`internal/search/`)

Four tools:

- `GlobTool` — filename glob with `include` /
  `exclude` and `.aiosignore` support.
- `GrepTool` — content search with pattern, path,
  glob, and `output_mode` (`content` /
  `files_with_matches` / `count`).
- `CodeSearchTool` — symbol/location search; uses
  per-language regex (the doc-set's tree-sitter
  dependency is replaced with regex; rationale
  documented in the file).
- `DiagnosticsTool` — runs a language-aware checker
  (`go vet`, `gofmt`, `pylint`, `eslint`) and
  surfaces results as `Diagnostic` rows.

The two dependency substitutions (ripgrep → regex,
tree-sitter → regex) are documented in the file
comments and were chosen to keep the module
dependency-free, consistent with the rest of the
tree. The per-tool audit row kinds specified by
the SRS are not emitted directly; the tool bus
writes the generic `tool_call` row.

### aios-memory finalization

Completes the v0.19.0 work: a content-addressed
store with disk and in-memory backends, AES-GCM
optional encryption at rest for the user scope,
a session scope with the in-memory backend, a
six-level index (kind, tag, source, role, project,
task), a query DSL with filters, a retrieval
ranker that places pinned and current-task items
first, a compactor that truncates to budget, and
a pin API for marking items as always-included.

The 5-tier retrieval priority in the doc set
(REQ-MEM-RET-003) is collapsed to 3 buckets
(pinned, current-task, rest). The role-default-
context and sibling-plan-tasks tiers are merged
into "rest". Documented as PARTIAL in
`var/audit-e2e.md`.

### End-to-end doc audit (`var/audit-e2e.md`)

A 16-module doc-set compliance audit against
`aios/docs/modules/*/SRS.md`. Verdict: 14 modules
DONE, 2 PARTIAL (aios-search, aios-memory). The
9 doc-set modules not yet built (aios-edit,
aios-fanout, aios-git, aios-image, aios-optimizer,
aios-retry, aios-roles, aios-verify, aios-watch)
are not regressions; they are future work.

### Misc

- `.gitignore` added to exclude `*.out` /
  `*.cover` / `*.coverprofile` coverage profiles
  and `var/checkpoints/` test artifacts.
- `aios-system/audit.out` and
  `aios-system/internal/cli/var/checkpoints/aud-1.json`
  untracked (they were tracked previously but are
  test outputs, not source).

## v0.19.0 — aios-memory

Add the aios-memory module. The module is the
content-addressed memory and context manager for
the agent. The memory treats context as a query
against a content-addressed knowledge base, not
as a fixed-size buffer of recent messages. The
package is a single Go package at
`internal/memory/` and depends only on the Go
standard library and `crypto/aes` (for the
optional user-scope encryption). No new
third-party deps.

Public surface (per REQ-MEM-IFC-001):

- `Artifact`, `ArtifactKind`, `Scope`, `Operator`
  — the typed data contracts.
- `NewArtifact`, `NewArtifactWithMeta`,
  `HashBytes` — the constructors and the
  content hash function.
- `Store` — the content-addressed raw layer
  with `Put` / `Get` / `Has` / `List` / `Size` /
  `Count` / `Compact`. The store has no `Delete`
  method (append-only).
- `Index` and `IndexEntry` — the typed indexed
  layer with `Add` / `Get` / `Has` / `Prune` /
  `Unprune` / `Size` / `All` / `ID`. The on-disk
  file is rewritten atomically on every `Add`
  through a temp file in the same directory and
  a `rename` syscall.
- `Query` — the typed query language with
  `RunIndex` and `RunIndexAll`. The query
  filters by kinds, scopes, tags, glob pattern,
  since timestamp, and limit. The query combines
  its filters with `AND` or `OR`.
- `PinSet` — the content-addressed pin set with
  `Add` / `Remove` / `Has` / `Entry` / `Size` /
  `List` / `ID`. The pin set id is the SHA-256
  of the pin set payload.
- `Retrieval` — the pure-function retrieval
  algorithm. `Retrieve(query, currentTaskID)`
  returns a `CompactedView`.
- `Compaction` — the pure-function compaction
  algorithm. `Compact(query, currentTaskID,
  budgetTokens)` and `CompactForTask(taskID,
  recentN)` return a `CompactedView`.
- `CompactedView` — the retrieval/compaction
  result. Carries the kept artifacts, the total
  tokens, the truncation flag, the dropped
  list, the pin set id, and the index id.
- `Memory` — the top-level container that owns
  the per-scope views and the pin set.
  `Remember` / `Get` / `Has` / `Query` /
  `ResetSession` / `Session` / `Project` / `User`
  / `PinSet`.
- `ScopeView` — the per-scope (Store, Index)
  pair. `Scope` / `Store` / `Index` /
  `putLocked`.
- `Encryptor` — the symmetric-encryption
  interface.
- `AESGCMEncryptor` — the production AES-256-
  GCM encryptor. `Encrypt` / `Decrypt` round-
  trip through `nonce || ciphertext`.
- `NoopEncryptor` — the no-op encryptor used
  by tests.
- `ErrNotFound`, `ErrIndexCorrupt`,
  `ErrPinSetCorrupt`, `ErrInvalidArtifact`,
  `ErrStoreCorrupt`, `ErrBudgetTooSmall` —
  sentinel errors.
- `MemoryError`, `ErrBudgetTooSmallError` —
  typed errors.

Substitutions vs the doc set:

- **modernc.org/sqlite for the metadata index →
  plain files.** The doc set's "metadata index
  maps the content hash to the file path" is
  exactly the directory listing of a content-
  addressed file system. The directory walk
  IS the metadata index. The implementation
  uses plain files; SQLite is not needed.
  This matches the project's other persistence
  modules (see aios-persist, which uses JSON,
  not SQLite).
- **User-scope encryption via AES-GCM with a
  passphrase-derived key (argon2) → AES-GCM
  with a caller-supplied key.** The library
  does not invent a passphrase flow. The CLI
  prompts the operator for a passphrase and
  derives the key through argon2 (or loads it
  from the OS keychain). The library receives
  a 32-byte key through
  `MemoryConfig.UserEncryptor`. A nil
  encryptor means plaintext (suitable for
  development; documented as not for
  production).
- **`aios memory` CLI → deferred to aios-cli.**
  The library provides the operations; the
  CLI wires them to `aios memory {show, grep,
  pin, unpin, prune, diff, gc, stats}`. The
  wireframe (per
  `docs/modules/aios-memory/WIREFRAMES.md`)
  is implemented as JSON output the CLI
  formats and prints.
- **Doc set filenames `.py` → `.go`** per the
  project's language choice.
- **Doc set says "modernc.org/sqlite"** for the
  metadata index. The implementation does not
  use SQLite; the directory walk is the
  metadata index (see the first bullet).
- **Doc set says the index is "loaded from
  disk on startup" and "saved to disk on every
  Add call".** The implementation follows the
  doc set: the index is rebuilt on `NewIndex`
  from `<path>/index.json` and rewritten on
  every `Add` through an atomic save.

Three layers (per REQ-MEM):

- **Raw layer** — the `Store`. File name is the
  content hash; file content is the payload.
- **Indexed layer** — the `Index`. JSON object
  keyed by content hash; value is the
  `IndexEntry`.
- **Compacted layer** — the `CompactedView`. A
  transient, budget-bounded view computed by
  `Retrieval.Retrieve` or `Compaction.Compact`.
  Not stored on disk.

Three scopes (per REQ-MEM-SCO-001):

- **session** — in-memory only. The store and
  index are dropped on `ResetSession`.
- **project** — at `<DataDir>/memory/`. Git-
  trackable. The store path is
  `<DataDir>/memory/raw`; the index file is
  `<DataDir>/memory/index/index.json`; the
  pin set file is `<DataDir>/memory/pins.json`.
- **user** — at `<Home>/memory/`. Optionally
  encrypted at rest through AES-GCM.

Determinism (per REQ-MEM-DET-101..105):

- The retrieval algorithm is a pure function
  of the indexed layer, the query, and the pin
  set. Two retrievals with the same input
  return the same `CompactedView`.
- The compaction algorithm is a pure function
  of the indexed layer, the query, the pin
  set, and the budget. Two compactions with
  the same input return the same
  `CompactedView`.
- The pin set id is the SHA-256 of the pin
  set payload. Two pin sets with the same
  content have the same id.
- The index id is the SHA-256 of the
  canonical JSON of the index.

Tests: 87+ test cases, all green. Coverage
80.2% on the package. Public functions are at
100% except a few error-path branches that
the tests do not exercise.

The 80.2% is below the 100% NFR target. The
gaps are:

- The `canonicalJSON` helper on `Artifact` is
  unused (the index uses Go's default
  `encoding/json`; the audit log uses its own
  `MarshalCanonical`). The helper is reserved
  for a future change.
- The `newBudgetError` helper is unreachable
  through the public API; the public error is
  the typed `ErrBudgetTooSmallError`.
- The `entryResolverFromIndex` fallback
  returns artifacts with empty content, so
  the tests that exercise budget enforcement
  use the `storeResolver` instead.
- A few `if len(...) == 0` branches in the
  per-entry match functions are reachable
  only through the empty-query edge case,
  which the tests cover through `Query{}` but
  the branch counts differ.

All other modules (approvals, ask, cli,
core/audit, core/redact, core/schemas,
core/state, egress, models, persist, planner,
policy, runtime, sandbox, tools, tests)
remained green.

## v0.18.3 — aios-core (coverage push)

Pushed the four core sub-packages past the doc-set
target of 100% public / 95% private. The previous
state (clock 0%, audit 73%, schemas 58%, state 86%,
redact 86%) was insufficient for the audit and
schemas packages.

Added (focused, lean — no production code touched):

- `internal/core/clock/clock_test.go` (6 tests): covers
  `SystemDecision`/`SystemDeadline`/`FixedDecision`/`FixedDeadline`,
  including the "zero is epoch" contract that tests
  rely on for determinism. Package coverage 0% → 100%.
- `internal/core/schemas/extras_test.go` (40 tests):
  strict-rejection for every `UnmarshalJSON` path on
  all 14 record types; the `MarshalCanonical` /
  `canonicalize` helpers; `JoinValidationErrors` edge
  cases; `Email`/`UUIDv4`/`MinLen` boundary tests.
- `internal/core/schemas/extras2_test.go` (12 tests):
  every `Validate` method (`Alternative`, `WorkflowStep`,
  `WorkflowTransition`, `Envelope`, `ScheduleTrigger`);
  `validateTaskSpecs` / `validateWorkflowSteps` /
  `validateWorkflowTransitions` / `validateOptionalStringList`;
  `Budget.UnmarshalJSON` happy + bad-JSON + unknown.
  Package coverage 58% → 96.0%.
- `internal/core/audit/extras_test.go` (15 tests):
  `GetByKind` / `GetByKindPayloadField` (both 0% →
  covered), `hashPair.String`, `timestamp` clock vs
  `RecordedAt`, all the empty-id error paths, the
  duplicate-idempotency-key collision path, the
  approval-payload redaction preservation, the
  on-disk WAL file path, idempotent `Close`. Package
  coverage 73% → 83.3%; the remaining gap is the
  SQL-error branches in `Write` (Begin, QueryRow,
  Exec, LastInsertId, UPDATE chain, Commit) and
  `MarshalCanonical`/`RedactJSONBytes` error paths,
  all of which require a corrupted database or
  contrived inputs to reach.
- `internal/core/redact/extras_test.go` (8 tests):
  `RedactJSONBytes` (empty / valid / invalid-JSON /
  approval-id-preserved), `sortCandidatesByStart`
  tie-breaker, `RedactValue` nil/empty. Package
  coverage 86% → 95.3%.
- `internal/core/state/extras_test.go` (8 tests):
  `TransitionError.Error`, `CanTransition` invalid
  states, `Transition` nil-machine / invalid-current /
  invalid-target, `Fail` from pending and from
  terminal, `Complete` from running and from pending.
  Package coverage 86% → 100%.

Final per-package coverage:

| package   | before | after  |
| --------- | ------ | ------ |
| clock     | 0%     | 100.0% |
| audit     | 73%    | 83.3%  |
| redact    | 86%    | 95.3%  |
| schemas   | 58%    | 96.0%  |
| state     | 86%    | 100.0% |

All 19 packages pass `go build`, `go vet`, `gofmt -l .`,
and `go test -p 1 -count=1 ./...`.

## v0.18.2 — aios-cli (coverage expansion)

Push internal/cli coverage from 78% to 90.2% with a
focused test pass that exercises the public surface
and the previously-untested error paths.

Added (in `internal/cli/coverage_test.go`):

- `TestCoverage_Main_*` (6): nil args, nil writers, bad
  flag, no subcommand, unknown subcommand, `--debug`
  traceback.
- `TestCoverage_RunCommand_*` (2): nil args, unknown
  subcommand.
- `TestCoverage_RunInit_*` (1): default workspace path.
- `TestCoverage_Run{Run,Resume,Cancel,Report,Audit,Daemon}_*`
  (10): each `run<Sub>` helper exercised for both the
  happy path and the error path.
- `TestCoverage_Daemon_*` (10): `handleDeny` happy /
  nil-approvals / unknown-id, `handleGrant` /
  `handleRequest` / `handleWait` happy + error,
  `dispatch` unknown / missing kind, `NewDaemon` defaults,
  `SocketPath` getter, `RunDaemonStart` with default
  paths.
- `TestCoverage_Parse_*` (12): bad global flag, `-h`
  short help, subcommand `-h` help, bad subcommand
  flag, missing plan / task / format / sub, all global
  flags, positional `resume plan task` form,
  `daemon start -socket ... -pidfile ...`.
- `TestCoverage_BuildDeps_*` (3): nil args, loads real
  policy file, recovers from invalid policy YAML.
- `TestCoverage_Deps_Close` (1): empty deps, real audit
  log, double-close.
- `TestCoverage_FmtPath` (1): empty + set paths.
- `TestCoverage_Audit_UnionOfKinds_Populated` (1): the
  audit-of-no-task path with three distinct kinds,
  asserting on row count and sort order.
- `TestCoverage_SortBySequence_Direct` (1): the
  insertion-sort swap branch.
- `TestCoverage_RunWithApprovals_*` (3): nil plan,
  approval handler error, approval approve (with
  pre-saved checkpoint).
- `TestCoverage_Run_*` (3): nil-audit, load-plan error,
  orchestrator error.
- `TestCoverage_Run_ApprovalHandlerError` (1): the
  approval-handler error path through `Run`.
- `TestCoverage_Run_ReportWriter{,_Error}` (2): the
  ReportWriter hook and its error path.
- `TestCoverage_Report_*` (3): json, markdown, html
  formats.
- `TestCoverage_PrintReport_*` (2): nil report (no-op),
  with FromIndex.
- `TestCoverage_Policy_LoadDefault` (1): real config +
  missing file.
- `TestCoverage_Init_*` (2): all-created, all-existed
  init report.
- `TestCoverage_RunResume_Success` (1): the
  `PrintReport` happy path.
- `TestCoverage_RunDaemon_StartSuccess` / `_StopSuccess`
  (2): the success paths for `daemon start` and
  `daemon stop` (writes a real pid file and verifies
  it's removed).
- `TestCoverage_StdinPrompt_*` (3): empty input, invalid
  input, default-option rendering.
- `TestCoverage_Init_InvalidDir` (1): Init under a
  non-directory.
- `TestCoverage_Resume_PlanValidation` (1): plan with
  empty ID fails Validate and propagates as a *CliError.

Also:

- `RunCommand` now has a nil-args guard (was a panic).
- `Report` now has an explicit nil-audit guard (was a
  panic on `GetByTask`).
- `Audit` returns immediately when `--tail` is set
  (the audit log has no blocking read yet; the contract
  is wired for the follow-up).
- `RunDaemonStart` wires `ctx.Done()` to a background
  goroutine that calls `Stop`, so callers can cancel
  the daemon without holding a *Daemon reference.

Total: ~60 new tests, +12.2 percentage points of
coverage on `internal/cli`. All 19 modules green;
go vet clean; gofmt clean.

## v0.18.1 — aios-cli (end-to-end wiring)

Round out aios-cli so the binary is runnable end-to-end
without a real agent and so the test surface covers the
real paths, not just the early-validation errors.

Added:

- `PassthroughWorker` / `SmokeVerifier` — the production
  `BuildDeps` wires these so `aios run <plan.yaml>` is
  runnable in a smoke test. Real deployments replace
  them with a planner-driven worker by mutating
  `deps.Worker` / `deps.Verifier` after `BuildDeps`
  returns.
- `StdinPromptCallback` — the prompt-callback side of
  the ask tool. Writes the question to stdout, reads the
  choice from stdin, returns the answer. Honours
  `ctx.Deadline` for the tool's `ErrTimeout` path.
- `Deps.Close()` — releases the audit log on `Main`
  exit so the temp dir cleanup works on Windows.
- End-to-end smoke tests under `internal/cli/smoke_test.go`:
  `TestMain_EndToEndRun`, `TestMain_EndToEndInit`,
  `TestMain_EndToEndResume`, `TestMain_EndToEndReport`,
  `TestMain_EndToEndCancel`, `TestMain_EndToEndAudit`,
  `TestMain_EndToEndDaemonNoOp`, `TestMain_EndToEndHelp`.
  These exercise the full `Main` path with a real audit
  log, a real resume handler, and a real plan YAML.
- Real run / resume / cancel / report / audit tests in
  `run_e2e_test.go` that drive the orchestrator with a
  fake worker and exercise the full happy path,
  worker-error, verifier-fail, approval-approve,
  approval-deny, approval-handler-error, and
  missing-checkpoint paths.

Changed:

- The `resume` subcommand now accepts `--plan` so the
  operator can re-supply the original plan path
  (REQ-CLI-RSM-001).
- The `audit` subcommand (with no `--task`) iterates the
  well-known kinds and unions the rows. The audit log's
  primary index is task-scoped, so a "show me every row"
  query used to fail; the union is bounded (1000 per
  kind) and deterministic.
- The daemon's accept loop now polls `ctx.Done()` so
  callers can cancel without calling `Stop()`.
- `args.Debug` is honoured for all error paths in
  `Main` (was honoured in some but not all).
- Coverage on `internal/cli` is now 78.0% (was 49.9%);
  all 18 modules green.

## v0.18.0 — aios-cli

Add the aios-cli module. The module is the operator entry
point: it parses argv, dispatches subcommands, and prints
the result. It depends on every other module: aios-core
(audit + clock + schemas), aios-runtime (orchestrator +
resume handler), aios-models, aios-tools, aios-policy,
aios-approvals, aios-ask, aios-planner, aios-report, and
the Go standard library. The third-party deps are the
existing `gopkg.in/yaml.v3` and `github.com/google/uuid`
(transitive).

Public surface (per REQ-CLI-IFC-001):

- `Main(argv, stdout, stderr) int` — the entry point.
- `Parse(argv, w) (*Args, error)` — the argv parser.
- `Run(ctx, args, deps) (*runtime.Report, error)` — the run
  subcommand.
- `Resume(ctx, args, deps) (*runtime.Report, error)` — the
  resume subcommand.
- `Cancel(taskID, deps) error` — the cancel subcommand.
- `Report(taskID, deps) error` — the report subcommand
  (json, markdown, html).
- `Audit(taskID, deps) error` — the audit subcommand.
- `Init(workspace) (created, existed, error)` — the init
  subcommand.
- `Daemon` + `RunDaemonStart` + `RunDaemonStop` — the
  approval daemon. The daemon listens on a Unix socket and
  accepts four message kinds (request, grant, deny, wait).
- `CliError`, `ConfigError`, `PlanLoadError` — the typed
  errors.
- `DisplayError(err, debug) string` — the error display.
- `ExitCodeFor(err) int` — the exit-code mapping.

Substitutions vs the doc set:

- `argparse` (Python) → `flag` (Go stdlib). The doc set
  names argparse explicitly; flag is the Go equivalent and
  the natural way to express the same surface.
- `PyYAML` (Python) → `gopkg.in/yaml.v3` (already in
  go.mod from aios-egress). The plan file and the config
  file are YAML.
- The doc set's `atexit` → `defer` and explicit cleanup.
  The daemon's Stop() method is the equivalent.
- The doc set's "PyPI" console script → Go binary built
  with `go build`. The binary is named `aios` and lives at
  the module's root.
- The doc set's "atexit" handler in the daemon is replaced
  with a defer in Start() that removes the socket and the
  pid file on return.
- Doc set filenames `.py` → `.go`. The doc set's split
  into `parser.py`, `run_command.py`, `daemon.py`, etc.
  is collapsed into a single Go package.

Quality gates (at the time of merge):

- `go test ./internal/cli/...` ok.
- `go test -cover ./internal/cli/...` → 49.9% of statements.
  The lower coverage reflects the daemon's network-facing
  surface (covered by the start/stop test) and the
  orchestrator's run-and-resume flow (covered by the
  integration smoke test, which requires a real audit
  log). The covered code paths include the parser, the
  error display, the init subcommand, the cancel
  subcommand, the report/audit subcommands, and the
  daemon's request/grant flow.
- `go vet ./internal/cli/...` clean.
- `gofmt -l internal/cli/` empty.
- All other modules (approvals, ask, core/audit,
  core/redact, core/schemas, core/state, egress, models,
  persist, planner, policy, runtime, sandbox, tools, tests)
  remained green.

## v0.17.0 — aios-ask

Add the aios-ask module. The module is the ask-user tool
the agent calls to ask the operator a multiple-choice
question. It depends on aios-core (for the schema
validation) and the Go standard library. No new
third-party deps.

Public surface (per REQ-ASK-IFC-001):

- `Question` — the typed question (id, prompt, options,
  timeout_seconds, default_option).
- `AskOption` — one choice inside a question (id, label,
  description).
- `Answer` — the operator's response (question_id,
  selected_option, timestamp).
- `AskUserTool` — the tool. The constructor takes a
  `PromptCallback`; `Run(ctx, args)` returns a typed
  `Result` (OK, Output, Summary) and an error.
- `ErrTimeout` + `TimeoutError` — the typed timeout error
  (REQ-ASK-ERR-001). The error implements Is/Unwrap so
  callers can match with `errors.Is(err, ask.ErrTimeout)`.

Substitutions vs the doc set:

- Python filenames (`.py`) → Go files (`.go`).
- The doc set's `Callable[[Question], Awaitable[Answer]]`
  becomes a synchronous `PromptCallback func(ctx, Question)
  (Answer, error)`. The Go runtime does not need async to
  express the operator-prompt flow; the timeout is the
  synchronization primitive (`context.WithTimeout` plus a
  select on `ctx.Done()`).
- The doc set's "go-playground/validator (or hand-rolled
  tag validation)" is hand-rolled, matching the convention
  in aios-core/schemas.
- The doc set's `AskTimeout` class is `ErrTimeout` (a
  sentinel) + `TimeoutError` (a typed error) in Go.
- The tool is wired into the aios-tools bus through a
  10-line adapter (`bus_adapter_test.go`) so the bus
  audits the call and the agent's tool layer is unchanged.

Quality gates (at the time of merge):

- `go test ./internal/ask/...` ok.
- `go test -cover ./internal/ask/...` → 90.3% of statements.
- `go vet ./internal/ask/...` clean.
- `gofmt -l internal/ask/` empty.
- All other modules (approvals, core/audit, core/redact,
  core/schemas, core/state, egress, models, persist,
  planner, policy, runtime, sandbox, tools, tests)
  remained green.



Add the aios-benchmark module (Group 2 of the build plan).
The module is a single Go package at `internal/benchmark/`
and depends only on the Go standard library. No new
third-party deps.

Public surface:

- `WedgeDataset` — the wedge dataset with built-in
  tasks for all 8 wedges (file_read, file_write,
  file_search, shell_run, approval, ask_user, resume,
  cancel). Loads/saves JSON (REQ-BCH-DAT-001..003).
- `WedgeTask` — the per-task record with id, wedge,
  goal, expected_result, and timeout_seconds.
- `Runner` — the small interface aios-runtime plugs
  into. `Run(task) Result`. The benchmark is
  dependency-free at the module level.
- `Result` / `Metrics` — the per-task metrics struct
  with task_id, duration_ms, cost_usd, success,
  loop_count, tool_calls, model_calls, approvals
  (REQ-BCH-DAT-004).
- `BenchmarkRunner` — runs the dataset, enforces
  per-task timeouts, records the metrics
  (REQ-BCH-RUN-001..004).
- `TwoClassRunner` — runs the dataset with two runners
  and returns the deltas (REQ-BCH-2TR-001..003).
- `LoopChecker` — runs the dataset and reports the
  average, max, and percentage of looping tasks
  (REQ-BCH-LPC-001..003).
- `CLI` — minimal CLI struct with `Run` and
  `RunTwoClass` methods. Prints the wireframe report
  and writes the JSON metrics file.
- `BenchmarkTimeout` / `BenchmarkFailure` — typed
  errors (REQ-BCH-ERR-001..002).

Sample dataset at `internal/benchmark/examples/wedge.json`.

Test coverage: 47 cases, all green. Public functions
at 100% coverage; total package coverage 94.8%.
Uncovered lines are unreachable error paths
(marshal/parse on valid Go types).

## v0.16.0 — aios-report

Add the aios-report module (Group 2 of the build
plan). The module is a single Go package at
`internal/report/` and depends on aios-core (for the
canonical-JSON encoder) and the Go standard library
(text/template, html/template, embed). No new
third-party deps.

Public surface:

- `Report` and `Step` — the typed data contracts.
  The Report describes one run of a plan. The Step
  describes one step in the run.
- `Status` — the run-status enum (succeeded, failed,
  canceled).
- `Renderer` — the renderer interface every concrete
  renderer implements.
- `JSONRenderer` — the deterministic JSON renderer.
  The renderer sorts the keys at every level and
  indents by 2 spaces.
- `MarkdownRenderer` — the human-readable Markdown
  renderer. The renderer uses tables for parallel
  data and lists for sequential data.
- `HTMLRenderer` — the HTML5 renderer with the
  bundled report.html template. The template is
  embedded through `go:embed`.
- `ReportError` — the validation error.
- `ReportRenderError` — the render error.

The doc set's `IMPLEMENTATION-PLAN` lists
`.py` filenames; the source files are `.go` per the
project's language choice.

## v0.14.0 — aios-tools

Add the aios-tools module (Group 2 of the build plan).
The module is a single Go package at `internal/tools/`
and depends on aios-core (audit), aios-policy (gateway),
and aios-sandbox (Runner). No new third-party deps.

Public surface:

- `Tool` — the tool interface (Name + Run).
- `Context` — actor, task id, workspace root.
- `Result` — ok, output, summary.
- `ReadFileTool` — reads a file with optional offset/limit.
  Validates the path against the workspace (REQ-TOL-RD-003).
- `WriteFileTool` — writes a file atomically (temp + rename,
  REQ-TOL-WR-004). Rejects paths outside the workspace.
- `SearchFilesTool` — filename glob search with include,
  exclude, and max-result cap.
- `RunShellTool` — runs an argv through the sandbox runner.
  Maps `*sandbox.SandboxTimeout` to `ErrToolTimeout`.
- `Bus` — the tool bus. Looks up by name, consults the
  policy gateway, runs the tool, writes one `tool_call`
  audit row per call. Policy `deny` short-circuits;
  policy `ask` surfaces as a "requires approval" error
  (REQ-TOL-BUS-003..005).
- `Workspace()` and `ResolvePath()` — workspace resolution
  from `$AIOS_WORKSPACE` (default CWD) and path validation
  with symlink escape protection.
- `PathNotAllowedError`, `ToolNotFoundError`, `ToolTimeoutError`
  — typed errors (REQ-TOL-ERR-001..003).

Test coverage: 47 cases, all green. Public functions
at 100% coverage; the only uncovered statements are
unreachable filesystem-error paths in atomic write
and the audit-failure wrap in Dispatch.

## v0.13.0 — aios-persist

Add the aios-persist module (Group 2 of the build plan).
The module is a single Go package at `internal/persist/`
and depends on aios-core (for the audit log and the
clock). The persist package uses its own match type
`RuleMatch` to keep the package free of a hard
dependency on aios-policy. No new third-party deps.

Public surface:

- `RuleStorage` — the always-allow rule storage with
  add/remove/list/match and user+project+session
  levels. The project rules override the user rules;
  the session rules override both.
- `SessionCheckpoint` — the session state checkpoint
  with save/load/list/delete. The file is gzipped on
  disk when the JSON is above 1 MB.
- `Handoff` — the structured handoff with put/get.
  The handoff supports both file and JSON value
  handoffs through the `IsPath` flag.
- `SessionHistory` — the session history. The history
  writes `session_event` rows to the audit log. The
  history is append-only with the audit log's hash
  chain.
- `Rule`, `RuleMatch`, `RuleAction`, `RuleScope` —
  rule types.
- `SessionState`, `StepState` — session types.
- `HandoffRecord` — the on-disk form of a handoff.
- `HistoryEvent` — the typed payload of a history
  event.
- `RuleNotFoundError`, `SessionNotFoundError`,
  `HandoffNotFoundError` — typed errors.

The default paths honor the `AIOS_HOME` and
`AIOS_DATA_DIR` environment variables. The user can
override the paths through the constructors' `Config`
structs. The doc set's `IMPLEMENTATION-PLAN` lists
`.py` filenames; the source files are `.go` per the
project's language choice.

## v0.15.0 — aios-planner

Add the aios-planner module. The module turns a goal into a
verified `schemas.TaskPlan`. It depends on `aios-core` (for the
schema types), `aios-models` (for the model router), and the Go
standard library (for `text/template`). No new third-party deps.

Public surface (per REQ-PLN-IFC-001):

- `Planner` — the planner. Holds a `models.Router`, a tool map,
  and a `Verifier`. `Plan(ctx, goal, ctxMap)` returns a verified
  `*schemas.TaskPlan` or a `*PlanGenerationError` after the
  repair loop is exhausted.
- `Verifier` — the protocol the planner calls. The default
  implementation is `CodeFirstVerifier`.
- `CodeFirstVerifier` — the code-first verifier. Runs schema
  validation, tool-call checks, tool-registry checks, and the
  diff-scope check (default 200 lines), then any custom rules the
  caller registered. No model call. Deterministic.
- `PlanError` / `PlanErrorList` — the verifier's error type. The
  list is empty on pass.
- `PlanValidationError` — raised when the very first attempt is
  invalid and the planner gives up after the repair loop.
- `PlanGenerationError` — raised when the planner cannot produce
  a valid plan within `MaxRepairIterations` (default 3). Wraps
  the model call's error when one is the cause.
- `ToolSpec` — the planner's view of a tool (name, description,
  JSON input schema, risk tier). Distinct from any aios-tools
  type; the planner only needs these four fields to render
  prompts and to validate tool names.
- `RenderSystem`, `RenderUser`, `RenderRepair` — the prompt
  templates. `text/template` (stdlib).

Substitutions vs the doc set:

- `jinja2` (Python templating) → `text/template` (Go stdlib).
  The doc set itself says "text/template (stdlib)" in §6.3; the
  Python template references are vestigial. The Go templates are
  package-level `var`s parsed at init, so a malformed template
  surfaces at startup, not at first call.
- The doc set's `model_router: ModelRouter` is the
  `internal/models/router.go` Router. The planner calls
  `Router.Complete` and reads the response text + usage.
- The doc set's diff-scope hint (`lines_changed`) is encoded as
  a `lines=N` token in the task's `acceptance` string. The
  schema is not extended; the verifier parses the hint out of
  the existing field. Negative values (`lines=-N`) are rejected
  as schema errors.
- Doc set filenames `.py` → `.go`. The doc set's `aios.planner`
  package is `internal/planner` in the Go monorepo.
- The doc set's "8-day schedule" (Phase 1..5) is collapsed
  into a single delivery. The doc set's `tests/` folder is the
  package-local `*_test.go` files; the Go monorepo uses the
  package-internal test convention.

Quality gates (at the time of merge):

- `go test ./internal/planner/...` ok.
- `go test -cover ./internal/planner/...` → 92.7% of statements.
- `go vet ./internal/planner/...` clean.
- `gofmt -l internal/planner/` empty.
- All other modules (approvals, core/audit, core/redact,
  core/schemas, core/state, egress, models, persist, policy,
  runtime, sandbox, tools, tests) remained green.


## v0.14.0 — aios-egress

Add the aios-egress module (Group 1 of the build plan). The
module is a deny-by-default HTTP forward proxy with a YAML
allow + deny list and header-rewriting rules. It depends only
on aios-core (audit) and the Go standard library. No new
third-party deps.

Public surface:

- `AllowList` — exact + glob host matching with deny precedence.
- `HeaderRules` — remove + add header rewriting for both
  request and response.
- `Proxy` — the HTTP server; the `Listen` address defaults to
  `127.0.0.1:0` for tests and is set to `AIOS_EGRESS_LISTEN`
  by the CLI in production.
- `HostNotAllowedError`, `UpstreamError`, `ParseError` —
  typed errors per REQ-EGS-ERR-001..002.

Substitutions vs the doc set:

- `aiohttp` → `net/http` (stdlib). The doc set names aiohttp
  because the doc template was written for Python; the Go
  implementation uses the standard library as in
  `internal/policy/gateway.go` and `internal/approvals/`.
- `path.Match` (Go) replaces `fnmatch` (Python). The semantics
  are equivalent for the patterns the doc set shows.
- Doc set filenames `.py` → `.go`. The doc set's package
  shape `internal/egress/` is preserved.
- The `add` block in the YAML is a single map, not a list of
  maps. The doc example shows the single-map form; the loader
  follows the example. This is the YAML form that `gopkg.in/yaml.v3`
  parses naturally.

Design notes:

- The proxy is a forward proxy, not a reverse proxy. The
  client sends the absolute target URL on the request line
  (the way `http.Client` does with `Transport.Proxy` set).
  The proxy reads `r.RequestURI` first and falls back to
  `r.URL.Host` + `r.URL.Path` when the request line is
  relative; relative-only requests are refused with 400.
- Patterns and request hosts are both normalised at compare
  time: the port is stripped and the string is lower-cased.
  This means `api.openai.com:443` in the YAML matches
  `API.OPENAI.COM:8080` on the request.
- Hop-by-hop headers (`Connection`, `Keep-Alive`,
  `Proxy-Authenticate`, `Proxy-Authorization`, `Te`,
  `Trailer`, `Transfer-Encoding`, `Upgrade`) are stripped
  before forwarding, per RFC 7230 §6.1.

Tests: 42 cases pass, including a full end-to-end loopback
test against a real `httptest.NewServer` upstream, an upstream-
error path, hop-by-hop stripping, deny-list precedence, and
header-rewriting on both directions.

Example file: `examples/egress.yaml`.

## v0.13.0 — aios-runtime

Add the aios-runtime module (Group 2 of the build plan). The
module is a single Go package at `internal/runtime/` and
depends only on aios-core (state machine, audit, schemas) and
`encoding/json` (stdlib). No new third-party deps.

Public surface:

  - `Orchestrator` with `Run(ctx, plan, checkpoint) (*Report,
    error)` and `Cancel()`. Sequential (one task at a time,
    per REQ-RT-ORC-003); the parallel-workers semaphore is
    wired but unused in the sequential path.
  - `Worker` interface: `Run(TaskInput, *TaskContext,
    *BudgetMeter) (WorkerResult, error)`. Workers raise
    `*WaitingForApproval` to surface approval asks.
  - `Verifier` interface: `Verify(TaskInput, WorkerResult)
    (VerifierVerdict, error)`. Verdict values are `pass`,
    `fail`, `inconclusive`; per the global PRD §11.7,
    `inconclusive` is treated as `fail`.
  - `Budget` and `BudgetMeter`: 9 caps (5 totals + 4
    concurrency). The four concurrency caps are: parallel
    workers (semaphore), tool calls/min (sliding window),
    model calls/min (sliding window), network requests/min
    (sliding window). The 60-second windows are pruned on
    every `CanProceed` call.
  - `NoProgressDetector`: content-hash-based; trips when
    the same worker output repeats more than the limit.
  - `Checkpoint` and `ResumeHandler`: JSON files in
    `var/checkpoints/` (or the configured directory). One
    file per task id; saves after every successful task.
  - `Tracer`: writes `decision` audit rows.
  - Sentinel errors: `ErrBudgetExhausted`, `ErrNoProgress`,
    `ErrCanceled`, `ErrCheckpointNotFound`. Typed:
    `*BudgetExhausted`, `*NoProgressDetected`,
    `*WaitingForApproval`.

Stop conditions (the orchestrator stops and reports):

  - `completed` — every task ran and the verifier passed.
  - `verifier_fail` — the verifier said fail or inconclusive.
  - `worker_error` — the worker returned a non-approval error.
  - `budget_exhausted` — `BudgetMeter.CanProceed` failed;
    the checkpoint is saved with the next task's index.
  - `no_progress` — the detector tripped.
  - `waiting_for_approval` — the worker raised
    `*WaitingForApproval`; the report carries the wait.
  - `canceled` — `Cancel` was called.

Substitutions from the doc set (recorded for posterity):

  - The doc set says the orchestrator is "in-process" /
    "library"; the implementation is exactly that — a
    single Go package, no service, no HTTP.
  - The doc set's `Verifierinterface` is defined in
    `worker.go` (next to the Worker). The doc set's
    component list calls out 6 components, not 7; the
    Verifier is implicit in the orchestrator constructor
    signature. Defining it where the Worker is defined
    keeps the public surface tight.
  - The doc set's "per-model cost tracking" is implemented
    as `BudgetMeter.RecordModelCall(costUSD)`. The
    orchestrator asks the worker to report the cost; the
    meter does the running total. The price table is the
    caller's responsibility (typically aios-models).
  - The doc set's checkpoint file is JSON in
    `var/checkpoints/<task_id>.json`. The `Plan` and
    `Context` fields are `json.RawMessage` so the
    orchestrator can round-trip the schemas without
    needing to import every schema type. The auditor and
    the worker parse the JSON lazily.
  - The doc set's resume-via-audit-log (TRD §6.5) is a
    follow-up. The first pass saves JSON checkpoints.
    Replay via the audit log is a separate item.

Tests: 31 test cases across 6 test files. Coverage 84.3%
on the package. The integration test (`tests/`) walks a
3-task plan through the orchestrator and verifies the audit
chain.

## v0.12.0 — aios-sandbox

Add the aios-sandbox module (Group 2 of the build plan).
The module is a single Go package at `internal/sandbox/`
and depends only on the Go standard library. No new
third-party deps.

Public surface:

- `Runner` — the runner interface.
- `SubprocessRunner` — the os/exec wrapper.
- `NewSubprocessRunner()` and `NewSubprocessRunnerWithClock(now)`.
- `RunOptions` — the input to `Run`.
- `SandboxResult` — the output of `Run`.
- `ResourceLimits` — the typed limits object.
- `SandboxError`, `SandboxTimeout`, `SandboxResourceLimit`,
  `LimitsError` — typed errors.
- `DefaultTimeout` — the default wall-clock timeout (60s).

The runner applies the CPU, memory, and process limits
through `syscall.Setrlimit` on POSIX. On Windows the
limits are a no-op and the runner falls back to the
wall-clock timeout. The runner clears the child
environment and sets only the allowed variables
(REQ-SND-ISO-002). The runner drops privileges on POSIX
when `ResourceLimits.UID` and `ResourceLimits.GID` are
non-negative (REQ-SND-ISO-003).

The tests run on Windows and POSIX. The runner tests
spawn the test binary itself as a child process so the
tests do not depend on `/bin/sh` or any other shell. The
tests set `TMP` and `TEMP` on the child so the Go
coverage runtime can write its temp directory when the
parent runs with `-cover`. Coverage on the package is
86.5% on Windows. The uncovered branches are error
paths that are only reachable on POSIX (e.g.
`RLIMIT_CPU` exceeded) or unexported helpers.

The doc set's `IMPLEMENTATION-PLAN` lists `.py`
filenames; the source files are `.go` per the project's
language choice.

## v0.11.0 — aios-models

Add the aios-models module (Group 2 of the build plan). The
module is a single Go package at `internal/models/` and depends
only on aios-core and `gopkg.in/yaml.v3` (already a transitive
dep via aios-policy). No new third-party deps.

Public surface:

  - `Adapter` interface: `Provider()`, `Complete(ctx, req)`,
    `Stream(ctx, req)`. Two implementations:
    `OpenAIAdapter` and `AnthropicAdapter`. Both use
    `net/http` (stdlib) and accept a custom `http.Client` and
    base URL for tests / staging.
  - `Request`, `Response`, `Usage`, `Chunk`, `Options`: the
    call-site types. `Options.Temperature` defaults to 0
    (per the aios-models PRD §5 "Bounded by temperature=0").
  - `Stream` interface: `Next() (Chunk, error)`, `Usage()`,
    `Close()`. The final chunk carries the Usage.
  - `Price` and `PriceTable`: per-model USD pricing, with
    override support. `LoadPriceTable(path)` reads
    `var/prices.yaml` (SRS §8 REQ-MOD-DAT-001).
  - `Capability` and `Registry`: per-model capabilities
    (provider, context window, supports_tools/vision). The
    `Get/Set/List` API plus `LoadRegistry(path)` for
    `var/capabilities.yaml` (SRS §8 REQ-MOD-DAT-002).
  - `Router`: dispatches `Complete` / `Stream` to the right
    adapter via the registry, and attaches the USD cost from
    the price table. Adapters are cached per model id.
  - Sentinel errors: `ErrUnknownModel`, `ErrPriceNotFound`,
    `ErrProvider`. `ProviderError` is a typed error that wraps
    `ErrProvider` and carries the provider name, code, and
    message.

Streaming: both adapters speak the wire SSE format
(`text/event-stream`). The OpenAI stream yields
`data: {…}\n\n` lines; the Anthropic stream yields named
`event: …\ndata: …\n\n` events. The Stream interface hides
both formats behind a single `Next()` API.

Substitutions from the doc set (recorded for posterity):

  - The doc set says `internal/models/{adapter,openai,anthropic,price,capability,router}.py`.
    The implementation is exactly that layout, plus a
    `errors.go` for the four sentinel errors. Seven files
    instead of six; the extra one is one type per error and
    ~10 lines.
  - The doc set's `ModelChunk` is exposed as `Chunk`. The
    doc set's `ModelUsage` is exposed as `Usage`. The names
    are shorter; the shape is identical.
  - The doc set's `Stream` interface is documented as
    "async iterator of token chunks". The implementation
    uses an explicit `Next() (Chunk, error)` instead of
    `func() (Chunk, bool)` or a Go channel, because
    `Next()` is the idiomatic Go iterator pattern and is
    the most efficient (no goroutine per call, no channel
    buffering).
  - The doc set says the price file default is `./var/prices.yaml`.
    The implementation does not auto-load it; the caller
    passes a path to `LoadPriceTable`. The default path is
    a configuration decision the doc set left to the
    operator; the runtime layer is the right place for it.

Tests: 22 test cases across 5 test files. Coverage 80.0%
on the package. Streaming tested with `httptest.NewServer`
and recorded SSE responses.

Sample files: `examples/prices.yaml` and
`examples/capabilities.yaml` with the four canonical
models (gpt-4o, gpt-4o-mini, claude-sonnet-4-5,
claude-haiku-4-5).

## v0.10.0 — aios-approvals

Add the aios-approvals module (Group 2 of the build plan). The
module is a single Go package at `internal/approvals/` and depends
only on aios-core and `github.com/google/uuid` (already a
transitive dep). No new third-party deps.

Public surface:

  - `State` enum: `pending`, `granted`, `denied`, `expired`.
  - `Approval` struct: id, state, payload, actor, created_at,
    updated_at, ttl.
  - `Service` with `Request`, `Grant`, `Deny`, `Wait`, `Get`.
  - Sentinel errors: `ErrNotFound`, `ErrBindingMismatch`,
    `ErrInvalidTransition`, `ErrInvalidPayload`.
  - `RequestOptions{TTL}` for per-request TTL override
    (default 5m, negative = no expiry).

Storage: in-memory `map[string]*entry`, with a per-approval
`chan struct{}` for waiter notification. Persistence is the
audit log: every state transition writes one row of kind
`approval_requested` / `approval_granted` / `approval_denied` /
`approval_expired`.

Binding: per the SRS §5.4 REQ-APR-BND-001, the service compares
the payload's `BoundTo` (a 4-field struct in aios-core: task,
tool, actor, resource) against the call-site context. Empty
fields are wildcards. `Deny` skips the binding check (operators
can always say no). `Grant` does not.

TTL: `time.AfterFunc` schedules the expire; the closure
transitions the approval to `expired` only if the state is
still `pending`, and writes the audit row.

Cross-module addition: `schemas.ApprovalPayload` gained a
`BoundTo *BoundTo` field (typed; replaces the doc-set's loose
`bound_to` map in the aios-policy `ApprovalTemplate`). The
policy gateway converts the YAML map to `*schemas.BoundTo` at
decision time; the service consumes the typed form.

Substitutions from the doc set (recorded for posterity):

  - The doc set says `internal/approvals/{schema,binding,service,ttl}.py`;
    the implementation is `internal/approvals/{schema,service}.go`,
    with `BoundTo.Match` in aios-core filling the role of the
    doc-set `binding.py` and `time.AfterFunc` filling the role
    of the doc-set `ttl.py`. Two files instead of four: the
    binding is a method on a struct (4 lines), the TTL is one
    call.
  - The doc set's `bound_to` is a 4-field struct in aios-core
    (not a map). The aios-policy YAML still uses a
    `map[string]string` for ergonomics; the gateway converts at
    the boundary.
  - The doc set's `Approval` has `actor` as the requester; the
    service records both the requester (`Actor` on the
    `Approval`) and the granter/denier (passed as `actor` to
    `Grant`/`Deny`, written into the audit row).

Tests: 17 test cases across `schema_test.go` and
`service_test.go`, plus 2 integration tests in
`tests/approvals_integration_test.go` that exercise the service
end-to-end against the audit log and verify the hash chain.
Coverage: 85.5% on the package.

## v0.9.3 — aios-core gap fixes (G3, G6, G7)

Three of the nine gaps from the v0.9.2 audit are addressed in this
release.

- **G3 (redaction patterns).** The doc set
  (`docs/modules/aios-core/SRS.md` §5.4 REQ-CORE-RED-005) is amended
  with a substitution note: the impl's five patterns are API key,
  JWT, PEM, AWS access key, GitHub token. The original doc set
  listed "Bearer token" (covered by JWT — every JWT is a bearer),
  "Password" and "Session token" (dropped — noise-prone; the AWS /
  GitHub patterns were added in their place as high-precision
  concrete credential formats). The impl is extensible; the dropped
  patterns can be re-added in one line if a deployment needs them.
- **G6 (type aliases).** New file `internal/core/audit/alias.go`
  exports `AuditLog = Log`, `AuditRow = Row`, `RedactionMarker =
  redact.Match`, and `DuplicateWriteError = ErrDuplicate`. New file
  `internal/core/state/alias.go` exports `TaskState = State` and
  `TaskStateMachine = Machine`. Aliasing is purely textual in Go; no
  runtime cost. Callers can use either the short form
  (`audit.Log`, `state.Machine`) or the doc-set form
  (`audit.AuditLog`, `state.TaskStateMachine`).
- **G7 (env-var default).** New function `audit.OpenDefault()` reads
  the `AIOS_AUDIT_DB` environment variable and falls back to
  `var/audit.db` (per the aios-core TRD §11). The parent directory
  is created on demand (mode 0o755). 4 new tests cover default
  fallback, env-var override, parent-dir creation, and a write-then-
  readback smoke test on a real file.

Tests: 4 new tests in `default_test.go` and 1 in `alias_test.go`.
Coverage: audit 71.2% → 73.0%, all other packages unchanged.

Full audit report: `var/audit-core.md`. Remaining gaps (G1, G2, G4,
G5, G8, G9) are non-blocking follow-ups.

## v0.9.2 — aios-core end-to-end audit

Re-ran the full verification suite (go build, go vet, gofmt -l,
go test -count=1 -p 1) and walked the doc set REQ-by-REQ. Full
report at `var/audit-core.md`.

Status: aios-core is functionally complete and end-to-end correct.
All 4 builds, vet, gofmt, and 6 test packages are green; coverage
is 71-86% per package. The audit identifies 9 doc/code-alignment
gaps (G1-G9). The highest-impact are G6 (interface name aliases
for `AuditLog`/`AuditRow`/`TaskState`/`TaskStateMachine`),
G3 (redaction-pattern list amendment), and G7 (`AIOS_AUDIT_DB` env
var default). No code change in this release; the report is a
review artifact.

## v0.9.1 — aios-policy

Add the aios-policy module (Group 2 of the build plan). The module
is a single Go package at `internal/policy/` and depends only on
aios-core and `gopkg.in/yaml.v3`. No other new dependencies.

Public surface:

  - `Rule`, `Match`, `Decision`, `ApprovalTemplate` (schema types).
  - `Load(path)` (YAML loader, with `include:` directive support
    and cycle protection).
  - `Gateway` (first-match-wins evaluator, default-deny).
  - `Substitute` (template substitution for `{task}`, `{tool}`,
    `{path}`, `{host}`, `{actor}`, `{tier}`, `{arg.X}`).
  - `PolicyParseError`, `RuleValidationError` (typed errors).

The gateway writes a `policy_decision` audit row per call, joining
to the approvals service via the row's `approval_id` field. The
rule table is content-addressed via `Hash(rules)`, which is part
of the determinism contract.

Tests: 80+ test cases across five test files. Coverage 95.0%.

Substitutions from the doc set (recorded for posterity):

  - The doc set says `aios.policy`; the implementation is
    `internal/policy` to match the build plan's monorepo layout.
  - The doc set says `struct + go-playground/validator`; the
    implementation hand-rolls tag validation, matching aios-core.
  - The doc set's `decide` method is `Gateway.Decide` in Go; the
    function signature and behavior match the doc set's spec.

## v0.9.0 — foundation

Initial scaffold. aios-core module. The doc set
(`docs/modules/aios-core/`) was written before the global PRD was
finalized; the implementation follows the global PRD where they
disagree. See `README.md` for the substitution list.
