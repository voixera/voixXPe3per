#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
IOS_DIR="$ROOT_DIR/ios"

if ! command -v xcodegen >/dev/null 2>&1; then
  echo "xcodegen belum terinstall. Jalankan: brew install xcodegen" >&2
  exit 1
fi

cd "$IOS_DIR"
xcodegen generate
xcodebuild -scheme voiXPe3per -destination 'generic/platform=iOS' build
