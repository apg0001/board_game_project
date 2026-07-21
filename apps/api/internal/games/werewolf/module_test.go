package werewolf

import (
	"context"
	"testing"
	"time"

	"board-game-platform/apps/api/internal/gamecore"
)

func TestPublicStateHidesOtherRoles(t *testing.T) {
	module := NewModule()
	state := module.CreateInitialState(testContext()).(State)

	public := module.PublicState(state, "p1").(State)
	if public.Players[0].OriginalRole == "" {
		t.Fatal("viewer should see own original role")
	}
	if public.Players[1].OriginalRole != "" {
		t.Fatal("viewer should not see another player role")
	}
	if public.Center[0] != "hidden" {
		t.Fatal("center cards should be hidden")
	}
}

func TestRobberSwapsAndLearnsNewRole(t *testing.T) {
	module := NewModule()
	state := fixedState()

	result, err := module.ApplyAction(context.Background(), state, gamecore.Action{
		Type:     ActionRob,
		PlayerID: "p1",
		Payload:  map[string]any{"targetPlayerId": "p2"},
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}
	next := result.State.(State)
	if next.Players[0].CurrentRole != RoleWerewolf || next.Players[1].CurrentRole != RoleRobber {
		t.Fatalf("expected swapped roles, got %+v", next.Players)
	}
	if next.Players[0].SeenRoles["self:current"] != RoleWerewolf {
		t.Fatal("robber should learn new role")
	}
}

func TestVoteKillsWerewolfAndVillageWins(t *testing.T) {
	module := NewModule()
	state := fixedState()
	state.Phase = PhaseDiscussion

	for _, voter := range []string{"p1", "p2", "p3"} {
		result, err := module.ApplyAction(context.Background(), state, gamecore.Action{
			Type:     ActionVote,
			PlayerID: gamecore.PlayerID(voter),
			Payload:  map[string]any{"targetPlayerId": "p2"},
		}, testContext())
		if err != nil {
			t.Fatal(err)
		}
		state = result.State.(State)
	}

	if !state.Finished || state.WinningTeam != "village" {
		t.Fatalf("expected village win, got %+v", state)
	}
}

func fixedState() State {
	return State{
		Phase: PhaseNight,
		Players: []PlayerState{
			{PlayerID: "p1", OriginalRole: RoleRobber, CurrentRole: RoleRobber, SeenRoles: map[string]string{}, Active: true},
			{PlayerID: "p2", OriginalRole: RoleWerewolf, CurrentRole: RoleWerewolf, SeenRoles: map[string]string{}, Active: true},
			{PlayerID: "p3", OriginalRole: RoleVillager, CurrentRole: RoleVillager, SeenRoles: map[string]string{}, Active: true},
		},
		Center:           []string{RoleSeer, RoleDrunk, RoleVillager},
		CompletedActions: map[string]bool{},
		Votes:            map[string]string{},
	}
}

func testContext() gamecore.Context {
	return gamecore.Context{
		GameID: "werewolf",
		Players: []gamecore.Player{
			{ID: "p1", SeatIndex: 0, DisplayName: "P1", Connected: true},
			{ID: "p2", SeatIndex: 1, DisplayName: "P2", Connected: true},
			{ID: "p3", SeatIndex: 2, DisplayName: "P3", Connected: true},
		},
		Now:        time.Date(2026, 7, 21, 1, 0, 0, 0, time.UTC),
		RandomSeed: "seed",
	}
}
