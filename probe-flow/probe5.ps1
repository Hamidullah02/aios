$ErrorActionPreference = "Continue"
# The daemon grant journey: the operator starts the approval daemon
# in one terminal, runs a plan in a second, and decides the gate
# from a third with `aios approve <id>`.
$aios = "E:\sadi\projects\aiagent\aios-system\var\aios.exe"
$root = "E:\sadi\projects\aiagent\aios-system"
$d = "E:\sadi\projects\aiagent\probe-flow\daemon-gate"
$sock = "C:\Users\Home\AppData\Local\Temp\probe-approval.sock"
$pidf = "C:\Users\Home\AppData\Local\Temp\probe-approval.pid"

Remove-Item -Recurse -Force $d -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Force -Path $d | Out-Null
$real = Get-Content "$root\.env" -Raw
[System.IO.File]::WriteAllText("$d\.env", $real, (New-Object System.Text.UTF8Encoding($false)))

$plan = @"
id: probe-1
goal: write a marker file using the shell
tasks:
  - id: t1
    description: create marker.txt with the word shellran
    acceptance: marker.txt contains shellran
    tool: run_shell
    inputs:
      - command=cmd,/c,echo,shellran>marker.txt
"@
[System.IO.File]::WriteAllText("$d\plan.yaml", $plan)
Set-Location $d

Write-Output "########## STEP 1: the daemon starts on a custom socket ##########"
# The globals go AFTER the subcommand here, exactly as the tool's
# own help text advertises: `aios daemon start|stop [--socket=path
# --pidfile=path]`.
$daemon = Start-Process -FilePath $aios `
  -ArgumentList "daemon","start","--socket=$sock","--pidfile=$pidf" `
  -WorkingDirectory $d -PassThru -RedirectStandardOutput "$d\d.out" -RedirectStandardError "$d\d.err"
$deadline = (Get-Date).AddSeconds(15)
while ((Get-Date) -lt $deadline -and -not (Test-Path $sock)) { Start-Sleep -Milliseconds 400 }
Write-Output ("  socket exists: " + (Test-Path $sock))
Write-Output ("  daemon said: " + (Get-Content "$d\d.out" -Raw -ErrorAction SilentlyContinue))
Write-Output ("  daemon err: " + (Get-Content "$d\d.err" -Raw -ErrorAction SilentlyContinue))

Write-Output ""
Write-Output "########## STEP 2: a run in another process parks an approval ##########"
$run = Start-Process -FilePath $aios `
  -ArgumentList "--socket=$sock","run","plan.yaml" `
  -WorkingDirectory $d -PassThru -RedirectStandardOutput "$d\r.out" -RedirectStandardError "$d\r.err"

# The run announces the approval id it is waiting on, so the
# operator can act on it from another terminal.
$approvalId = $null
$deadline = (Get-Date).AddSeconds(30)
while ((Get-Date) -lt $deadline -and -not $approvalId) {
  Start-Sleep -Milliseconds 500
  $err = Get-Content "$d\r.err" -Raw -ErrorAction SilentlyContinue
  if ($err -match "approval ([0-9a-f\-]{36})") { $approvalId = $matches[1] }
}
Write-Output ("  approval id: " + $approvalId)
Write-Output ("  run said so far: " + (Get-Content "$d\r.err" -Raw -ErrorAction SilentlyContinue))

Write-Output ""
Write-Output "########## STEP 3: the operator grants it from a third terminal ##########"
if ($approvalId) {
  $approve = & $aios approve $approvalId --socket=$sock 2>&1 | Out-String
  Write-Output ("  approve: " + $approve.Trim())
  Write-Output ("  exit: " + $LASTEXITCODE)
} else {
  Write-Output "  no approval id; cannot grant"
}

Write-Output ""
Write-Output "########## STEP 4: the run continues and finishes ##########"
$deadline = (Get-Date).AddSeconds(40)
while ((Get-Date) -lt $deadline) {
  if ($run.HasExited) { break }
  Start-Sleep -Milliseconds 500
}
Write-Output ("  run exited: " + $run.HasExited)
Write-Output ("  run said: " + (Get-Content "$d\r.out" -Raw -ErrorAction SilentlyContinue))
Write-Output ("  run err: " + (Get-Content "$d\r.err" -Raw -ErrorAction SilentlyContinue))
Write-Output ("  marker.txt exists: " + (Test-Path "$d\marker.txt"))
if (Test-Path "$d\marker.txt") { Write-Output ("  marker.txt: " + (Get-Content "$d\marker.txt" -Raw).Trim()) }

Write-Output ""
Write-Output "########## cleanup ##########"
if ($run -and -not $run.HasExited) { Stop-Process -Id $run.Id -Force -ErrorAction SilentlyContinue }
if ($daemon -and -not $daemon.HasExited) { Stop-Process -Id $daemon.Id -Force -ErrorAction SilentlyContinue }
Remove-Item -Force $sock,$pidf -ErrorAction SilentlyContinue
Write-Output "done"
