#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT/relay"

echo "Starting local relay for development only."
echo "Production/default relay uses: wss://voixpe3per-relay.onrender.com/ws"

if [[ ! -d node_modules ]]; then
  npm install
fi

npm start
