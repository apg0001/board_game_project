package onecard

import (
	"testing"
	"time"

	"board-game-platform/apps/api/internal/gamecore"
)

func TestCreateInitialStateDealsSevenCards(t *testing.T) {
	module := NewModule()
	state := module.CreateInitialState(testContext()).(State)
	if len(state.Players[0].Hand) != 7 {
		t.Fatalf("expected seven cards, got %d", len(state.Players[0].Hand))
	}
	if len(state.DiscardPile) != 1 {
		t.Fatalf("expected one top card, got %d", len(state.DiscardPile))
	}
}

func TestCanPlayMatchesSuitOrRank(t *testing.T) {
	top := Card{Suit: "heart", Rank: "7"}
	if !canPlay(Card{Suit: "heart", Rank: "2"}, top) {
		t.Fatal("expected same suit playable")
	}
	if !canPlay(Card{Suit: "spade", Rank: "7"}, top) {
		t.Fatal("expected same rank playable")
	}
	if canPlay(Card{Suit: "spade", Rank: "3"}, top) {
		t.Fatal("expected different suit and rank rejected")
	}
}

func testContext() gamecore.Context {
	return gamecore.Context{
		GameID: "onecard",
		Players: []gamecore.Player{
			{ID: "p1", SeatIndex: 0, DisplayName: "P1", Connected: true},
			{ID: "p2", SeatIndex: 1, DisplayName: "P2", Connected: true},
		},
		Now:        time.Date(2026, 7, 21, 1, 0, 0, 0, time.UTC),
		RandomSeed: "seed",
	}
}
