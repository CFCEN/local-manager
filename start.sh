#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")"

PID_FILE="server.pid"
BIN="bin/server"
LOG="server.log"
PORT="${PORT:-9527}"

is_running() {
  [[ -f "$PID_FILE" ]] || return 1
  local pid
  pid="$(cat "$PID_FILE")"
  [[ -n "$pid" ]] && kill -0 "$pid" 2>/dev/null
}

stop_server() {
  if [[ -f "$PID_FILE" ]]; then
    local pid
    pid="$(cat "$PID_FILE")"
    if [[ -n "$pid" ]] && kill -0 "$pid" 2>/dev/null; then
      kill "$pid" 2>/dev/null || true
      local _
      for _ in 1 2 3 4 5 6 7 8 9 10; do
        kill -0 "$pid" 2>/dev/null || break
        sleep 0.2
      done
      if kill -0 "$pid" 2>/dev/null; then
        kill -9 "$pid" 2>/dev/null || true
      fi
    fi
    rm -f "$PID_FILE"
  fi
}

start_server() {
  mkdir -p data bin
  CGO_ENABLED=1 go build -o "$BIN" ./cmd/server
  stop_server
  nohup "./$BIN" >>"$LOG" 2>&1 &
  echo $! >"$PID_FILE"
  disown || true

  local _
  for _ in $(seq 1 25); do
    if curl -sf "http://127.0.0.1:${PORT}/api/workspace" >/dev/null; then
      echo "started pid=$(cat "$PID_FILE") http://127.0.0.1:${PORT}"
      return 0
    fi
    if ! is_running; then
      echo "process exited, see $LOG" >&2
      return 1
    fi
    sleep 0.2
  done
  echo "health check failed, see $LOG" >&2
  return 1
}

status_server() {
  if is_running; then
    echo "running pid=$(cat "$PID_FILE") port=${PORT}"
  else
    echo "stopped port=${PORT}"
    return 1
  fi
}

case "${1:-start}" in
  start) start_server ;;
  stop)
    stop_server
    echo "stopped"
    ;;
  status) status_server ;;
  *)
    echo "usage: $0 [start|stop|status]" >&2
    exit 1
    ;;
esac
