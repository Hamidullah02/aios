$env:GOROOT  = "E:\sadi\GO\go"
$env:GOPATH  = "E:\sadi\GO\home"
$env:GOCACHE = "E:\sadi\GO\cache"
$env:CGO_ENABLED = "0"
$env:PATH = "$env:GOROOT\bin;$env:PATH"

Set-Location 'E:\sadi\projects\aiagent\aios-system'

$pkgs = @('internal\core\audit', 'internal\core\redact', 'internal\core\schemas', 'internal\core\state')
foreach ($p in $pkgs) {
  Write-Host "=== $p ==="
  & $env:GOROOT\bin\go.exe test -count=1 -p 1 -coverprofile="$PWD\c.out" "./$p" 2>&1 | Out-Null
  & $env:GOROOT\bin\go.exe tool cover -func="$PWD\c.out" 2>&1
  Remove-Item "$PWD\c.out" -Force
}
