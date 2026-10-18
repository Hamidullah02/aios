$env:GOROOT  = "E:\sadi\GO\go"
$env:GOPATH  = "E:\sadi\GO\home"
$env:GOCACHE = "E:\sadi\GO\cache"
$env:CGO_ENABLED = "0"
$env:PATH = "$env:GOROOT\bin;$env:PATH"

Set-Location 'E:\sadi\projects\aiagent\aios-system'

Write-Host "=== go build ===" -ForegroundColor Cyan
go build ./... 2>&1
Write-Host "=== go vet ===" -ForegroundColor Cyan
go vet ./... 2>&1
Write-Host "=== gofmt -l . ===" -ForegroundColor Cyan
gofmt -l . 2>&1
Write-Host "=== go test ===" -ForegroundColor Cyan
go test -count=1 -p 1 -timeout 30s ./... 2>&1
Write-Host "=== Done ===" -ForegroundColor Cyan
