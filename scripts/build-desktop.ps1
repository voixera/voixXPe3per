param(
    [switch]$InstallGlobal
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot
$desktop = Join-Path $root "desktop"

Push-Location $desktop
try {
    if (-not (Get-Command wails -ErrorAction SilentlyContinue)) {
        throw "Wails CLI tidak ditemukan. Install: go install github.com/wailsapp/wails/v2/cmd/wails@latest"
    }

    wails build -platform windows/amd64 -clean
    Write-Host "EXE tersedia di: $desktop\build\bin\voiXPe3per.exe"

    if ($InstallGlobal) {
        & (Join-Path $root "scripts\install-global.ps1")
    }
}
finally {
    Pop-Location
}
