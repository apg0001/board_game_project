package tutorial

type Step struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	ActionHint  string `json:"actionHint"`
}

type Guide struct {
	GameID  string   `json:"gameId"`
	Title   string   `json:"title"`
	Summary string   `json:"summary"`
	Tips    []string `json:"tips"`
	Steps   []Step   `json:"steps"`
}

type Service struct {
	guides map[string]Guide
}

func NewService() *Service {
	return &Service{guides: defaultGuides()}
}

func (s *Service) Find(gameID string) (Guide, bool) {
	guide, ok := s.guides[gameID]
	return guide, ok
}

func (s *Service) All() []Guide {
	guides := make([]Guide, 0, len(s.guides))
	for _, guide := range s.guides {
		guides = append(guides, guide)
	}
	return guides
}

func defaultGuides() map[string]Guide {
	return map[string]Guide{
		"dalmuti": {
			GameID:  "dalmuti",
			Title:   "위대한 달무티",
			Summary: "낮은 숫자가 더 강한 계급 카드 게임입니다. 같은 계급 카드를 같은 장수로 내며 가장 먼저 손패를 비우면 다음 라운드의 높은 계급이 됩니다.",
			Tips: []string{
				"카드 숫자는 계급이자 장수입니다. 1이 가장 강하고 12가 가장 약합니다.",
				"이전 플레이와 같은 장수이면서 더 강한 계급의 세트만 낼 수 있습니다.",
				"조커는 단독이면 가장 약하지만 다른 카드와 함께 내면 그 계급을 따라갑니다.",
				"라운드가 끝나면 순위에 따라 좌석과 세금 교환이 다시 정해집니다.",
			},
			Steps: []Step{
				{Title: "계급 확인", Description: "라운드 시작 전 좌석과 계급을 확인하고 세금 교환 또는 혁명 여부를 처리합니다.", ActionHint: "계급/세금 확인"},
				{Title: "세트 내기", Description: "리드 플레이어는 같은 계급 카드를 원하는 장수로 내고, 이후 플레이어는 같은 장수의 더 강한 세트를 냅니다.", ActionHint: "카드 세트 선택"},
				{Title: "패스와 새 리드", Description: "낼 수 없거나 아끼고 싶으면 패스합니다. 모두 연속으로 패스하면 마지막으로 낸 플레이어가 새 리드를 잡습니다.", ActionHint: "패스"},
				{Title: "라운드 종료", Description: "손패를 먼저 비운 순서대로 다음 라운드의 계급이 정해집니다.", ActionHint: "순위 확인"},
			},
		},
		"werewolf": {
			GameID:  "werewolf",
			Title:   "한 밤의 늑대인간",
			Summary: "모두가 한 번씩 비밀 역할 행동을 한 뒤 토론과 투표로 늑대인간을 찾아내는 짧은 블러핑 게임입니다.",
			Tips: []string{
				"자기 역할뿐 아니라 밤 행동 후 바뀐 역할 가능성까지 추리해야 합니다.",
				"마을 팀은 늑대인간을 처형하면 승리하고, 늑대인간 팀은 늑대가 처형되지 않으면 승리합니다.",
				"역할 행동은 정해진 밤 순서대로 처리되며, 투표 전에는 공개 정보와 발언만 사용할 수 있습니다.",
				"중앙 카드 3장은 누구에게도 배정되지 않은 역할 후보입니다.",
			},
			Steps: []Step{
				{Title: "역할 확인", Description: "각 플레이어는 비밀 역할 1장을 받고 중앙에는 역할 3장이 놓입니다.", ActionHint: "내 역할 확인"},
				{Title: "밤 행동", Description: "늑대인간, 예언자, 강도, 말썽쟁이 등 역할별 효과를 정해진 순서로 해결합니다.", ActionHint: "역할 행동"},
				{Title: "토론", Description: "아침이 되면 모든 플레이어가 발언하며 현재 자기 역할과 늑대 위치를 추리합니다.", ActionHint: "채팅/이모지"},
				{Title: "투표와 승패", Description: "동시에 투표하고 최다 득표자를 처형합니다. 처형 결과에 따라 마을 또는 늑대 팀 승패가 결정됩니다.", ActionHint: "투표"},
			},
		},
		"rummikub": {
			GameID:  "rummikub",
			Title:   "루미큐브",
			Summary: "숫자 타일을 세트나 런으로 테이블에 내려놓고 재배열하면서 가장 먼저 받침대의 타일을 모두 없애는 게임입니다.",
			Tips: []string{
				"첫 등록은 자기 타일만으로 합계 30 이상을 만들어야 합니다.",
				"세트는 같은 숫자 다른 색 3장 이상, 런은 같은 색 연속 숫자 3장 이상입니다.",
				"첫 등록 뒤에는 테이블의 기존 조합을 재배열할 수 있지만 모든 조합은 유효하게 남아야 합니다.",
				"낼 수 없거나 내지 않으면 타일을 1개 가져옵니다.",
			},
			Steps: []Step{
				{Title: "초기 손패", Description: "각 플레이어는 타일 14개를 받고 자기 받침대에 숨겨 둡니다.", ActionHint: "손패 확인"},
				{Title: "첫 등록", Description: "합계 30 이상인 유효 조합을 자기 타일만으로 내려놓습니다.", ActionHint: "조합 등록"},
				{Title: "테이블 조작", Description: "등록 후에는 기존 테이블 조합을 쪼개고 붙여 새 유효 조합을 만들 수 있습니다.", ActionHint: "타일 이동"},
				{Title: "라운드 종료", Description: "누군가 손패를 모두 비우면 남은 타일 점수로 라운드 점수를 계산합니다.", ActionHint: "점수 확인"},
			},
		},
		"bang": {
			GameID:  "bang",
			Title:   "뱅!",
			Summary: "서부극 정체 은닉 카드 게임입니다. 보안관, 부관, 무법자, 배신자가 각자의 승리 조건을 향해 장비와 액션 카드를 사용합니다.",
			Tips: []string{
				"보안관만 정체를 공개하고 나머지 직업은 비공개로 시작합니다.",
				"기본적으로 자기 턴에는 카드 2장을 뽑고 원하는 만큼 사용할 수 있지만 BANG!은 보통 턴당 1장 제한입니다.",
				"공격 가능 여부는 무기 사거리와 플레이어 간 거리로 결정됩니다.",
				"체력이 0이 되면 맥주 등 즉시 회복 카드가 없을 때 탈락합니다.",
			},
			Steps: []Step{
				{Title: "직업과 캐릭터", Description: "직업 카드를 받고 캐릭터의 생명력과 능력을 확인합니다.", ActionHint: "역할 확인"},
				{Title: "카드 뽑기", Description: "자기 턴 시작에 덱에서 카드 2장을 뽑습니다.", ActionHint: "카드 뽑기"},
				{Title: "카드 사용", Description: "BANG!, 빗나감!, 맥주, 장비, 갈색 액션 카드를 규칙에 맞게 사용합니다.", ActionHint: "카드 사용"},
				{Title: "승리 조건", Description: "보안관 팀, 무법자, 배신자의 목표가 달라 끝까지 정체와 의도를 숨기는 것이 중요합니다.", ActionHint: "승패 확인"},
			},
		},
		"davinci": {
			GameID:  "davinci",
			Title:   "다빈치 코드",
			Summary: "상대의 숨겨진 숫자 타일을 추리하고, 내 코드는 끝까지 숨기는 숫자 추리 게임입니다.",
			Tips: []string{
				"기본 타일은 흰색/검정 0부터 11까지이며, 같은 숫자는 검정이 왼쪽에 놓입니다.",
				"자기 턴에는 먼저 타일을 1개 뽑고 그 타일을 임시로 숨겨 둔 뒤 추측합니다.",
				"정답이면 계속 추측하거나 턴을 끝낼 수 있고, 오답이면 이번 턴에 뽑은 타일을 공개해 삽입합니다.",
				"고급 규칙에서는 대시 타일이 조커처럼 코드 안 원하는 위치에 들어갑니다.",
			},
			Steps: []Step{
				{Title: "초기 코드", Description: "2-3인은 4개, 4인은 3개의 타일을 뽑아 오름차순으로 세웁니다.", ActionHint: "내 코드 확인"},
				{Title: "타일 뽑기", Description: "자기 턴 시작에 중앙에서 타일 1개를 뽑아 자신만 확인합니다.", ActionHint: "타일 뽑기"},
				{Title: "상대 추측", Description: "상대의 비공개 타일 하나를 골라 색상과 숫자 또는 대시를 추측합니다.", ActionHint: "추측하기"},
				{Title: "계속 또는 종료", Description: "맞히면 추가 추측이나 턴 종료를 선택하고, 틀리면 뽑은 타일을 공개한 채 삽입합니다.", ActionHint: "턴 종료"},
			},
		},
		"halli-galli": {
			GameID:  "halli-galli",
			Title:   "할리갈리",
			Summary: "펼쳐진 같은 과일의 합이 정확히 5가 되는 순간 종을 치는 순발력 게임입니다.",
			Tips: []string{
				"56장 덱을 모두 나누고 각자 자기 앞에 뒤집은 더미를 만듭니다.",
				"자기 차례에는 자기 더미 맨 위 카드 1장을 펼쳐 기존 공개 카드를 덮습니다.",
				"같은 과일이 정확히 5개 보이면 가장 먼저 종을 친 플레이어가 모든 공개 카드를 가져갑니다.",
				"잘못 종을 치면 다른 활성 플레이어들에게 카드 1장씩 벌칙으로 줍니다.",
			},
			Steps: []Step{
				{Title: "카드 펼치기", Description: "차례가 오면 카드 한 장을 공개합니다.", ActionHint: "카드 펼치기"},
				{Title: "합계 확인", Description: "펼쳐진 맨 위 카드들의 과일별 합계를 봅니다.", ActionHint: "과일 수 확인"},
				{Title: "종 치기", Description: "합계가 정확히 5인 과일이 있으면 빠르게 종을 치고 공개 카드를 모두 가져갑니다.", ActionHint: "종 치기"},
				{Title: "탈락과 승리", Description: "카드가 모두 떨어진 플레이어는 더 이상 넘길 수 없고, 마지막까지 카드를 가진 플레이어가 승리합니다.", ActionHint: "카드 수 확인"},
			},
		},
		"splendor": {
			GameID:  "splendor",
			Title:   "스플랜더",
			Summary: "보석 토큰으로 개발 카드를 구매해 영구 보너스와 명성을 얻고, 15점 도달 후 최종 승자를 가리는 엔진 빌딩 게임입니다.",
			Tips: []string{
				"한 턴에는 토큰 가져오기, 개발 카드 구매, 카드 예약 중 하나만 합니다.",
				"금 토큰은 예약할 때 얻는 와일드 토큰입니다.",
				"행동 후 토큰이 10개를 넘으면 초과분을 반납한 뒤 턴이 넘어갑니다.",
				"개발 카드 보너스는 이후 구매 비용을 줄이고 귀족 방문 조건에도 쓰입니다.",
				"누군가 15점 이상이 되면 시작 플레이어 전까지 같은 턴 수를 맞춘 뒤 승자를 정합니다.",
			},
			Steps: []Step{
				{Title: "시장 준비", Description: "레벨별 개발 카드 4장씩과 인원수에 맞는 귀족 타일을 공개합니다.", ActionHint: "시장 확인"},
				{Title: "토큰 선택", Description: "서로 다른 보석 3개 또는 같은 보석 2개 등 가능한 조합으로 토큰을 가져옵니다.", ActionHint: "토큰 가져오기"},
				{Title: "구매와 예약", Description: "비용을 지불해 개발 카드를 구매하거나 카드를 예약해 손에 보관하고 금 토큰을 받습니다.", ActionHint: "카드 구매"},
				{Title: "귀족과 종료", Description: "턴 종료 시 조건을 만족한 귀족을 받고, 15점 이상 도달 후 마지막 라운드를 진행합니다.", ActionHint: "점수 확인"},
			},
		},
		"sutda": {
			GameID:  "sutda",
			Title:   "섯다",
			Summary: "화투 20장으로 2장 조합의 족보를 겨루는 한국식 베팅 게임입니다. 베팅 판단과 족보 기억이 핵심입니다.",
			Tips: []string{
				"기본 덱은 1월부터 10월까지 각 2장씩 총 20장입니다.",
				"가장 강한 족보는 삼팔광땡이고, 땡, 알리, 독사, 구삥, 장삥, 장사, 세륙, 끗 순으로 비교합니다.",
				"암행어사, 땡잡이, 구사 같은 특수 족보는 특정 강한 족보를 잡거나 재경기를 만들 수 있습니다.",
				"베팅 규칙은 방 옵션으로 제한을 두고, 쇼다운에서 남은 플레이어끼리 족보를 비교합니다.",
			},
			Steps: []Step{
				{Title: "카드 받기", Description: "각 플레이어는 비공개 화투 카드 2장을 받아 족보를 확인합니다.", ActionHint: "패 확인"},
				{Title: "베팅", Description: "콜, 레이즈, 다이 등 방 규칙에 맞는 베팅 행동을 선택합니다.", ActionHint: "베팅"},
				{Title: "족보 비교", Description: "남은 플레이어가 카드를 공개하고 특수 족보와 기본 족보 순서대로 비교합니다.", ActionHint: "쇼다운"},
				{Title: "정산", Description: "가장 높은 족보가 판돈을 가져가고 다음 판 딜러와 선을 정합니다.", ActionHint: "결과 확인"},
			},
		},
		"gostop": {
			GameID:  "gostop",
			Title:   "고스톱",
			Summary: "같은 월의 화투를 맞춰 가져가며 광, 띠, 열끗, 피 점수를 만들고 고 또는 스톱을 선택하는 한국 전통 카드 게임입니다.",
			Tips: []string{
				"자기 차례에는 손패 1장을 내고 더미 1장을 뒤집어 각각 바닥패와 같은 월을 맞춥니다.",
				"점수는 광, 띠, 열끗, 피 조합으로 계산하며 3인 기본은 보통 3점부터 고/스톱을 선택합니다.",
				"고를 외치면 판을 계속하고 보너스 배율을 노리지만 상대가 먼저 스톱할 위험이 커집니다.",
				"따닥, 쪽, 폭탄, 흔들기 등 특수 상황은 방 옵션으로 단계적으로 켤 수 있게 관리합니다.",
			},
			Steps: []Step{
				{Title: "패 배치", Description: "인원수에 맞춰 손패, 바닥패, 더미를 나누고 보너스패를 처리합니다.", ActionHint: "패 확인"},
				{Title: "월 맞추기", Description: "손에서 카드 1장을 내고 더미 1장을 뒤집어 같은 월 카드와 함께 획득합니다.", ActionHint: "카드 내기"},
				{Title: "점수 계산", Description: "획득패의 광, 띠, 열끗, 피 점수를 갱신합니다.", ActionHint: "점수판 확인"},
				{Title: "고 또는 스톱", Description: "기준 점수 이상이면 계속할지 종료할지 선택하고, 스톱하면 최종 점수를 정산합니다.", ActionHint: "고/스톱"},
			},
		},
		"onecard": {
			GameID:  "onecard",
			Title:   "원카드",
			Summary: "같은 문양이나 숫자의 카드를 내며 손패를 먼저 비우는 게임입니다. 지역별 차이가 큰 공격/방어 규칙은 시작 전 투표로 확정합니다.",
			Tips: []string{
				"로비에서 공격카드, 방어카드, 조커 공격 장수, 공격 누적 여부를 투표합니다.",
				"공격이 누적되면 정해진 방어카드로 막거나 누적 장수만큼 카드를 뽑습니다.",
				"특수카드로 방향 전환, 건너뛰기, 추가 턴, 문양 변경 같은 효과를 처리합니다.",
				"투표가 동률이면 서버가 무작위로 선택하고 모두에게 공지합니다.",
			},
			Steps: []Step{
				{Title: "룰 투표", Description: "방 로비에서 이번 판에 사용할 공격, 방어, 조커, 누적 규칙을 고릅니다.", ActionHint: "룰 투표"},
				{Title: "카드 내기", Description: "버린 더미의 문양 또는 숫자와 맞는 카드를 내거나 와일드 카드를 냅니다.", ActionHint: "카드 선택"},
				{Title: "공격 대응", Description: "공격 카드가 나오면 방어카드를 내거나 확정된 장수만큼 카드를 뽑습니다.", ActionHint: "카드 뽑기"},
				{Title: "승리", Description: "마지막 카드를 처리해 손패를 모두 비운 플레이어가 라운드에서 승리합니다.", ActionHint: "결과 확인"},
			},
		},
		"jokerdraw": {
			GameID:  "jokerdraw",
			Title:   "조커뽑기",
			Summary: "같은 숫자 쌍을 버리고 옆 사람 손패에서 카드를 뽑습니다. 끝까지 조커를 가진 플레이어가 패배하는 Old Maid 계열 게임입니다.",
			Tips: []string{
				"처음 받은 손패에서 같은 숫자 쌍을 모두 버리고 시작합니다.",
				"자기 차례에는 다음 플레이어의 뒷면 카드 중 1장을 뽑습니다.",
				"뽑은 카드로 새 쌍이 생기면 즉시 버립니다.",
				"모든 쌍이 사라진 뒤 조커만 남긴 플레이어가 패배합니다.",
			},
			Steps: []Step{
				{Title: "쌍 버리기", Description: "초기 손패와 이후 뽑은 카드에서 같은 숫자 쌍을 자동으로 제거합니다.", ActionHint: "쌍 확인"},
				{Title: "카드 뽑기", Description: "차례가 오면 지정된 상대의 비공개 손패에서 카드 1장을 선택합니다.", ActionHint: "카드 뽑기"},
				{Title: "탈출", Description: "손패가 모두 없어지면 게임에서 빠져나가고 남은 플레이어끼리 계속합니다.", ActionHint: "대기"},
				{Title: "패자 결정", Description: "마지막에 조커를 들고 있는 플레이어가 패배합니다.", ActionHint: "결과 확인"},
			},
		},
	}
}
