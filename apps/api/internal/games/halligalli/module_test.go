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
