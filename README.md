# aios

The aios CLI. It turns a sentence into a plan, runs the plan, and
shows you what it did -- in a browser, without a config file.

The Go module lives in `aios-system/`. The install scripts at the
repository root are what you want.

## Install

macOS and Linux:

```
curl -fsSL https://raw.githubusercontent.com/Hamidullah02/aios/main/install.sh | sh
```

Windows (PowerShell):

```powershell
irm https://raw.githubusercontent.com/Hamidullah02/aios/main/install.ps1 | iex
```

The installer downloads a prebuilt binary for your machine, checks
that it runs, and puts it on your PATH. There is no Go toolchain
needed and nothing to compile.

From source instead:

```
go install github.com/Hamidullah02/aios/aios-system/cmd/aios@latest
```

The Go module lives in the repository's `aios-system/` subdirectory,
so its path carries both the repository name and the subdirectory.
The release pipeline tags it accordingly -- see **Releasing** below.

To see which build you have:

```
aios --version
```

## Your model key

The system needs one model key to turn a sentence into a plan. Put
it in a `.env` file in the directory you will run `aios` from:

```
OPENROUTER_API_KEY=sk-or-...
```

On Windows, write it with `-Encoding ascii` so the file does not
start with a byte-order mark:

```powershell
Set-Content -Path .env -Value 'OPENROUTER_API_KEY=sk-or-...' -Encoding ascii
```

The `.env` is read on startup, and it is searched for from the
directory of your `--config` file and from the directory you are
standing in, upwards. A key you exported in your shell wins over
the file, so you can override it for one command.

Three providers work: **OpenRouter** (`OPENROUTER_API_KEY`),
**OpenAI** (`OPENAI_API_KEY`), and **Anthropic**
(`ANTHROPIC_API_KEY`).

**OpenRouter is the default.** The default model is
`nvidia/nemotron-3-ultra-550b-a55b:free` -- a free NVIDIA Nemotron
that is good at the one thing the planner asks of it: emitting a
JSON object and nothing else. A first run costs nothing. Use
`--model` to pick another one:

```
aios fix "summarise reports/q3.csv" --model nvidia/nemotron-3-ultra-550b-a55b
```

## First run

There is nothing to set up. Start it in the directory you want the
agent to work in:

```
aios web
```

The first run creates a `var/` folder beside you holding the
policy, the price table, and the audit log, and tells you it has
done so. **The agent works in the directory you are standing in**,
so a file it creates lands next to you. Its own settings stay in
`var/`.

The policy it writes allows reading, searching, and editing files
in that directory. Running a shell command is gated: you are asked
before it happens. Edit `var/policy.yaml` to change any of that.

Running `aios init` is still useful if you want to set the policy
up before starting -- but it is not a step you have to discover.

## The operator UI

The UI is embedded in the binary -- no npm, no sidecar assets. It
binds loopback by default and **requires a session token** on
every request:

```
aios web
```

The command prints a URL carrying a one-time session token. Open
it: the server trades the token for an `HttpOnly` cookie and
redirects to the clean URL. Treat that link as a credential -- it
grants full control of the UI, including approving tool calls.

For a headless setup, pin the token in the environment so it does
not change on every start:

```
AIOS_WEB_TOKEN=... aios web
```

Scripts can authenticate with the `X-Aios-Token` header instead of
the cookie.

The server also refuses a request whose `Host` header is not one it
answers to (the DNS-rebinding defence) and a state-changing
request whose `Origin` is not its own origin. `aios web --addr`
must stay on loopback unless you understand what you are exposing;
`webui.Config.AllowedHosts` is the escape hatch for an operator
who reaches the UI through their own hostname.

### Giving it something to do

The page opens with one field: **what do you want done?** Type a
sentence and press Start. The system works out the rest -- it
turns the sentence into a plan, repairs the plan if the plan is
wrong, and runs it. There is no plan file to write, no policy to
choose, and no terminal to open.

```
aios web
```

then type `summarise the sales numbers in reports/q3.csv` and
press Start. The task appears on the **Your tasks** panel with the
steps it took, and a card moves from planning to running to done
while you watch.

The same thing over HTTP, for a script or another tool:

```
curl -X POST http://127.0.0.1:7755/api/tasks \
  -H "X-Aios-Token: $AIOS_WEB_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"goal":"summarise the sales numbers in reports/q3.csv"}'
```

`POST /api/messages` with `"from":"operator"` does the same thing
and returns the task id as well. A message from an *agent* stays a
note between agents and starts nothing.

