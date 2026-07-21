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

echo "Docker Compose 개발 서버를 종료합니다."
compose -f "$COMPOSE_FILE" down "$@"
echo "종료 완료"
