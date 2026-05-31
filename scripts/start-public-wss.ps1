Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$Root = Split-Path -Parent $PSScriptRoot
$ToolsDir = Join-Path $Root ".tools\public-wss"
$Cloudflared = Join-Path $ToolsDir "cloudflared.exe"
$TunnelOut = Join-Path $ToolsDir "cloudflared.out.log"
$TunnelErr = Join-Path $ToolsDir "cloudflared.err.log"
$TunnelPid = Join-Path $ToolsDir "cloudflared.pid"
$PublicUrlFile = Join-Path $ToolsDir "public-wss-url.txt"
$InstalledExe = Join-Path $env:LOCALAPPDATA "Programs\voiXPe3per\voiXPe3per.exe"

New-Item -ItemType Directory -Force -Path $ToolsDir | Out-Null

if (!(Test-Path -LiteralPath $InstalledExe)) {
    throw "voiXPe3per.exe belum terinstall global. Jalankan scripts\install-global.ps1 dulu."
}

function Stop-ExistingProcessByPidFile {
    param([string]$PidFile)

    if (!(Test-Path -LiteralPath $PidFile)) {
        return
    }

    $rawPid = Get-Content -LiteralPath $PidFile -ErrorAction SilentlyContinue | Select-Object -First 1
    if ($rawPid -and ($rawPid -as [int])) {
        Stop-Process -Id ([int]$rawPid) -Force -ErrorAction SilentlyContinue
    }
    Remove-Item -LiteralPath $PidFile -Force -ErrorAction SilentlyContinue
}

function Ensure-Cloudflared {
    if (Test-Path -LiteralPath $Cloudflared) {
        return
    }

    $downloadUrl = "https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-windows-amd64.exe"
    Write-Host "Downloading cloudflared..."
    Invoke-WebRequest -Uri $downloadUrl -OutFile $Cloudflared
}

function Read-TunnelUrl {
    $content = ""
    if (Test-Path -LiteralPath $TunnelOut) {
        $content += Get-Content -Raw -LiteralPath $TunnelOut -ErrorAction SilentlyContinue
    }
    if (Test-Path -LiteralPath $TunnelErr) {
        $content += "`n" + (Get-Content -Raw -LiteralPath $TunnelErr -ErrorAction SilentlyContinue)
    }

    $match = [regex]::Match($content, "https://[a-z0-9-]+\.trycloudflare\.com")
    if ($match.Success) {
        return $match.Value
    }
    return $null
}

Ensure-Cloudflared
Stop-ExistingProcessByPidFile -PidFile $TunnelPid

Remove-Item -LiteralPath $TunnelOut, $TunnelErr -Force -ErrorAction SilentlyContinue

$tunnel = Start-Process `
    -FilePath $Cloudflared `
    -ArgumentList @("tunnel", "--url", "http://127.0.0.1:8080", "--no-autoupdate") `
    -RedirectStandardOutput $TunnelOut `
    -RedirectStandardError $TunnelErr `
    -WindowStyle Hidden `
    -PassThru
Set-Content -LiteralPath $TunnelPid -Value $tunnel.Id -Encoding ASCII

$publicHttp = $null
for ($i = 0; $i -lt 60; $i++) {
    Start-Sleep -Seconds 1
    $publicHttp = Read-TunnelUrl
    if ($publicHttp) {
        break
    }
}

if (!$publicHttp) {
    throw "Cloudflare tunnel belum memberi URL publik. Cek log: $TunnelErr"
}

$publicUri = [Uri]$publicHttp
$publicWs = "wss://$($publicUri.Host)/ws"
Set-Content -LiteralPath $PublicUrlFile -Value $publicWs -Encoding ASCII

Get-CimInstance Win32_Process |
    Where-Object { $_.ExecutablePath -eq $InstalledExe } |
    ForEach-Object { Stop-Process -Id $_.ProcessId -Force -ErrorAction SilentlyContinue }

$env:VOIXPE3PER_PAIRING_MODE = "direct"
$env:VOIXPE3PER_PUBLIC_WS_URL = $publicWs
$env:VOIXPE3PER_PAIRING_PAGE_URL = "https://voixxpe3per.vercel.app/pair"
Start-Process -FilePath $InstalledExe

Write-Host "Public WSS tunnel ready:"
Write-Host $publicWs
Write-Host "Desktop dibuka dengan mode direct public WSS. Scan QR baru dari aplikasi desktop."
