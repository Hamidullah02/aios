$env:GOROOT  = "E:\sadi\GO\go"
$env:GOPATH  = "E:\sadi\GO\home"
$env:GOCACHE = "E:\sadi\GO\cache"
$env:CGO_ENABLED = "0"
$env:PATH = "$env:GOROOT\bin;$env:PATH"

Set-Location 'E:\sadi\projects\aiagent\aios-system'

& $env:GOROOT\bin\go.exe test -count=1 -p 1 -coverprofile="$PWD\cli-cov.out" ./internal/cli 2>&1 | Out-Null
& $env:GOROOT\bin\go.exe tool cover -func="$PWD\cli-cov.out" 2>&1
