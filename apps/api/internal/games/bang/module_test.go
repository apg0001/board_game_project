package bang

import (
	"context"
	"testing"
	"time"

	"board-game-platform/apps/api/internal/gamecore"
)

func TestCreateInitialStateRevealsSheriff(t *testing.T) {
	module := NewModule()
	state := module.CreateInitialState(testContext()).(State)
	if state.Players[0].Role != RoleSheriff {
		t.Fatalf("expected first player sheriff, got %s", state.Players[0].Role)
	}
	if state.Players[0].HP != 5 {
		t.Fatalf("expected sheriff 5 hp, got %d", state.Players[0].HP)
	}
}

func TestBangDamageCanBeMissed(t *testing.T) {
	state := fixedState()
	next := damageTargetBy(state, "p2", 1, "p1")
	if next.Players[1].HP != 4 || len(next.Players[1].Hand) != 0 {
		t.Fatalf("expected missed to prevent damage, got %+v", next.Players[1])
	}
	if len(next.Discard) != 1 || next.Discard[0].Type != CardMissed {
		t.Fatalf("expected missed card to move to discard pile, got %+v", next.Discard)
	}
}

func TestOutlawsWinWhenSheriffDies(t *testing.T) {
	state := fixedState()
	state.Players[0].HP = 1
	state.Players[0].Hand = []Card{}
	next := checkEnd(damageTarget(state, "p1", 1))
	if !next.Finished || next.Winner != "outlaw" {
		t.Fatalf("expected outlaw win, got %+v", next)
	}
}

func TestEliminatedPlayerHandGoesToDiscardPile(t *testing.T) {
	state := fixedState()
	state.Players[1].HP = 1
	state.Players[1].Hand = []Card{{ID: "gunfighter-1", Type: "gunfighter"}, {ID: "saloon-1", Type: "saloon"}}

	next := damageTarget(state, "p2", 1)
	if next.Players[1].Alive {
		t.Fatal("expected target to be eliminated")
	}
	if len(next.Players[1].Hand) != 0 || next.Players[1].HandSize != 0 {
		t.Fatalf("expected eliminated player's hand to be cleared, got %+v", next.Players[1])
	}
	if len(next.Discard) != 2 {
		t.Fatalf("expected eliminated player's cards to move to the discard pile, got %d", len(next.Discard))
	}
}

func TestApplyTimeoutMovesEliminatedHandToDiscardPile(t *testing.T) {
	module := NewModule()
	state := fixedState()
	state.Players[1].Hand = []Card{{ID: "gunfighter-1", Type: "gunfighter"}}

	result, err := module.ApplyTimeout(context.Background(), state, "p2", testContext())
	if err != nil {
		t.Fatal(err)
	}
	next := result.State.(State)
	if len(next.Players[1].Hand) != 0 {
		t.Fatal("expected timed-out player's hand to be cleared")
	}
	if len(next.Discard) != 1 {
		t.Fatalf("expected timed-out player's cards to move to the discard pile, got %d", len(next.Discard))
	}
}

