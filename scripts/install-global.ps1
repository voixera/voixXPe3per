Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$ProjectRoot = Split-Path -Parent $PSScriptRoot
$BuiltExe = Join-Path $ProjectRoot "desktop\build\bin\voiXPe3per.exe"
$InstallDir = Join-Path $env:LOCALAPPDATA "Programs\voiXPe3per"
$InstalledExe = Join-Path $InstallDir "voiXPe3per.exe"
$WindowsApps = Join-Path $env:LOCALAPPDATA "Microsoft\WindowsApps"

if (Test-Path -LiteralPath $BuiltExe) {
    $SourceExe = $BuiltExe
} elseif (Test-Path -LiteralPath $InstalledExe) {
    $SourceExe = $InstalledExe
} else {
    throw "EXE belum ada. Jalankan build desktop dulu, lalu ulangi install-global.ps1."
}

New-Item -ItemType Directory -Force -Path $InstallDir, $WindowsApps | Out-Null
if ((Resolve-Path -LiteralPath $SourceExe).Path -ne (Resolve-Path -LiteralPath $InstalledExe -ErrorAction SilentlyContinue).Path) {
    Get-Process -Name "voiXPe3per" -ErrorAction SilentlyContinue |
        Where-Object { $_.Path -eq $InstalledExe } |
        Stop-Process -Force
    Copy-Item -LiteralPath $SourceExe -Destination $InstalledExe -Force
}

$cmd = "@echo off`r`nstart """" ""$InstalledExe"" %*`r`n"
Set-Content -LiteralPath (Join-Path $WindowsApps "voixpe3per.cmd") -Value $cmd -Encoding ASCII
Set-Content -LiteralPath (Join-Path $WindowsApps "voiXPe3per.cmd") -Value $cmd -Encoding ASCII

function Add-UserPathEntry {
    param([Parameter(Mandatory = $true)][string]$PathEntry)

    $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
    $pathParts = @()
    if ($userPath) {
        $pathParts = $userPath -split ";" | Where-Object { $_ }
    }

    if ($pathParts -notcontains $PathEntry) {
        $newPath = (@($pathParts) + $PathEntry) -join ";"
        [Environment]::SetEnvironmentVariable("Path", $newPath, "User")
        $env:Path = $env:Path + ";" + $PathEntry
    }
}

Add-UserPathEntry -PathEntry $InstallDir
Add-UserPathEntry -PathEntry $WindowsApps

$appPathKey = "HKCU:\Software\Microsoft\Windows\CurrentVersion\App Paths\voiXPe3per.exe"
New-Item -Path $appPathKey -Force | Out-Null
Set-ItemProperty -Path $appPathKey -Name "(default)" -Value $InstalledExe
Set-ItemProperty -Path $appPathKey -Name "Path" -Value $InstallDir

Write-Host "voiXPe3per installed globally:"
Write-Host $InstalledExe
Write-Host "Run from any new terminal with: voiXPe3per.exe"
Write-Host "Alias also available: voixpe3per"
