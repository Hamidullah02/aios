$env:GOROOT  = "E:\sadi\GO\go"
$env:GOPATH  = "E:\sadi\GO\home"
$env:GOCACHE = "E:\sadi\GO\cache"
$env:CGO_ENABLED = "0"
$env:PATH = "$env:GOROOT\bin;$env:PATH"

Set-Location 'E:\sadi\projects\aiagent\aios-system'

Write-Host "=== Core sub-packages ==="
foreach ($pkg in @('core/clock','core/audit','core/redact','core/schemas','core/state')) {
    $out = "$PWD\cov_$($pkg -replace '/','_').out"
    & $env:GOROOT\bin\go.exe test -count=1 -p 1 -coverprofile="$out" "./internal/$pkg" 2>&1 | Out-Null
    $total = & $env:GOROOT\bin\go.exe tool cover -func="$out" 2>&1 | Select-String -Pattern 'total:' | ForEach-Object { ($_ -replace '\s+',' ').Trim() }
    Write-Host "$pkg  $total"
}
