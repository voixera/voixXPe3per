Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$InstallDir = Join-Path $env:LOCALAPPDATA "Programs\voiXPe3per"
$WindowsApps = Join-Path $env:LOCALAPPDATA "Microsoft\WindowsApps"
$appPathKey = "HKCU:\Software\Microsoft\Windows\CurrentVersion\App Paths\voiXPe3per.exe"

Remove-Item -LiteralPath (Join-Path $WindowsApps "voixpe3per.cmd") -Force -ErrorAction SilentlyContinue
Remove-Item -LiteralPath (Join-Path $WindowsApps "voiXPe3per.cmd") -Force -ErrorAction SilentlyContinue
Remove-Item -Path $appPathKey -Recurse -Force -ErrorAction SilentlyContinue
Remove-Item -LiteralPath $InstallDir -Recurse -Force -ErrorAction SilentlyContinue

Write-Host "voiXPe3per global install removed."
