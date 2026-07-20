package catalog

import "testing"

func TestRecommendFiltersUnsupportedGames(t *testing.T) {
	c := NewInMemoryCatalog(DefaultGames())

	games := c.Recommend(2, "")

	for _, game := range games {
		if !game.Supports(2) {
			t.Fatalf("unsupported game returned: %s", game.ID)
		}
	}
}

func TestRecommendPrioritizesRecommendedPlayerCounts(t *testing.T) {
	c := NewInMemoryCatalog(DefaultGames())

	games := c.Recommend(4, "")
	if len(games) == 0 {
		t.Fatal("expected recommended games")
	}

	if !games[0].IsRecommendedFor(4) {
		t.Fatalf("first game should be recommended for player count, got %s", games[0].ID)
	}
}

func TestRecommendFiltersByCategory(t *testing.T) {
	c := NewInMemoryCatalog(DefaultGames())

	games := c.Recommend(4, CategoryBluffing)
	if len(games) == 0 {
		t.Fatal("expected bluffing games")
	}

	for _, game := range games {
		if !hasCategory(game, CategoryBluffing) {
			t.Fatalf("unexpected category result: %s", game.ID)
		}
	}
}
