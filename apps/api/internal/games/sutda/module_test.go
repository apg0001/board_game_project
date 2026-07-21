package sutda

import (
	"context"
	"testing"
	"time"

	"board-game-platform/apps/api/internal/gamecore"
)

func TestEvaluateRanks(t *testing.T) {
	rank, name := evaluate([]Card{{Month: 3, Gwang: true}, {Month: 8, Gwang: true}})
	if name != "38광땡" || rank < 1000 {
		t.Fatalf("expected 38 gwang, got %s %d", name, rank)
	}
	rank, name = evaluate([]Card{{Month: 1}, {Month: 2}})
	if name != "알리" || rank != 700 {
		t.Fatalf("expected ali, got %s %d", name, rank)
	}
	rank, name = evaluate([]Card{{Month: 4}, {Month: 5}})
	if name != "갑오" || rank != 609 {
		t.Fatalf("expected gap-o, got %s %d", name, rank)
	}
}

func TestCompareHands(t *testing.T) {
	left := []Card{{Month: 10}, {Month: 10}}
	right := []Card{{Month: 1}, {Month: 2}}
	if compare(left, right) <= 0 {
		t.Fatal("expected jang-ttaeng to beat ali")
	}
}

func TestTimeoutCurrentPlayerAdvancesTurn(t *testing.T) {
	module := NewModule()
	state := State{
		CurrentPlayerIndex: 0,
		Players: []PlayerState{
			{PlayerID: "p1", Hand: []Card{{Month: 1}, {Month: 2}}, Active: true},
			{PlayerID: "p2", Hand: []Card{{Month: 3}, {Month: 4}}, Active: true},
			{PlayerID: "p3", Hand: []Card{{Month: 5}, {Month: 6}}, Active: true},
		},
	}

	result, err := module.ApplyTimeout(context.Background(), state, "p1", testContext())
	if err != nil {
		t.Fatal(err)
	}

	next := result.State.(State)
	if next.CurrentPlayerIndex != 1 || next.Finished {
		t.Fatalf("expected turn to advance to p2 without finishing, got %+v", next)
	}
}

func testContext() gamecore.Context {
	return gamecore.Context{
		GameID: "sutda",
		Players: []gamecore.Player{
			{ID: "p1", SeatIndex: 0, DisplayName: "P1", Connected: true},
			{ID: "p2", SeatIndex: 1, DisplayName: "P2", Connected: true},
			{ID: "p3", SeatIndex: 2, DisplayName: "P3", Connected: true},
		},
		Now:        time.Date(2026, 7, 21, 1, 0, 0, 0, time.UTC),
		RandomSeed: "seed",
	}
}
