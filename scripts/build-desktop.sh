#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT/desktop"

if ! command -v wails >/dev/null 2>&1; then
  echo "Wails CLI tidak ditemukan. Install: go install github.com/wailsapp/wails/v2/cmd/wails@latest" >&2
  exit 1
fi

wails build -platform windows/amd64 -clean
echo "EXE tersedia di: $ROOT/desktop/build/bin/voiXPe3per.exe"