Earlier requests stay in the conversation, so a follow-up can
refer back to them: after `the numbers are in reports/q3.csv`, a
request for `summarise it` is planned with that file in mind.

**The server needs a model key to plan.** Without one the goal is
accepted and the task card says so in words -- no AI model key is
configured -- and it names the variable to set. `aios web` still
serves and still shows past activity; it just cannot start new work.
See **Your model key** above.

## Network egress

Every subprocess a run starts is pointed at an egress proxy, and the
proxy decides what that subprocess may reach. The allow list lives in
`var/egress.yaml`:

```yaml
allow:
  - proxy.golang.org
  - registry.npmjs.org
deny: []
```

The default is `--egress=proxy`: the policy applies, and the allow
list decides. `--egress=deny` blocks subprocess network access
outright. `--egress=off` removes the policy entirely, and is only
appropriate when you are deliberately testing the tools against the
open network.

**The default fails closed.** A workspace with no `var/egress.yaml`
has an empty allow list, and an empty allow list denies everything --
including the hosts a build needs. The first thing you will hit is a
build that cannot download its dependencies. Add the hosts to
`var/egress.yaml`; that is the whole fix.

HTTPS is enforced too. The proxy handles `CONNECT`, so a tunnel to a
host that is not on the list is refused before a socket is opened.

**What this is not.** It is not a firewall. A child that ignores
`HTTP_PROXY`, opens a raw socket, or speaks a protocol other than
HTTP is not stopped by the proxy. The honest claim is: *a
well-behaved child in a gated tool call can only reach the hosts you
listed*. The proxy is the network boundary on every platform.

## Process isolation

Every subprocess a run starts is contained. `--isolation` picks the
level:

- `auto` (the default) -- use the Linux namespaces when the
  platform and the kernel allow them, and fall back to the weaker
  containment when they do not.
- `namespaces` -- require the namespaces. A run that cannot have
  them fails rather than running with less isolation than you
  asked for.
- `none` -- no namespace isolation. The process-group and Job
  Object containment still apply.

What is in force either way:

- **The process tree dies with the run.** The child gets its own
  process group on Linux and macOS, and its own Job Object on
  Windows. A command that backgrounds work -- `npm run dev &`, a
  shell that spawns a compiler -- does not leave that work running
  after the timeout.
- **On Windows the kernel enforces the limits.** The Job Object
  caps the process count and the job's committed memory, and kills
  every process in the job when it closes.
- **On Linux the child is in its own namespaces.** User, mount and
  pid. The user namespace maps the child's uid 0 onto an
  unprivileged uid on the host, so a child that finds a setuid
  binary is confined to the identity you granted. The mount
  namespace gives the child its own filesystem view, and the pid
  namespace makes the kernel reap the child's orphans.
- **The network namespace is not used.** It would give the child
  an empty network stack, but the egress proxy lives in the
  parent's network namespace and a child in its own network
  namespace cannot reach it. So the child would be offline rather
  than fenced by the allow list, and the allow list would never
  run. The proxy is the network boundary; that is deliberate, and
  it is the same on every platform.

**What this is not.** It is not a container. There is no chroot, no
seccomp filter and no cgroup memory cap on Linux. A child in a
user namespace can still read every file you can read. For a hard
guarantee, run aios itself inside a container.

## Layout

```
aios-system/
  go.mod                    Module root
  README.md                 This file
  CHANGELOG.md              Change log
  aios-env.ps1              Windows helper: sets GOROOT, GOPATH, PATH
  internal/
    core/                   aios-core module (this is Group 1 in the build plan)
      schemas/              Typed data contracts
      audit/                Hash-chained audit log
      state/                Task state machine
      redact/               Secret redaction
      clock/                Decision clock (injected for tests/replay)
    policy/                 aios-policy module (Group 2)
    approvals/              aios-approvals module (Group 2)
    models/                 aios-models module (Group 2)
    sandbox/                aios-sandbox module (Group 2)
    persist/                aios-persist module (Group 2)
    tools/                  aios-tools module (Group 2)
    search/                 aios-search module (Group 2)
    edit/                   aios-edit module (Group 2)
    report/                 aios-report module (Group 2)
    benchmark/              aios-benchmark module (Group 2)
    roles/                  aios-roles module (Group 2)
    memory/                 aios-memory module (Group 2)
    runtime/                aios-runtime module (Group 2)
    fanout/                 aios-fanout module (Group 2)
    verify/                 aios-verify module (Group 2)
    watch/                  aios-watch module (Group 2)
  tests/                    Black-box tests
```

