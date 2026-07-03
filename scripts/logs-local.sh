#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
LOG_DIR="$ROOT_DIR/.logs"
BACKEND_LOG="$LOG_DIR/backend.log"
FRONTEND_LOG="$LOG_DIR/frontend.log"

mkdir -p "$LOG_DIR"

echo "===== backend.log ====="
tail -n 120 "$BACKEND_LOG" 2>/dev/null || echo "暂无 backend 日志"
echo
echo "===== frontend.log ====="
tail -n 120 "$FRONTEND_LOG" 2>/dev/null || echo "暂无 frontend 日志"
