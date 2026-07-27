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

func TestCreateInitialStateDealsFiveCardsForThreePlusPlayers(t *testing.T) {
	module := NewModule()
	ctx := testContext()
	ctx.Players = append(ctx.Players, gamecore.Player{ID: "p3", SeatIndex: 2, DisplayName: "P3", Connected: true})
	state := module.CreateInitialState(ctx).(State)
	for _, player := range state.Players {
		if len(player.Hand) != 5 {
			t.Fatalf("expected five cards per player for 3+ players, got %d", len(player.Hand))
		}
	}
}

func TestNonStackingAttackPassesTurnToVictimForDefense(t *testing.T) {
	module := NewModule()
	state := State{
		CurrentPlayerIndex: 0,
		Direction:          1,
		Rules:              ruleConfigFromOptions(map[string]any{"attackCards": []string{"2", "A", "JOKER"}, "stacking": false}),
		Players: []PlayerState{
			{PlayerID: "p1", Hand: []Card{{ID: "heart-2", Suit: "heart", Rank: "2"}, {ID: "heart-9", Suit: "heart", Rank: "9"}}, Active: true},
			{PlayerID: "p2", Hand: []Card{{ID: "spade-3", Suit: "spade", Rank: "3"}}, Active: true},
			{PlayerID: "p3", Hand: []Card{{ID: "club-3", Suit: "club", Rank: "3"}}, Active: true},
		},
		DrawPile:    []Card{{ID: "club-4", Suit: "club", Rank: "4"}, {ID: "club-5", Suit: "club", Rank: "5"}},
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
	if next.CurrentPlayerIndex != 1 {
		t.Fatalf("expected turn to pass to the victim (index 1) for a defense chance, got %d", next.CurrentPlayerIndex)
	}
	if next.PendingDraw != 2 {
		t.Fatalf("expected pending draw of 2 for the victim to resolve, got %d", next.PendingDraw)
	}
	if len(next.Players[1].Hand) != 1 {
		t.Fatalf("victim should not be force-drawn before getting a chance to defend, got hand %+v", next.Players[1].Hand)
	}
}

func TestKingCardGrantsExtraTurn(t *testing.T) {
	module := NewModule()
	state := State{
		CurrentPlayerIndex: 0,
		Direction:          1,
		Rules:              ruleConfigFromOptions(nil),
		Players: []PlayerState{
			{PlayerID: "p1", Hand: []Card{{ID: "heart-king", Suit: "heart", Rank: "K"}}, Active: true},
			{PlayerID: "p2", Hand: []Card{{ID: "spade-3", Suit: "spade", Rank: "3"}}, Active: true},
		},
		DiscardPile: []Card{{ID: "heart-7", Suit: "heart", Rank: "7"}},
	}

	result, err := module.ApplyAction(context.Background(), state, gamecore.Action{
		Type:     ActionPlay,
		PlayerID: "p1",
		Payload:  map[string]any{"cardId": "heart-king"},
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}
	next := result.State.(State)
	if next.CurrentPlayerIndex != 0 {
		t.Fatalf("expected K to grant an extra turn to the same player, got index %d", next.CurrentPlayerIndex)
	}
}

func TestQueenReversesDirectionAndAceDoesNot(t *testing.T) {
	module := NewModule()
	state := State{
		CurrentPlayerIndex: 0,
		Direction:          1,
		Rules:              ruleConfigFromOptions(map[string]any{"attackCards": []string{"2", "A"}, "stacking": true}),
		Players: []PlayerState{
			{PlayerID: "p1", Hand: []Card{{ID: "heart-q", Suit: "heart", Rank: "Q"}, {ID: "spade-a", Suit: "spade", Rank: "A"}}, Active: true},
			{PlayerID: "p2", Hand: []Card{{ID: "club-3", Suit: "club", Rank: "3"}}, Active: true},
			{PlayerID: "p3", Hand: []Card{{ID: "diamond-4", Suit: "diamond", Rank: "4"}}, Active: true},
		},
		DiscardPile: []Card{{ID: "heart-5", Suit: "heart", Rank: "5"}},
	}

	result, err := module.ApplyAction(context.Background(), state, gamecore.Action{
		Type:     ActionPlay,
		PlayerID: "p1",
		Payload:  map[string]any{"cardId": "heart-q"},
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}
	next := result.State.(State)
	if next.Direction != -1 {
		t.Fatalf("expected Q to reverse direction, got %d", next.Direction)
	}

	next.CurrentPlayerIndex = 0
	next.Direction = 1
	next.DiscardPile = []Card{{ID: "spade-5", Suit: "spade", Rank: "5"}}
	result, err = module.ApplyAction(context.Background(), next, gamecore.Action{
		Type:     ActionPlay,
		PlayerID: "p1",
		Payload:  map[string]any{"cardId": "spade-a"},
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}
	afterAce := result.State.(State)
	if afterAce.Direction != 1 {
		t.Fatalf("expected A to keep direction while applying attack, got %d", afterAce.Direction)
	}
	if afterAce.PendingDraw != 3 {
		t.Fatalf("expected A attack to add 3 pending cards, got %d", afterAce.PendingDraw)
	}
}

func TestApplyTimeoutResolvesPendingDrawBeforePassingTurn(t *testing.T) {
	module := NewModule()
	state := State{
		CurrentPlayerIndex: 0,
		Direction:          1,
		Rules:              ruleConfigFromOptions(nil),
		PendingDraw:        2,
		Players: []PlayerState{
			{PlayerID: "p1", Hand: []Card{}, Active: true},
			{PlayerID: "p2", Hand: []Card{}, Active: true},
		},
		DrawPile:    []Card{{ID: "club-4"}, {ID: "club-5"}},
		DiscardPile: []Card{{ID: "heart-7"}},
	}

	result, err := module.ApplyTimeout(context.Background(), state, "p1", testContext())
	if err != nil {
		t.Fatal(err)
	}
	next := result.State.(State)
	if len(next.Players[0].Hand) != 2 {
		t.Fatalf("expected timed-out player to draw the pending penalty, got hand %+v", next.Players[0].Hand)
	}
	if next.PendingDraw != 0 {
		t.Fatal("pending draw should be cleared after timeout resolves it")
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
		{UserID: "p1", Choices: map[string]string{"attackCards": "two", "defenseMode": "same-rank", "jokerDrawCount": "7", "stacking": "off", "changeSuitCards": "off", "oneCardPenalty": "off"}},
		{UserID: "p2", Choices: map[string]string{"attackCards": "two-ace-joker", "defenseMode": "attack-or-joker", "jokerDrawCount": "5", "stacking": "on", "changeSuitCards": "seven-joker", "oneCardPenalty": "on"}},
	}, "room_seed")

	if len(resolution.Announcements) < 2 {
		t.Fatalf("expected tie announcements, got %#v", resolution.Announcements)
	}
	if resolution.Options["jokerDrawCount"] == nil || resolution.Options["stacking"] == nil || resolution.Options["changeSuitCards"] == nil || resolution.Options["oneCardPenalty"] == nil {
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

func TestFirstDiscardSkipsSpecialCards(t *testing.T) {
	deck := []Card{
		{ID: "heart-2", Suit: "heart", Rank: "2"},
		{ID: "heart-a", Suit: "heart", Rank: "A"},
		{ID: "heart-j", Suit: "heart", Rank: "J"},
		{ID: "heart-q", Suit: "heart", Rank: "Q"},
		{ID: "heart-k", Suit: "heart", Rank: "K"},
		{ID: "heart-7", Suit: "heart", Rank: "7"},
		{ID: "club-9", Suit: "club", Rank: "9"},
	}
	if index := firstDiscardIndex(deck); index != 6 {
		t.Fatalf("expected first non-special card index 6, got %d", index)
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

func TestPlayToOneCanDeclareOne(t *testing.T) {
	module := NewModule()
	state := State{
		CurrentPlayerIndex: 0,
		Direction:          1,
		Rules:              ruleConfigFromOptions(nil),
		Players: []PlayerState{
			{PlayerID: "p1", Hand: []Card{{ID: "heart-9", Suit: "heart", Rank: "9"}, {ID: "heart-5", Suit: "heart", Rank: "5"}}, Active: true},
			{PlayerID: "p2", Hand: []Card{{ID: "spade-3", Suit: "spade", Rank: "3"}}, Active: true},
		},
		DiscardPile: []Card{{ID: "heart-7", Suit: "heart", Rank: "7"}},
	}

	result, err := module.ApplyAction(context.Background(), state, gamecore.Action{
		Type:     ActionPlay,
		PlayerID: "p1",
		Payload:  map[string]any{"cardId": "heart-9", "declareOne": true},
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}

	next := result.State.(State)
	if !next.DeclaredOne["p1"] {
		t.Fatalf("expected p1 one-card declaration to be recorded, got %#v", next.DeclaredOne)
	}
}

func TestCalloutOnePenaltyDrawsCards(t *testing.T) {
	module := NewModule()
	state := State{
		CurrentPlayerIndex: 1,
		Direction:          1,
		Rules:              ruleConfigFromOptions(nil),
		DeclaredOne:        map[string]bool{"p1": false},
		Players: []PlayerState{
			{PlayerID: "p1", Hand: []Card{{ID: "heart-5", Suit: "heart", Rank: "5"}}, Active: true},
			{PlayerID: "p2", Hand: []Card{{ID: "spade-3", Suit: "spade", Rank: "3"}}, Active: true},
		},
		DrawPile:    []Card{{ID: "club-4", Suit: "club", Rank: "4"}, {ID: "diamond-6", Suit: "diamond", Rank: "6"}},
		DiscardPile: []Card{{ID: "heart-7", Suit: "heart", Rank: "7"}},
	}

	if err := module.ValidateAction(context.Background(), state, gamecore.Action{
		Type:     ActionCalloutOne,
		PlayerID: "p2",
		Payload:  map[string]any{"targetPlayerId": "p1"},
	}, testContext()); err != nil {
		t.Fatal(err)
	}
	result, err := module.ApplyAction(context.Background(), state, gamecore.Action{
		Type:     ActionCalloutOne,
		PlayerID: "p2",
		Payload:  map[string]any{"targetPlayerId": "p1"},
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}

	next := result.State.(State)
	if len(next.Players[0].Hand) != 3 {
		t.Fatalf("expected p1 to draw two penalty cards, got %+v", next.Players[0].Hand)
	}
	if next.DeclaredOne["p1"] {
		t.Fatal("penalized player should no longer be marked as declared")
	}
}

func TestChangeSuitRequiresDeclaredSuit(t *testing.T) {
	module := NewModule()
	state := State{
		CurrentPlayerIndex: 0,
		Direction:          1,
		Rules:              ruleConfigFromOptions(nil),
		Players: []PlayerState{
			{PlayerID: "p1", Hand: []Card{{ID: "heart-7", Suit: "heart", Rank: "7"}}, Active: true},
			{PlayerID: "p2", Hand: []Card{{ID: "spade-3", Suit: "spade", Rank: "3"}}, Active: true},
		},
		DiscardPile: []Card{{ID: "heart-5", Suit: "heart", Rank: "5"}},
	}

	err := module.ValidateAction(context.Background(), state, gamecore.Action{
		Type:     ActionPlay,
		PlayerID: "p1",
		Payload:  map[string]any{"cardId": "heart-7"},
	}, testContext())
	if err == nil {
		t.Fatal("expected changing-suit card to require a declared suit")
	}
}

func TestDeclaredSuitControlsNextPlayableSuit(t *testing.T) {
	module := NewModule()
	state := State{
		CurrentPlayerIndex: 0,
		Direction:          1,
		Rules:              ruleConfigFromOptions(nil),
		Players: []PlayerState{
			{PlayerID: "p1", Hand: []Card{{ID: "heart-7", Suit: "heart", Rank: "7"}, {ID: "club-4", Suit: "club", Rank: "4"}}, Active: true},
			{PlayerID: "p2", Hand: []Card{{ID: "spade-3", Suit: "spade", Rank: "3"}}, Active: true},
		},
		DiscardPile: []Card{{ID: "heart-5", Suit: "heart", Rank: "5"}},
	}

	result, err := module.ApplyAction(context.Background(), state, gamecore.Action{
		Type:     ActionPlay,
		PlayerID: "p1",
		Payload:  map[string]any{"cardId": "heart-7", "declaredSuit": "spade"},
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}
	next := result.State.(State)
	if next.DeclaredSuit != "spade" {
		t.Fatalf("expected declared suit spade, got %q", next.DeclaredSuit)
	}
	if !canPlay(Card{ID: "spade-9", Suit: "spade", Rank: "9"}, next) {
		t.Fatal("expected declared suit to allow spade")
	}
	if canPlay(Card{ID: "heart-9", Suit: "heart", Rank: "9"}, next) {
		t.Fatal("expected previous physical suit to be ignored after declared suit")
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
