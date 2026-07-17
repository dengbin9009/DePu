#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CHANGE="table-playability-hardening"
TASKS="$ROOT/openspec/changes/$CHANGE/tasks.md"
PROMPT="$ROOT/.loop/worker.md"
LOG_DIR="$ROOT/.loop/logs"
MAX_ROUNDS="${1:-12}"
NODE_BIN="${NODE_BIN:-$HOME/.nvm/versions/node/v20.19.4/bin}"
RUN_ID="$(date '+%Y%m%d-%H%M%S')"

if [[ ! -x "$NODE_BIN/node" || ! -x "$NODE_BIN/npx" ]]; then
  echo "Node 20 runtime not found at $NODE_BIN. Set NODE_BIN to a usable Node 20 bin directory."
  exit 5
fi

export PATH="$NODE_BIN:$PATH"

mkdir -p "$LOG_DIR"

for round in $(seq 1 "$MAX_ROUNDS"); do
  if ! rg -q '^- \[ \]' "$TASKS"; then
    echo "Loop complete: no unfinished tasks."
    exit 0
  fi

  log="$LOG_DIR/$CHANGE-$RUN_ID-$round.log"
  echo "Starting loop round $round/$MAX_ROUNDS"

  codex exec --sandbox danger-full-access -c 'approval_policy="never"' -c 'service_tier="fast"' -c 'model_reasoning_effort="xhigh"' -C "$ROOT" - < "$PROMPT" | tee "$log"

  if rg -q '^LOOP_STATUS: BLOCKED$' "$log"; then
    echo "Loop blocked in round $round. See $log"
    exit 2
  fi

  if ! rg -q '^LOOP_STATUS: (CONTINUE|DONE)$' "$log"; then
    echo "Loop worker did not emit a valid status. See $log"
    exit 3
  fi

  git -C "$ROOT" diff --check
  npx --yes @fission-ai/openspec validate "$CHANGE" --strict --no-interactive
done

if rg -q '^- \[ \]' "$TASKS"; then
  echo "Loop stopped after $MAX_ROUNDS rounds with unfinished tasks."
  exit 4
fi

echo "Loop complete."
