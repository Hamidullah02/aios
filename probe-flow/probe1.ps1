$ErrorActionPreference = "Continue"
$aios = "E:\sadi\projects\aiagent\aios-system\var\aios.exe"
Set-Location "E:\sadi\projects\aiagent\probe-flow"
$sock = "C:\Users\Home\AppData\Local\Temp\aios-approval.sock"

Write-Output "=== 1. aios approve <id> --socket=X  (global flag AFTER the subcommand) ==="
& $aios approve someid --socket=$sock 2>&1 | Out-String
Write-Output "exit=$LASTEXITCODE"

Write-Output ""
Write-Output "=== 2. aios web --addr --socket=X (global after subcommand) ==="
& $aios web --addr=127.0.0.1:18899 --socket=$sock 2>&1 | Out-String
Write-Output "exit=$LASTEXITCODE"

Write-Output ""
Write-Output "=== 3. aios audit --config=X (global after subcommand) ==="
& $aios audit --config=./var/config.yaml 2>&1 | Out-String
Write-Output "exit=$LASTEXITCODE"

Write-Output ""
Write-Output "=== 4. CONTROL: aios run plan.yaml --socket=X, globals BEFORE the subcommand ==="
& $aios --socket=$sock run plan.yaml 2>&1 | Out-String
Write-Output "exit=$LASTEXITCODE"
