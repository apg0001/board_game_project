#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RUN_DIR="$ROOT_DIR/.run"
API_PID_FILE="$RUN_DIR/api.pid"
WEB_PID_FILE="$RUN_DIR/web.pid"
API_LOG_FILE="$RUN_DIR/api.log"
WEB_LOG_FILE="$RUN_DIR/web.log"

cd "$ROOT_DIR"
mkdir -p "$RUN_DIR"

detect_lan_ip() {
  if [[ -n "${LAN_IP:-}" ]]; then
    echo "$LAN_IP"
    return
  fi

  if command -v ipconfig >/dev/null 2>&1; then
    ipconfig getifaddr en0 2>/dev/null && return
    ipconfig getifaddr en1 2>/dev/null && return
  fi

  if command -v hostname >/dev/null 2>&1; then
    hostname -I 2>/dev/null | awk '{print $1}' && return
  fi
}

is_running() {
  local pid_file="$1"
  [[ -f "$pid_file" ]] && kill -0 "$(cat "$pid_file")" >/dev/null 2>&1
}

ensure_port_free() {
  local port="$1"
  local label="$2"
  local pid_file="$3"

  if is_running "$pid_file"; then
    return
  fi

  if command -v lsof >/dev/null 2>&1 && lsof -nP -iTCP:"$port" -sTCP:LISTEN >/dev/null 2>&1; then
    echo "${label} 포트 ${port}가 이미 사용 중입니다. 먼저 해당 프로세스를 종료하거나 포트를 바꿔주세요." >&2
    exit 1
  fi
}

API_PORT="${API_PORT:-4000}"
WEB_PORT="${WEB_PORT:-5173}"
HTTP_ADDR="${HTTP_ADDR:-:${API_PORT}}"
LAN_IP="$(detect_lan_ip || true)"

if [[ -n "$LAN_IP" ]]; then
  export VITE_API_URL="${VITE_API_URL:-http://${LAN_IP}:${API_PORT}}"
  export VITE_WS_URL="${VITE_WS_URL:-ws://${LAN_IP}:${API_PORT}/ws}"
  export CORS_ORIGINS="${CORS_ORIGINS:-http://localhost:${WEB_PORT},http://127.0.0.1:${WEB_PORT},http://${LAN_IP}:${WEB_PORT}}"
else
  export VITE_API_URL="${VITE_API_URL:-http://localhost:${API_PORT}}"
  export VITE_WS_URL="${VITE_WS_URL:-ws://localhost:${API_PORT}/ws}"
  export CORS_ORIGINS="${CORS_ORIGINS:-http://localhost:${WEB_PORT},http://127.0.0.1:${WEB_PORT}}"
fi

export HTTP_ADDR
export API_PORT
export WEB_PORT

ensure_port_free "$API_PORT" "API" "$API_PID_FILE"
ensure_port_free "$WEB_PORT" "Web" "$WEB_PID_FILE"

if [[ ! -d "$ROOT_DIR/node_modules" ]]; then
  echo "node_modules가 없어 npm ci를 실행합니다."
  npm ci
fi

if is_running "$API_PID_FILE"; then
  echo "API는 이미 실행 중입니다. PID $(cat "$API_PID_FILE")"
else
  echo "Go API를 시작합니다."
  (
    cd "$ROOT_DIR/apps/api"
    go run ./cmd/api
  ) >"$API_LOG_FILE" 2>&1 &
  echo $! >"$API_PID_FILE"
fi

if is_running "$WEB_PID_FILE"; then
  echo "Web은 이미 실행 중입니다. PID $(cat "$WEB_PID_FILE")"
else
  echo "Vite 웹 서버를 시작합니다."
  (
    cd "$ROOT_DIR"
    npm run dev -- --host 0.0.0.0 --port "$WEB_PORT"
  ) >"$WEB_LOG_FILE" 2>&1 &
  echo $! >"$WEB_PID_FILE"
fi

echo
echo "직접 실행 개발 서버가 시작되었습니다."
echo "API: ${VITE_API_URL}"
echo "Web: http://localhost:${WEB_PORT}"
if [[ -n "$LAN_IP" ]]; then
  echo "Mobile/LAN: http://${LAN_IP}:${WEB_PORT}"
fi
echo "API 로그: tail -f ${API_LOG_FILE}"
echo "Web 로그: tail -f ${WEB_LOG_FILE}"
echo "종료: ./stop-local.sh"
