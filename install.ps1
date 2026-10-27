<#
.SYNOPSIS
    Installs the aios CLI on Windows.

.DESCRIPTION
    This is the Windows answer to "there is no installer". It
    downloads the release build for this machine, checks it runs,
    and puts it on the PATH.

    Run it from PowerShell:

        irm https://raw.githubusercontent.com/Hamidullah02/aios/main/install.ps1 | iex

    Or, to see what it will do before it does it:

        ./install.ps1

.PARAMETER Version
    The release tag to install. Defaults to the latest release.

.PARAMETER InstallDir
    The directory to install into. Defaults to a per-user
    directory under $env:LOCALAPPDATA, which needs no elevation.

.PARAMETER Repo
    The GitHub repo to download from. Defaults to Hamidullah02/aios.

    It is the GitHub repo, not the Go module path. The module
    lives in its aios-system/ subdirectory, so the module path
    carries both: github.com/Hamidullah02/aios/aios-system. That
    is the path `go install` needs.
#>
[CmdletBinding()]
param(
    [string]$Version = "",
    [string]$InstallDir = "",
    [string]$Repo = "Hamidullah02/aios"
)

$ErrorActionPreference = "Stop"

function Say([string]$Message) { Write-Host $Message }
function Die([string]$Message) {
    Write-Host "aios: $Message" -ForegroundColor Red
    exit 1
}

# --- pick the platform -------------------------------------------------

# The release workflow builds exactly one Windows target:
# windows/amd64. An installer that asked for an asset nobody
# builds would download a URL that 404s and fail with a download
# error that reads like a broken release rather than an
# unsupported machine.
#
# So the installer offers only what is built. A machine on any
# other architecture gets a clear message instead of a failed
# download.
$arch = $env:PROCESSOR_ARCHITECTURE
switch ($arch) {
    "AMD64" { $cpu = "amd64" }
    default {
        Die "this installer has a build for x64 Windows only (PROCESSOR_ARCHITECTURE=$arch).
Build from source instead: go install github.com/Hamidullah02/aios/aios-system/cmd/aios@latest"
    }
}

$asset = "aios_windows_$cpu.zip"

# --- resolve the version ------------------------------------------------

if (-not $Version) {
    Say "aios: finding the latest release"
    try {
        $release = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest" -Headers @{ "User-Agent" = "aios-installer" }
        $Version = $release.tag_name
    } catch {
        Die "could not reach GitHub to find the latest release: $($_.Exception.Message)"
    }
}

if (-not $Version) { Die "could not determine which version to install" }

$url = "https://github.com/$Repo/releases/download/$Version/$asset"
Say "aios: downloading $asset ($Version)"

# --- download -----------------------------------------------------------

$tmp = Join-Path ([System.IO.Path]::GetTempPath()) ("aios-install-" + [System.Guid]::NewGuid().ToString("N"))
New-Item -ItemType Directory -Force -Path $tmp | Out-Null

try {
    $zip = Join-Path $tmp $asset
    try {
        Invoke-WebRequest -Uri $url -OutFile $zip -UseBasicParsing
    } catch {
        Die "could not download $url
The release may not have a build for this CPU yet.
Build from source instead: go install github.com/Hamidullah02/aios/aios-system/cmd/aios@latest"
    }

    Expand-Archive -Path $zip -DestinationPath $tmp -Force
    $exe = Join-Path $tmp "aios.exe"
    if (-not (Test-Path $exe)) { Die "the download did not contain an 'aios.exe'" }

    # --- install --------------------------------------------------------

    if (-not $InstallDir) {
        $InstallDir = Join-Path $env:LOCALAPPDATA "aios\bin"
    }
    New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
    Copy-Item -Path $exe -Destination (Join-Path $InstallDir "aios.exe") -Force

    # --- check ----------------------------------------------------------

    $installed = Join-Path $InstallDir "aios.exe"
    try {
        & $installed --help | Out-Null
    } catch {
        Die "the installed binary did not run. Try: $installed --help"
    }

    # Report what was actually installed. An operator who needs to
    # file a bug report needs the version of the binary they are
    # looking at, not the version they thought they were running.
    $installedVersion = (& $installed --version 2>$null) -join ""
    if (-not $installedVersion) { $installedVersion = "unknown" }

    Say ""
    Say "aios is installed at $installed"
    Say "  $installedVersion"

    # The PATH is only useful if the operator's shell will see it.
    $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
    if ($userPath -notlike "*$InstallDir*") {
        [Environment]::SetEnvironmentVariable("Path", "$userPath;$InstallDir", "User")
        Say "NOTE: $InstallDir was added to your user PATH."
        Say "      Open a NEW terminal for it to take effect."
    }

    Say ""
    Say "Next:"
    Say "  1. Put your model key in a .env file next to where you will run aios:"
    Say "       Set-Content -Path .env -Value 'OPENROUTER_API_KEY=sk-or-...' -Encoding ascii"
    Say "  2. Start it:"
    Say "       aios web"
    Say ""
    Say "The UI opens in your browser. Type what you want done."
}
finally {
    Remove-Item -Recurse -Force $tmp -ErrorAction SilentlyContinue
}
