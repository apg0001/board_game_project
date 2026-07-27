package sutda

import (
	"context"
	"testing"
	"time"

	"board-game-platform/apps/api/internal/gamecore"
)

func TestEvaluateRanks(t *testing.T) {
	rank, name := evaluate([]Card{{Month: 3, Gwang: true}, {Month: 8, Gwang: true}})
	if name != "38광땡" || rank < 1000 {
		t.Fatalf("expected 38 gwang, got %s %d", name, rank)
	}
	rank, name = evaluate([]Card{{Month: 1}, {Month: 2}})
	if name != "알리" || rank != 700 {
		t.Fatalf("expected ali, got %s %d", name, rank)
	}
	rank, name = evaluate([]Card{{Month: 4}, {Month: 5}})
	if name != "갑오" || rank != 609 {
		t.Fatalf("expected gap-o, got %s %d", name, rank)
	}
}

func TestCompareHands(t *testing.T) {
	left := []Card{{Month: 10}, {Month: 10}}
	right := []Card{{Month: 1}, {Month: 2}}
	if compare(left, right) <= 0 {
		t.Fatal("expected jang-ttaeng to beat ali")
	}
}

func TestAmhaengEosaBeatsNonThirtyEightGwangTtaeng(t *testing.T) {
	amhaeng := []Card{{Month: 4}, {Month: 7}}
	gwang := []Card{{Month: 1, Gwang: true}, {Month: 3, Gwang: true}}
	thirtyEight := []Card{{Month: 3, Gwang: true}, {Month: 8, Gwang: true}}

	if rank, name := evaluate(amhaeng); name != "암행어사" || rank != 501 {
		t.Fatalf("expected amhaeng-eosa label, got %s %d", name, rank)
	}
	if compare(amhaeng, gwang) <= 0 {
		t.Fatal("expected amhaeng-eosa to beat non-38 gwang-ttaeng")
	}
	if compare(amhaeng, thirtyEight) >= 0 {
		t.Fatal("expected amhaeng-eosa to lose to 38 gwang-ttaeng")
	}
}

func TestTtaengJabiBeatsOrdinaryTtaengOnly(t *testing.T) {
	ttaengJabi := []Card{{Month: 3}, {Month: 7}}
	ordinaryTtaeng := []Card{{Month: 9}, {Month: 9}}
	gwangTtaeng := []Card{{Month: 1, Gwang: true}, {Month: 8, Gwang: true}}

	if rank, name := evaluate(ttaengJabi); name != "땡잡이" || rank != 500 {
		t.Fatalf("expected ttaeng-jabi label, got %s %d", name, rank)
	}
	if compare(ttaengJabi, ordinaryTtaeng) <= 0 {
		t.Fatal("expected ttaeng-jabi to beat ordinary ttaeng")
	}
	if compare(ttaengJabi, gwangTtaeng) >= 0 {
		t.Fatal("expected ttaeng-jabi to lose to gwang-ttaeng")
	}
}

func TestFinishMarksShowdownDrawWhenBestHandsTie(t *testing.T) {
	module := NewModule()
	state := State{
		Players: []PlayerState{
			{PlayerID: "p1", Hand: []Card{{ID: "p1-1", Month: 1}, {ID: "p1-2", Month: 2}}, Active: true},
			{PlayerID: "p2", Hand: []Card{{ID: "p2-1", Month: 1}, {ID: "p2-2", Month: 2}}, Active: true},
			{PlayerID: "p3", Hand: []Card{{ID: "p3-1", Month: 4}, {ID: "p3-2", Month: 5}}, Active: true},
		},
	}

	done := finish(state)
	if !done.Draw || done.WinnerID != "" || len(done.WinnerIDs) != 2 || done.WinnerIDs[0] != "p1" || done.WinnerIDs[1] != "p2" {
		t.Fatalf("expected p1/p2 draw, got %+v", done)
	}
	results := module.CalculateResult(done, testContext())
	for _, result := range results {
		if result.PlayerID == "p1" || result.PlayerID == "p2" {
			if result.Outcome != gamecore.OutcomeDraw || result.Rank != 1 {
				t.Fatalf("expected tied winners to be draw rank 1, got %+v", result)
			}
		}
		if result.PlayerID == "p3" && result.Outcome != gamecore.OutcomeLose {
			t.Fatalf("expected lower hand to lose, got %+v", result)
		}
	}
}

