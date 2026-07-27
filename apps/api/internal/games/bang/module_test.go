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
	for _, player := range state.Players {
		if player.CharacterID == "" || player.CharacterName == "" {
			t.Fatalf("expected character assignment, got %+v", player)
		}
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

func TestIndiansRequiresBangResponsesInOrder(t *testing.T) {
	module := NewModule()
	state := fixedState()
	state.Players[0].Drawn = true
	state.Players[0].Hand = []Card{{ID: "indians-1", Type: CardIndians}}
	state.Players[1].Hand = []Card{{ID: "bang-2", Type: CardBang}}
	state.Players[2].Hand = []Card{}
	state.Players[3].Hand = []Card{{ID: "bang-4", Type: CardBang}}

	result, err := module.ApplyAction(context.Background(), state, gamecore.Action{
		Type:     ActionPlay,
		PlayerID: "p1",
		Payload:  map[string]any{"cardId": "indians-1"},
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}
	first := result.State.(State)
	if first.PendingAttack == nil || first.PendingAttack.TargetPlayerID != "p2" {
		t.Fatalf("expected p2 bang response, got %+v", first.PendingAttack)
	}

	result, err = module.ApplyAction(context.Background(), first, gamecore.Action{
		Type:     ActionUseBang,
		PlayerID: "p2",
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}
	second := result.State.(State)
	if second.Players[2].HP != 3 {
		t.Fatalf("expected p3 without bang to take damage, got %+v", second.Players[2])
	}
	if second.PendingAttack == nil || second.PendingAttack.TargetPlayerID != "p4" {
		t.Fatalf("expected p4 bang response, got %+v", second.PendingAttack)
	}

	result, err = module.ApplyAction(context.Background(), second, gamecore.Action{
		Type:     ActionTakeHit,
		PlayerID: "p4",
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}
	done := result.State.(State)
	if done.PendingAttack != nil || done.Players[3].HP != 3 || len(done.Players[3].Hand) != 1 {
		t.Fatalf("expected p4 to take damage and keep bang, got pending=%+v player=%+v", done.PendingAttack, done.Players[3])
	}
	if len(done.Discard) != 2 || done.Discard[0].Type != CardIndians || done.Discard[1].Type != CardBang {
		t.Fatalf("expected indians and p2 bang in discard, got %+v", done.Discard)
	}
}

func TestDuelAlternatesBangResponsesUntilFailure(t *testing.T) {
	module := NewModule()
	state := fixedState()
	state.Players[0].Drawn = true
	state.Players[0].Hand = []Card{{ID: "duel-1", Type: CardDuel}, {ID: "bang-1", Type: CardBang}}
	state.Players[1].Hand = []Card{{ID: "bang-2", Type: CardBang}}

	result, err := module.ApplyAction(context.Background(), state, gamecore.Action{
		Type:     ActionPlay,
		PlayerID: "p1",
		Payload:  map[string]any{"cardId": "duel-1", "targetPlayerId": "p2"},
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}
	first := result.State.(State)
	if first.PendingAttack == nil || first.PendingAttack.TargetPlayerID != "p2" {
		t.Fatalf("expected challenged player to answer first, got %+v", first.PendingAttack)
	}

	result, err = module.ApplyAction(context.Background(), first, gamecore.Action{
		Type:     ActionUseBang,
		PlayerID: "p2",
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}
	second := result.State.(State)
	if second.PendingAttack == nil || second.PendingAttack.TargetPlayerID != "p1" {
		t.Fatalf("expected challenger to answer after target bang, got %+v", second.PendingAttack)
	}

	result, err = module.ApplyAction(context.Background(), second, gamecore.Action{
		Type:     ActionUseBang,
		PlayerID: "p1",
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}
	done := result.State.(State)
	if done.PendingAttack != nil || done.Players[1].HP != 3 {
		t.Fatalf("expected p2 to lose duel after running out of bang cards, got pending=%+v player=%+v", done.PendingAttack, done.Players[1])
	}
	if len(done.Discard) != 3 {
		t.Fatalf("expected duel and two bang cards discarded, got %+v", done.Discard)
	}
}

func TestGeneralStoreRevealsCardsAndChoosesInSeatOrder(t *testing.T) {
	module := NewModule()
	state := fixedState()
	state.Players[0].Drawn = true
	state.Players[0].Hand = []Card{{ID: "store-1", Type: CardGeneralStore}}
	state.Deck = []Card{
		{ID: "offer-1", Type: CardBang},
		{ID: "offer-2", Type: CardBeer},
		{ID: "offer-3", Type: CardMissed},
		{ID: "offer-4", Type: CardPanic},
	}

	result, err := module.ApplyAction(context.Background(), state, gamecore.Action{
		Type:     ActionPlay,
		PlayerID: "p1",
		Payload:  map[string]any{"cardId": "store-1"},
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}
	next := result.State.(State)
	if next.PendingGeneralStore == nil || next.PendingGeneralStore.CurrentChooserID != "p1" || len(next.PendingGeneralStore.Offer) != 4 {
		t.Fatalf("expected p1 to choose from four offers, got %+v", next.PendingGeneralStore)
	}

	err = module.ValidateAction(context.Background(), next, gamecore.Action{
		Type:     ActionChoose,
		PlayerID: "p2",
		Payload:  map[string]any{"cardId": "offer-1"},
	}, testContext())
	if err == nil {
		t.Fatal("expected non-current chooser to be rejected")
	}

	for _, choice := range []struct {
		playerID string
		cardID   string
		nextID   string
	}{
		{playerID: "p1", cardID: "offer-2", nextID: "p2"},
		{playerID: "p2", cardID: "offer-1", nextID: "p3"},
		{playerID: "p3", cardID: "offer-3", nextID: "p4"},
		{playerID: "p4", cardID: "offer-4", nextID: ""},
	} {
		result, err = module.ApplyAction(context.Background(), next, gamecore.Action{
			Type:     ActionChoose,
			PlayerID: gamecore.PlayerID(choice.playerID),
			Payload:  map[string]any{"cardId": choice.cardID},
		}, testContext())
		if err != nil {
			t.Fatal(err)
		}
		next = result.State.(State)
		if choice.nextID == "" {
			if next.PendingGeneralStore != nil {
				t.Fatalf("expected general store to finish, got %+v", next.PendingGeneralStore)
			}
			continue
		}
		if next.PendingGeneralStore == nil || next.PendingGeneralStore.CurrentChooserID != choice.nextID {
			t.Fatalf("expected next chooser %s, got %+v", choice.nextID, next.PendingGeneralStore)
		}
	}
	if len(next.Players[0].Hand) != 1 || next.Players[0].Hand[0].ID != "offer-2" {
		t.Fatalf("expected p1 to receive selected offer, got %+v", next.Players[0].Hand)
	}
	if len(next.Deck) != 0 {
		t.Fatalf("expected offers to leave deck, got %d", len(next.Deck))
	}
}

func TestBarrelHeartDrawAvoidsBangBeforeMissed(t *testing.T) {
	module := NewModule()
	state := fixedState()
	state.Players[0].Drawn = true
	state.Players[0].Hand = []Card{{ID: "bang-1", Type: CardBang}}
	state.Players[1].Hand = []Card{}
	state.Players[1].Equipment = []Card{{ID: "barrel-1", Type: CardBarrel}}
	state.Deck = []Card{{ID: "check-1", Type: CardBeer, Suit: SuitHeart, Rank: 6}}

	result, err := module.ApplyAction(context.Background(), state, gamecore.Action{
		Type:     ActionPlay,
		PlayerID: "p1",
		Payload:  map[string]any{"cardId": "bang-1", "targetPlayerId": "p2"},
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}
	next := result.State.(State)
	if next.PendingAttack != nil || next.Players[1].HP != 4 {
		t.Fatalf("expected barrel heart draw to avoid attack, got pending=%+v player=%+v", next.PendingAttack, next.Players[1])
	}
	if len(next.Discard) != 2 || next.Discard[1].ID != "check-1" {
		t.Fatalf("expected bang and draw check card in discard, got %+v", next.Discard)
	}
}

func TestBarrelFailedDrawFallsBackToMissedResponse(t *testing.T) {
	module := NewModule()
	state := fixedState()
	state.Players[0].Drawn = true
	state.Players[0].Hand = []Card{{ID: "bang-1", Type: CardBang}}
	state.Players[1].Hand = []Card{{ID: "missed-1", Type: CardMissed}}
	state.Players[1].Equipment = []Card{{ID: "barrel-1", Type: CardBarrel}}
	state.Deck = []Card{{ID: "check-1", Type: CardBang, Suit: SuitSpade, Rank: 4}}

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
		t.Fatalf("expected missed response after failed barrel, got %+v", next.PendingAttack)
	}
	if len(next.Discard) != 2 || next.Discard[1].ID != "check-1" {
		t.Fatalf("expected failed draw check in discard, got %+v", next.Discard)
	}
}

func TestJailCannotTargetSheriff(t *testing.T) {
	module := NewModule()
	state := fixedState()
	state.CurrentPlayerIndex = 1
	state.Players[1].Drawn = true
	state.Players[1].Hand = []Card{{ID: "jail-1", Type: CardJail}}

	err := module.ValidateAction(context.Background(), state, gamecore.Action{
		Type:     ActionPlay,
		PlayerID: "p2",
		Payload:  map[string]any{"cardId": "jail-1", "targetPlayerId": "p1"},
	}, testContext())
	if err == nil {
		t.Fatal("expected jail on sheriff to be rejected")
	}
}

func TestJailFailedDrawSkipsTurn(t *testing.T) {
	module := NewModule()
	state := fixedState()
	state.Players[0].Equipment = []Card{{ID: "jail-1", Type: CardJail}}
	state.Deck = []Card{{ID: "check-1", Type: CardBang, Suit: SuitSpade, Rank: 4}}

	result, err := module.ApplyAction(context.Background(), state, gamecore.Action{
		Type:     ActionDraw,
		PlayerID: "p1",
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}
	next := result.State.(State)
	if next.CurrentPlayerIndex != 1 || next.Round != 1 || next.Players[0].Drawn {
		t.Fatalf("expected jail to skip p1 turn, got index=%d round=%d player=%+v", next.CurrentPlayerIndex, next.Round, next.Players[0])
	}
	if len(next.Players[0].Equipment) != 0 || len(next.Discard) != 2 {
		t.Fatalf("expected jail and draw check discarded, got equipment=%+v discard=%+v", next.Players[0].Equipment, next.Discard)
	}
}

func TestDynamiteExplodesBeforeDrawAndPlayerContinuesIfAlive(t *testing.T) {
	module := NewModule()
	state := fixedState()
	state.Players[0].Equipment = []Card{{ID: "dynamite-1", Type: CardDynamite}}
	state.Deck = []Card{
		{ID: "check-1", Type: CardBang, Suit: SuitSpade, Rank: 5},
		{ID: "draw-1", Type: CardBang},
		{ID: "draw-2", Type: CardBeer},
	}

	result, err := module.ApplyAction(context.Background(), state, gamecore.Action{
		Type:     ActionDraw,
		PlayerID: "p1",
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}
	next := result.State.(State)
	if next.Players[0].HP != 2 || !next.Players[0].Drawn || len(next.Players[0].Hand) != 2 {
		t.Fatalf("expected p1 to lose three hp then draw, got %+v", next.Players[0])
	}
	if len(next.Players[0].Equipment) != 0 || len(next.Discard) != 2 {
		t.Fatalf("expected dynamite and draw check discarded, got equipment=%+v discard=%+v", next.Players[0].Equipment, next.Discard)
	}
}

func TestDynamiteSafeDrawMovesToNextAlivePlayer(t *testing.T) {
	module := NewModule()
	state := fixedState()
	state.Players[0].Equipment = []Card{{ID: "dynamite-1", Type: CardDynamite}}
	state.Deck = []Card{
		{ID: "check-1", Type: CardBeer, Suit: SuitHeart, Rank: 6},
		{ID: "draw-1", Type: CardBang},
		{ID: "draw-2", Type: CardBeer},
	}

	result, err := module.ApplyAction(context.Background(), state, gamecore.Action{
		Type:     ActionDraw,
		PlayerID: "p1",
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}
	next := result.State.(State)
	if len(next.Players[0].Equipment) != 0 || len(next.Players[1].Equipment) != 1 || next.Players[1].Equipment[0].Type != CardDynamite {
		t.Fatalf("expected dynamite to move to p2, got p1=%+v p2=%+v", next.Players[0].Equipment, next.Players[1].Equipment)
	}
	if !next.Players[0].Drawn || len(next.Players[0].Hand) != 2 {
		t.Fatalf("expected p1 to continue draw phase, got %+v", next.Players[0])
	}
}

func TestLuckyDukeChoosesSafeDynamiteDraw(t *testing.T) {
	module := NewModule()
	state := fixedState()
	state.Players[0].CharacterID = CharacterLucky
	state.Players[0].Equipment = []Card{{ID: "dynamite-1", Type: CardDynamite}}
	state.Deck = []Card{
		{ID: "check-bad", Type: CardBang, Suit: SuitSpade, Rank: 5},
		{ID: "check-good", Type: CardBeer, Suit: SuitHeart, Rank: 6},
		{ID: "draw-1", Type: CardBang},
		{ID: "draw-2", Type: CardBeer},
	}

	result, err := module.ApplyAction(context.Background(), state, gamecore.Action{
		Type:     ActionDraw,
		PlayerID: "p1",
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}
	next := result.State.(State)
	if next.Players[0].HP != 5 || len(next.Players[1].Equipment) != 1 || next.Players[1].Equipment[0].Type != CardDynamite {
		t.Fatalf("expected lucky duke to choose safe dynamite result, got p1=%+v p2=%+v", next.Players[0], next.Players[1])
	}
	if len(next.Discard) != 2 {
		t.Fatalf("expected both draw check cards discarded, got %+v", next.Discard)
	}
}

func TestLuckyDukeChoosesHeartToEscapeJail(t *testing.T) {
	module := NewModule()
	state := fixedState()
	state.Players[0].CharacterID = CharacterLucky
	state.Players[0].Equipment = []Card{{ID: "jail-1", Type: CardJail}}
	state.Deck = []Card{
		{ID: "check-bad", Type: CardBang, Suit: SuitSpade, Rank: 4},
		{ID: "check-good", Type: CardBeer, Suit: SuitHeart, Rank: 6},
		{ID: "draw-1", Type: CardBang},
		{ID: "draw-2", Type: CardBeer},
	}

	result, err := module.ApplyAction(context.Background(), state, gamecore.Action{
		Type:     ActionDraw,
		PlayerID: "p1",
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}
	next := result.State.(State)
	if next.CurrentPlayerIndex != 0 || !next.Players[0].Drawn || len(next.Players[0].Hand) != 2 {
		t.Fatalf("expected lucky duke to escape jail and draw, got index=%d player=%+v", next.CurrentPlayerIndex, next.Players[0])
	}
	if len(next.Discard) != 3 {
		t.Fatalf("expected two draw checks and jail discarded, got %+v", next.Discard)
	}
}

func TestWillyTheKidCanPlayMultipleBangCards(t *testing.T) {
	module := NewModule()
	state := fixedState()
	state.Players[0].CharacterID = CharacterWilly
	state.Players[0].Drawn = true
	state.Players[0].Hand = []Card{{ID: "bang-1", Type: CardBang}, {ID: "bang-2", Type: CardBang}}
	state.Players[1].Hand = []Card{}

	result, err := module.ApplyAction(context.Background(), state, gamecore.Action{
		Type:     ActionPlay,
		PlayerID: "p1",
		Payload:  map[string]any{"cardId": "bang-1", "targetPlayerId": "p2"},
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}
	next := result.State.(State)
	err = module.ValidateAction(context.Background(), next, gamecore.Action{
		Type:     ActionPlay,
		PlayerID: "p1",
		Payload:  map[string]any{"cardId": "bang-2", "targetPlayerId": "p2"},
	}, testContext())
	if err != nil {
		t.Fatalf("expected willy to allow second bang, got %v", err)
	}
}

func TestRoseAndPaulAdjustDistanceLikeScopeAndMustang(t *testing.T) {
	module := NewModule()
	state := fixedState()
	state.Players[0].CharacterID = CharacterRose
	state.Players[0].Drawn = true
	state.Players[0].Hand = []Card{{ID: "bang-1", Type: CardBang}}

	err := module.ValidateAction(context.Background(), state, gamecore.Action{
		Type:     ActionPlay,
		PlayerID: "p1",
		Payload:  map[string]any{"cardId": "bang-1", "targetPlayerId": "p3"},
	}, testContext())
	if err != nil {
		t.Fatalf("expected rose to see p3 at distance one, got %v", err)
	}

	state.Players[0].CharacterID = ""
	state.Players[2].CharacterID = CharacterPaul
	err = module.ValidateAction(context.Background(), state, gamecore.Action{
		Type:     ActionPlay,
		PlayerID: "p1",
		Payload:  map[string]any{"cardId": "bang-1", "targetPlayerId": "p2"},
	}, testContext())
	if err != nil {
		t.Fatalf("expected paul not to affect p2, got %v", err)
	}
	err = module.ValidateAction(context.Background(), state, gamecore.Action{
		Type:     ActionPlay,
		PlayerID: "p1",
		Payload:  map[string]any{"cardId": "bang-1", "targetPlayerId": "p3"},
	}, testContext())
	if err == nil {
		t.Fatal("expected paul to push p3 farther away")
	}
}

func TestJourdonnaisHasBuiltInBarrel(t *testing.T) {
	module := NewModule()
	state := fixedState()
	state.Players[0].Drawn = true
	state.Players[0].Hand = []Card{{ID: "bang-1", Type: CardBang}}
	state.Players[1].CharacterID = CharacterJourdonnais
	state.Players[1].Hand = []Card{}
	state.Deck = []Card{{ID: "check-1", Type: CardBeer, Suit: SuitHeart, Rank: 6}}

	result, err := module.ApplyAction(context.Background(), state, gamecore.Action{
		Type:     ActionPlay,
		PlayerID: "p1",
		Payload:  map[string]any{"cardId": "bang-1", "targetPlayerId": "p2"},
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}
	next := result.State.(State)
	if next.PendingAttack != nil || next.Players[1].HP != 4 {
		t.Fatalf("expected jourdonnais barrel ability to avoid bang, got pending=%+v player=%+v", next.PendingAttack, next.Players[1])
	}
}

func TestBlackJackDrawsExtraOnRedSecondCard(t *testing.T) {
	module := NewModule()
	state := fixedState()
	state.Players[0].CharacterID = CharacterBlackJack
	state.Deck = []Card{
		{ID: "draw-1", Type: CardBang, Suit: SuitSpade, Rank: 4},
		{ID: "draw-2", Type: CardBeer, Suit: SuitHeart, Rank: 6},
		{ID: "draw-3", Type: CardMissed, Suit: SuitClub, Rank: 10},
	}

	result, err := module.ApplyAction(context.Background(), state, gamecore.Action{
		Type:     ActionDraw,
		PlayerID: "p1",
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}
	next := result.State.(State)
	if len(next.Players[0].Hand) != 3 || len(next.Deck) != 0 {
		t.Fatalf("expected black jack to draw three cards, got hand=%+v deck=%+v", next.Players[0].Hand, next.Deck)
	}
}

func TestKitCarlsonChoosesTwoOfThreeDrawCards(t *testing.T) {
	module := NewModule()
	state := fixedState()
	state.Players[0].CharacterID = CharacterKit
	state.Deck = []Card{
		{ID: "kit-1", Type: CardBang},
		{ID: "kit-2", Type: CardBeer},
		{ID: "kit-3", Type: CardMissed},
		{ID: "after-1", Type: CardPanic},
	}

	result, err := module.ApplyAction(context.Background(), state, gamecore.Action{
		Type:     ActionDraw,
		PlayerID: "p1",
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}
	next := result.State.(State)
	if next.PendingCharacterChoice == nil || next.PendingCharacterChoice.ChoiceType != ChoiceKitDraw {
		t.Fatalf("expected kit draw choice, got %+v", next.PendingCharacterChoice)
	}
	if next.Players[0].Drawn || len(next.Players[0].Hand) != 0 {
		t.Fatalf("expected draw phase to wait for kit choice, got %+v", next.Players[0])
	}

	publicForOther := module.PublicState(next, "p2").(State)
	if len(publicForOther.PendingCharacterChoice.Cards) != 3 || publicForOther.PendingCharacterChoice.Cards[0].Type != "hidden" {
		t.Fatalf("expected kit choices masked for other player, got %+v", publicForOther.PendingCharacterChoice.Cards)
	}
	publicForKit := module.PublicState(next, "p1").(State)
	if publicForKit.PendingCharacterChoice.Cards[0].ID != "kit-1" {
		t.Fatalf("expected kit player to see choices, got %+v", publicForKit.PendingCharacterChoice.Cards)
	}

	result, err = module.ApplyAction(context.Background(), next, gamecore.Action{
		Type:     ActionChooseCharacter,
		PlayerID: "p1",
		Payload:  map[string]any{"cardIds": []any{"kit-2", "kit-3"}},
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}
	done := result.State.(State)
	if done.PendingCharacterChoice != nil || !done.Players[0].Drawn || len(done.Players[0].Hand) != 2 {
		t.Fatalf("expected kit choice to complete draw, got pending=%+v player=%+v", done.PendingCharacterChoice, done.Players[0])
	}
	if len(done.Deck) != 2 || done.Deck[0].ID != "kit-1" || done.Deck[1].ID != "after-1" {
		t.Fatalf("expected unchosen card on top of deck, got %+v", done.Deck)
	}
}

func TestPedroRamirezMayDrawFirstCardFromDiscard(t *testing.T) {
	module := NewModule()
	state := fixedState()
	state.Players[0].CharacterID = CharacterPedro
	state.Discard = []Card{{ID: "old-1", Type: CardBang}, {ID: "top-discard", Type: CardBeer}}
	state.Deck = []Card{{ID: "deck-1", Type: CardMissed}}

	result, err := module.ApplyAction(context.Background(), state, gamecore.Action{
		Type:     ActionDraw,
		PlayerID: "p1",
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}
	next := result.State.(State)
	if next.PendingCharacterChoice == nil || next.PendingCharacterChoice.ChoiceType != ChoicePedroDraw {
		t.Fatalf("expected pedro choice, got %+v", next.PendingCharacterChoice)
	}

	result, err = module.ApplyAction(context.Background(), next, gamecore.Action{
		Type:     ActionChooseCharacter,
		PlayerID: "p1",
		Payload:  map[string]any{"cardId": "top-discard", "useDiscard": true},
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}
	done := result.State.(State)
	if done.PendingCharacterChoice != nil || !done.Players[0].Drawn || len(done.Players[0].Hand) != 2 {
		t.Fatalf("expected pedro draw to complete, got pending=%+v player=%+v", done.PendingCharacterChoice, done.Players[0])
	}
	if done.Players[0].Hand[0].ID != "top-discard" || done.Players[0].Hand[1].ID != "deck-1" {
		t.Fatalf("expected discard then deck draw, got %+v", done.Players[0].Hand)
	}
	if len(done.Discard) != 1 || done.Discard[0].ID != "old-1" {
		t.Fatalf("expected top discard removed only, got %+v", done.Discard)
	}
}

func TestPedroRamirezMaySkipDiscardDraw(t *testing.T) {
	module := NewModule()
	state := fixedState()
	state.Players[0].CharacterID = CharacterPedro
	state.Discard = []Card{{ID: "top-discard", Type: CardBeer}}
	state.Deck = []Card{{ID: "deck-1", Type: CardMissed}, {ID: "deck-2", Type: CardBang}}

	result, err := module.ApplyAction(context.Background(), state, gamecore.Action{
		Type:     ActionDraw,
		PlayerID: "p1",
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}
	next := result.State.(State)
	result, err = module.ApplyAction(context.Background(), next, gamecore.Action{
		Type:     ActionChooseCharacter,
		PlayerID: "p1",
		Payload:  map[string]any{"useDiscard": false},
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}
	done := result.State.(State)
	if len(done.Players[0].Hand) != 2 || done.Players[0].Hand[0].ID != "deck-1" || len(done.Discard) != 1 {
		t.Fatalf("expected pedro to draw from deck and keep discard pile, got hand=%+v discard=%+v", done.Players[0].Hand, done.Discard)
	}
}

func TestSidKetchumDiscardsTwoCardsToHeal(t *testing.T) {
	module := NewModule()
	state := fixedState()
	state.Players[1].CharacterID = CharacterSid
	state.Players[1].HP = 2
	state.Players[1].Hand = []Card{{ID: "sid-1", Type: CardBang}, {ID: "sid-2", Type: CardBeer}, {ID: "sid-3", Type: CardMissed}}

	result, err := module.ApplyAction(context.Background(), state, gamecore.Action{
		Type:     ActionSidHeal,
		PlayerID: "p2",
		Payload:  map[string]any{"cardIds": []any{"sid-1", "sid-3"}},
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}
	next := result.State.(State)
	if next.Players[1].HP != 3 || len(next.Players[1].Hand) != 1 || len(next.Discard) != 2 {
		t.Fatalf("expected sid to heal by discarding two cards, got player=%+v discard=%+v", next.Players[1], next.Discard)
	}
}

func TestBartCassidyDrawsWhenLosingHP(t *testing.T) {
	state := fixedState()
	state.Players[1].CharacterID = CharacterBart
	state.Players[1].Hand = []Card{}
	state.Deck = []Card{{ID: "bart-1", Type: CardBeer}}

	next := damageTargetWithoutMissed(state, "p2", 1, "p1")
	if next.Players[1].HP != 3 || len(next.Players[1].Hand) != 1 {
		t.Fatalf("expected bart to draw after losing hp, got %+v", next.Players[1])
	}
}

func TestVultureSamTakesEliminatedPlayerCards(t *testing.T) {
	state := fixedState()
	state.Players[2].CharacterID = CharacterVulture
	state.Players[1].HP = 1
	state.Players[1].Hand = []Card{{ID: "loot-1", Type: CardCatBalou}}
	state.Players[1].Equipment = []Card{{ID: "loot-2", Type: CardBarrel}}

	next := damageTargetWithoutMissed(state, "p2", 1, "p1")
	if next.Players[1].Alive {
		t.Fatal("expected p2 to be eliminated")
	}
	if len(next.Players[2].Hand) != 2 {
		t.Fatalf("expected vulture sam to take two cards, got %+v", next.Players[2].Hand)
	}
}

func TestCalamityJanetUsesMissedAsBangAndBangAsMissed(t *testing.T) {
	module := NewModule()
	state := fixedState()
	state.Players[0].CharacterID = CharacterCalamity
	state.Players[0].Drawn = true
	state.Players[0].Hand = []Card{{ID: "missed-1", Type: CardMissed}}
	state.Players[1].Hand = []Card{}

	err := module.ValidateAction(context.Background(), state, gamecore.Action{
		Type:     ActionPlay,
		PlayerID: "p1",
		Payload:  map[string]any{"cardId": "missed-1", "targetPlayerId": "p2"},
	}, testContext())
	if err != nil {
		t.Fatalf("expected calamity to play missed as bang, got %v", err)
	}

	state = fixedState()
	state.Players[1].CharacterID = CharacterCalamity
	state.Players[1].Hand = []Card{{ID: "bang-2", Type: CardBang}}
	state.PendingAttack = &PendingAttack{
		SourcePlayerID: "p1",
		TargetPlayerID: "p2",
		CardType:       CardBang,
		Damage:         1,
	}
	err = module.ValidateAction(context.Background(), state, gamecore.Action{
		Type:     ActionUseMissed,
		PlayerID: "p2",
	}, testContext())
	if err != nil {
		t.Fatalf("expected calamity to use bang as missed, got %v", err)
	}
}

func TestSlabTheKillerRequiresTwoMissedCards(t *testing.T) {
	module := NewModule()
	state := fixedState()
	state.Players[0].CharacterID = CharacterSlab
	state.Players[0].Drawn = true
	state.Players[0].Hand = []Card{{ID: "bang-1", Type: CardBang}}
	state.Players[1].Hand = []Card{{ID: "missed-1", Type: CardMissed}, {ID: "missed-2", Type: CardMissed}}

	result, err := module.ApplyAction(context.Background(), state, gamecore.Action{
		Type:     ActionPlay,
		PlayerID: "p1",
		Payload:  map[string]any{"cardId": "bang-1", "targetPlayerId": "p2"},
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}
	next := result.State.(State)
	if next.PendingAttack == nil || next.PendingAttack.RequiredResponseCount != 2 {
		t.Fatalf("expected two missed responses, got %+v", next.PendingAttack)
	}

	result, err = module.ApplyAction(context.Background(), next, gamecore.Action{
		Type:     ActionUseMissed,
		PlayerID: "p2",
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}
	next = result.State.(State)
	if next.PendingAttack == nil || next.PendingAttack.RequiredResponseCount != 1 || len(next.Players[1].Hand) != 1 {
		t.Fatalf("expected one more missed response, got pending=%+v player=%+v", next.PendingAttack, next.Players[1])
	}

	result, err = module.ApplyAction(context.Background(), next, gamecore.Action{
		Type:     ActionUseMissed,
		PlayerID: "p2",
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}
	done := result.State.(State)
	if done.PendingAttack != nil || done.Players[1].HP != 4 || len(done.Players[1].Hand) != 0 {
		t.Fatalf("expected two missed cards to avoid slab bang, got pending=%+v player=%+v", done.PendingAttack, done.Players[1])
	}
}

func TestSlabTheKillerBarrelSuccessCountsAsOneMissedOnly(t *testing.T) {
	module := NewModule()
	state := fixedState()
	state.Players[0].CharacterID = CharacterSlab
	state.Players[0].Drawn = true
	state.Players[0].Hand = []Card{{ID: "bang-1", Type: CardBang}}
	state.Players[1].Hand = []Card{{ID: "missed-1", Type: CardMissed}}
	state.Players[1].Equipment = []Card{{ID: "barrel-1", Type: CardBarrel}}
	state.Deck = []Card{{ID: "check-1", Type: CardBeer, Suit: SuitHeart, Rank: 6}}

	result, err := module.ApplyAction(context.Background(), state, gamecore.Action{
		Type:     ActionPlay,
		PlayerID: "p1",
		Payload:  map[string]any{"cardId": "bang-1", "targetPlayerId": "p2"},
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}
	next := result.State.(State)
	if next.PendingAttack == nil || next.PendingAttack.RequiredResponseCount != 1 {
		t.Fatalf("expected barrel to satisfy one of two missed responses, got %+v", next.PendingAttack)
	}

	result, err = module.ApplyAction(context.Background(), next, gamecore.Action{
		Type:     ActionUseMissed,
		PlayerID: "p2",
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}
	done := result.State.(State)
	if done.PendingAttack != nil || done.Players[1].HP != 4 {
		t.Fatalf("expected barrel plus missed to avoid slab bang, got pending=%+v player=%+v", done.PendingAttack, done.Players[1])
	}
}

func TestSlabTheKillerBarrelAloneStillDealsDamage(t *testing.T) {
	module := NewModule()
	state := fixedState()
	state.Players[0].CharacterID = CharacterSlab
	state.Players[0].Drawn = true
	state.Players[0].Hand = []Card{{ID: "bang-1", Type: CardBang}}
	state.Players[1].Hand = []Card{}
	state.Players[1].Equipment = []Card{{ID: "barrel-1", Type: CardBarrel}}
	state.Deck = []Card{{ID: "check-1", Type: CardBeer, Suit: SuitHeart, Rank: 6}}

	result, err := module.ApplyAction(context.Background(), state, gamecore.Action{
		Type:     ActionPlay,
		PlayerID: "p1",
		Payload:  map[string]any{"cardId": "bang-1", "targetPlayerId": "p2"},
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}
	next := result.State.(State)
	if next.PendingAttack != nil || next.Players[1].HP != 3 {
		t.Fatalf("expected barrel alone not to stop slab bang, got pending=%+v player=%+v", next.PendingAttack, next.Players[1])
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

func TestBangRequiresTargetWithinRange(t *testing.T) {
	module := NewModule()
	state := fixedState()
	state.Players[0].Drawn = true
	state.Players[0].Hand = []Card{{ID: "bang-1", Type: CardBang}}

	err := module.ValidateAction(context.Background(), state, gamecore.Action{
		Type:     ActionPlay,
		PlayerID: "p1",
		Payload:  map[string]any{"cardId": "bang-1", "targetPlayerId": "p3"},
	}, testContext())
	if err == nil {
		t.Fatal("expected default range one to reject distance two target")
	}
}

func TestWeaponExtendsBangRange(t *testing.T) {
	module := NewModule()
	state := fixedState()
	state.Players[0].Drawn = true
	state.Players[0].Hand = []Card{{ID: "bang-1", Type: CardBang}}
	state.Players[0].Equipment = []Card{{ID: "schofield-1", Type: CardSchofield}}

	err := module.ValidateAction(context.Background(), state, gamecore.Action{
		Type:     ActionPlay,
		PlayerID: "p1",
		Payload:  map[string]any{"cardId": "bang-1", "targetPlayerId": "p3"},
	}, testContext())
	if err != nil {
		t.Fatalf("expected range two weapon to allow p3 target, got %v", err)
	}
}

func TestScopeAndMustangAdjustDistance(t *testing.T) {
	module := NewModule()
	state := fixedState()
	state.Players[0].Drawn = true
	state.Players[0].Hand = []Card{{ID: "bang-1", Type: CardBang}}
	state.Players[0].Equipment = []Card{{ID: "scope-1", Type: CardScope}}

	err := module.ValidateAction(context.Background(), state, gamecore.Action{
		Type:     ActionPlay,
		PlayerID: "p1",
		Payload:  map[string]any{"cardId": "bang-1", "targetPlayerId": "p3"},
	}, testContext())
	if err != nil {
		t.Fatalf("expected scope to bring p3 into default range, got %v", err)
	}

	state.Players[2].Equipment = []Card{{ID: "mustang-1", Type: CardMustang}}
	err = module.ValidateAction(context.Background(), state, gamecore.Action{
		Type:     ActionPlay,
		PlayerID: "p1",
		Payload:  map[string]any{"cardId": "bang-1", "targetPlayerId": "p3"},
	}, testContext())
	if err == nil {
		t.Fatal("expected target mustang to push p3 out of default range")
	}
}

func TestPlayingWeaponEquipsAndReplacesOldWeapon(t *testing.T) {
	module := NewModule()
	state := fixedState()
	state.Players[0].Drawn = true
	state.Players[0].Hand = []Card{{ID: "remington-1", Type: CardRemington}}
	state.Players[0].Equipment = []Card{{ID: "schofield-1", Type: CardSchofield}}

	result, err := module.ApplyAction(context.Background(), state, gamecore.Action{
		Type:     ActionPlay,
		PlayerID: "p1",
		Payload:  map[string]any{"cardId": "remington-1"},
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}
	next := result.State.(State)
	if len(next.Players[0].Equipment) != 1 || next.Players[0].Equipment[0].Type != CardRemington {
		t.Fatalf("expected remington equipped, got %+v", next.Players[0].Equipment)
	}
	if len(next.Discard) != 1 || next.Discard[0].Type != CardSchofield {
		t.Fatalf("expected old weapon discarded, got %+v", next.Discard)
	}
}

func TestVolcanicAllowsMultipleBangCardsInOneTurn(t *testing.T) {
	module := NewModule()
	state := fixedState()
	state.Players[0].Drawn = true
	state.Players[0].Hand = []Card{{ID: "bang-1", Type: CardBang}, {ID: "bang-2", Type: CardBang}}
	state.Players[0].Equipment = []Card{{ID: "volcanic-1", Type: CardVolcanic}}
	state.Players[1].Hand = []Card{}

	result, err := module.ApplyAction(context.Background(), state, gamecore.Action{
		Type:     ActionPlay,
		PlayerID: "p1",
		Payload:  map[string]any{"cardId": "bang-1", "targetPlayerId": "p2"},
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}
	next := result.State.(State)
	err = module.ValidateAction(context.Background(), next, gamecore.Action{
		Type:     ActionPlay,
		PlayerID: "p1",
		Payload:  map[string]any{"cardId": "bang-2", "targetPlayerId": "p2"},
	}, testContext())
	if err != nil {
		t.Fatalf("expected volcanic to allow another bang, got %v", err)
	}

	result, err = module.ApplyAction(context.Background(), next, gamecore.Action{
		Type:     ActionPlay,
		PlayerID: "p1",
		Payload:  map[string]any{"cardId": "bang-2", "targetPlayerId": "p2"},
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}
	next = result.State.(State)
	if next.Players[1].HP != 2 {
		t.Fatalf("expected two bang damage, got %+v", next.Players[1])
	}
}

func TestStagecoachAndWellsFargoDrawCards(t *testing.T) {
	module := NewModule()
	state := fixedState()
	state.Players[0].Drawn = true
	state.Players[0].Hand = []Card{{ID: "stagecoach-1", Type: CardStagecoach}, {ID: "wells-1", Type: CardWellsFargo}}
	state.Deck = []Card{
		{ID: "draw-1", Type: CardBang},
		{ID: "draw-2", Type: CardMissed},
		{ID: "draw-3", Type: CardBeer},
		{ID: "draw-4", Type: CardBang},
		{ID: "draw-5", Type: CardBang},
	}

	result, err := module.ApplyAction(context.Background(), state, gamecore.Action{
		Type:     ActionPlay,
		PlayerID: "p1",
		Payload:  map[string]any{"cardId": "stagecoach-1"},
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}
	next := result.State.(State)
	if len(next.Players[0].Hand) != 3 || len(next.Deck) != 3 {
		t.Fatalf("expected stagecoach to draw two cards, got hand=%+v deck=%d", next.Players[0].Hand, len(next.Deck))
	}

	result, err = module.ApplyAction(context.Background(), next, gamecore.Action{
		Type:     ActionPlay,
		PlayerID: "p1",
		Payload:  map[string]any{"cardId": "wells-1"},
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}
	next = result.State.(State)
	if len(next.Players[0].Hand) != 5 || len(next.Deck) != 0 {
		t.Fatalf("expected wells fargo to draw three cards, got hand=%+v deck=%d", next.Players[0].Hand, len(next.Deck))
	}
}

func TestSaloonHealsAllAlivePlayers(t *testing.T) {
	module := NewModule()
	state := fixedState()
	state.Players[0].Drawn = true
	state.Players[0].Hand = []Card{{ID: "saloon-1", Type: CardSaloon}}
	state.Players[0].HP = 4
	state.Players[1].HP = 2
	state.Players[2].HP = 4
	state.Players[3].HP = 0
	state.Players[3].Alive = false

	result, err := module.ApplyAction(context.Background(), state, gamecore.Action{
		Type:     ActionPlay,
		PlayerID: "p1",
		Payload:  map[string]any{"cardId": "saloon-1"},
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}
	next := result.State.(State)
	if next.Players[0].HP != 5 || next.Players[1].HP != 3 || next.Players[2].HP != 4 || next.Players[3].HP != 0 {
		t.Fatalf("expected saloon to heal alive damaged players only, got %+v", next.Players)
	}
}

func TestCatBalouDiscardsTargetCard(t *testing.T) {
	module := NewModule()
	state := fixedState()
	state.Players[0].Drawn = true
	state.Players[0].Hand = []Card{{ID: "cat-1", Type: CardCatBalou}}
	state.Players[1].Hand = []Card{{ID: "missed-1", Type: CardMissed}}
	state.Players[1].Equipment = []Card{{ID: "scope-1", Type: CardScope}}

	result, err := module.ApplyAction(context.Background(), state, gamecore.Action{
		Type:     ActionPlay,
		PlayerID: "p1",
		Payload:  map[string]any{"cardId": "cat-1", "targetPlayerId": "p2", "targetCardId": "scope-1"},
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}
	next := result.State.(State)
	if len(next.Players[1].Equipment) != 0 || len(next.Players[1].Hand) != 1 {
		t.Fatalf("expected cat balou to discard selected equipment only, got %+v", next.Players[1])
	}
	if len(next.Discard) != 2 || next.Discard[1].ID != "scope-1" {
		t.Fatalf("expected discarded cat balou and target card, got %+v", next.Discard)
	}
}

func TestPanicStealsDistanceOneTargetCard(t *testing.T) {
	module := NewModule()
	state := fixedState()
	state.Players[0].Drawn = true
	state.Players[0].Hand = []Card{{ID: "panic-1", Type: CardPanic}}
	state.Players[1].Hand = []Card{{ID: "beer-1", Type: CardBeer}}

	result, err := module.ApplyAction(context.Background(), state, gamecore.Action{
		Type:     ActionPlay,
		PlayerID: "p1",
		Payload:  map[string]any{"cardId": "panic-1", "targetPlayerId": "p2", "targetCardId": "beer-1"},
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}
	next := result.State.(State)
	if len(next.Players[1].Hand) != 0 || len(next.Players[0].Hand) != 1 || next.Players[0].Hand[0].ID != "beer-1" {
		t.Fatalf("expected panic to steal target card, got actor=%+v target=%+v", next.Players[0], next.Players[1])
	}

	state = fixedState()
	state.Players[0].Drawn = true
	state.Players[0].Hand = []Card{{ID: "panic-1", Type: CardPanic}}
	state.Players[2].Hand = []Card{{ID: "beer-1", Type: CardBeer}}
	err = module.ValidateAction(context.Background(), state, gamecore.Action{
		Type:     ActionPlay,
		PlayerID: "p1",
		Payload:  map[string]any{"cardId": "panic-1", "targetPlayerId": "p3"},
	}, testContext())
	if err == nil {
		t.Fatal("expected panic to reject distance two target")
	}
}

func TestImplementedDeckUsesOfficialCoreActionCounts(t *testing.T) {
	counts := map[string]int{}
	for _, card := range shuffledDeck("counts") {
		counts[card.Type]++
	}
	expected := map[string]int{
		CardBang:         25,
		CardMissed:       12,
		CardBeer:         6,
		CardGatling:      1,
		CardBarrel:       2,
		CardJail:         3,
		CardDynamite:     1,
		CardStagecoach:   2,
		CardWellsFargo:   1,
		CardSaloon:       1,
		CardCatBalou:     4,
		CardPanic:        4,
		CardDuel:         3,
		CardIndians:      2,
		CardGeneralStore: 2,
		CardMustang:      2,
		CardScope:        1,
		CardVolcanic:     2,
		CardSchofield:    3,
		CardRemington:    1,
		CardCarabine:     1,
		CardWinchester:   1,
	}
	for cardType, want := range expected {
		if counts[cardType] != want {
			t.Fatalf("expected %s count %d, got %d in %+v", cardType, want, counts[cardType], counts)
		}
	}
	if len(shuffledDeck("counts")) != 80 {
		t.Fatalf("expected 80 base cards, got %d", len(shuffledDeck("counts")))
	}
	for _, card := range shuffledDeck("counts") {
		if card.Suit == "" || card.Rank == 0 {
			t.Fatalf("expected official suit/rank on card %+v", card)
		}
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
