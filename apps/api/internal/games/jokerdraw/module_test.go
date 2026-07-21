package jokerdraw

import (
	"context"
	"testing"
	"time"

	"board-game-platform/apps/api/internal/gamecore"
)

func TestRemovePairsKeepsUnmatchedAndJoker(t *testing.T) {
	hand := []Card{{ID: "A-1", Rank: "A"}, {ID: "A-2", Rank: "A"}, {ID: "K-1", Rank: "K"}, {ID: "joker", Rank: "joker", Joker: true}}
	result := removePairs(hand)
	if len(result) != 2 {
		t.Fatalf("expected unmatched king and joker, got %d", len(result))
	}
	if result[0].Rank != "K" || !result[1].Joker {
		t.Fatalf("expected stable unmatched order, got %+v", result)
	}
}

func TestShuffledDeckHasJoker(t *testing.T) {
	deck := shuffledDeck("seed")
	if len(deck) != 53 {
		t.Fatalf("expected 53 cards, got %d", len(deck))
	}
	found := false
	for _, card := range deck {
		found = found || card.Joker
	}
	if !found {
		t.Fatal("expected joker")
	}
}

func TestPublicStateDoesNotMutatePrivateHands(t *testing.T) {
	module := NewModule()
	state := State{
		Players: []PlayerState{
			{PlayerID: "p1", Hand: []Card{{ID: "A-1", Rank: "A"}}, Active: true},
			{PlayerID: "p2", Hand: []Card{{ID: "joker", Rank: "joker", Joker: true}}, Active: true},
		},
	}

	public := module.PublicState(state, "p1").(State)
	if len(public.Players[1].Hand) != 1 || public.Players[1].Hand[0].ID != "" {
		t.Fatalf("expected masked opponent hand, got %+v", public.Players[1].Hand)
	}
	if !state.Players[1].Hand[0].Joker {
		t.Fatal("public state must not mutate private joker")
	}
}

func TestFinalizeFindsJokerLoser(t *testing.T) {
	state := finalizeIfComplete(State{
		Players: []PlayerState{
			{PlayerID: "p1", Hand: []Card{}, Out: true, Active: false},
			{PlayerID: "p2", Hand: []Card{{ID: "joker", Rank: "joker", Joker: true}}, Active: true},
		},
	})

	if !state.Finished || state.LoserID != "p2" {
		t.Fatalf("expected joker holder loser, got %+v", state)
	}
}

func TestDrawRemovesPairAndFinishes(t *testing.T) {
	module := NewModule()
	state := State{
		CurrentPlayerIndex: 0,
		Players: []PlayerState{
			{PlayerID: "p1", Hand: []Card{{ID: "A-1", Rank: "A"}}, Active: true},
			{PlayerID: "p2", Hand: []Card{{ID: "A-2", Rank: "A"}, {ID: "joker", Rank: "joker", Joker: true}}, Active: true},
		},
	}

	result, err := module.ApplyAction(context.Background(), state, gamecore.Action{
		Type:     ActionDraw,
		PlayerID: "p1",
		Payload:  map[string]any{"targetPlayerId": "p2", "cardIndex": float64(0)},
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}

	next := result.State.(State)
	if !next.Finished || next.LoserID != "p2" {
		t.Fatalf("expected p2 to lose with joker, got %+v", next)
	}
}

func TestValidateDrawRequiresNextActiveTarget(t *testing.T) {
	module := NewModule()
	state := State{
		CurrentPlayerIndex: 0,
		Players: []PlayerState{
			{PlayerID: "p1", Hand: []Card{{ID: "A-1", Rank: "A"}}, Active: true},
			{PlayerID: "p2", Hand: []Card{{ID: "K-1", Rank: "K"}}, Active: true},
			{PlayerID: "p3", Hand: []Card{{ID: "joker", Rank: "joker", Joker: true}}, Active: true},
		},
	}

	err := module.ValidateAction(context.Background(), state, gamecore.Action{
		Type:     ActionDraw,
		PlayerID: "p1",
		Payload:  map[string]any{"targetPlayerId": "p3", "cardIndex": 0},
	}, testContextWithPlayers("p1", "p2", "p3"))
	if err == nil {
		t.Fatal("expected non-next target to be rejected")
	}
}

func TestTimeoutSetsTimedOutLoser(t *testing.T) {
	module := NewModule()
	state := State{
		CurrentPlayerIndex: 0,
		Players: []PlayerState{
			{PlayerID: "p1", Hand: []Card{{ID: "joker", Rank: "joker", Joker: true}}, Active: true},
			{PlayerID: "p2", Hand: []Card{{ID: "A-1", Rank: "A"}}, Active: true},
		},
	}

	result, err := module.ApplyTimeout(context.Background(), state, "p1", testContext())
	if err != nil {
		t.Fatal(err)
	}
	next := result.State.(State)
	if !next.Finished || next.LoserID != "p1" {
		t.Fatalf("expected timed out player loser, got %+v", next)
	}
}

func testContext() gamecore.Context {
	return testContextWithPlayers("p1", "p2")
}

func testContextWithPlayers(ids ...string) gamecore.Context {
	players := make([]gamecore.Player, 0, len(ids))
	for index, id := range ids {
		players = append(players, gamecore.Player{ID: gamecore.PlayerID(id), SeatIndex: index, DisplayName: id, Connected: true})
	}
	return gamecore.Context{
		GameID:     "jokerdraw",
		Players:    players,
		Now:        time.Date(2026, 7, 21, 1, 0, 0, 0, time.UTC),
		RandomSeed: "seed",
	}
}
