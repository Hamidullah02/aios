$env:GOROOT  = "E:\sadi\GO\go"
$env:GOPATH  = "E:\sadi\GO\home"
$env:GOCACHE = "E:\sadi\GO\cache"
$env:CGO_ENABLED = "0"
$env:PATH = "$env:GOROOT\bin;$env:PATH"

Set-Location 'E:\sadi\projects\aiagent\aios-system'

& go build ./internal/runtime/... ./internal/approvals/... ./internal/policy/... ./internal/models/... ./internal/core/... ./tests/... 2>&1
& go vet ./internal/runtime/... ./internal/approvals/... ./internal/policy/... ./internal/models/... ./internal/core/... ./tests/... 2>&1
& gofmt -l internal/runtime internal/approvals internal/policy internal/models internal/core tests 2>&1
& go test -count=1 -p 1 -timeout 30s ./internal/runtime/... ./internal/approvals/... ./internal/policy/... ./internal/models/... ./internal/core/... ./tests/... 2>&1
