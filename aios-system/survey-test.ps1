$env:GOROOT  = "E:\sadi\GO\go"
$env:GOPATH  = "E:\sadi\GO\home"
$env:GOCACHE = "E:\sadi\GO\cache"
$env:CGO_ENABLED = "0"
$env:PATH = "$env:GOROOT\bin;$env:PATH"

Set-Location 'E:\sadi\projects\aiagent\aios-system'

# Run tests first
$testOut = & $env:GOROOT\bin\go.exe test -p 1 -count=1 -timeout 60s ./... 2>&1
$testOut | ForEach-Object { Write-Host $_ }

Write-Host '=== coverage ==='
& $env:GOROOT\bin\go.exe test -p 1 -count=1 -timeout 60s -coverprofile="$PWD\cover-all.out" ./... 2>&1 | Where-Object { $_ -match 'coverage|ok |FAIL' }
Write-Host '--- per-package coverage ---'
$packages = @(
  'internal\approvals',
  'internal\ask',
  'internal\benchmark',
  'internal\cli',
  'internal\core\audit',
  'internal\core\redact',
  'internal\core\schemas',
  'internal\core\state',
  'internal\egress',
  'internal\models',
  'internal\persist',
  'internal\planner',
  'internal\policy',
  'internal\report',
  'internal\runtime',
  'internal\sandbox',
  'internal\tools',
  'tests'
)
foreach ($p in $packages) {
  & $env:GOROOT\bin\go.exe test -count=1 -p 1 -coverprofile="$PWD\cover-$($p.Replace('\','-')).out" "./$p" 2>&1 | Where-Object { $_ -match 'coverage|FAIL|no test' }
}
