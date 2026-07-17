#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CALLER_DIR="$PWD"

if [[ $# -eq 0 ]]; then
  echo "Usage: $0 command [args...]" >&2
  exit 2
fi

export DEPU_TEST_RUN_ID="${DEPU_TEST_RUN_ID:-$(date -u '+%Y%m%dT%H%M%SZ')-$$}"

HELPER_BIN="$(mktemp "${TMPDIR:-/tmp}/depu-test-mysql.XXXXXX")"
trap 'rm -f "$HELPER_BIN"' EXIT

cd "$ROOT_DIR/backend"
go build -o "$HELPER_BIN" ./cmd/depu-test-mysql
"$HELPER_BIN" -label multi_account -cwd "$CALLER_DIR" -- "$@"
