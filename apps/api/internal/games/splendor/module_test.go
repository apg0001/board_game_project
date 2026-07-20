package splendor

import (
	"context"
	"testing"
	"time"

	"board-game-platform/apps/api/internal/gamecore"
)

func TestCreateInitialStateBuildsMarket(t *testing.T) {
	module := NewModule()
	state := module.CreateInitialState(testContext()).(State)
	if len(state.Market) != 4 {
		t.Fatalf("expected 4 market cards, got %d", len(state.Market))
	}
	if state.Bank["white"] != 4 {
		t.Fatalf("expected two-player bank amount 4, got %d", state.Bank["white"])
	}
}

func TestTakeTokenAdvancesTurn(t *testing.T) {
	module := NewModule()
	state := module.CreateInitialState(testContext()).(State)
	result, err := module.ApplyAction(context.Background(), state, gamecore.Action{
		Type:     ActionTakeToken,
		PlayerID: "p1",
		Payload:  map[string]any{"color": "white"},
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}
	next := result.State.(State)
	if next.Players[0].Tokens["white"] != 1 {
		t.Fatal("expected token")
	}
	if next.CurrentPlayerIndex != 1 {
		t.Fatalf("expected next player, got %d", next.CurrentPlayerIndex)
	}
}

func TestBuyCardUsesTokens(t *testing.T) {
	module := NewModule()
	state := module.CreateInitialState(testContext()).(State)
	card := Card{ID: "test", Color: "white", Points: 1, Cost: map[string]int{"blue": 1}}
	state.Market[0] = card
	state.Players[0].Tokens["blue"] = 1
	result, err := module.ApplyAction(context.Background(), state, gamecore.Action{
		Type:     ActionBuyCard,
		PlayerID: "p1",
		Payload:  map[string]any{"marketIndex": float64(0)},
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}
	next := result.State.(State)
	if next.Players[0].Score != 1 || next.Players[0].Bonuses["white"] != 1 {
		t.Fatalf("expected bought card effects, got %+v", next.Players[0])
	}
}

func testContext() gamecore.Context {
	return gamecore.Context{
		GameID: "splendor",
		Players: []gamecore.Player{
			{ID: "p1", SeatIndex: 0, DisplayName: "P1", Connected: true},
			{ID: "p2", SeatIndex: 1, DisplayName: "P2", Connected: true},
		},
		Now:        time.Date(2026, 7, 20, 1, 0, 0, 0, time.UTC),
		RandomSeed: "seed",
	}
}
