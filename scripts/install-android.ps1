Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$Root = Split-Path -Parent $PSScriptRoot
$Apk = Join-Path $Root "android\app\build\outputs\apk\debug\app-debug.apk"

if (!(Test-Path -LiteralPath $Apk)) {
    & (Join-Path $PSScriptRoot "build-android.ps1")
}

if (!(Get-Command adb -ErrorAction SilentlyContinue)) {
    throw "adb tidak ditemukan. Install Android Studio Platform Tools atau install APK manual dari: $Apk"
}

adb install -r $Apk
Write-Host "APK terinstall ke device Android yang tersambung:"
Write-Host $Apk
