package onecard

import (
	"context"
	"testing"
	"time"

	"board-game-platform/apps/api/internal/gamecore"
)

func TestCreateInitialStateDealsSevenCards(t *testing.T) {
	module := NewModule()
	state := module.CreateInitialState(testContext()).(State)
	if len(state.Players[0].Hand) != 7 {
		t.Fatalf("expected seven cards, got %d", len(state.Players[0].Hand))
	}
	if len(state.DiscardPile) != 1 {
		t.Fatalf("expected one top card, got %d", len(state.DiscardPile))
	}
}

func TestCanPlayMatchesSuitOrRank(t *testing.T) {
	state := State{
		Rules:       ruleConfigFromOptions(nil),
		DiscardPile: []Card{{Suit: "heart", Rank: "7"}},
	}
	if !canPlay(Card{Suit: "heart", Rank: "2"}, state) {
		t.Fatal("expected same suit playable")
	}
	if !canPlay(Card{Suit: "spade", Rank: "7"}, state) {
		t.Fatal("expected same rank playable")
	}
	if canPlay(Card{Suit: "spade", Rank: "3"}, state) {
		t.Fatal("expected different suit and rank rejected")
	}
}

func TestResolveRulesUsesVotesAndTieBreaks(t *testing.T) {
	module := NewModule()
	resolution := module.ResolveRules([]gamecore.RuleVote{
		{UserID: "p1", Choices: map[string]string{"attackCards": "two", "defenseMode": "same-rank", "jokerDrawCount": "7", "stacking": "off"}},
		{UserID: "p2", Choices: map[string]string{"attackCards": "two-ace-joker", "defenseMode": "attack-or-joker", "jokerDrawCount": "5", "stacking": "on"}},
	}, "room_seed")

	if len(resolution.Announcements) < 2 {
		t.Fatalf("expected tie announcements, got %#v", resolution.Announcements)
	}
	if resolution.Options["jokerDrawCount"] == nil || resolution.Options["stacking"] == nil {
		t.Fatalf("expected resolved options, got %#v", resolution.Options)
	}
}

func TestDeckIncludesJokers(t *testing.T) {
	deck := shuffledDeck("seed")
	jokers := 0
	for _, card := range deck {
		if card.Joker {
			jokers++
		}
	}
	if len(deck) != 54 || jokers != 2 {
		t.Fatalf("expected 54 cards with two jokers, got len=%d jokers=%d", len(deck), jokers)
	}
}

func TestAttackCardAddsPendingDraw(t *testing.T) {
	module := NewModule()
	state := State{
		CurrentPlayerIndex: 0,
		Direction:          1,
		Rules:              ruleConfigFromOptions(map[string]any{"attackCards": []string{"2", "A", "JOKER"}, "stacking": true}),
		Players: []PlayerState{
			{PlayerID: "p1", Hand: []Card{{ID: "heart-2", Suit: "heart", Rank: "2"}}, Active: true},
			{PlayerID: "p2", Hand: []Card{{ID: "spade-3", Suit: "spade", Rank: "3"}}, Active: true},
		},
		DrawPile:    []Card{{ID: "club-4", Suit: "club", Rank: "4"}},
		DiscardPile: []Card{{ID: "heart-7", Suit: "heart", Rank: "7"}},
	}

	result, err := module.ApplyAction(context.Background(), state, gamecore.Action{
		Type:     ActionPlay,
		PlayerID: "p1",
		Payload:  map[string]any{"cardId": "heart-2"},
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}

	next := result.State.(State)
	if next.PendingDraw != 2 {
		t.Fatalf("expected pending draw 2, got %d", next.PendingDraw)
	}
}

func TestJokerAttackUsesVotedDrawCount(t *testing.T) {
	rules := ruleConfigFromOptions(map[string]any{"attackCards": []string{"2", "A", "JOKER"}, "jokerDrawCount": 7})
	if amount := attackAmount(Card{Joker: true, Rank: "JOKER"}, rules); amount != 7 {
		t.Fatalf("expected joker attack 7, got %d", amount)
	}
}

func TestPendingAttackCanBeDefendedByConfiguredCard(t *testing.T) {
	state := State{
		Rules:       ruleConfigFromOptions(map[string]any{"attackCards": []string{"2", "A", "JOKER"}, "defenseMode": "attack-or-joker"}),
		PendingDraw: 2,
		DiscardPile: []Card{{ID: "heart-2", Suit: "heart", Rank: "2"}},
	}
	if !canPlay(Card{ID: "joker-black", Suit: "joker", Rank: "JOKER", Joker: true}, state) {
		t.Fatal("expected joker to defend pending attack")
	}
	if canPlay(Card{ID: "heart-7", Suit: "heart", Rank: "7"}, state) {
		t.Fatal("expected normal card rejected while attack is pending")
	}
}

func TestAttackOrJokerDefenseAllowsJokerOutsideAttackSet(t *testing.T) {
	state := State{
		Rules:       ruleConfigFromOptions(map[string]any{"attackCards": []string{"2"}, "defenseMode": "attack-or-joker"}),
		PendingDraw: 2,
		DiscardPile: []Card{{ID: "heart-2", Suit: "heart", Rank: "2"}},
	}
	if !canPlay(Card{ID: "joker-black", Suit: "joker", Rank: "JOKER", Joker: true}, state) {
		t.Fatal("expected joker to defend when defense mode allows attack cards or joker")
	}
}

func TestAnyAttackDefenseRequiresConfiguredAttackCard(t *testing.T) {
	state := State{
		Rules:       ruleConfigFromOptions(map[string]any{"attackCards": []string{"2"}, "defenseMode": "any-attack"}),
		PendingDraw: 2,
		DiscardPile: []Card{{ID: "heart-2", Suit: "heart", Rank: "2"}},
	}
	if canPlay(Card{ID: "joker-black", Suit: "joker", Rank: "JOKER", Joker: true}, state) {
		t.Fatal("expected joker to be rejected when it is not a configured attack card")
	}
	if !canPlay(Card{ID: "spade-2", Suit: "spade", Rank: "2"}, state) {
		t.Fatal("expected configured attack card to defend")
	}
}

func testContext() gamecore.Context {
	return gamecore.Context{
		GameID: "onecard",
		Players: []gamecore.Player{
			{ID: "p1", SeatIndex: 0, DisplayName: "P1", Connected: true},
			{ID: "p2", SeatIndex: 1, DisplayName: "P2", Connected: true},
		},
		Now:        time.Date(2026, 7, 21, 1, 0, 0, 0, time.UTC),
		RandomSeed: "seed",
	}
}
