# Board Table

친구들과 모바일/데스크톱에서 같이 즐기는 PWA 보드게임 플랫폼입니다. 프론트엔드는 React PWA, 백엔드는 Go API/WebSocket 서버로 구성했습니다.

## 현재 구성

```txt
.
  src/                  # React PWA 클라이언트
  public/               # manifest, service worker, PWA 아이콘
  apps/api/             # Go 백엔드
    cmd/api             # 서버 엔트리포인트
    internal/catalog    # 게임 목록/추천 도메인
    internal/gamecore   # 게임 모듈 인터페이스
    internal/httpapi    # REST/WebSocket 라우터
    internal/realtime   # WebSocket 허브
  docker-compose.dev.yml
```

## 실행

### 프론트엔드

```bash
npm install
npm run dev
```

브라우저에서 `http://localhost:5173`으로 접속합니다.

### Go 백엔드

```bash
cd apps/api
go run ./cmd/api
```

기본 주소는 `http://localhost:4000`입니다.

### Docker Compose

```bash
docker compose -f docker-compose.dev.yml up --build
```

같은 Wi-Fi의 모바일에서 접속하려면 PC의 로컬 IP를 확인한 뒤 다음처럼 환경변수를 조정합니다.

```bash
export CORS_ORIGINS="http://localhost:5173,http://192.168.0.21:5173"
export VITE_API_URL="http://192.168.0.21:4000"
export VITE_WS_URL="ws://192.168.0.21:4000/ws"
```

모바일 브라우저에서는 `http://192.168.0.21:5173`으로 접속합니다.

## PWA

- `public/manifest.webmanifest`로 홈 화면 설치를 지원합니다.
- `public/sw.js`로 앱 셸을 캐싱합니다.
- `viewport-fit=cover`와 `safe-area` 처리를 넣어 iOS/Android 설치형 화면에서도 하단 액션바가 잘리지 않게 했습니다.
- 프로덕션 빌드에서만 서비스워커를 등록합니다.

## API

```txt
GET /health
GET /api/games
GET /api/games/recommend?players=4
GET /ws?room=lobby&user=Guest_8391
```

추천 API 응답 예시:

```json
{
  "playerCount": 4,
  "games": [
    {
      "id": "davinci",
      "title": "다빈치 코드",
      "minPlayers": 2,
      "maxPlayers": 4,
      "recommendedPlayers": [2, 3, 4],
      "estimatedMinutes": 15,
      "difficulty": "쉬움",
      "categories": ["숫자/조합", "전략"]
    }
  ]
}
```

## 게임 모듈 원칙

신규 게임은 `apps/api/internal/gamecore.Module`을 구현합니다.

- 상태 생성: `CreateInitialState`
- 공개/개인 상태 분리: `PublicState`
- 액션 검증: `ValidateAction`
- 상태 전이: `ApplyAction`
- 종료 판단: `IsFinished`
- 결과 계산: `CalculateResult`

프론트 UI는 게임 액션만 서버로 보내고, 실제 룰 검증과 상태 변경은 Go 백엔드의 게임 모듈에서만 처리합니다.

## 검증

```bash
npm run lint
npm run build

cd apps/api
go test ./...
```

## Git 작업 흐름

현재 작업 브랜치:

```txt
feature/#1-pwa-go-backend
```

커밋 메시지는 한국어 본문에 What, How, Why를 포함합니다.
