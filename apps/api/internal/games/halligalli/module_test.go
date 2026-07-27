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

func TestShuffledDeckUsesOfficialDistribution(t *testing.T) {
	deck := shuffledDeck("seed")
	if len(deck) != 56 {
		t.Fatalf("expected 56 cards, got %d", len(deck))
	}
	counts := map[int]int{}
	for _, card := range deck {
		if card.Fruit == "banana" {
			counts[card.Count]++
		}
	}
	expected := map[int]int{1: 5, 2: 3, 3: 3, 4: 2, 5: 1}
	for count, want := range expected {
		if counts[count] != want {
			t.Fatalf("expected banana count %d to appear %d times, got %d", count, want, counts[count])
		}
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

func TestCorrectRingCollectsFaceUpIntoDeck(t *testing.T) {
	module := NewModule()
	state := State{
		Players: []PlayerState{
			{PlayerID: "p1", Deck: []Card{{Fruit: "banana", Count: 1}}, FaceUp: []Card{{Fruit: "banana", Count: 2}}, Active: true},
			{PlayerID: "p2", Deck: []Card{{Fruit: "lime", Count: 1}}, FaceUp: []Card{{Fruit: "banana", Count: 3}}, Active: true},
		},
	}

	result, err := module.ApplyAction(context.Background(), state, gamecore.Action{
		Type:     ActionRing,
		PlayerID: "p1",
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}

	next := result.State.(State)
	if len(next.Players[0].Deck) != 3 || len(next.Players[0].FaceUp) != 0 || len(next.Players[1].FaceUp) != 0 {
		t.Fatalf("expected p1 to collect face-up piles, got %+v", next.Players)
	}
	if !next.BellSettled || next.LastBellWinnerID != "p1" {
		t.Fatalf("expected bell settlement to record first winner, got settled=%v winner=%q", next.BellSettled, next.LastBellWinnerID)
	}
}

func TestLateRingAfterCorrectBellIsRejectedWithoutPenalty(t *testing.T) {
	module := NewModule()
	state := State{
		Players: []PlayerState{
			{PlayerID: "p1", Deck: []Card{{Fruit: "banana", Count: 1}}, FaceUp: []Card{{Fruit: "banana", Count: 2}}, Active: true},
			{PlayerID: "p2", Deck: []Card{{Fruit: "lime", Count: 1}}, FaceUp: []Card{{Fruit: "banana", Count: 3}}, Active: true},
		},
	}

	result, err := module.ApplyAction(context.Background(), state, gamecore.Action{
		Type:     ActionRing,
		PlayerID: "p1",
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}
	settled := result.State.(State)
	beforeP2Deck := len(settled.Players[1].Deck)

	err = module.ValidateAction(context.Background(), settled, gamecore.Action{
		Type:     ActionRing,
		PlayerID: "p2",
	}, testContext())
	if err == nil {
		t.Fatal("expected late bell after settlement to be rejected")
	}
	late, err := module.ApplyAction(context.Background(), settled, gamecore.Action{
		Type:     ActionRing,
		PlayerID: "p2",
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}
	next := late.State.(State)
	if len(next.Players[1].Deck) != beforeP2Deck {
		t.Fatalf("late ring should not apply wrong-ring penalty after bell settlement, got p2 deck %d want %d", len(next.Players[1].Deck), beforeP2Deck)
	}
}

func TestWrongRingPaysOneCardToEachActiveOpponent(t *testing.T) {
	module := NewModule()
	state := State{
		Players: []PlayerState{
			{PlayerID: "p1", Deck: []Card{{Fruit: "banana", Count: 1}, {Fruit: "lime", Count: 1}}, FaceUp: []Card{{Fruit: "banana", Count: 2}}, Active: true},
			{PlayerID: "p2", Deck: []Card{{Fruit: "plum", Count: 1}}, FaceUp: []Card{{Fruit: "banana", Count: 2}}, Active: true},
			{PlayerID: "p3", Deck: []Card{{Fruit: "strawberry", Count: 1}}, FaceUp: []Card{}, Active: true},
		},
	}

	result, err := module.ApplyAction(context.Background(), state, gamecore.Action{
		Type:     ActionRing,
		PlayerID: "p1",
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}

	next := result.State.(State)
	if len(next.Players[0].Deck) != 0 || len(next.Players[1].Deck) != 2 || len(next.Players[2].Deck) != 2 {
		t.Fatalf("expected p1 to pay one card to each opponent, got %+v", next.Players)
	}
}

func TestWrongRingCanOnlyBeAttemptedOncePerTableBySamePlayer(t *testing.T) {
	module := NewModule()
	state := State{
		Players: []PlayerState{
			{PlayerID: "p1", Deck: []Card{{Fruit: "banana", Count: 1}, {Fruit: "lime", Count: 1}, {Fruit: "plum", Count: 1}}, FaceUp: []Card{{Fruit: "banana", Count: 2}}, Active: true},
			{PlayerID: "p2", Deck: []Card{{Fruit: "plum", Count: 1}}, FaceUp: []Card{{Fruit: "banana", Count: 2}}, Active: true},
		},
		RingAttempts: map[string]bool{},
	}

	result, err := module.ApplyAction(context.Background(), state, gamecore.Action{
		Type:     ActionRing,
		PlayerID: "p1",
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}
	afterFirst := result.State.(State)
	if !afterFirst.RingAttempts["p1"] {
		t.Fatalf("expected p1 ring attempt to be recorded, got %+v", afterFirst.RingAttempts)
	}
	beforeDeck := len(afterFirst.Players[0].Deck)

	err = module.ValidateAction(context.Background(), afterFirst, gamecore.Action{
		Type:     ActionRing,
		PlayerID: "p1",
	}, testContext())
	if err == nil {
		t.Fatal("expected repeated ring on the same table to be rejected")
	}
	result, err = module.ApplyAction(context.Background(), afterFirst, gamecore.Action{
		Type:     ActionRing,
		PlayerID: "p1",
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}
	afterSecond := result.State.(State)
	if len(afterSecond.Players[0].Deck) != beforeDeck {
		t.Fatalf("repeated ring should not apply a second penalty, got deck=%d want %d", len(afterSecond.Players[0].Deck), beforeDeck)
	}
}

func TestFlipClearsRingAttemptsForNewTable(t *testing.T) {
	module := NewModule()
	state := State{
		CurrentPlayerIndex: 0,
		RingAttempts:       map[string]bool{"p1": true},
		Players: []PlayerState{
			{PlayerID: "p1", Deck: []Card{{Fruit: "banana", Count: 1}}, Active: true},
			{PlayerID: "p2", Deck: []Card{{Fruit: "plum", Count: 1}}, Active: true},
		},
	}

	result, err := module.ApplyAction(context.Background(), state, gamecore.Action{
		Type:     ActionFlip,
		PlayerID: "p1",
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}
	next := result.State.(State)
	if len(next.RingAttempts) != 0 {
		t.Fatalf("expected ring attempts to reset after a new flip, got %+v", next.RingAttempts)
	}
}

func TestInactivePlayerCannotRing(t *testing.T) {
	module := NewModule()
	state := State{
		Players: []PlayerState{
			{PlayerID: "p1", Deck: []Card{}, FaceUp: []Card{}, Active: false},
			{PlayerID: "p2", Deck: []Card{{Fruit: "lime", Count: 1}}, Active: true},
		},
	}

	err := module.ValidateAction(context.Background(), state, gamecore.Action{
		Type:     ActionRing,
		PlayerID: "p1",
	}, testContext())
	if err == nil {
		t.Fatal("expected inactive player ring to be rejected")
	}
}

func TestTimedOutPlayerStaysForfeitedAfterFlippingBeforeDisconnect(t *testing.T) {
	module := NewModule()
	state := State{
		CurrentPlayerIndex: 1,
		Players: []PlayerState{
			{PlayerID: "p1", Deck: []Card{{Fruit: "lime", Count: 1}}, FaceUp: []Card{{Fruit: "banana", Count: 2}}, Active: true},
			{PlayerID: "p2", Deck: []Card{{Fruit: "plum", Count: 1}}, Active: true},
		},
	}

	result, err := module.ApplyTimeout(context.Background(), state, "p1", testContext())
	if err != nil {
		t.Fatal(err)
	}
	next := result.State.(State)
	if next.Players[0].Active {
		t.Fatal("timed-out player with leftover face-up cards should stay inactive, not be resurrected by refresh")
	}

	err = module.ValidateAction(context.Background(), next, gamecore.Action{
		Type:     ActionRing,
		PlayerID: "p1",
	}, testContext())
	if err == nil {
		t.Fatal("forfeited player should not be allowed to ring after timing out")
	}
}

func TestPublicStateDoesNotMutatePrivateDeck(t *testing.T) {
	module := NewModule()
	state := State{Players: []PlayerState{{PlayerID: "p1", Deck: []Card{{Fruit: "banana", Count: 1}}, Active: true}}}
	public := module.PublicState(state, "p1").(State)

	if public.Players[0].Deck[0].Fruit != "" {
		t.Fatalf("expected masked deck card, got %+v", public.Players[0].Deck[0])
	}
	if state.Players[0].Deck[0].Fruit != "banana" {
		t.Fatal("public state must not mutate private deck")
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
