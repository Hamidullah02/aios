$env:GOROOT  = "E:\sadi\GO\go"
$env:GOPATH  = "E:\sadi\GO\home"
$env:GOCACHE = "E:\sadi\GO\cache"
$env:CGO_ENABLED = "0"
$env:PATH = "$env:GOROOT\bin;$env:PATH"

Set-Location 'E:\sadi\projects\aiagent\aios-system'

Write-Host "=== Per-Package Coverage ==="
Write-Host ""

$pkgs = & $env:GOROOT\bin\go.exe list ./internal/... 2>&1 | Where-Object { $_ -match '/internal/' }
foreach ($pkg in $pkgs) {
    $out = "$PWD\cov_$([System.IO.Path]::GetRandomFileName()).out"
    & $env:GOROOT\bin\go.exe test -count=1 -p 1 -coverprofile="$out" "$pkg" 2>&1 | Out-Null
    $total = & $env:GOROOT\bin\go.exe tool cover -func="$out" 2>&1 | Select-String -Pattern 'total:' | ForEach-Object { ($_ -replace '\s+',' ').Trim() }
    $short = $pkg -replace 'github.com/sadi/aios-system/', ''
    Write-Host ("{0,-32}  {1}" -f $short, $total)
    Remove-Item $out -Force -ErrorAction SilentlyContinue
}
