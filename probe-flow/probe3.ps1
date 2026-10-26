$ErrorActionPreference = "Continue"
$aios = "E:\sadi\projects\aiagent\aios-system\var\aios.exe"
Set-Location "E:\sadi\projects\aiagent\probe-flow"

Write-Output "=== `aios daemon start --socket=X --pidfile=Y` (exactly as subUsage documents it) ==="
& $aios daemon start --socket="C:\Users\Home\AppData\Local\Temp\probe.sock" --pidfile="C:\Users\Home\AppData\Local\Temp\probe.pid" 2>&1 | Out-String
Write-Output "exit=$LASTEXITCODE"

Write-Output ""
Write-Output "=== `aios approve <id> --socket=X` with a REAL daemon on a custom socket ==="
Write-Output "(the approve path is documented as 'over the daemon socket')"
