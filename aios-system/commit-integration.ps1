$env:GOROOT  = "E:\sadi\GO\go"
$env:GOPATH  = "E:\sadi\GO\home"
$env:GOCACHE = "E:\sadi\GO\cache"
$env:CGO_ENABLED = "0"
$env:PATH = "$env:GOROOT\bin;$env:PATH"

Set-Location 'E:\sadi\projects\aiagent\aios-system'

& git add -A
& git status | Out-String | Select-Object -First 30
Write-Host ""
Write-Host "=== commit ==="
& git commit -m "v0.21.0: end-to-end wire audit + BusWorker + BuildDeps hardening

- New internal/cli/bus_worker.go wires runtime -> tools.Bus
  with 5 end-to-end tests covering allow/deny/ask policy
  decisions, real shell subprocess through sandbox, and
  ask_user through the bus adapter.
- cli.BuildDeps no longer silently swallows audit.Open
  and policy.Load errors; both surface as *CliError
  with the binary exiting 2. Audit is required for
  run/resume/cancel/report/audit; init/daemon stay
  audit-free.
- cli parser --config default is now empty (policy
  optional per doc set); smoke tests updated.
- TestCoverage_BuildDeps_PolicyError was asserting
  the wrong contract (expected no error on bad YAML);
  fixed. New TestCoverage_BuildDeps_AuditError
  asserts the new audit-failure contract.
- var/audit-integration.md documents the full
  cross-module wire audit (7 findings, all fixed).
- Counts: 1090 PASS, 0 FAIL, 1 SKIP (Windows symlink).
  Integration x10: 80/80 PASS." 2>&1 | Out-String
Write-Host ""
Write-Host "=== push ==="
& git push origin main 2>&1 | Out-String
