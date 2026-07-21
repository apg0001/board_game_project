package rummikub

import (
	"context"
	"testing"
	"time"

	"board-game-platform/apps/api/internal/gamecore"
)

func TestCreateInitialStateDealsFourteenTiles(t *testing.T) {
	module := NewModule()
	state := module.CreateInitialState(testContext()).(State)
	if len(state.Players[0].Rack) != 14 {
		t.Fatalf("expected 14 tiles, got %d", len(state.Players[0].Rack))
	}
	if len(state.Pool) != 78 {
		t.Fatalf("expected pool 78 for two players, got %d", len(state.Pool))
	}
}

func TestValidGroupAndRun(t *testing.T) {
	group := []Tile{{Color: "black", Number: 7}, {Color: "blue", Number: 7}, {Color: "red", Number: 7}}
	run := []Tile{{Color: "blue", Number: 4}, {Color: "blue", Number: 5}, {Color: "blue", Number: 6}}
	if !validSet(group) {
		t.Fatal("expected valid group")
	}
	if !validSet(run) {
		t.Fatal("expected valid run")
	}
}

func TestInitialMeldRequiresThirtyPoints(t *testing.T) {
	low := []Tile{{Color: "blue", Number: 1}, {Color: "blue", Number: 2}, {Color: "blue", Number: 3}}
	if meldValue(low) >= 30 {
		t.Fatal("expected low meld under 30")
	}
}

func TestRunAllowsJokerToFillGapAfterOne(t *testing.T) {
	run := []Tile{{Color: "blue", Number: 1}, {Color: "blue", Number: 3}, {ID: "joker", Joker: true}}
	if !validSet(run) {
		t.Fatal("expected joker to complete 1-2-3 run")
	}
}

func TestDrawSkipsInactivePlayer(t *testing.T) {
	module := NewModule()
	state := State{
		CurrentPlayerIndex: 0,
		Players: []PlayerState{
			{PlayerID: "p1", Rack: []Tile{}, Active: true},
			{PlayerID: "p2", Rack: []Tile{}, Active: false},
			{PlayerID: "p3", Rack: []Tile{}, Active: true},
		},
		Pool: []Tile{{ID: "blue-1", Color: "blue", Number: 1}},
	}

	result, err := module.ApplyAction(context.Background(), state, gamecore.Action{
		Type:     ActionDraw,
		PlayerID: "p1",
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}

	next := result.State.(State)
	if next.CurrentPlayerIndex != 2 {
		t.Fatalf("expected turn to skip inactive player, got %d", next.CurrentPlayerIndex)
	}
}

func testContext() gamecore.Context {
	return gamecore.Context{
		GameID: "rummikub",
		Players: []gamecore.Player{
			{ID: "p1", SeatIndex: 0, DisplayName: "P1", Connected: true},
			{ID: "p2", SeatIndex: 1, DisplayName: "P2", Connected: true},
		},
		Now:        time.Date(2026, 7, 21, 1, 0, 0, 0, time.UTC),
		RandomSeed: "seed",
	}
}
