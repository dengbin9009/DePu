#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
LOG_DIR="$ROOT_DIR/.logs"
BIN_DIR="$ROOT_DIR/.bin"
BACKEND_LOG="$LOG_DIR/backend.log"
FRONTEND_LOG="$LOG_DIR/frontend.log"
BACKEND_BIN="$BIN_DIR/depu-server"
RUN_MODE="${1:-foreground}"

mkdir -p "$LOG_DIR" "$BIN_DIR"

takeover_port() {
  local port="$1"
  local pids
  pids="$(lsof -tiTCP:"$port" -sTCP:LISTEN 2>/dev/null || true)"
  if [[ -n "$pids" ]]; then
    echo "[port:$port] killing existing listener(s): $pids"
    kill $pids || true
    sleep 1
    pids="$(lsof -tiTCP:"$port" -sTCP:LISTEN 2>/dev/null || true)"
    if [[ -n "$pids" ]]; then
      kill -9 $pids || true
    fi
  fi
}

build_backend() {
  takeover_port 5174
  echo "[backend] building binary"
  (
    cd "$ROOT_DIR/backend"
    go build -o "$BACKEND_BIN" ./cmd/depu-server >>"$BACKEND_LOG" 2>&1
  )
}

build_frontend() {
  takeover_port 5175
  echo "[frontend] building"
  (
    cd "$ROOT_DIR/frontend"
    npm run build >>"$FRONTEND_LOG" 2>&1
  )
}

check_health() {
  echo "[check] backend health"
  curl -fsS http://127.0.0.1:5174/health >/dev/null
  echo "[check] frontend home"
  curl -fsS http://127.0.0.1:5175/ >/dev/null
  echo "[check] frontend route fallback"
  curl -fsS http://127.0.0.1:5175/login >/dev/null
  echo "[check] frontend api proxy"
  curl -fsS http://127.0.0.1:5175/api/rulesets >/dev/null
}

start_foreground() {
  build_backend
  build_frontend

  echo "[backend] starting on :5174"
  (
    cd "$ROOT_DIR/backend"
    "$BACKEND_BIN" >>"$BACKEND_LOG" 2>&1
  ) &
  local backend_pid=$!

  echo "[frontend] serving on :5175"
  (
    cd "$ROOT_DIR/frontend"
    python3 serve_app.py >>"$FRONTEND_LOG" 2>&1
  ) &
  local frontend_pid=$!

  echo "$backend_pid" >"$LOG_DIR/backend.pid"
  echo "$frontend_pid" >"$LOG_DIR/frontend.pid"

  cleanup() {
    echo
    echo "[stop] stopping local services"
    kill "$frontend_pid" "$backend_pid" 2>/dev/null || true
    wait "$frontend_pid" "$backend_pid" 2>/dev/null || true
  }
  trap cleanup INT TERM EXIT

  sleep 2
  check_health

  echo
  echo "启动成功，保持这个终端窗口打开："
  echo "- 前端: http://127.0.0.1:5175"
  echo "- 后端: http://127.0.0.1:5174"
  echo "- 后端日志: $BACKEND_LOG"
  echo "- 前端日志: $FRONTEND_LOG"
  echo "- 后端 PID: $backend_pid"
  echo "- 前端 PID: $frontend_pid"
  echo
  echo "按 Ctrl+C 停止服务。"

  wait "$backend_pid" "$frontend_pid"
}

start_background() {
  build_backend
  build_frontend

  echo "[backend] starting on :5174"
  (
    cd "$ROOT_DIR/backend"
    nohup "$BACKEND_BIN" >>"$BACKEND_LOG" 2>&1 < /dev/null &
    echo $! >"$LOG_DIR/backend.pid"
  )

  echo "[frontend] serving on :5175"
  (
    cd "$ROOT_DIR/frontend"
    nohup python3 serve_app.py >>"$FRONTEND_LOG" 2>&1 < /dev/null &
    echo $! >"$LOG_DIR/frontend.pid"
  )

  sleep 2
  check_health

  echo
  echo "后台启动成功："
  echo "- 前端: http://127.0.0.1:5175"
  echo "- 后端: http://127.0.0.1:5174"
  echo "- 后端日志: $BACKEND_LOG"
  echo "- 前端日志: $FRONTEND_LOG"
  echo "- 后端 PID: $(cat "$LOG_DIR/backend.pid")"
  echo "- 前端 PID: $(cat "$LOG_DIR/frontend.pid")"
}

case "$RUN_MODE" in
  foreground|fg)
    start_foreground
    ;;
  background|bg)
    start_background
    ;;
  *)
    echo "Usage: $0 [foreground|background]" >&2
    exit 2
    ;;
esac
