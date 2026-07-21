package catalog

import (
	"sort"
)

type Catalog interface {
	Recommend(playerCount int, category Category) []Game
	All() []Game
}

type InMemoryCatalog struct {
	games []Game
}

func NewInMemoryCatalog(games []Game) *InMemoryCatalog {
	return &InMemoryCatalog{games: append([]Game(nil), games...)}
}

func (c *InMemoryCatalog) All() []Game {
	return append([]Game(nil), c.games...)
}

func (c *InMemoryCatalog) Recommend(playerCount int, category Category) []Game {
	result := make([]Game, 0, len(c.games))
	for _, game := range c.games {
		if !game.Supports(playerCount) {
			continue
		}
		if category != "" && !hasCategory(game, category) {
			continue
		}
		result = append(result, game)
	}

	sort.SliceStable(result, func(i, j int) bool {
		left := result[i].IsRecommendedFor(playerCount)
		right := result[j].IsRecommendedFor(playerCount)
		if left != right {
			return left
		}
		return result[i].EstimatedMinutes < result[j].EstimatedMinutes
	})

	return result
}

func hasCategory(game Game, category Category) bool {
	for _, item := range game.Categories {
		if item == category {
			return true
		}
	}
	return false
}

func DefaultGames() []Game {
	return []Game{
		{ID: "dalmuti", Title: "위대한 달무티", MinPlayers: 4, MaxPlayers: 8, RecommendedPlayers: []int{5, 6, 7}, EstimatedMinutes: 20, Difficulty: "쉬움", Categories: []Category{CategoryCard, CategoryParty}},
		{ID: "werewolf", Title: "한 밤의 늑대인간", MinPlayers: 3, MaxPlayers: 10, RecommendedPlayers: []int{5, 6, 7}, EstimatedMinutes: 10, Difficulty: "보통", Categories: []Category{CategoryBluffing, CategoryParty}},
		{ID: "rummikub", Title: "루미큐브", MinPlayers: 2, MaxPlayers: 4, RecommendedPlayers: []int{3, 4}, EstimatedMinutes: 35, Difficulty: "보통", Categories: []Category{CategoryNumber, CategoryStrategy}},
		{ID: "bang", Title: "뱅!", MinPlayers: 4, MaxPlayers: 7, RecommendedPlayers: []int{5, 6, 7}, EstimatedMinutes: 40, Difficulty: "어려움", Categories: []Category{CategoryBluffing, CategoryCard}},
		{ID: "davinci", Title: "다빈치 코드", MinPlayers: 2, MaxPlayers: 4, RecommendedPlayers: []int{2, 3, 4}, EstimatedMinutes: 15, Difficulty: "쉬움", Categories: []Category{CategoryNumber, CategoryStrategy}},
		{ID: "halli-galli", Title: "할리갈리", MinPlayers: 2, MaxPlayers: 6, RecommendedPlayers: []int{3, 4, 5}, EstimatedMinutes: 10, Difficulty: "쉬움", Categories: []Category{CategoryReaction, CategoryParty}},
		{ID: "splendor", Title: "스플랜더", MinPlayers: 2, MaxPlayers: 4, RecommendedPlayers: []int{3, 4}, EstimatedMinutes: 30, Difficulty: "보통", Categories: []Category{CategoryStrategy, CategoryCard}},
		{ID: "sutda", Title: "섯다", MinPlayers: 2, MaxPlayers: 10, RecommendedPlayers: []int{3, 4, 5}, EstimatedMinutes: 8, Difficulty: "보통", Categories: []Category{CategoryBluffing, CategoryCard}},
		{ID: "gostop", Title: "고스톱", MinPlayers: 2, MaxPlayers: 3, RecommendedPlayers: []int{3}, EstimatedMinutes: 20, Difficulty: "보통", Categories: []Category{CategoryStrategy, CategoryCard}},
		{ID: "onecard", Title: "원카드", MinPlayers: 2, MaxPlayers: 6, RecommendedPlayers: []int{3, 4, 5}, EstimatedMinutes: 10, Difficulty: "쉬움", Categories: []Category{CategoryCard, CategoryParty}},
		{ID: "jokerdraw", Title: "조커뽑기", MinPlayers: 2, MaxPlayers: 8, RecommendedPlayers: []int{4, 5, 6}, EstimatedMinutes: 8, Difficulty: "쉬움", Categories: []Category{CategoryCard, CategoryParty}},
	}
}
