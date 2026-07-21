package tutorial

import (
	"testing"

	"board-game-platform/apps/api/internal/catalog"
)

func TestFindGuide(t *testing.T) {
	service := NewService()
	guide, ok := service.Find("davinci")
	if !ok {
		t.Fatal("expected davinci guide")
	}
	if len(guide.Steps) == 0 {
		t.Fatal("expected guide steps")
	}
}

func TestAllCatalogGamesHaveRuleGuides(t *testing.T) {
	service := NewService()

	for _, game := range catalog.DefaultGames() {
		guide, ok := service.Find(game.ID)
		if !ok {
			t.Fatalf("expected guide for %s", game.ID)
		}
		if guide.Title == "" {
			t.Fatalf("expected title for %s", game.ID)
		}
		if guide.Summary == "" {
			t.Fatalf("expected summary for %s", game.ID)
		}
		if len(guide.Tips) < 3 {
			t.Fatalf("expected at least 3 tips for %s", game.ID)
		}
		if len(guide.Steps) < 3 {
			t.Fatalf("expected at least 3 steps for %s", game.ID)
		}
	}
}
