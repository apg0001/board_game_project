package dalmuti

import (
	"context"
	"testing"
	"time"

	"board-game-platform/apps/api/internal/gamecore"
)

func TestCreateInitialStateDealsWholeDeck(t *testing.T) {
	module := NewModule()
	state := module.CreateInitialState(testContext()).(State)
	total := 0
	for _, player := range state.Players {
		total += len(player.Hand)
	}
	if total != 80 {
		t.Fatalf("expected 80 dealt cards, got %d", total)
	}
}

func TestPlayRequiresSameCountAndStrongerRank(t *testing.T) {
	module := NewModule()
	state := fixedState()

	_, err := module.ApplyAction(context.Background(), state, gamecore.Action{
		Type:     ActionPlay,
		PlayerID: "p1",
		Payload:  map[string]any{"rank": float64(10), "count": float64(2)},
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}

	err = module.ValidateAction(context.Background(), State{
		CurrentPlayerIndex: 1,
		Players:            state.Players,
		CurrentTrick:       Trick{Rank: 10, Count: 2, PlayerID: "p1"},
	}, gamecore.Action{
		Type:     ActionPlay,
		PlayerID: "p2",
		Payload:  map[string]any{"rank": float64(11), "count": float64(2)},
	}, testContext())
	if err == nil {
		t.Fatal("expected weaker rank to be rejected")
	}
}

func TestJesterCanCompleteSetAsWild(t *testing.T) {
	state := fixedState()
	err := validatePlay(state.Players[1], Trick{Rank: 10, Count: 2, PlayerID: "p1"}, PlayPayload{Rank: 9, Count: 2})
	if err != nil {
		t.Fatal(err)
	}
}

func fixedState() State {
	return State{
		CurrentPlayerIndex: 0,
		Players: []PlayerState{
			{PlayerID: "p1", Hand: []Card{{ID: "10-1", Rank: 10}, {ID: "10-2", Rank: 10}}, Active: true},
			{PlayerID: "p2", Hand: []Card{{ID: "9-1", Rank: 9}, {ID: "jester-1", Rank: 13}}, Active: true},
			{PlayerID: "p3", Hand: []Card{{ID: "8-1", Rank: 8}, {ID: "8-2", Rank: 8}}, Active: true},
			{PlayerID: "p4", Hand: []Card{{ID: "7-1", Rank: 7}, {ID: "7-2", Rank: 7}}, Active: true},
		},
	}
}

func testContext() gamecore.Context {
	return gamecore.Context{
		GameID: "dalmuti",
		Players: []gamecore.Player{
			{ID: "p1", SeatIndex: 0, DisplayName: "P1", Connected: true},
			{ID: "p2", SeatIndex: 1, DisplayName: "P2", Connected: true},
			{ID: "p3", SeatIndex: 2, DisplayName: "P3", Connected: true},
			{ID: "p4", SeatIndex: 3, DisplayName: "P4", Connected: true},
		},
		Now:        time.Date(2026, 7, 21, 1, 0, 0, 0, time.UTC),
		RandomSeed: "seed",
	}
}
