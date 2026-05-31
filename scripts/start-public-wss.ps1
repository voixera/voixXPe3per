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

function Wait-HttpOk {
    param(
        [Parameter(Mandatory = $true)][string]$Url,
        [Parameter(Mandatory = $true)][int]$Seconds,
        [Parameter(Mandatory = $true)][string]$Name
    )

    for ($i = 0; $i -lt $Seconds; $i++) {
        try {
            $response = Invoke-WebRequest -Uri $Url -UseBasicParsing -TimeoutSec 3
            if ($response.StatusCode -ge 200 -and $response.StatusCode -lt 300) {
                return
            }
        }
        catch {
            Start-Sleep -Seconds 1
        }
    }

    throw "$Name belum sehat: $Url"
}

function Test-PublicWebSocket {
    param([Parameter(Mandatory = $true)][string]$Url)

    $client = [System.Net.WebSockets.ClientWebSocket]::new()
    $timeout = [System.Threading.CancellationTokenSource]::new([TimeSpan]::FromSeconds(20))
    try {
        $client.ConnectAsync([Uri]$Url, $timeout.Token).GetAwaiter().GetResult()
        if ($client.State -ne [System.Net.WebSockets.WebSocketState]::Open) {
            throw "state $($client.State)"
        }
        $client.CloseAsync(
            [System.Net.WebSockets.WebSocketCloseStatus]::NormalClosure,
            "probe done",
            [System.Threading.CancellationToken]::None
        ).GetAwaiter().GetResult()
    }
    finally {
        $client.Dispose()
        $timeout.Dispose()
    }
}

function Wait-PublicWebSocket {
    param(
        [Parameter(Mandatory = $true)][string]$Url,
        [Parameter(Mandatory = $true)][int]$Seconds
    )

    $lastError = $null
    for ($i = 0; $i -lt $Seconds; $i += 2) {
        try {
            Test-PublicWebSocket -Url $Url
            return
        }
        catch {
            $lastError = $_.Exception.Message
            Start-Sleep -Seconds 2
        }
    }

    throw "Public WSS belum bisa upgrade: $Url ($lastError)"
}

function Stop-InstalledApp {
    Get-CimInstance Win32_Process |
        Where-Object { $_.ExecutablePath -eq $InstalledExe } |
        ForEach-Object { Stop-Process -Id $_.ProcessId -Force -ErrorAction SilentlyContinue }
}

Ensure-Cloudflared

$lastError = $null
for ($attempt = 1; $attempt -le 3; $attempt++) {
    Stop-ExistingProcessByPidFile -PidFile $TunnelPid
    Stop-InstalledApp

    Remove-Item -LiteralPath $TunnelOut, $TunnelErr, $PublicUrlFile -Force -ErrorAction SilentlyContinue

    Write-Host "Starting public WSS tunnel attempt $attempt..."
    $tunnel = Start-Process `
        -FilePath $Cloudflared `
        -ArgumentList @("tunnel", "--url", "http://127.0.0.1:8080", "--protocol", "http2", "--no-autoupdate") `
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
        $lastError = "Cloudflare tunnel belum memberi URL publik. Cek log: $TunnelErr"
        Write-Warning $lastError
        continue
    }

    $publicUri = [Uri]$publicHttp
    $publicWs = "wss://$($publicUri.Host)/ws"
    Set-Content -LiteralPath $PublicUrlFile -Value $publicWs -Encoding ASCII

    $env:VOIXPE3PER_PAIRING_MODE = "direct"
    $env:VOIXPE3PER_PUBLIC_WS_URL = $publicWs
    $env:VOIXPE3PER_PAIRING_PAGE_URL = "https://voixxpe3per.vercel.app/pair"
    Start-Process -FilePath $InstalledExe

    try {
        Wait-HttpOk -Url "http://127.0.0.1:8080/health" -Seconds 30 -Name "Desktop local server"
        Wait-HttpOk -Url "https://$($publicUri.Host)/health" -Seconds 120 -Name "Public tunnel"
        Wait-PublicWebSocket -Url $publicWs -Seconds 120

        Write-Host "Public WSS tunnel ready:"
        Write-Host $publicWs
        Write-Host "Desktop dibuka dengan mode direct public WSS. Scan QR baru dari aplikasi desktop."
        exit 0
    }
    catch {
        $lastError = $_.Exception.Message
        Write-Warning "Attempt $attempt failed: $lastError"
    }
}

throw "Public WSS tunnel belum siap setelah 3 percobaan. Terakhir: $lastError"
