Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot
$relay = Join-Path $root "relay"

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
