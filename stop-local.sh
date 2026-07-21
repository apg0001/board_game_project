#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RUN_DIR="$ROOT_DIR/.run"

stop_process() {
  local label="$1"
  local pid_file="$2"

  if [[ ! -f "$pid_file" ]]; then
    echo "${label}: PID 파일 없음"
    return
  fi

  local pid
  pid="$(cat "$pid_file")"
  if [[ -z "$pid" ]]; then
    rm -f "$pid_file"
    echo "${label}: 빈 PID 파일 정리"
    return
  fi

  if ! kill -0 "$pid" >/dev/null 2>&1; then
    rm -f "$pid_file"
    echo "${label}: 이미 종료됨"
    return
  fi

  echo "${label}: PID ${pid} 종료 중"
  kill "$pid" >/dev/null 2>&1 || true

  for _ in {1..20}; do
    if ! kill -0 "$pid" >/dev/null 2>&1; then
      rm -f "$pid_file"
      echo "${label}: 종료 완료"
      return
    fi
    sleep 0.2
  done

  echo "${label}: 정상 종료가 지연되어 강제 종료합니다."
  kill -9 "$pid" >/dev/null 2>&1 || true
  rm -f "$pid_file"
}

stop_process "Web" "$RUN_DIR/web.pid"
stop_process "API" "$RUN_DIR/api.pid"

echo "직접 실행 개발 서버 종료 완료"
