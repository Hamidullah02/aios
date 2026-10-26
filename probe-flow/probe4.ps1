$ErrorActionPreference = "Continue"
# A real operator drives the approval daemon from a SECOND terminal.
# The daemon is started with the globals BEFORE the subcommand (the
# only form the parser currently accepts), then a run parks an
# approval with it, and `aios approve <id>` grants it from here.
$aios = "E:\sadi\projects\aiagent\aios-system\var\aios.exe"
$root = "E:\sadi\projects\aiagent\aios-system"
$d = "E:\sadi\projects\aiagent\probe-flow\daemon-gate"
$sock = "C:\Users\Home\AppData\Local\Temp\probe-approval.sock"
$pidf = "C:\Users\Home\AppData\Local\Temp\probe-approval.pid"

Remove-Item -Recurse -Force $d -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Force -Path $d | Out-Null
$real = Get-Content "$root\.env" -Raw
[System.IO.File]::WriteAllText("$d\.env", $real, (New-Object System.Text.UTF8Encoding($false)))
Set-Location $d

Write-Output "########## STEP 1: the daemon starts on a custom socket ##########"
$daemon = Start-Process -FilePath $aios `
  -ArgumentList "--socket=$sock","--pidfile=$pidf","daemon","start" `
  -WorkingDirectory $d -PassThru -RedirectStandardOutput "$d\d.out" -RedirectStandardError "$d\d.err"
$deadline = (Get-Date).AddSeconds(15)
while ((Get-Date) -lt $deadline -and -not (Test-Path $sock)) { Start-Sleep -Milliseconds 400 }
Write-Output ("  socket exists: " + (Test-Path $sock))
Write-Output ("  daemon said: " + (Get-Content "$d\d.out" -Raw -ErrorAction SilentlyContinue))

Write-Output ""
Write-Output "########## STEP 2: a run raises an approval through the daemon ##########"
# The run is in its own process, attached to the daemon's socket.
$run = Start-Process -FilePath $aios `
  -ArgumentList "--socket=$sock","run","plan.yaml" `
  -WorkingDirectory $d -PassThru -RedirectStandardOutput "$d\r.out" -RedirectStandardError "$d\r.err"
Start-Sleep -Seconds 12
Write-Output ("  run said: " + (Get-Content "$d\r.out" -Raw -ErrorAction SilentlyContinue))
Write-Output ("  run err: " + (Get-Content "$d\r.err" -Raw -ErrorAction SilentlyContinue))

Write-Output ""
Write-Output "########## STEP 3: the operator lists the pending queue ##########"
# The daemon's own `list` verb is what the UI polls. A script can
# speak it: one JSON object per line over the socket.
Write-Output "(the queue is read through the socket, not a CLI command)"

Write-Output ""
Write-Output "########## STEP 4: cleanup ##########"
if ($run -and -not $run.HasExited) { Stop-Process -Id $run.Id -Force -ErrorAction SilentlyContinue }
if ($daemon -and -not $daemon.HasExited) { Stop-Process -Id $daemon.Id -Force -ErrorAction SilentlyContinue }
Remove-Item -Force $sock,$pidf -ErrorAction SilentlyContinue
Write-Output "done"
