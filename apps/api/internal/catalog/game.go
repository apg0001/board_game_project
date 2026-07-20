package catalog

type Category string

const (
	CategoryBluffing Category = "블러핑"
	CategoryStrategy Category = "전략"
	CategoryReaction Category = "순발력"
	CategoryNumber   Category = "숫자/조합"
	CategoryParty    Category = "파티"
	CategoryCard     Category = "카드"
)

type Game struct {
	ID                 string     `json:"id"`
	Title              string     `json:"title"`
	MinPlayers         int        `json:"minPlayers"`
	MaxPlayers         int        `json:"maxPlayers"`
	RecommendedPlayers []int      `json:"recommendedPlayers"`
	EstimatedMinutes   int        `json:"estimatedMinutes"`
	Difficulty         string     `json:"difficulty"`
	Categories         []Category `json:"categories"`
}

func (g Game) Supports(playerCount int) bool {
	return g.MinPlayers <= playerCount && playerCount <= g.MaxPlayers
}

func (g Game) IsRecommendedFor(playerCount int) bool {
	for _, count := range g.RecommendedPlayers {
		if count == playerCount {
			return true
		}
	}
	return false
}
