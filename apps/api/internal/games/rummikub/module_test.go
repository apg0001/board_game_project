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

func TestInitialMeldJokerUsesRepresentedTileValue(t *testing.T) {
	lowGroupWithJoker := []Tile{
		{ID: "black-1", Color: "black", Number: 1},
		{ID: "blue-1", Color: "blue", Number: 1},
		{ID: "joker-1", Joker: true},
	}
	if meldValue(lowGroupWithJoker) != 3 {
		t.Fatalf("expected joker to represent a one-point tile in group, got %d", meldValue(lowGroupWithJoker))
	}

	runWithJoker := []Tile{
		{ID: "blue-11", Color: "blue", Number: 11},
		{ID: "blue-13", Color: "blue", Number: 13},
		{ID: "joker-1", Joker: true},
	}
	if meldValue(runWithJoker) != 36 {
		t.Fatalf("expected joker to represent blue 12 in run, got %d", meldValue(runWithJoker))
	}
}

func TestInitialMeldRejectsLowJokerGroupUnderThirty(t *testing.T) {
	module := NewModule()
	state := State{
		Players: []PlayerState{
			{
				PlayerID: "p1",
				Rack: []Tile{
					{ID: "black-1", Color: "black", Number: 1},
					{ID: "blue-1", Color: "blue", Number: 1},
					{ID: "joker-1", Joker: true},
				},
				Active: true,
			},
			{PlayerID: "p2", Rack: []Tile{}, Active: true},
		},
	}

	err := module.ValidateAction(context.Background(), state, gamecore.Action{
		Type:     ActionMeld,
		PlayerID: "p1",
		Payload: map[string]any{
			"groups": []any{
				[]any{"black-1", "blue-1", "joker-1"},
			},
		},
	}, testContext())
	if err == nil {
		t.Fatal("expected low joker group to remain under the initial 30-point requirement")
	}
}

