$env:GOROOT  = "E:\sadi\GO\go"
$env:GOPATH  = "E:\sadi\GO\home"
$env:GOCACHE = "E:\sadi\GO\cache"
$env:CGO_ENABLED = "0"
$env:PATH = "$env:GOROOT\bin;$env:PATH"

Set-Location 'E:\sadi\projects\aiagent\aios-system'

& go build ./internal/models/... ./internal/approvals/... ./internal/policy/... ./internal/core/... ./tests/...
& go vet ./internal/models/... ./internal/approvals/... ./internal/policy/... ./internal/core/... ./tests/...
& gofmt -l internal/models internal/approvals internal/policy internal/core tests
& go test -count=1 -p 1 -timeout 30s ./internal/models/... ./internal/approvals/... ./internal/policy/... ./internal/core/... ./tests/...
