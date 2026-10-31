<#
.SYNOPSIS
    Builds aios from this source tree and installs it, locally.

.DESCRIPTION
    This is the install path for a machine that has the source and
    nothing else. It builds the Go module in ./aios-system, checks
    the binary runs, and puts it on the PATH.

    It is the local counterpart to install.ps1. That one downloads a
    prebuilt binary from a GitHub release, which is right for a
    machine that has no source and no Go toolchain. This one needs
    neither the network nor a release -- it builds what is in front
    of it.

    The two differ in one more way that matters: this script stamps
    the version from the tree it built, so `aios --version` names the
    commit the binary came from. A binary that cannot say what it
    is makes every bug report start with "which build is this?".

.PARAMETER InstallDir
    Where to put the binary. Defaults to a per-user directory under
    $env:LOCALAPPDATA, which needs no elevation.

.PARAMETER Version
    The version to stamp into the binary. Defaults to the tag the
    tree is on, or "dev" when the tree is not on a tag.

.PARAMETER NoPathUpdate
    Install the binary without adding it to the user PATH.

    The PATH update is a per-user registry write, and it changes the
    environment of every process the user starts afterwards, on this
    machine, for good. That is what an installer is for, and it is
    why the automated tests must not let it happen.

.PARAMETER SkipTests
    Build and install without running the test suite first.

    The default is to run it: a binary that fails its own tests is
    not one to put on the PATH. This flag exists for the case where
    the suite was just run and re-running it would only cost time.
#>
[CmdletBinding()]
param(
    [string]$InstallDir = "",
    [string]$Version = "",
    [switch]$NoPathUpdate,
    [switch]$SkipTests
)

$ErrorActionPreference = "Stop"

function Say([string]$Message) { Write-Host $Message }
function Die([string]$Message) {
    Write-Host "aios: $Message" -ForegroundColor Red
    exit 1
}

# --- find the source -----------------------------------------------------

$moduleDir = Join-Path $PSScriptRoot "aios-system"
if (-not (Test-Path (Join-Path $moduleDir "go.mod"))) {
    Die "cannot find the Go module at $moduleDir
This script must sit next to the aios-system directory."
}

# --- find Go -------------------------------------------------------------

# The Go toolchain is not on the PATH on every machine. The README
# says where it lives on this one, so it is checked before giving
# up -- a "go is not recognised" message would be a poor reason to
# fail when the toolchain is right there.
$go = Get-Command go -ErrorAction SilentlyContinue
if (-not $go) {
    foreach ($candidate in @(
        "E:\sadi\GO\go\bin\go.exe",
        "$env:ProgramFiles\Go\bin\go.exe",
        "$env:LOCALAPPDATA\Programs\Go\bin\go.exe"
    )) {
        if (Test-Path $candidate) {
            $go = $candidate
            break
        }
    }
}
if (-not $go) {
    Die "Go is not on the PATH and was not found in the usual places.
Install Go 1.23 or later, or put it on the PATH:
  https://go.dev/dl/"
}
Say "using Go: $go"

# --- work out the version ------------------------------------------------

Push-Location $PSScriptRoot
$commit = ""
try {
    $commit = (& git rev-parse --short HEAD 2>$null | Out-String).Trim()
} catch {
    # Not a git checkout. The build is still fine; it just cannot
    # name a commit.
}
if (-not $Version) {
    $Version = "dev"
    try {
        $tag = (& git describe --tags --exact-match 2>$null | Out-String).Trim()
        if ($tag) { $Version = $tag }
    } catch { }
}
Pop-Location

Say "aios: building from $moduleDir (version $Version)"

# --- build ---------------------------------------------------------------

$env:CGO_ENABLED = "0"
$build = Join-Path $env:TEMP ("aios-build-" + [System.Guid]::NewGuid().ToString("N") + ".exe")
try {
    Push-Location $moduleDir
    & $go build -trimpath `
        -ldflags "-s -w -X github.com/Hamidullah02/aios/aios-system/internal/version.Version=$Version -X github.com/Hamidullah02/aios/aios-system/internal/version.Commit=$commit" `
        -o $build ./cmd/aios
    if ($LASTEXITCODE -ne 0) { Die "the build failed" }
    Pop-Location
} finally {
    Pop-Location
}

