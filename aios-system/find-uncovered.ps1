$env:GOROOT  = "E:\sadi\GO\go"
$env:GOPATH  = "E:\sadi\GO\home"
$env:GOCACHE = "E:\sadi\GO\cache"
$env:CGO_ENABLED = "0"
$env:PATH = "$env:GOROOT\bin;$env:PATH"

Set-Location 'E:\sadi\projects\aiagent\aios-system'

$out = & $env:GOROOT\bin\go.exe tool cover -func=cover-tools.out 2>&1
$out | Where-Object { $_ -match 'sadi/aios-system/internal/tools/' -and $_ -notmatch '100\.0%' } | Where-Object { $_ -notmatch 'total:' }
