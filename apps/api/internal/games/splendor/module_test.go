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

func TestBuyPayloadAcceptsIntegerMarketIndex(t *testing.T) {
	index, err := buyPayload(map[string]any{"marketIndex": 0})
	if err != nil {
		t.Fatal(err)
	}
	if index != 0 {
		t.Fatalf("expected market index 0, got %d", index)
	}
}

func TestPublicStateMasksDeckAndDoesNotMutatePrivateState(t *testing.T) {
	module := NewModule()
	state := module.CreateInitialState(testContext()).(State)
	public := module.PublicState(state, "p1").(State)

	if len(public.Deck) != len(state.Deck) {
		t.Fatalf("expected masked deck length %d, got %d", len(state.Deck), len(public.Deck))
	}
	if len(public.Deck) > 0 && public.Deck[0].ID != "" {
		t.Fatalf("expected hidden deck card, got %+v", public.Deck[0])
	}
	if len(state.Deck) > 0 && state.Deck[0].ID == "" {
		t.Fatal("public state must not mutate private deck")
	}

	public.Bank["white"] = 99
	public.Market[0].Cost["blue"] = 99
	if state.Bank["white"] == 99 || state.Market[0].Cost["blue"] == 99 {
		t.Fatal("public state must not share mutable maps with private state")
	}
}

func TestTimeoutReturnsTokensToBank(t *testing.T) {
	module := NewModule()
	state := module.CreateInitialState(testContext()).(State)
	state.Players[0].Tokens["white"] = 2
	state.Bank["white"] -= 2

	result, err := module.ApplyTimeout(context.Background(), state, "p1", testContext())
	if err != nil {
		t.Fatal(err)
	}

	next := result.State.(State)
	if next.Players[0].Tokens["white"] != 0 || next.Bank["white"] != 4 {
		t.Fatalf("expected timed out player's tokens returned, got tokens=%d bank=%d", next.Players[0].Tokens["white"], next.Bank["white"])
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
