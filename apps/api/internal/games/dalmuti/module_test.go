package dalmuti

import (
	"context"
	"testing"
	"time"

	"board-game-platform/apps/api/internal/gamecore"
)

func TestCreateInitialStateDealsWholeDeck(t *testing.T) {
	module := NewModule()
	state := module.CreateInitialState(testContext()).(State)
	total := 0
	for _, player := range state.Players {
		total += len(player.Hand)
	}
	if total != 80 {
		t.Fatalf("expected 80 dealt cards, got %d", total)
	}
}

func TestShuffledDeckKeepsOfficialRankCounts(t *testing.T) {
	deck := shuffledDeck("rank-counts")
	if len(deck) != 80 {
		t.Fatalf("expected 80 card dalmuti deck, got %d", len(deck))
	}
	counts := map[int]int{}
	for _, card := range deck {
		counts[card.Rank]++
	}
	for rank := 1; rank <= 12; rank++ {
		if counts[rank] != rank {
			t.Fatalf("expected rank %d to have %d copies, got %d", rank, rank, counts[rank])
		}
	}
	if counts[13] != 2 {
		t.Fatalf("expected two jesters, got %d", counts[13])
	}
}

func TestOpeningTaxExchangesPeonBestCardsForDalmutiSelectedCards(t *testing.T) {
	state := applyOpeningTaxAndRevolution(State{
		Players: []PlayerState{
			{PlayerID: "greater-dalmuti", Hand: []Card{{ID: "9-1", Rank: 9}, {ID: "12-1", Rank: 12}, {ID: "jester-1", Rank: 13}}, Active: true},
			{PlayerID: "lesser-dalmuti", Hand: []Card{{ID: "8-1", Rank: 8}, {ID: "11-1", Rank: 11}}, Active: true},
			{PlayerID: "lesser-peon", Hand: []Card{{ID: "2-1", Rank: 2}, {ID: "10-1", Rank: 10}}, Active: true},
			{PlayerID: "greater-peon", Hand: []Card{{ID: "1-1", Rank: 1}, {ID: "3-1", Rank: 3}, {ID: "12-2", Rank: 12}}, Active: true},
		},
	})

	if state.PendingTax == nil || state.PendingTax.CurrentChooserID != "greater-dalmuti" || state.TaxApplied || state.Revolution {
		t.Fatalf("expected pending taxation without revolution, got %+v", state)
	}
	afterGreater := applyTaxChoice(state, "greater-dalmuti", []string{"12-1", "jester-1"})
	if afterGreater.PendingTax == nil || afterGreater.PendingTax.CurrentChooserID != "lesser-dalmuti" {
		t.Fatalf("expected lesser dalmuti tax choice next, got %+v", afterGreater.PendingTax)
	}
	done := applyTaxChoice(afterGreater, "lesser-dalmuti", []string{"11-1"})
	if !done.TaxApplied || done.PendingTax != nil || done.Revolution {
		t.Fatalf("expected taxation to complete without revolution, got %+v", done)
	}
	if !hasRank(done.Players[0].Hand, 1) || !hasRank(done.Players[0].Hand, 3) {
		t.Fatalf("greater dalmuti should receive greater peon's best two cards, got %+v", done.Players[0].Hand)
	}
	if !hasRank(done.Players[3].Hand, 12) || !hasRank(done.Players[3].Hand, 13) {
		t.Fatalf("greater peon should receive greater dalmuti's selected cards, got %+v", done.Players[3].Hand)
	}
	if !hasRank(done.Players[1].Hand, 2) || !hasRank(done.Players[2].Hand, 11) {
		t.Fatalf("lesser tax exchange was not applied correctly, players=%+v", done.Players)
	}
}

func TestPendingTaxBlocksPlayAndValidatesChooserCards(t *testing.T) {
	module := NewModule()
	state := applyOpeningTaxAndRevolution(fixedState())

	err := module.ValidateAction(context.Background(), state, gamecore.Action{
		Type:     ActionPlay,
		PlayerID: "p1",
		Payload:  map[string]any{"rank": 10, "count": 1},
	}, testContext())
	if err == nil {
		t.Fatal("expected play to be blocked during tax selection")
	}

	err = module.ValidateAction(context.Background(), state, gamecore.Action{
		Type:     ActionTax,
		PlayerID: "p2",
		Payload:  map[string]any{"cardIds": []any{"9-1", "jester-1"}},
	}, testContext())
	if err == nil {
		t.Fatal("expected non-current tax chooser to be rejected")
	}

	err = module.ValidateAction(context.Background(), state, gamecore.Action{
		Type:     ActionTax,
		PlayerID: "p1",
		Payload:  map[string]any{"cardIds": []any{"10-1", "10-1"}},
	}, testContext())
	if err == nil {
		t.Fatal("expected duplicate tax card to be rejected")
	}
}

func TestOpeningGreaterRevolutionReversesPlayerOrderAndSkipsTax(t *testing.T) {
	state := applyOpeningTaxAndRevolution(State{
		Players: []PlayerState{
			{PlayerID: "greater-dalmuti", Hand: []Card{{ID: "1-1", Rank: 1}}, Active: true},
			{PlayerID: "merchant-a", Hand: []Card{{ID: "8-1", Rank: 8}}, Active: true},
			{PlayerID: "merchant-b", Hand: []Card{{ID: "9-1", Rank: 9}}, Active: true},
			{PlayerID: "greater-peon", Hand: []Card{{ID: "jester-1", Rank: 13}, {ID: "jester-2", Rank: 13}}, Active: true},
		},
	})

	if !state.Revolution || !state.GreaterRevolution || state.TaxApplied {
		t.Fatalf("expected greater revolution without taxation, got %+v", state)
	}
	if state.Players[0].PlayerID != "greater-peon" || state.Players[3].PlayerID != "greater-dalmuti" {
		t.Fatalf("expected player order to reverse, got %+v", state.Players)
	}
}

