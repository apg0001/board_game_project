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
	next := damageTarget(state, "p2", 1)
	if next.Players[1].HP != 4 || len(next.Players[1].Hand) != 0 {
		t.Fatalf("expected missed to prevent damage, got %+v", next.Players[1])
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
