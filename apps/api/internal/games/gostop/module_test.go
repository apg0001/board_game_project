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

func TestScoreRainBrightThreeGwang(t *testing.T) {
	cards := []Card{
		{Kind: "bright"},
		{Kind: "bright"},
		{Kind: "bright", Tags: []string{"rain"}},
	}
	if score(cards) != 2 {
		t.Fatalf("expected rain three-gwang to score 2, got %d", score(cards))
	}
}

func TestScoreGodoriAndRibbonSets(t *testing.T) {
	cards := []Card{
		{Month: 2, Kind: "animal", Tags: []string{"bird"}},
		{Month: 4, Kind: "animal", Tags: []string{"bird"}},
		{Month: 8, Kind: "animal", Tags: []string{"bird"}},
		{Kind: "ribbon", Tags: []string{"red"}},
		{Kind: "ribbon", Tags: []string{"red"}},
		{Kind: "ribbon", Tags: []string{"red"}},
		{Kind: "ribbon", Tags: []string{"blue"}},
		{Kind: "ribbon", Tags: []string{"blue"}},
		{Kind: "ribbon", Tags: []string{"blue"}},
	}
	if score(cards) != 13 {
		t.Fatalf("expected godori 5 + six ribbons 2 + two ribbon sets 6 = 13, got %d", score(cards))
	}
}

func TestScoreDoubleJunk(t *testing.T) {
	cards := []Card{
		{Kind: "junk", JunkValue: 2},
		{Kind: "junk", JunkValue: 2},
		{Kind: "junk"}, {Kind: "junk"}, {Kind: "junk"}, {Kind: "junk"}, {Kind: "junk"}, {Kind: "junk"},
	}
	if score(cards) != 1 {
		t.Fatalf("expected double junk to count as ten junk and score 1, got %d", score(cards))
	}
}

func TestStandardDeckHasTaggedScoringCards(t *testing.T) {
	deck := standardDeck()
	if len(deck) != 48 {
		t.Fatalf("expected 48 cards, got %d", len(deck))
	}
	required := map[string]bool{"2-animal": false, "3-ribbon": false, "12-bright": false, "11-junk-double": false}
	for _, card := range deck {
		if _, ok := required[card.ID]; ok {
			required[card.ID] = true
		}
	}
	for id, ok := range required {
		if !ok {
			t.Fatalf("expected standard deck to include %s", id)
		}
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

func TestFinalScoreAppliesGoAndPenaltyMultipliers(t *testing.T) {
	module := NewModule()
	state := State{
		Players: []PlayerState{
			{
				PlayerID: "p1",
				Score:    3,
				GoCount:  3,
				Captured: append(
					[]Card{{Kind: "bright"}, {Kind: "bright"}, {Kind: "bright"}},
					[]Card{
						{Kind: "junk"}, {Kind: "junk"}, {Kind: "junk"}, {Kind: "junk"}, {Kind: "junk"},
						{Kind: "junk"}, {Kind: "junk"}, {Kind: "junk"}, {Kind: "junk"}, {Kind: "junk"},
					}...,
				),
				Active: true,
			},
			{
				PlayerID: "p2",
				GoCount:  1,
				Captured: []Card{{Kind: "junk"}, {Kind: "junk"}, {Kind: "junk"}, {Kind: "junk"}, {Kind: "junk"}},
				Active:   true,
			},
		},
	}

	done := finishWithWinner(state, "p1")
	if done.Players[0].FinalScore != 80 {
		t.Fatalf("expected (3+2) * 3go/pibak/gwangbak/gobak multipliers = 80, got %+v", done.Players[0])
	}
	for _, tag := range []string{"3고", "피박", "광박", "고박"} {
		if !hasPenaltyTag(done.Players[0].PenaltyTags, tag) {
			t.Fatalf("expected penalty tag %s in %+v", tag, done.Players[0].PenaltyTags)
		}
	}
	results := module.CalculateResult(done, testContext())
	if results[0].Score != 80 || results[0].Outcome != gamecore.OutcomeWin {
		t.Fatalf("expected final score to be recorded in result, got %+v", results[0])
	}
}

func TestCaptureMonthTakesAllMatchingFieldCards(t *testing.T) {
	player := &PlayerState{PlayerID: "p1"}
	field := []Card{
		{ID: "5-junk-a", Month: 5, Kind: "junk"},
		{ID: "5-junk-b", Month: 5, Kind: "junk"},
		{ID: "9-junk", Month: 9, Kind: "junk"},
	}
	captureMonth(player, &field, Card{ID: "5-bright", Month: 5, Kind: "bright"})

	if len(player.Captured) != 3 {
		t.Fatalf("expected all three month-5 cards captured, got %+v", player.Captured)
	}
	if len(field) != 1 || field[0].ID != "9-junk" {
		t.Fatalf("expected only the unrelated month-9 card left on the field, got %+v", field)
	}
}

func TestApplyTimeoutSkipsCurrentPlayerWithoutEndingGameForOthers(t *testing.T) {
	module := NewModule()
	state := State{
		CurrentPlayerIndex: 1,
		Players: []PlayerState{
			{PlayerID: "p1", Active: true},
			{PlayerID: "p2", Active: true},
			{PlayerID: "p3", Active: true},
		},
	}

	result, err := module.ApplyTimeout(context.Background(), state, "p2", testContext())
	if err != nil {
		t.Fatal(err)
	}
	next := result.State.(State)
	if next.Finished {
		t.Fatal("game should continue while two or more players remain active")
	}
	if next.CurrentPlayerIndex != 2 {
		t.Fatalf("expected turn to skip the timed-out current player, got index %d", next.CurrentPlayerIndex)
	}
	if next.Players[1].Active {
		t.Fatal("timed-out player should be marked inactive")
	}
}

func TestApplyTimeoutDoesNotEndGameForNonCurrentPlayer(t *testing.T) {
	module := NewModule()
	state := State{
		CurrentPlayerIndex: 0,
		Players: []PlayerState{
			{PlayerID: "p1", Active: true},
			{PlayerID: "p2", Active: true},
			{PlayerID: "p3", Active: true},
		},
	}

	result, err := module.ApplyTimeout(context.Background(), state, "p3", testContext())
	if err != nil {
		t.Fatal(err)
	}
	next := result.State.(State)
	if next.Finished {
		t.Fatal("a non-current player timing out should not end the game for everyone else")
	}
	if next.CurrentPlayerIndex != 0 {
		t.Fatal("turn should not move when the timed-out player was not holding the turn")
	}
}

func hasPenaltyTag(tags []string, expected string) bool {
	for _, tag := range tags {
		if tag == expected {
			return true
		}
	}
	return false
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
