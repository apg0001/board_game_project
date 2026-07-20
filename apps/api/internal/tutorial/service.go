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
		"davinci": {
			GameID:  "davinci",
			Title:   "다빈치 코드",
			Summary: "상대의 숨겨진 숫자 타일을 추리하고, 내 타일은 끝까지 숨기는 숫자 추리 게임입니다.",
			Tips: []string{
				"내 타일은 항상 보이지만 상대의 비공개 타일은 ?로 표시됩니다.",
				"상대 타일을 선택한 뒤 색상과 숫자를 고르고 추측하세요.",
				"오답이면 내 숨겨진 타일 하나가 공개됩니다.",
			},
			Steps: []Step{
				{Title: "타일 확인", Description: "내 타일의 색상과 숫자를 확인하고 상대의 공개 타일을 관찰합니다.", ActionHint: "내 타일 영역 확인"},
				{Title: "상대 타일 선택", Description: "상대의 ? 타일 중 하나를 선택합니다.", ActionHint: "상대 타일 탭"},
				{Title: "색상/숫자 추측", Description: "검정 또는 흰색과 0부터 11 사이 숫자를 입력합니다.", ActionHint: "추측하기"},
			},
		},
		"halli-galli": {
			GameID:  "halli-galli",
			Title:   "할리갈리",
			Summary: "펼쳐진 같은 과일의 합이 정확히 5가 되는 순간 종을 치는 순발력 게임입니다.",
			Tips: []string{
				"자기 차례에는 카드를 한 장 펼칩니다.",
				"같은 과일 합계가 5면 종을 치세요.",
				"잘못 누르면 점수 페널티를 받습니다.",
			},
			Steps: []Step{
				{Title: "카드 펼치기", Description: "차례가 오면 카드 한 장을 공개합니다.", ActionHint: "카드 펼치기"},
				{Title: "합계 확인", Description: "펼쳐진 맨 위 카드들의 과일별 합계를 봅니다.", ActionHint: "과일 수 확인"},
				{Title: "종 치기", Description: "합계가 정확히 5인 과일이 있으면 빠르게 종을 칩니다.", ActionHint: "종 치기"},
			},
		},
	}
}
