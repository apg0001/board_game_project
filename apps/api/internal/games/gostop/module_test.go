package gostop

import (
	"testing"
	"time"

	"board-game-platform/apps/api/internal/gamecore"
)

func TestCreateInitialStateDealsGoStop(t *testing.T) {
	module := NewModule()
	state := module.CreateInitialState(testContext()).(State)
	if len(state.Players[0].Hand) != 7 {
		t.Fatalf("expected 7 cards for three-player go-stop, got %d", len(state.Players[0].Hand))
	}
	if len(state.Field) != 6 {
		t.Fatalf("expected 6 field cards, got %d", len(state.Field))
	}
}

func TestScoreBrightAndJunk(t *testing.T) {
	cards := []Card{
		{Kind: "bright"}, {Kind: "bright"}, {Kind: "bright"},
		{Kind: "junk"}, {Kind: "junk"}, {Kind: "junk"}, {Kind: "junk"}, {Kind: "junk"},
		{Kind: "junk"}, {Kind: "junk"}, {Kind: "junk"}, {Kind: "junk"}, {Kind: "junk"},
	}
	if score(cards) != 4 {
		t.Fatalf("expected 4 points, got %d", score(cards))
	}
}

func testContext() gamecore.Context {
	return gamecore.Context{
		GameID: "gostop",
		Players: []gamecore.Player{
			{ID: "p1", SeatIndex: 0, DisplayName: "P1", Connected: true},
			{ID: "p2", SeatIndex: 1, DisplayName: "P2", Connected: true},
			{ID: "p3", SeatIndex: 2, DisplayName: "P3", Connected: true},
		},
		Now:        time.Date(2026, 7, 21, 1, 0, 0, 0, time.UTC),
		RandomSeed: "seed",
	}
}
