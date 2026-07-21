#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
COMPOSE_FILE="${COMPOSE_FILE:-docker-compose.dev.yml}"

cd "$ROOT_DIR"

compose() {
  if docker compose version >/dev/null 2>&1; then
    docker compose "$@"
    return
  fi

  if command -v docker-compose >/dev/null 2>&1; then
    docker-compose "$@"
    return
  fi

  echo "Docker Compose를 찾을 수 없습니다. Docker Desktop을 실행하거나 docker compose를 설치해주세요." >&2
  exit 1
}

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

print_access_links() {
  echo
  echo "접속 링크"
  echo "  PC 웹: http://localhost:${WEB_PORT}"
  if [[ -n "$LAN_IP" ]]; then
    echo "  모바일/같은 Wi-Fi: http://${LAN_IP}:${WEB_PORT}"
  fi
  echo "  API: ${VITE_API_URL}"
  echo "  WebSocket: ${VITE_WS_URL}"
}

API_PORT="${API_PORT:-4000}"
WEB_PORT="${WEB_PORT:-5173}"
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

export API_PORT
export WEB_PORT

echo "Board Table 개발 서버를 시작합니다."

compose -f "$COMPOSE_FILE" up --build -d

echo
echo "실행 완료"
print_access_links
echo
echo "로그 보기: docker compose -f ${COMPOSE_FILE} logs -f"
echo "종료: ./stop.sh"
