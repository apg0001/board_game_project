package gostop

import (
	"context"
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

func TestScoringPlayWaitsForGoStopDecision(t *testing.T) {
	module := NewModule()
	state := State{
		CurrentPlayerIndex: 0,
		Players: []PlayerState{
			{
				PlayerID: "p1",
				Hand:     []Card{{ID: "8-bright", Month: 8, Kind: "bright"}, {ID: "9-junk", Month: 9, Kind: "junk"}},
				Captured: []Card{{ID: "1-bright", Kind: "bright"}, {ID: "3-bright", Kind: "bright"}},
				Active:   true,
			},
			{PlayerID: "p2", Hand: []Card{{ID: "2-junk", Month: 2, Kind: "junk"}}, Active: true},
		},
		Field: []Card{{ID: "8-junk", Month: 8, Kind: "junk"}},
		Deck: []Card{
			{ID: "12-junk", Month: 12, Kind: "junk"},
			{ID: "11-junk", Month: 11, Kind: "junk"},
		},
	}

	result, err := module.ApplyAction(context.Background(), state, gamecore.Action{
		Type:     ActionPlay,
		PlayerID: "p1",
		Payload:  map[string]any{"cardId": "8-bright"},
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}

	next := result.State.(State)
	if !next.AwaitingDecision || next.CurrentPlayerIndex != 0 {
		t.Fatalf("expected p1 to choose go/stop before turn advances, got %+v", next)
	}
}

func TestGoDecisionAdvancesTurn(t *testing.T) {
	module := NewModule()
	state := State{
		CurrentPlayerIndex: 0,
		AwaitingDecision:   true,
		Players: []PlayerState{
			{PlayerID: "p1", Score: 3, Active: true},
			{PlayerID: "p2", Score: 0, Active: true},
		},
	}

	result, err := module.ApplyAction(context.Background(), state, gamecore.Action{
		Type:     ActionGo,
		PlayerID: "p1",
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}

	next := result.State.(State)
	if next.AwaitingDecision || next.CurrentPlayerIndex != 1 || next.Players[0].GoCount != 1 {
		t.Fatalf("expected go to clear decision and advance turn, got %+v", next)
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
