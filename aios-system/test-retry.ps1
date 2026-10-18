$env:GOROOT='E:\sadi\GO\go'
$env:GOPATH='E:\sadi\GO\home'
$env:GOCACHE='E:\sadi\GO\cache'
$env:CGO_ENABLED='0'
$env:PATH="$env:GOROOT\bin;$env:PATH"
Set-Location 'E:\sadi\projects\aiagent\aios-system'
go test -count=1 -v ./internal/retry/...