func TestMustDrawBeforePlayingOrEndingTurn(t *testing.T) {
	module := NewModule()
	state := fixedState()
	state.Deck = []Card{{ID: "beer-1", Type: CardBeer}}
	state.Players[0].Hand = []Card{{ID: "bang-1", Type: CardBang}}

	err := module.ValidateAction(context.Background(), state, gamecore.Action{
		Type:     ActionPlay,
		PlayerID: "p1",
		Payload:  map[string]any{"cardId": "bang-1", "targetPlayerId": "p2"},
	}, testContext())
	if err == nil {
		t.Fatal("expected play before draw to be rejected")
	}

	err = module.ValidateAction(context.Background(), state, gamecore.Action{
		Type:     ActionEndTurn,
		PlayerID: "p1",
	}, testContext())
	if err == nil {
		t.Fatal("expected ending turn before draw to be rejected")
	}

	result, err := module.ApplyAction(context.Background(), state, gamecore.Action{
		Type:     ActionDraw,
		PlayerID: "p1",
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}
	next := result.State.(State)
	if !next.Players[0].Drawn {
		t.Fatal("expected player to enter play phase after drawing")
	}
}

func TestDrawRecyclesDiscardWhenDeckIsEmpty(t *testing.T) {
	module := NewModule()
	state := fixedState()
	state.Deck = []Card{}
	state.Discard = []Card{{ID: "beer-1", Type: CardBeer}, {ID: "bang-1", Type: CardBang}}

	result, err := module.ApplyAction(context.Background(), state, gamecore.Action{
		Type:     ActionDraw,
		PlayerID: "p1",
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}

	next := result.State.(State)
	if len(next.Players[0].Hand) != 2 || len(next.Discard) != 0 {
		t.Fatalf("expected recycled discard to be drawn, got hand=%d discard=%d", len(next.Players[0].Hand), len(next.Discard))
	}
}

func TestEliminatingOutlawDrawsThreeCardReward(t *testing.T) {
	state := fixedState()
	state.Players[3].HP = 1
	state.Deck = []Card{
		{ID: "reward-1", Type: CardBang},
		{ID: "reward-2", Type: CardMissed},
		{ID: "reward-3", Type: CardBeer},
	}

	next := damageTargetBy(state, "p4", 1, "p1")
	if next.Players[3].Alive {
		t.Fatal("expected outlaw to be eliminated")
	}
	if len(next.Players[0].Hand) != 3 {
		t.Fatalf("expected sheriff to draw three reward cards, got %+v", next.Players[0].Hand)
	}
	if len(next.Deck) != 0 {
		t.Fatalf("expected reward cards to leave deck, got %d", len(next.Deck))
	}
}

func TestSheriffEliminatingDeputyDiscardsOwnHand(t *testing.T) {
	state := fixedState()
	state.Players[1].Role = RoleDeputy
	state.Players[1].HP = 1
	state.Players[1].Hand = []Card{}
	state.Players[0].Hand = []Card{{ID: "bang-1", Type: CardBang}, {ID: "beer-1", Type: CardBeer}}

	next := damageTargetBy(state, "p2", 1, "p1")
	if next.Players[1].Alive {
		t.Fatal("expected deputy to be eliminated")
	}
	if len(next.Players[0].Hand) != 0 || next.Players[0].HandSize != 0 {
		t.Fatalf("expected sheriff hand to be discarded, got %+v", next.Players[0])
	}
	if len(next.Discard) != 2 {
		t.Fatalf("expected sheriff hand cards in discard pile, got %+v", next.Discard)
	}
}

func TestBeerCannotBePlayedWithTwoPlayersLeft(t *testing.T) {
	module := NewModule()
	state := fixedState()
	state.Players[0].HP = 4
	state.Players[0].Drawn = true
	state.Players[0].Hand = []Card{{ID: "beer-1", Type: CardBeer}}
	state.Players[2].Alive = false
	state.Players[3].Alive = false

	err := module.ValidateAction(context.Background(), state, gamecore.Action{
		Type:     ActionPlay,
		PlayerID: "p1",
		Payload:  map[string]any{"cardId": "beer-1"},
	}, testContext())
	if err == nil {
		t.Fatal("expected beer to be rejected with only two alive players")
	}
}

func TestBeerDoesNotSaveEliminationWithTwoPlayersLeft(t *testing.T) {
	state := fixedState()
	state.Players[1].HP = 1
	state.Players[1].Hand = []Card{{ID: "beer-1", Type: CardBeer}}
	state.Players[2].Alive = false
	state.Players[3].Alive = false

	next := damageTarget(state, "p2", 1)
	if next.Players[1].Alive {
		t.Fatal("expected beer not to save target with only two alive players")
	}
	if len(next.Discard) != 1 || next.Discard[0].Type != CardBeer {
		t.Fatalf("expected eliminated beer card to move to discard pile, got %+v", next.Discard)
	}
}

func TestGatlingDoesNotRequireSingleTarget(t *testing.T) {
	module := NewModule()
	state := fixedState()
	state.Players[0].Drawn = true
	state.Players[0].Hand = []Card{{ID: "gatling-1", Type: CardGatling}}

	err := module.ValidateAction(context.Background(), state, gamecore.Action{
		Type:     ActionPlay,
		PlayerID: "p1",
		Payload:  map[string]any{"cardId": "gatling-1"},
	}, testContext())
	if err != nil {
		t.Fatalf("expected gatling without target to be valid, got %v", err)
	}
}

func TestBangPlayWaitsForTargetMissedResponse(t *testing.T) {
	module := NewModule()
	state := fixedState()
	state.Players[0].Drawn = true
	state.Players[0].Hand = []Card{{ID: "bang-1", Type: CardBang}}

	result, err := module.ApplyAction(context.Background(), state, gamecore.Action{
		Type:     ActionPlay,
		PlayerID: "p1",
		Payload:  map[string]any{"cardId": "bang-1", "targetPlayerId": "p2"},
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}

	next := result.State.(State)
	if next.PendingAttack == nil || next.PendingAttack.TargetPlayerID != "p2" {
		t.Fatalf("expected pending missed response for p2, got %+v", next.PendingAttack)
	}
	if next.Players[1].HP != 4 || len(next.Players[1].Hand) != 1 {
		t.Fatalf("expected p2 damage to wait for response, got %+v", next.Players[1])
	}
	if len(next.Discard) != 1 || next.Discard[0].Type != CardBang {
		t.Fatalf("expected bang card to be discarded, got %+v", next.Discard)
	}
}

func TestOnlyPendingTargetCanRespondToAttack(t *testing.T) {
	module := NewModule()
	state := fixedState()
	state.PendingAttack = &PendingAttack{
		SourcePlayerID: "p1",
		TargetPlayerID: "p2",
		CardType:       CardBang,
		Damage:         1,
	}

	err := module.ValidateAction(context.Background(), state, gamecore.Action{
		Type:     ActionUseMissed,
		PlayerID: "p3",
	}, testContext())
	if err == nil {
		t.Fatal("expected non-target response to be rejected")
	}

	err = module.ValidateAction(context.Background(), state, gamecore.Action{
		Type:     ActionPlay,
		PlayerID: "p1",
		Payload:  map[string]any{"cardId": "bang-1", "targetPlayerId": "p2"},
	}, testContext())
	if err == nil {
		t.Fatal("expected normal turn action to be blocked while response is pending")
	}

	err = module.ValidateAction(context.Background(), state, gamecore.Action{
		Type:     ActionUseMissed,
		PlayerID: "p2",
	}, testContext())
	if err != nil {
		t.Fatalf("expected target missed response to be valid, got %v", err)
	}
}

func TestPendingTargetCanUseMissed(t *testing.T) {
	module := NewModule()
	state := fixedState()
	state.PendingAttack = &PendingAttack{
		SourcePlayerID: "p1",
		TargetPlayerID: "p2",
		CardType:       CardBang,
		Damage:         1,
	}

	result, err := module.ApplyAction(context.Background(), state, gamecore.Action{
		Type:     ActionUseMissed,
		PlayerID: "p2",
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}
	next := result.State.(State)
	if next.PendingAttack != nil {
		t.Fatalf("expected pending attack to resolve, got %+v", next.PendingAttack)
	}
	if next.Players[1].HP != 4 || len(next.Players[1].Hand) != 0 {
		t.Fatalf("expected missed to avoid damage and leave hand, got %+v", next.Players[1])
	}
	if len(next.Discard) != 1 || next.Discard[0].Type != CardMissed {
		t.Fatalf("expected missed in discard pile, got %+v", next.Discard)
	}
}

func TestPendingTargetMayTakeHitWithoutSpendingMissed(t *testing.T) {
	module := NewModule()
	state := fixedState()
	state.PendingAttack = &PendingAttack{
		SourcePlayerID: "p1",
		TargetPlayerID: "p2",
		CardType:       CardBang,
		Damage:         1,
	}

	result, err := module.ApplyAction(context.Background(), state, gamecore.Action{
		Type:     ActionTakeHit,
		PlayerID: "p2",
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}
	next := result.State.(State)
	if next.PendingAttack != nil {
		t.Fatalf("expected pending attack to resolve, got %+v", next.PendingAttack)
	}
	if next.Players[1].HP != 3 || len(next.Players[1].Hand) != 1 {
		t.Fatalf("expected p2 to take damage without spending missed, got %+v", next.Players[1])
	}
	if len(next.Discard) != 0 {
		t.Fatalf("expected missed card to stay in hand, got discard %+v", next.Discard)
	}
}

func TestGatlingProcessesPendingResponsesInOrder(t *testing.T) {
	module := NewModule()
	state := fixedState()
	state.Players[0].Drawn = true
	state.Players[0].Hand = []Card{{ID: "gatling-1", Type: CardGatling}}
	state.Players[3].Hand = []Card{{ID: "missed-4", Type: CardMissed}}

	result, err := module.ApplyAction(context.Background(), state, gamecore.Action{
		Type:     ActionPlay,
		PlayerID: "p1",
		Payload:  map[string]any{"cardId": "gatling-1"},
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}
	first := result.State.(State)
	if first.PendingAttack == nil || first.PendingAttack.TargetPlayerID != "p2" {
		t.Fatalf("expected p2 to respond first, got %+v", first.PendingAttack)
	}

	result, err = module.ApplyAction(context.Background(), first, gamecore.Action{
		Type:     ActionUseMissed,
		PlayerID: "p2",
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}
	second := result.State.(State)
	if second.Players[2].HP != 3 {
		t.Fatalf("expected p3 without missed to take damage, got %+v", second.Players[2])
	}
	if second.PendingAttack == nil || second.PendingAttack.TargetPlayerID != "p4" {
		t.Fatalf("expected p4 to respond after p3 damage, got %+v", second.PendingAttack)
	}
}

func TestEndTurnRequiresDiscardDownToCurrentHP(t *testing.T) {
	module := NewModule()
	state := fixedState()
	state.Players[0].HP = 3
	state.Players[0].Drawn = true
	state.Players[0].Hand = []Card{
		{ID: "c1", Type: CardBang},
		{ID: "c2", Type: CardBeer},
		{ID: "c3", Type: CardMissed},
		{ID: "c4", Type: CardBang},
		{ID: "c5", Type: CardBeer},
	}

	result, err := module.ApplyAction(context.Background(), state, gamecore.Action{
		Type:     ActionEndTurn,
		PlayerID: "p1",
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}
	next := result.State.(State)
	if next.PendingDiscardID != "p1" || next.PendingDiscardCount != 2 {
		t.Fatalf("expected pending discard of two cards, got id=%s count=%d", next.PendingDiscardID, next.PendingDiscardCount)
	}
	if next.CurrentPlayerIndex != 0 {
		t.Fatalf("expected turn to wait for discard, got index %d", next.CurrentPlayerIndex)
	}
}

func TestPendingDiscardBlocksOtherActions(t *testing.T) {
	module := NewModule()
	state := fixedState()
	state.PendingDiscardID = "p1"
	state.PendingDiscardCount = 1
	state.Players[0].Drawn = true
	state.Players[0].Hand = []Card{{ID: "c1", Type: CardBang}}

	err := module.ValidateAction(context.Background(), state, gamecore.Action{
		Type:     ActionDraw,
		PlayerID: "p1",
	}, testContext())
	if err == nil {
		t.Fatal("expected normal action to be blocked during pending discard")
	}

	err = module.ValidateAction(context.Background(), state, gamecore.Action{
		Type:     ActionDiscard,
		PlayerID: "p2",
		Payload:  map[string]any{"cardIds": []any{"c1"}},
	}, testContext())
	if err == nil {
		t.Fatal("expected other player discard to be rejected")
	}
}

func TestPendingDiscardAdvancesTurnAfterSelectedCards(t *testing.T) {
	module := NewModule()
	state := fixedState()
	state.PendingDiscardID = "p1"
	state.PendingDiscardCount = 2
	state.Players[0].HP = 1
	state.Players[0].Drawn = true
	state.Players[0].BangUsed = true
	state.Players[0].Hand = []Card{
		{ID: "c1", Type: CardBang},
		{ID: "c2", Type: CardBeer},
		{ID: "c3", Type: CardMissed},
	}

	result, err := module.ApplyAction(context.Background(), state, gamecore.Action{
		Type:     ActionDiscard,
		PlayerID: "p1",
		Payload:  map[string]any{"cardIds": []any{"c1", "c2"}},
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}
	next := result.State.(State)
	if next.PendingDiscardID != "" || next.PendingDiscardCount != 0 {
		t.Fatalf("expected pending discard to clear, got id=%s count=%d", next.PendingDiscardID, next.PendingDiscardCount)
	}
	if next.CurrentPlayerIndex == 0 || next.Round != 1 {
		t.Fatalf("expected turn to advance, got index=%d round=%d", next.CurrentPlayerIndex, next.Round)
	}
	if next.Players[0].Drawn || next.Players[0].BangUsed {
		t.Fatalf("expected turn flags to reset, got %+v", next.Players[0])
	}
	if len(next.Players[0].Hand) != 1 || len(next.Discard) != 2 {
		t.Fatalf("expected two discarded cards and one remaining card, got hand=%v discard=%v", next.Players[0].Hand, next.Discard)
	}
}

func TestPendingDiscardRejectsWrongCardCount(t *testing.T) {
	module := NewModule()
	state := fixedState()
	state.PendingDiscardID = "p1"
	state.PendingDiscardCount = 2
	state.Players[0].Hand = []Card{{ID: "c1", Type: CardBang}, {ID: "c2", Type: CardBeer}}

	err := module.ValidateAction(context.Background(), state, gamecore.Action{
		Type:     ActionDiscard,
		PlayerID: "p1",
		Payload:  map[string]any{"cardIds": []any{"c1"}},
	}, testContext())
	if err == nil {
		t.Fatal("expected wrong discard count to be rejected")
	}
}

func fixedState() State {
	return State{
		Players: []PlayerState{
			{PlayerID: "p1", Role: RoleSheriff, HP: 5, MaxHP: 5, Alive: true, Active: true},
			{PlayerID: "p2", Role: RoleOutlaw, HP: 4, MaxHP: 4, Hand: []Card{{ID: "missed-1", Type: CardMissed}}, Alive: true, Active: true},
			{PlayerID: "p3", Role: RoleRenegade, HP: 4, MaxHP: 4, Alive: true, Active: true},
			{PlayerID: "p4", Role: RoleOutlaw, HP: 4, MaxHP: 4, Alive: true, Active: true},
		},
	}
}

func testContext() gamecore.Context {
	return gamecore.Context{
		GameID: "bang",
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