## Build environment

- Go 1.23 lives at `E:\sadi\GO\go\`. **Do not install Go on C:.**
- Module cache: `E:\sadi\GO\cache\`.
- GOPATH: `E:\sadi\GO\home\`.
- `aios-env.ps1` sets these in the current shell. Source it before any `go`
  command: `powershell -ExecutionPolicy Bypass -Command "& .\aios-env.ps1; go test ./..."`.
- `CGO_ENABLED=0`. The audit log uses the pure-Go `modernc.org/sqlite` driver.
- The race detector needs cgo, so it needs a C compiler. One is present:
  `C:\msys64\ucrt64\bin\gcc.exe`. To run it:
  ```powershell
  $env:CGO_ENABLED='1'
  $env:CC='C:\msys64\ucrt64\bin\gcc.exe'
  $env:PATH='C:\msys64\ucrt64\bin;' + $env:PATH
  go test -race ./...
  ```
  `CC` alone is not enough: cgo also shells out to `gcc` by name, so the
  compiler's directory has to be on `PATH`. Run this before trusting any
  concurrency change — it has already caught two real data races that the
  plain test suite did not.

## CI

`.github/workflows/ci.yml` runs on every push to `main` and every pull
request. It has three jobs:

| job | what it does |
| --- | --- |
| `test` | `go vet ./...` and `go test ./...` with `CGO_ENABLED=0`, matching how the binaries are built |
| `race` | `go test -race ./...` with `CGO_ENABLED=1` |
| `cross-compile` | builds and vets all five release targets (linux/darwin on amd64+arm64, windows on amd64) |

The release workflow runs the same checks, including the race detector,
before it packages a binary. It is not enough to have CI on pull requests:
a tag can be pushed without one ever being opened.

## Releasing

Push a plain `v` tag and the pipeline does the rest:

```
git tag v0.56.0
git push origin v0.56.0
```

That builds all five targets, stamps the version and commit into each
binary, publishes the release assets, and then pushes a **second** tag,
`aios-system/v0.56.0`.

**Why two tags.** The Go module lives in this repository's
`aios-system/` subdirectory, and the Go tooling only looks for a
module in a subdirectory under a tag whose name carries the
subdirectory. With only the plain tag, `go install` fails with
"module ... found, but does not contain package .../cmd/aios" -- it
finds the repository at the plain version tag and looks for the
package in the wrong place.

The plain tag is what the installers and the release page use. The
prefixed tag is what `go install` resolves. Both point at the same
commit.

The release job also **runs each built binary and checks what it
prints**, so a broken version stamp fails the release rather than
shipping a binary that cannot say what it is.

## Notes on the doc set

The aios-core doc set (`docs/modules/aios-core/`) was written before several
rounds of correction to the global PRD (`docs/global/PRD.md`). The
implementation follows the global PRD where the two disagree, and the
following substitutions apply:

- The doc set's hash is `SHA256(prev_hash + type + actor + payload)`. The
  global PRD §11.2 splits this into `content_hash` (pure function of inputs)
  and `chain_hash` (content + previous chain_hash + sequence_number). The
  implementation follows the global PRD.
- The doc set's IMPLEMENTATION-PLAN lists `.py` filenames. The stack is
  Go, so the source files are `.go`.
- The doc set says the schemas use `struct + go-playground/validator (or
  hand-rolled tag validation)`. The implementation hand-rolls validation in
  `internal/core/schemas/validate.go` so the module has no third-party
  dependencies. This matches REQ-CORE-NFR-002.
- The doc set has a single audit store. The global PRD §12.5.8 splits the
  store into a redacted store and an encrypted replay store. The
  redacted store is `aios-core`'s job; the replay store is `aios-persist`'s
  job. The doc set is updated to reflect this.
- The doc set does not have a clock interface. The global PRD §11.11
  splits the clock into decision and deadline clocks. The implementation
  provides a `Clock` interface (`internal/core/clock/`) and uses it in
  the audit row's `ts` field.
- The doc set does not define fail-closed behaviour. The global PRD §11.12
  is honored: if the audit DB is locked, full, or unwritable, the call
  returns an error; the caller is responsible for treating the error as
  a run block per the fail-closed contract.
- The doc set's idempotency key is caller-supplied. The global PRD §11.14
  requires the key to default to `content_hash` of the row. The
  implementation accepts an explicit key but defaults to the content hash
  when the caller passes the empty string.

These substitutions are noted in `CHANGELOG.md` under v0.9.0.
