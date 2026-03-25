#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
cd "$ROOT_DIR"

if [[ -x "$ROOT_DIR/.tools/go/bin/go" ]]; then
  GO_BIN="$ROOT_DIR/.tools/go/bin/go"
else
  GO_BIN="go"
fi

exec "$GO_BIN" test ./...
