$env:GOROOT  = "E:\sadi\GO\go"
$env:GOPATH  = "E:\sadi\GO\home"
$env:GOCACHE = "E:\sadi\GO\cache"
$env:CGO_ENABLED = "0"
$env:PATH = "$env:GOROOT\bin;$env:PATH"

Set-Location 'E:\sadi\projects\aiagent\aios-system'

$out = & $env:GOROOT\bin\go.exe test -v -count=1 -timeout 120s ./... 2>&1 | Out-String
$lines = $out -split "`n"
$pass = ($lines | Where-Object { $_ -match '^--- PASS:' }).Count
$fail = ($lines | Where-Object { $_ -match '^--- FAIL:' }).Count
$skip = ($lines | Where-Object { $_ -match '^--- SKIP:' }).Count
Write-Host "PASS: $pass"
Write-Host "FAIL: $fail"
Write-Host "SKIP: $skip"
if ($fail -gt 0) {
    $lines | Where-Object { $_ -match 'FAIL|^---|panic|^    ' } | Select-Object -First 30 | ForEach-Object { Write-Host $_ }
}
