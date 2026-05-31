Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$Root = Split-Path -Parent $PSScriptRoot
$ToolsDir = Join-Path $Root ".tools\public-wss"
$TunnelPid = Join-Path $ToolsDir "cloudflared.pid"

if (Test-Path -LiteralPath $TunnelPid) {
    $rawPid = Get-Content -LiteralPath $TunnelPid -ErrorAction SilentlyContinue | Select-Object -First 1
    if ($rawPid -and ($rawPid -as [int])) {
        Stop-Process -Id ([int]$rawPid) -Force -ErrorAction SilentlyContinue
    }
    Remove-Item -LiteralPath $TunnelPid -Force -ErrorAction SilentlyContinue
}

Write-Host "Public WSS tunnel stopped."
