# 카드 에셋 관리

## 현재 적용

- 표준 트럼프 카드: `public/assets/cards/playing`
- 출처: https://github.com/hayeah/playing-cards-assets
- 원 출처: vector-playing-cards public domain 기반
- 포함 파일: 52장 SVG, 조커 2장 SVG, 카드 뒷면 PNG, 원본 MIT 라이선스 사본

## 적용 범위

- 원카드: 실제 트럼프 SVG 우선 렌더링
- 조커뽑기: 실제 트럼프 SVG 우선 렌더링
- 카드 뒷면: 조커뽑기 상대 카드 선택 버튼에 사용
- 화투/섯다/고스톱: 현재는 로컬 SVG 스타일 렌더러를 사용하며, 개별 화투 SVG 세트 확보 후 `public/assets/cards/hwatu`로 분리 예정
- 달무티/스플랜더/뱅!: 상용 게임 고유 아트 대신 룰 데이터 기반 로컬 카드 렌더러를 사용

## 원칙

- 앱은 외부 URL을 직접 핫링크하지 않고 로컬 `public/assets/cards` 아래의 파일만 참조한다.
- PWA 서비스워커는 트럼프 카드 에셋을 설치 시점에 프리캐시한다.
- 새 에셋을 추가할 때는 출처, 라이선스, 파일명 매핑 규칙을 이 문서에 같이 기록한다.
