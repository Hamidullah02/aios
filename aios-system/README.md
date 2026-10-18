# aios-system

The aios system. Foundation module is `internal/core/`.

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
    report/                 aios-report module (Group 2)
    benchmark/              aios-benchmark module (Group 2)
    memory/                 aios-memory module (Group 2)
    runtime/                aios-runtime module (Group 2)
    fanout/                 aios-fanout module (Group 2)
  tests/                    Black-box tests
```

## Build environment

- Go 1.23 lives at `E:\sadi\GO\go\`. **Do not install Go on C:.**
- Module cache: `E:\sadi\GO\cache\`.
- GOPATH: `E:\sadi\GO\home\`.
- `aios-env.ps1` sets these in the current shell. Source it before any `go`
  command: `powershell -ExecutionPolicy Bypass -Command "& .\aios-env.ps1; go test ./..."`.
- `CGO_ENABLED=0`. The audit log uses the pure-Go `modernc.org/sqlite` driver.

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