func TestInitialMeldCanUseMultipleGroupsTotalingThirty(t *testing.T) {
	module := NewModule()
	state := State{
		Players: []PlayerState{
			{
				PlayerID: "p1",
				Rack: []Tile{
					{ID: "black-8", Color: "black", Number: 8},
					{ID: "blue-8", Color: "blue", Number: 8},
					{ID: "red-8", Color: "red", Number: 8},
					{ID: "blue-1", Color: "blue", Number: 1},
					{ID: "blue-2", Color: "blue", Number: 2},
					{ID: "blue-3", Color: "blue", Number: 3},
				},
				Active: true,
			},
			{PlayerID: "p2", Rack: []Tile{}, Active: true},
		},
	}

	result, err := module.ApplyAction(context.Background(), state, gamecore.Action{
		Type:     ActionMeld,
		PlayerID: "p1",
		Payload: map[string]any{
			"groups": []any{
				[]any{"black-8", "blue-8", "red-8"},
				[]any{"blue-1", "blue-2", "blue-3"},
			},
		},
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}

	next := result.State.(State)
	if !next.Players[0].InitialMelded || len(next.Table) != 2 {
		t.Fatalf("expected two groups to be registered as first meld, got player=%+v table=%+v", next.Players[0], next.Table)
	}
	if len(next.Players[0].Rack) != 0 || !next.Finished {
		t.Fatalf("expected all rack tiles used and game finished, got rack=%+v finished=%v", next.Players[0].Rack, next.Finished)
	}
}

func TestMeldRejectsTileUsedTwiceAcrossGroups(t *testing.T) {
	module := NewModule()
	state := State{
		Players: []PlayerState{
			{
				PlayerID: "p1",
				Rack: []Tile{
					{ID: "black-8", Color: "black", Number: 8},
					{ID: "blue-8", Color: "blue", Number: 8},
					{ID: "red-8", Color: "red", Number: 8},
					{ID: "orange-8", Color: "orange", Number: 8},
				},
				Active: true,
			},
			{PlayerID: "p2", Rack: []Tile{}, Active: true},
		},
	}

	err := module.ValidateAction(context.Background(), state, gamecore.Action{
		Type:     ActionMeld,
		PlayerID: "p1",
		Payload: map[string]any{
			"groups": []any{
				[]any{"black-8", "blue-8", "red-8"},
				[]any{"black-8", "blue-8", "orange-8"},
			},
		},
	}, testContext())
	if err == nil {
		t.Fatal("expected duplicate tile usage across groups to be rejected")
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

func TestCalculateResultDoesNotRankForfeitedPlayerAsWinner(t *testing.T) {
	module := NewModule()
	state := State{
		Finished: true,
		Players: []PlayerState{
			{PlayerID: "p1", Rack: []Tile{{Color: "blue", Number: 2}}, Active: false},
			{PlayerID: "p2", Rack: []Tile{{Color: "blue", Number: 1}, {Color: "red", Number: 5}}, Active: true},
		},
	}

	results := module.CalculateResult(state, testContext())
	for _, result := range results {
		if result.PlayerID == "p1" && result.Outcome == gamecore.OutcomeWin {
			t.Fatal("a forfeited/disconnected player must not be ranked as the winner over a connected player")
		}
		if result.PlayerID == "p2" && result.Outcome != gamecore.OutcomeWin {
			t.Fatal("the connected player who stayed active should win when the other player forfeited")
		}
	}
}

func TestCalculateResultAwardsWinnerSumOfOpponentPenalties(t *testing.T) {
	module := NewModule()
	state := State{
		Finished: true,
		Players: []PlayerState{
			{PlayerID: "p1", Rack: []Tile{}, Active: true},
			{PlayerID: "p2", Rack: []Tile{{Color: "blue", Number: 1}, {Color: "red", Number: 5}}, Active: true},
			{PlayerID: "p3", Rack: []Tile{{Joker: true}}, Active: true},
		},
	}

	results := module.CalculateResult(state, testContext())
	if results[0].PlayerID != "p1" || results[0].Score != 36 || results[0].Outcome != gamecore.OutcomeWin {
		t.Fatalf("expected rack-empty winner to gain opponent penalties, got %+v", results[0])
	}
	for _, result := range results {
		if result.PlayerID == "p2" && result.Score != -6 {
			t.Fatalf("expected p2 score -6, got %+v", result)
		}
		if result.PlayerID == "p3" && result.Score != -30 {
			t.Fatalf("expected p3 joker penalty -30, got %+v", result)
		}
	}
}

func TestRearrangeCanReplaceJokerAndReuseItInValidGroup(t *testing.T) {
	module := NewModule()
	state := State{
		Players: []PlayerState{
			{
				PlayerID:      "p1",
				InitialMelded: true,
				Active:        true,
				Rack:          []Tile{{ID: "blue-2", Color: "blue", Number: 2}},
			},
			{PlayerID: "p2", Active: true},
		},
		Table: [][]Tile{
			{
				{ID: "blue-1", Color: "blue", Number: 1},
				{ID: "joker-1", Joker: true},
				{ID: "blue-3", Color: "blue", Number: 3},
			},
			{
				{ID: "black-7", Color: "black", Number: 7},
				{ID: "blue-7", Color: "blue", Number: 7},
				{ID: "red-7", Color: "red", Number: 7},
			},
		},
	}

	result, err := module.ApplyAction(context.Background(), state, gamecore.Action{
		Type:     ActionRearrange,
		PlayerID: "p1",
		Payload: map[string]any{
			"groups": []any{
				[]any{"blue-1", "blue-2", "blue-3"},
				[]any{"black-7", "blue-7", "red-7", "joker-1"},
			},
		},
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}

	next := result.State.(State)
	if len(next.Players[0].Rack) != 0 {
		t.Fatalf("expected replacement tile to leave rack, got %+v", next.Players[0].Rack)
	}
	if len(next.Table) != 2 || !validSet(next.Table[0]) || !validSet(next.Table[1]) {
		t.Fatalf("expected valid rearranged table, got %+v", next.Table)
	}
}

func TestRearrangeRejectsMissingOriginalTableTile(t *testing.T) {
	module := NewModule()
	state := State{
		Players: []PlayerState{
			{
				PlayerID:      "p1",
				InitialMelded: true,
				Active:        true,
				Rack:          []Tile{{ID: "blue-2", Color: "blue", Number: 2}},
			},
			{PlayerID: "p2", Active: true},
		},
		Table: [][]Tile{{
			{ID: "blue-1", Color: "blue", Number: 1},
			{ID: "joker-1", Joker: true},
			{ID: "blue-3", Color: "blue", Number: 3},
		}},
	}

	err := module.ValidateAction(context.Background(), state, gamecore.Action{
		Type:     ActionRearrange,
		PlayerID: "p1",
		Payload: map[string]any{
			"groups": []any{
				[]any{"blue-1", "blue-2", "blue-3"},
			},
		},
	}, testContext())
	if err == nil {
		t.Fatal("expected rearrange without original joker to be rejected")
	}
}

func TestRearrangeRequiresInitialMeld(t *testing.T) {
	module := NewModule()
	state := State{
		Players: []PlayerState{
			{PlayerID: "p1", Active: true, Rack: []Tile{{ID: "blue-2", Color: "blue", Number: 2}}},
			{PlayerID: "p2", Active: true},
		},
		Table: [][]Tile{{
			{ID: "blue-1", Color: "blue", Number: 1},
			{ID: "joker-1", Joker: true},
			{ID: "blue-3", Color: "blue", Number: 3},
		}},
	}

	err := module.ValidateAction(context.Background(), state, gamecore.Action{
		Type:     ActionRearrange,
		PlayerID: "p1",
		Payload: map[string]any{
			"groups": []any{
				[]any{"blue-1", "blue-2", "blue-3", "joker-1"},
			},
		},
	}, testContext())
	if err == nil {
		t.Fatal("expected rearrange before initial meld to be rejected")
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
