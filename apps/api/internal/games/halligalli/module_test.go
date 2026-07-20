package halligalli

import (
	"context"
	"testing"
	"time"

	"board-game-platform/apps/api/internal/gamecore"
)

func TestCreateInitialStateDealsDeck(t *testing.T) {
	module := NewModule()
	state := module.CreateInitialState(testContext()).(State)
	if len(state.Players) != 2 {
		t.Fatalf("expected two players, got %d", len(state.Players))
	}
	if len(state.Players[0].Deck) == 0 {
		t.Fatal("expected dealt cards")
	}
}

func TestFlipAdvancesTurn(t *testing.T) {
	module := NewModule()
	state := module.CreateInitialState(testContext()).(State)
	result, err := module.ApplyAction(context.Background(), state, gamecore.Action{
		Type:     ActionFlip,
		PlayerID: "p1",
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}
	next := result.State.(State)
	if next.CurrentPlayerIndex != 1 {
		t.Fatalf("expected next player, got %d", next.CurrentPlayerIndex)
	}
}

func TestTimeoutForfeitsPlayer(t *testing.T) {
	module := NewModule()
	state := module.CreateInitialState(testContext()).(State)

	result, err := module.ApplyTimeout(context.Background(), state, "p1", testContext())
	if err != nil {
		t.Fatal(err)
	}

	next := result.State.(State)
	if next.Players[0].Active {
		t.Fatal("timed out player should be inactive")
	}
	if len(next.Players[0].Deck) != 0 {
		t.Fatal("timed out player deck should be cleared")
	}
	if !next.Finished {
		t.Fatal("two-player game should finish after one forfeit")
	}
}

func testContext() gamecore.Context {
	return gamecore.Context{
		GameID: "halli-galli",
		Players: []gamecore.Player{
			{ID: "p1", SeatIndex: 0, DisplayName: "P1", Connected: true},
			{ID: "p2", SeatIndex: 1, DisplayName: "P2", Connected: true},
		},
		Now:        time.Date(2026, 7, 20, 1, 0, 0, 0, time.UTC),
		RandomSeed: "seed",
	}
}
