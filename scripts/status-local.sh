#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
LOG_DIR="$ROOT_DIR/.logs"
BACKEND_PID_FILE="$LOG_DIR/backend.pid"
FRONTEND_PID_FILE="$LOG_DIR/frontend.pid"

print_pid_status() {
  local name="$1"
  local pid_file="$2"
  if [[ ! -f "$pid_file" ]]; then
    echo "[$name] pid: missing"
    return
  fi

  local pid
  pid="$(cat "$pid_file" 2>/dev/null || true)"
  if [[ -n "$pid" ]] && kill -0 "$pid" 2>/dev/null; then
    echo "[$name] pid: $pid running"
    ps -p "$pid" -o pid,ppid,stat,etime,command=
  else
    echo "[$name] pid: $pid stale"
  fi
}

print_port_status() {
  local port="$1"
  local pids
  pids="$(lsof -tiTCP:"$port" -sTCP:LISTEN 2>/dev/null || true)"
  if [[ -n "$pids" ]]; then
    echo "[port:$port] listening pid(s): $pids"
    ps -p "$pids" -o pid,ppid,stat,etime,command= 2>/dev/null || true
  else
    echo "[port:$port] not listening"
  fi
}

check_url() {
  local name="$1"
  local url="$2"
  if curl -fsS --max-time 3 "$url" >/dev/null; then
    echo "[$name] ok: $url"
  else
    echo "[$name] failed: $url"
  fi
}

print_pid_status backend "$BACKEND_PID_FILE"
print_pid_status frontend "$FRONTEND_PID_FILE"
print_port_status 5174
print_port_status 5175
check_url backend-health http://127.0.0.1:5174/health
check_url frontend-home http://127.0.0.1:5175/
check_url frontend-login http://127.0.0.1:5175/login
check_url api-proxy http://127.0.0.1:5175/api/rulesets
