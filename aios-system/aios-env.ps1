# aios Go environment: sets GOROOT, GOPATH, PATH for this session.
# Usage: powershell -ExecutionPolicy Bypass -File aios-env.ps1 <command>
#   e.g. powershell -ExecutionPolicy Bypass -File aios-env.ps1 go version

$env:GOROOT = 'E:\sadi\GO\go'
$env:GOPATH = 'E:\sadi\GO\home'
$env:GOMODCACHE = 'E:\sadi\GO\cache'
$env:GOCACHE = 'E:\sadi\GO\cache'
$env:GOFLAGS = '-mod=mod'
$env:Path = 'E:\sadi\GO\go\bin;E:\sadi\GO\bin;' + $env:Path

# Pure Go build for aios-core. No cgo, no toolchain surprises.
$env:CGO_ENABLED = '0'

# This script only sets env vars. Source it with `. .\aios-env.ps1` and then
# run your go command in the same shell. Example:
#   powershell -ExecutionPolicy Bypass -Command "& .\aios-env.ps1; go version"
Write-Host "aios Go env loaded."
Write-Host "  GOROOT  = $env:GOROOT"
Write-Host "  GOPATH  = $env:GOPATH"
Write-Host "  GOCACHE = $env:GOCACHE"
Write-Host "  CGO     = $env:CGO_ENABLED"
