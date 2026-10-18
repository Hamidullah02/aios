$env:GOROOT  = "E:\sadi\GO\go"
$env:GOPATH  = "E:\sadi\GO\home"
$env:GOCACHE = "E:\sadi\GO\cache"
$env:CGO_ENABLED = "0"
$env:PATH = "$env:GOROOT\bin;$env:PATH"

Set-Location 'E:\sadi\projects\aiagent\aios-system'

& $env:GOROOT\bin\go.exe test -count=1 -p 1 -coverprofile="$PWD\schemas.out" ./internal/core/schemas 2>&1 | Out-Null
& $env:GOROOT\bin\go.exe tool cover -func="$PWD\schemas.out" 2>&1 | Select-String -Pattern 'total:'