func TestPlayRequiresSameCountAndStrongerRank(t *testing.T) {
	module := NewModule()
	state := fixedState()

	_, err := module.ApplyAction(context.Background(), state, gamecore.Action{
		Type:     ActionPlay,
		PlayerID: "p1",
		Payload:  map[string]any{"rank": float64(10), "count": float64(2)},
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}

	err = module.ValidateAction(context.Background(), State{
		CurrentPlayerIndex: 1,
		Players:            state.Players,
		CurrentTrick:       Trick{Rank: 10, Count: 2, PlayerID: "p1"},
	}, gamecore.Action{
		Type:     ActionPlay,
		PlayerID: "p2",
		Payload:  map[string]any{"rank": float64(11), "count": float64(2)},
	}, testContext())
	if err == nil {
		t.Fatal("expected weaker rank to be rejected")
	}
}

func TestJesterCanCompleteSetAsWild(t *testing.T) {
	state := fixedState()
	err := validatePlay(state.Players[1], Trick{Rank: 10, Count: 2, PlayerID: "p1"}, PlayPayload{Rank: 9, Count: 2})
	if err != nil {
		t.Fatal(err)
	}
}

func TestNewPlayClearsPreviousPasses(t *testing.T) {
	module := NewModule()
	state := State{
		CurrentPlayerIndex: 3,
		Players: []PlayerState{
			{PlayerID: "p1", Hand: []Card{{ID: "8-1", Rank: 8}}, Active: true},
			{PlayerID: "p2", Hand: []Card{{ID: "7-1", Rank: 7}}, Passed: true, Active: true},
			{PlayerID: "p3", Hand: []Card{{ID: "6-1", Rank: 6}}, Passed: true, Active: true},
			{PlayerID: "p4", Hand: []Card{{ID: "9-1", Rank: 9}}, Active: true},
		},
		CurrentTrick: Trick{Rank: 10, Count: 1, PlayerID: "p1"},
	}

	updated, err := module.ApplyAction(context.Background(), state, gamecore.Action{
		Type:     ActionPlay,
		PlayerID: "p4",
		Payload:  map[string]any{"rank": 9, "count": 1},
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}
	afterPlay := updated.State.(State)
	if afterPlay.Players[1].Passed || afterPlay.Players[2].Passed {
		t.Fatalf("previous passes must be cleared after a stronger play, got %+v", afterPlay.Players)
	}

	updated, err = module.ApplyAction(context.Background(), afterPlay, gamecore.Action{
		Type:     ActionPass,
		PlayerID: "p1",
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}
	afterPass := updated.State.(State)
	if afterPass.CurrentPlayerIndex != 1 {
		t.Fatalf("expected play to continue to p2 instead of ending the trick, got index %d state %+v", afterPass.CurrentPlayerIndex, afterPass)
	}
	if afterPass.CurrentTrick.PlayerID != "p4" {
		t.Fatalf("expected p4 to remain latest trick leader, got %+v", afterPass.CurrentTrick)
	}
}

func TestPlayPayloadAcceptsIntegerNumbers(t *testing.T) {
	module := NewModule()
	err := module.ValidateAction(context.Background(), fixedState(), gamecore.Action{
		Type:     ActionPlay,
		PlayerID: "p1",
		Payload:  map[string]any{"rank": 10, "count": 2},
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}
}

func TestPublicStateDoesNotMutatePrivateHands(t *testing.T) {
	module := NewModule()
	state := fixedState()
	public := module.PublicState(state, "p1").(State)

	if len(public.Players[1].Hand) != 0 {
		t.Fatalf("expected opponent hand to be hidden, got %+v", public.Players[1].Hand)
	}
	if len(state.Players[1].Hand) != 2 {
		t.Fatal("public state must not mutate private hands")
	}
}

func fixedState() State {
	return State{
		CurrentPlayerIndex: 0,
		Players: []PlayerState{
			{PlayerID: "p1", Hand: []Card{{ID: "10-1", Rank: 10}, {ID: "10-2", Rank: 10}}, Active: true},
			{PlayerID: "p2", Hand: []Card{{ID: "9-1", Rank: 9}, {ID: "jester-1", Rank: 13}}, Active: true},
			{PlayerID: "p3", Hand: []Card{{ID: "8-1", Rank: 8}, {ID: "8-2", Rank: 8}}, Active: true},
			{PlayerID: "p4", Hand: []Card{{ID: "7-1", Rank: 7}, {ID: "7-2", Rank: 7}}, Active: true},
		},
	}
}

func hasRank(hand []Card, rank int) bool {
	for _, card := range hand {
		if card.Rank == rank {
			return true
		}
	}
	return false
}

func testContext() gamecore.Context {
	return gamecore.Context{
		GameID: "dalmuti",
		Players: []gamecore.Player{
			{ID: "p1", SeatIndex: 0, DisplayName: "P1", Connected: true},
			{ID: "p2", SeatIndex: 1, DisplayName: "P2", Connected: true},
			{ID: "p3", SeatIndex: 2, DisplayName: "P3", Connected: true},
			{ID: "p4", SeatIndex: 3, DisplayName: "P4", Connected: true},
		},
		Now:        time.Date(2026, 7, 21, 1, 0, 0, 0, time.UTC),
		RandomSeed: "seed",
	}
}
