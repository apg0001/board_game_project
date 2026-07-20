package davinci

import (
	"context"
	"testing"
	"time"

	"board-game-platform/apps/api/internal/gamecore"
)

func TestCreateInitialStateDealsTiles(t *testing.T) {
	module := NewModule()
	state := module.CreateInitialState(testContext()).(State)

	if len(state.Players) != 2 {
		t.Fatalf("expected two players, got %d", len(state.Players))
	}
	if len(state.Players[0].Tiles) != 4 {
		t.Fatalf("expected four tiles for two-player game, got %d", len(state.Players[0].Tiles))
	}
}

func TestPublicStateHidesOpponentTiles(t *testing.T) {
	module := NewModule()
	state := module.CreateInitialState(testContext()).(State)

	public := module.PublicState(state, "p1").(State)

	if public.Players[0].Tiles[0].Value == -1 {
		t.Fatal("own tile should be visible")
	}
	if public.Players[1].Tiles[0].Value != -1 {
		t.Fatal("opponent hidden tile should be masked")
	}
}

func TestPassAdvancesTurn(t *testing.T) {
	module := NewModule()
	state := module.CreateInitialState(testContext()).(State)

	result, err := module.ApplyAction(context.Background(), state, gamecore.Action{
		Type:     ActionPass,
		PlayerID: "p1",
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}

	next := result.State.(State)
	if next.CurrentPlayerIndex != 1 {
		t.Fatalf("expected next player index 1, got %d", next.CurrentPlayerIndex)
	}
}

func TestWrongGuessRevealsOwnTile(t *testing.T) {
	module := NewModule()
	state := module.CreateInitialState(testContext()).(State)

	result, err := module.ApplyAction(context.Background(), state, gamecore.Action{
		Type:     ActionGuess,
		PlayerID: "p1",
		Payload: map[string]any{
			"targetPlayerId": "p2",
			"tileIndex":      float64(0),
			"color":          "black",
			"value":          float64(99),
		},
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}

	next := result.State.(State)
	if !next.Players[0].Tiles[0].Revealed {
		t.Fatal("wrong guess should reveal first hidden own tile")
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
	for _, tile := range next.Players[0].Tiles {
		if !tile.Revealed {
			t.Fatal("timed out player tiles should be revealed")
		}
	}
	if !next.Finished {
		t.Fatal("two-player game should finish after one forfeit")
	}
}

func testContext() gamecore.Context {
	return gamecore.Context{
		SessionID: "s1",
		GameID:    "davinci",
		Players: []gamecore.Player{
			{ID: "p1", SeatIndex: 0, DisplayName: "P1", Connected: true},
			{ID: "p2", SeatIndex: 1, DisplayName: "P2", Connected: true},
		},
		Now:        time.Date(2026, 7, 20, 1, 0, 0, 0, time.UTC),
		RandomSeed: "seed",
	}
}