func TestRaiseResetsOtherReadyAndCallsMatchCurrentBet(t *testing.T) {
	module := NewModule()
	state := State{
		CurrentPlayerIndex: 0,
		Pot:                3,
		CurrentBet:         1,
		Players: []PlayerState{
			{PlayerID: "p1", Hand: []Card{{Month: 1}, {Month: 2}}, Bet: 1, Active: true},
			{PlayerID: "p2", Hand: []Card{{Month: 3}, {Month: 4}}, Bet: 1, Ready: true, Active: true},
			{PlayerID: "p3", Hand: []Card{{Month: 5}, {Month: 6}}, Bet: 1, Ready: true, Active: true},
		},
	}

	result, err := module.ApplyAction(context.Background(), state, gamecore.Action{
		Type:     ActionRaise,
		PlayerID: "p1",
		Payload:  map[string]any{"amount": 2},
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}
	raised := result.State.(State)
	if raised.CurrentBet != 3 || raised.Pot != 5 || raised.Players[0].Bet != 3 || !raised.Players[0].Ready {
		t.Fatalf("expected p1 raise to set bet/pot/current bet, got %+v", raised)
	}
	if raised.Players[1].Ready || raised.Players[2].Ready || raised.CurrentPlayerIndex != 1 {
		t.Fatalf("expected other players to respond to raise, got %+v", raised.Players)
	}

	result, err = module.ApplyAction(context.Background(), raised, gamecore.Action{
		Type:     ActionCall,
		PlayerID: "p2",
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}
	called := result.State.(State)
	if called.Pot != 7 || called.Players[1].Bet != 3 || !called.Players[1].Ready || called.Finished {
		t.Fatalf("expected p2 call to match current bet without finishing yet, got %+v", called)
	}

	result, err = module.ApplyAction(context.Background(), called, gamecore.Action{
		Type:     ActionCall,
		PlayerID: "p3",
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}
	finished := result.State.(State)
	if !finished.Finished || finished.Pot != 9 || finished.Players[2].Bet != 3 {
		t.Fatalf("expected all calls to finish showdown with matched bets, got %+v", finished)
	}
}

func TestRaiseLimitAndAmountValidation(t *testing.T) {
	module := NewModule()
	state := State{
		CurrentPlayerIndex: 0,
		CurrentBet:         1,
		RaisesThisRound:    maxRaisesRound,
		Players: []PlayerState{
			{PlayerID: "p1", Bet: 1, Active: true},
			{PlayerID: "p2", Bet: 1, Active: true},
		},
	}

	err := module.ValidateAction(context.Background(), state, gamecore.Action{
		Type:     ActionRaise,
		PlayerID: "p1",
		Payload:  map[string]any{"amount": 1},
	}, testContext())
	if err == nil {
		t.Fatal("expected raise limit to be enforced")
	}

	state.RaisesThisRound = 0
	err = module.ValidateAction(context.Background(), state, gamecore.Action{
		Type:     ActionRaise,
		PlayerID: "p1",
		Payload:  map[string]any{"amount": maxRaiseAmount + 1},
	}, testContext())
	if err == nil {
		t.Fatal("expected raise amount limit to be enforced")
	}
}

func TestShowdownRequiresMatchedBets(t *testing.T) {
	module := NewModule()
	state := State{
		CurrentPlayerIndex: 0,
		Pot:                5,
		CurrentBet:         3,
		Players: []PlayerState{
			{PlayerID: "p1", Hand: []Card{{Month: 1}, {Month: 2}}, Bet: 3, Ready: true, Active: true},
			{PlayerID: "p2", Hand: []Card{{Month: 3}, {Month: 4}}, Bet: 1, Ready: false, Active: true},
		},
	}

	err := module.ValidateAction(context.Background(), state, gamecore.Action{
		Type:     ActionShowdown,
		PlayerID: "p1",
	}, testContext())
	if err == nil {
		t.Fatal("expected showdown to wait until active bets are matched")
	}

	state.Players[1].Bet = 3
	state.Players[1].Ready = true
	err = module.ValidateAction(context.Background(), state, gamecore.Action{
		Type:     ActionShowdown,
		PlayerID: "p1",
	}, testContext())
	if err != nil {
		t.Fatalf("expected showdown after bets match, got %v", err)
	}
}

func TestTimeoutCurrentPlayerAdvancesTurn(t *testing.T) {
	module := NewModule()
	state := State{
		CurrentPlayerIndex: 0,
		Players: []PlayerState{
			{PlayerID: "p1", Hand: []Card{{Month: 1}, {Month: 2}}, Active: true},
			{PlayerID: "p2", Hand: []Card{{Month: 3}, {Month: 4}}, Active: true},
			{PlayerID: "p3", Hand: []Card{{Month: 5}, {Month: 6}}, Active: true},
		},
	}

	result, err := module.ApplyTimeout(context.Background(), state, "p1", testContext())
	if err != nil {
		t.Fatal(err)
	}

	next := result.State.(State)
	if next.CurrentPlayerIndex != 1 || next.Finished {
		t.Fatalf("expected turn to advance to p2 without finishing, got %+v", next)
	}
}

func testContext() gamecore.Context {
	return gamecore.Context{
		GameID: "sutda",
		Players: []gamecore.Player{
			{ID: "p1", SeatIndex: 0, DisplayName: "P1", Connected: true},
			{ID: "p2", SeatIndex: 1, DisplayName: "P2", Connected: true},
			{ID: "p3", SeatIndex: 2, DisplayName: "P3", Connected: true},
		},
		Now:        time.Date(2026, 7, 21, 1, 0, 0, 0, time.UTC),
		RandomSeed: "seed",
	}
}
