$env:GOROOT  = "E:\sadi\GO\go"
$env:GOPATH  = "E:\sadi\GO\home"
$env:GOCACHE = "E:\sadi\GO\cache"
$env:CGO_ENABLED = "0"
$env:PATH = "$env:GOROOT\bin;$env:PATH"

Set-Location 'E:\sadi\projects\aiagent\aios-system'

Write-Host "=== E2E Verification ==="
Write-Host ""

# 1. go build
Write-Host "--- 1. go build ---"
& $env:GOROOT\bin\go.exe build ./... 2>&1
if ($LASTEXITCODE -ne 0) { Write-Host "FAIL"; exit 1 }
Write-Host "PASS"
Write-Host ""

# 2. go vet
Write-Host "--- 2. go vet ---"
& $env:GOROOT\bin\go.exe vet ./... 2>&1
if ($LASTEXITCODE -ne 0) { Write-Host "FAIL"; exit 1 }
Write-Host "PASS"
Write-Host ""

# 3. gofmt
Write-Host "--- 3. gofmt -l . ---"
$fmt = & $env:GOROOT\bin\go.exe fmt -l ./... 2>&1
if ($fmt) {
    Write-Host "DRIFT:"
    $fmt | ForEach-Object { Write-Host "  $_" }
} else {
    Write-Host "PASS"
}
Write-Host ""

# 4. go test
Write-Host "--- 4. go test (all packages) ---"
$failed = $false
$results = @()
& $env:GOROOT\bin\go.exe test -count=1 -p 1 -timeout 60s ./... 2>&1 | ForEach-Object {
    $line = $_
    if ($line -match '^(ok|FAIL|\?)\s+(\S+)\s+') {
        $results += $Matches[1] + "  " + $Matches[2]
        if ($Matches[1] -eq 'FAIL') { $failed = $true }
    }
}
$results | ForEach-Object { Write-Host "  $_" }
if ($failed) { Write-Host "FAIL"; exit 1 } else { Write-Host "PASS" }
Write-Host ""

# 5. integration tests
Write-Host "--- 5. integration tests (./tests) ---"
& $env:GOROOT\bin\go.exe test -count=1 -p 1 -timeout 60s ./tests 2>&1
if ($LASTEXITCODE -ne 0) { Write-Host "FAIL"; exit 1 } else { Write-Host "PASS" }
Write-Host ""

Write-Host "=== E2E Verification: PASS ==="
