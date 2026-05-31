Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot
$relay = Join-Path $root "relay"

Write-Host "Starting local relay for development only."
Write-Host "Production/default relay uses: wss://voixpe3per-relay.onrender.com/ws"

Push-Location $relay
try {
    if (!(Test-Path -LiteralPath "node_modules")) {
        npm install
    }
    npm start
}
finally {
    Pop-Location
}
