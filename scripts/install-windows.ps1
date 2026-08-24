param(
    [string]$InstallPath = (Join-Path $env:LOCALAPPDATA "voiXPe3per\source")
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

function Update-ProcessPath {
    $machinePath = [Environment]::GetEnvironmentVariable("Path", "Machine")
    $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
    $env:Path = "$machinePath;$userPath"
}

function Install-WingetPackage {
    param([Parameter(Mandatory = $true)][string]$Id)

    & winget install --id $Id --exact --silent --accept-package-agreements --accept-source-agreements
    if ($LASTEXITCODE -ne 0) {
        throw "Gagal menginstall $Id dengan winget. Install manual lalu jalankan script ini lagi."
    }
    Update-ProcessPath
}

if (-not (Get-Command winget -ErrorAction SilentlyContinue)) {
    throw "winget tidak ditemukan. Install App Installer dari Microsoft Store, lalu jalankan script ini lagi."
}

if (-not (Get-Command git -ErrorAction SilentlyContinue)) {
    Install-WingetPackage "Git.Git"
}
if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    Install-WingetPackage "GoLang.Go"
}
if (-not (Get-Command node -ErrorAction SilentlyContinue)) {
    Install-WingetPackage "OpenJS.NodeJS.LTS"
}
if (-not (Get-Command gcc -ErrorAction SilentlyContinue)) {
    Install-WingetPackage "BrechtSanders.WinLibs.POSIX.UCRT"
}

$wails = Join-Path $env:USERPROFILE "go\bin\wails.exe"
if (-not (Test-Path -LiteralPath $wails)) {
    & go install github.com/wailsapp/wails/v2/cmd/wails@latest
    if ($LASTEXITCODE -ne 0) {
        throw "Gagal menginstall Wails CLI."
    }
}
$env:Path = "$(Split-Path -Parent $wails);$env:Path"

if (Test-Path -LiteralPath $InstallPath) {
    & git -C $InstallPath pull --ff-only
} else {
    New-Item -ItemType Directory -Force -Path (Split-Path -Parent $InstallPath) | Out-Null
    & git clone https://github.com/voixera/voixXPe3per.git $InstallPath
}
if ($LASTEXITCODE -ne 0) {
    throw "Gagal mengambil source voiXPe3per."
}

& (Join-Path $InstallPath "scripts\build-desktop.ps1") -InstallGlobal
Write-Host "Selesai. Buka terminal baru, lalu jalankan: voixpe3per"
