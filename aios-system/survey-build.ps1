$env:GOROOT  = "E:\sadi\GO\go"
$env:GOPATH  = "E:\sadi\GO\home"
$env:GOCACHE = "E:\sadi\GO\cache"
$env:CGO_ENABLED = "0"
$env:PATH = "$env:GOROOT\bin;$env:PATH"

Set-Location 'E:\sadi\projects\aiagent\aios-system'

Write-Host '=== build + vet ==='
& go build ./... 2>&1
& go vet ./... 2>&1
Write-Host '=== gofmt -l . ==='
& gofmt -l . 2>&1