# --- test ----------------------------------------------------------------

if (-not $SkipTests) {
    Say "aios: running the test suite"
    Push-Location $moduleDir
    & $go test ./... -count=1
    $testExit = $LASTEXITCODE
    Pop-Location
    if ($testExit -ne 0) { Die "the test suite failed; nothing was installed" }
}

# --- check ---------------------------------------------------------------

# The check is on the exit code, not on the absence of output.
#
# Windows PowerShell raises a terminating error for a native command
# that writes to stderr when $ErrorActionPreference is "Stop", and it
# sets $LASTEXITCODE to -1 rather than the real exit code -- so a
# check that judged the binary by its output would reject a binary
# that is working perfectly. Start-Process reports the true exit
# code.
$probeOut = Join-Path $env:TEMP ("aios-probe-" + [System.Guid]::NewGuid().ToString("N") + ".out")
$probeErr = Join-Path $env:TEMP ("aios-probe-" + [System.Guid]::NewGuid().ToString("N") + ".err")
try {
    $proc = Start-Process -FilePath $build -ArgumentList "--help" `
        -RedirectStandardOutput $probeOut -RedirectStandardError $probeErr `
        -NoNewWindow -Wait -PassThru
    $probeExit = $proc.ExitCode
} catch {
    $probeExit = 1
    Set-Content -LiteralPath $probeErr -Value $_.Exception.Message
}
if ($probeExit -ne 0) {
    $why = (Get-Content -LiteralPath $probeErr -Raw -ErrorAction SilentlyContinue)
    if (-not $why) { $why = "(no error output)" }
    Die "the built binary did not run (exit code $probeExit).
$why"
}

# --- install -------------------------------------------------------------

if (-not $InstallDir) {
    $InstallDir = Join-Path $env:LOCALAPPDATA "aios\bin"
}
New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
$installed = Join-Path $InstallDir "aios.exe"
Copy-Item -Path $build -Destination $installed -Force

$verOut = Join-Path $env:TEMP ("aios-ver-" + [System.Guid]::NewGuid().ToString("N") + ".out")
try {
    $verProc = Start-Process -FilePath $installed -ArgumentList "--version" `
        -RedirectStandardOutput $verOut -RedirectStandardError "$verOut.err" `
        -NoNewWindow -Wait -PassThru
    $installedVersion = (Get-Content -LiteralPath $verOut -Raw -ErrorAction SilentlyContinue)
    if ($verProc.ExitCode -ne 0 -or -not $installedVersion) { $installedVersion = "unknown" }
} catch {
    $installedVersion = "unknown"
}
$installedVersion = ($installedVersion -replace "\s+", " ").Trim()

Say ""
Say "aios is installed at $installed"
Say "  $installedVersion"

# --- the PATH ------------------------------------------------------------

if ($NoPathUpdate) {
    Say ""
    Say "NOTE: $InstallDir was NOT added to your PATH (-NoPathUpdate)."
    Say "      Add it yourself if you want `aios` on your PATH:"
    Say "      [Environment]::SetEnvironmentVariable('Path', [Environment]::GetEnvironmentVariable('Path','User') + ';$InstallDir', 'User')"
} else {
    $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
    if ($userPath -notlike "*$InstallDir*") {
        [Environment]::SetEnvironmentVariable("Path", "$userPath;$InstallDir", "User")
        Say "NOTE: $InstallDir was added to your user PATH."
        Say "      Open a NEW terminal for it to take effect."
    }
}

Say ""
Say "Next:"
Say "  1. Put your model key in a .env file next to where you will run aios:"
Say "       Set-Content -Path .env -Value 'OPENROUTER_API_KEY=sk-or-...' -Encoding ascii"
Say "  2. Start it:"
Say "       aios web"
Say ""
Say "The UI opens in your browser. Type what you want done."
