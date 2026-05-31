#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT/android"

if [[ -x ./gradlew ]]; then
  ./gradlew assembleDebug
else
  gradle assembleDebug
fi

echo "APK debug tersedia di: $ROOT/android/app/build/outputs/apk/debug/app-debug.apk"
