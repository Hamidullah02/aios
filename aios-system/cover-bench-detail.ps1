$env:GOROOT  = "E:\sadi\GO\go"
$env:GOPATH  = "E:\sadi\GO\home"
$env:GOCACHE = "E:\sadi\GO\cache"
$env:CGO_ENABLED = "0"
$env:PATH = "$env:GOROOT\bin;$env:PATH"

Set-Location 'E:\sadi\projects\aiagent\aios-system'

& $env:GOROOT\bin\go.exe test -count=1 -p 1 -coverprofile="$PWD\cover-bench.out" ./internal/benchmark/... 2>&1
Write-Host '--- per-file ---'
& $env:GOROOT\bin\go.exe tool cover -func="$PWD\cover-bench.out" 2>&1
