$ErrorActionPreference = "Continue"
$aios = "E:\sadi\projects\aiagent\aios-system\var\aios.exe"
Set-Location "E:\sadi\projects\aiagent\probe-flow"

Write-Output "=== A. `aios --yes run plan.yaml` (global --yes BEFORE the subcommand) ==="
Remove-Item -Force marker.txt -ErrorAction SilentlyContinue
& $aios --yes run plan.yaml 2>&1 | Out-String
Write-Output "exit=$LASTEXITCODE"
Write-Output ("marker.txt exists: " + (Test-Path marker.txt))
if (Test-Path marker.txt) { Write-Output ("marker.txt: " + (Get-Content marker.txt -Raw).Trim()) }
Remove-Item -Force marker.txt -ErrorAction SilentlyContinue

Write-Output ""
Write-Output "=== B. `aios run plan.yaml --yes` (subcommand --yes AFTER the positional) ==="
& $aios run plan.yaml --yes 2>&1 | Out-String
Write-Output "exit=$LASTEXITCODE"
Write-Output ("marker.txt exists: " + (Test-Path marker.txt))
if (Test-Path marker.txt) { Write-Output ("marker.txt: " + (Get-Content marker.txt -Raw).Trim()) }
