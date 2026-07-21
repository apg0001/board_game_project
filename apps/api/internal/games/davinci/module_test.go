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
	if state.PendingTile == nil || state.PendingOwnerID != "p1" {
		t.Fatal("first player should draw a pending tile at turn start")
	}
}

func TestCreateInitialStateDealsFourTilesForThreePlayers(t *testing.T) {
	module := NewModule()
	ctx := testContext()
	ctx.Players = append(ctx.Players, gamecore.Player{ID: "p3", SeatIndex: 2, DisplayName: "P3", Connected: true})
	state := module.CreateInitialState(ctx).(State)

	if len(state.Players[0].Tiles) != 4 {
		t.Fatalf("expected four tiles for three-player game, got %d", len(state.Players[0].Tiles))
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

func TestPublicStateDoesNotMutatePrivateState(t *testing.T) {
	module := NewModule()
	state := module.CreateInitialState(testContext()).(State)
	originalOpponentValue := state.Players[1].Tiles[0].Value

	public := module.PublicState(state, "p1").(State)

	if public.Players[1].Tiles[0].Value != -1 {
		t.Fatal("opponent tile should be masked in public state")
	}
	if state.Players[1].Tiles[0].Value != originalOpponentValue {
		t.Fatal("public state must not mutate the private state")
	}
}

func TestCorrectGuessKeepsTurnAndAllowsEndTurn(t *testing.T) {
	module := NewModule()
	state := module.CreateInitialState(testContext()).(State)
	targetTile := state.Players[1].Tiles[0]

	result, err := module.ApplyAction(context.Background(), state, gamecore.Action{
		Type:     ActionGuess,
		PlayerID: "p1",
		Payload: map[string]any{
			"targetPlayerId": "p2",
			"tileIndex":      float64(0),
			"color":          targetTile.Color,
			"value":          float64(targetTile.Value),
		},
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}

	next := result.State.(State)
	if next.CurrentPlayerIndex != 0 {
		t.Fatalf("correct guess should keep current player turn, got %d", next.CurrentPlayerIndex)
	}
	if !next.CanEndTurn {
		t.Fatal("correct guess should allow the player to end their turn")
	}
	if !next.Players[1].Tiles[0].Revealed {
		t.Fatal("correctly guessed tile should be revealed")
	}
}

func TestEndTurnInsertsPendingTileHiddenAndAdvancesTurn(t *testing.T) {
	module := NewModule()
	state := module.CreateInitialState(testContext()).(State)
	pending := *state.PendingTile
	state.CanEndTurn = true

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
	if !hasTile(next.Players[0].Tiles, pending, false) {
		t.Fatal("ending a successful turn should insert the drawn tile hidden")
	}
	if next.PendingOwnerID != "p2" || next.PendingTile == nil {
		t.Fatal("next player should draw a new pending tile")
	}
}

func TestWrongGuessRevealsPendingTile(t *testing.T) {
	module := NewModule()
	state := module.CreateInitialState(testContext()).(State)
	pending := *state.PendingTile

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
	if !hasTile(next.Players[0].Tiles, pending, true) {
		t.Fatal("wrong guess should reveal and insert the pending tile")
	}
	if next.CurrentPlayerIndex != 1 {
		t.Fatalf("wrong guess should advance turn, got %d", next.CurrentPlayerIndex)
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

func hasTile(tiles []Tile, target Tile, revealed bool) bool {
	for _, tile := range tiles {
		if tile.Color == target.Color && tile.Value == target.Value && tile.Revealed == revealed {
			return true
		}
	}
	return false
}
