Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$Root = Split-Path -Parent $PSScriptRoot
$AndroidDir = Join-Path $Root "android"
$Apk = Join-Path $AndroidDir "app\build\outputs\apk\debug\app-debug.apk"

function Use-JavaIfNeeded {
    $javaHomeExe = if ($env:JAVA_HOME) { Join-Path $env:JAVA_HOME "bin\java.exe" } else { $null }
    if ($javaHomeExe -and (Test-Path -LiteralPath $javaHomeExe)) {
        return
    }

    if (Get-Command java -ErrorAction SilentlyContinue) {
        return
    }

    $candidates = @(
        (Join-Path $env:ProgramFiles "Android\Android Studio\jbr"),
        (Join-Path ${env:ProgramFiles(x86)} "Android\Android Studio\jbr")
    )

    foreach ($candidate in $candidates) {
        if ($candidate -and (Test-Path -LiteralPath (Join-Path $candidate "bin\java.exe"))) {
            $env:JAVA_HOME = $candidate
            $env:PATH = "$(Join-Path $candidate "bin");$env:PATH"
            return
        }
    }
}

Use-JavaIfNeeded

Push-Location $AndroidDir
try {
    if (Test-Path -LiteralPath ".\gradlew.bat") {
        & .\gradlew.bat assembleDebug
    }
    elseif (Get-Command gradle -ErrorAction SilentlyContinue) {
        gradle assembleDebug
    }
    else {
        throw "Gradle tidak ditemukan. Install Android Studio, lalu buka folder android/ dan build Debug APK."
    }
}
finally {
    Pop-Location
}

if (!(Test-Path -LiteralPath $Apk)) {
    throw "Build selesai tapi APK tidak ditemukan: $Apk"
}

Write-Host "APK Android siap:"
Write-Host $Apk
