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

func TestWerewolfMustSeeTeammateBeforeNightCanFinish(t *testing.T) {
	module := NewModule()
	state := State{
		Phase: PhaseNight,
		Players: []PlayerState{
			{PlayerID: "p1", OriginalRole: RoleWerewolf, CurrentRole: RoleWerewolf, SeenRoles: map[string]string{}, Active: true},
			{PlayerID: "p2", OriginalRole: RoleWerewolf, CurrentRole: RoleWerewolf, SeenRoles: map[string]string{}, Active: true},
			{PlayerID: "p3", OriginalRole: RoleVillager, CurrentRole: RoleVillager, SeenRoles: map[string]string{}, Active: true},
		},
		Center:           []string{RoleSeer, RoleDrunk, RoleVillager},
		CompletedActions: map[string]bool{},
		Votes:            map[string]string{},
	}

	err := module.ValidateAction(context.Background(), state, gamecore.Action{
		Type:     ActionFinishNight,
		PlayerID: "p3",
	}, testContext())
	if err == nil {
		t.Fatal("expected night finish to wait for werewolf actions")
	}

	result, err := module.ApplyAction(context.Background(), state, gamecore.Action{
		Type:     ActionSeeWerewolves,
		PlayerID: "p1",
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}
	state = result.State.(State)
	if state.Players[0].SeenRoles["player:p2"] != RoleWerewolf {
		t.Fatalf("expected p1 to see p2 as werewolf, got %+v", state.Players[0].SeenRoles)
	}

	result, err = module.ApplyAction(context.Background(), state, gamecore.Action{
		Type:     ActionSeeWerewolves,
		PlayerID: "p2",
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}
	state = result.State.(State)
	err = module.ValidateAction(context.Background(), state, gamecore.Action{
		Type:     ActionFinishNight,
		PlayerID: "p3",
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}
}

func TestLoneWerewolfMayViewOneCenterCard(t *testing.T) {
	module := NewModule()
	state := fixedState()

	result, err := module.ApplyAction(context.Background(), state, gamecore.Action{
		Type:     ActionLoneWolfCenter,
		PlayerID: "p2",
		Payload:  map[string]any{"centerIndexes": []any{1}},
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}
	next := result.State.(State)
	if next.Players[1].SeenRoles["center:1"] != RoleDrunk || !next.CompletedActions["p2"] {
		t.Fatalf("expected lone wolf to see one center card, got seen=%+v completed=%+v", next.Players[1].SeenRoles, next.CompletedActions)
	}

	err = module.ValidateAction(context.Background(), fixedState(), gamecore.Action{
		Type:     ActionSeeWerewolves,
		PlayerID: "p2",
	}, testContext())
	if err == nil {
		t.Fatal("expected lone wolf teammate check to be rejected")
	}
}

func TestNightRoleOrderBlocksLaterRoles(t *testing.T) {
	module := NewModule()
	state := fixedState()

	err := module.ValidateAction(context.Background(), state, gamecore.Action{
		Type:     ActionRob,
		PlayerID: "p1",
		Payload:  map[string]any{"targetPlayerId": "p3"},
	}, testContext())
	if err == nil {
		t.Fatal("expected robber to wait for lone werewolf action")
	}

	result, err := module.ApplyAction(context.Background(), state, gamecore.Action{
		Type:     ActionLoneWolfCenter,
		PlayerID: "p2",
		Payload:  map[string]any{"centerIndexes": []any{0}},
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}
	state = result.State.(State)

	err = module.ValidateAction(context.Background(), state, gamecore.Action{
		Type:     ActionRob,
		PlayerID: "p1",
		Payload:  map[string]any{"targetPlayerId": "p3"},
	}, testContext())
	if err != nil {
		t.Fatalf("expected robber to act after werewolf completes, got %v", err)
	}
}

func TestMinionSeesWerewolvesAndBlocksSeerUntilComplete(t *testing.T) {
	module := NewModule()
	state := State{
		Phase: PhaseNight,
		Players: []PlayerState{
			{PlayerID: "p1", OriginalRole: RoleWerewolf, CurrentRole: RoleWerewolf, SeenRoles: map[string]string{}, Active: true},
			{PlayerID: "p2", OriginalRole: RoleMinion, CurrentRole: RoleMinion, SeenRoles: map[string]string{}, Active: true},
			{PlayerID: "p3", OriginalRole: RoleSeer, CurrentRole: RoleSeer, SeenRoles: map[string]string{}, Active: true},
		},
		Center:           []string{RoleRobber, RoleDrunk, RoleVillager},
		CompletedActions: map[string]bool{},
		Votes:            map[string]string{},
	}

	result, err := module.ApplyAction(context.Background(), state, gamecore.Action{
		Type:     ActionLoneWolfCenter,
		PlayerID: "p1",
		Payload:  map[string]any{"centerIndexes": []any{0}},
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}
	state = result.State.(State)

	err = module.ValidateAction(context.Background(), state, gamecore.Action{
		Type:     ActionSeeCenter,
		PlayerID: "p3",
		Payload:  map[string]any{"centerIndexes": []any{0, 1}},
	}, testContext())
	if err == nil {
		t.Fatal("expected seer to wait for minion")
	}

	result, err = module.ApplyAction(context.Background(), state, gamecore.Action{
		Type:     ActionSeeMinion,
		PlayerID: "p2",
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}
	state = result.State.(State)
	if state.Players[1].SeenRoles["player:p1"] != RoleWerewolf {
		t.Fatalf("expected minion to see werewolf, got %+v", state.Players[1].SeenRoles)
	}
	err = module.ValidateAction(context.Background(), state, gamecore.Action{
		Type:     ActionSeeCenter,
		PlayerID: "p3",
		Payload:  map[string]any{"centerIndexes": []any{0, 1}},
	}, testContext())
	if err != nil {
		t.Fatalf("expected seer to act after minion, got %v", err)
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
	if !contains(state.WinningPlayerIDs, "p1") || contains(state.WinningPlayerIDs, "p2") {
		t.Fatalf("expected village players to be winners, got %+v", state.WinningPlayerIDs)
	}
}

func TestFinishNightRequiresRequiredRoleActions(t *testing.T) {
	module := NewModule()
	state := fixedState()

	err := module.ValidateAction(context.Background(), state, gamecore.Action{
		Type:     ActionFinishNight,
		PlayerID: "p2",
	}, testContext())
	if err == nil {
		t.Fatal("expected night finish to wait for required role actions")
	}

	state.CompletedActions["p1"] = true
	state.CompletedActions["p2"] = true
	err = module.ValidateAction(context.Background(), state, gamecore.Action{
		Type:     ActionFinishNight,
		PlayerID: "p2",
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}
}

func TestFinishVoteExecutesNobodyWhenEveryoneReceivesOneVote(t *testing.T) {
	state := fixedState()
	state.Phase = PhaseDiscussion
	state.Votes = map[string]string{"p1": "p2", "p2": "p3", "p3": "p1"}

	result := finishVote(state)
	if len(result.Executed) != 0 {
		t.Fatalf("expected no execution when every player receives one vote, got %+v", result.Executed)
	}
	if result.WinningTeam != "werewolf" {
		t.Fatalf("expected werewolf to win when no werewolf is executed, got %q", result.WinningTeam)
	}
}

func TestFinishVoteExecutesAllHighestTiedPlayersAboveOneVote(t *testing.T) {
	state := fixedState()
	state.Players = append(state.Players, PlayerState{PlayerID: "p4", OriginalRole: RoleVillager, CurrentRole: RoleVillager, SeenRoles: map[string]string{}, Active: true})
	state.Phase = PhaseDiscussion
	state.Votes = map[string]string{"p1": "p2", "p2": "p3", "p3": "p2", "p4": "p3"}

	result := finishVote(state)
	if len(result.Executed) != 2 || result.Executed[0] != "p2" || result.Executed[1] != "p3" {
		t.Fatalf("expected p2 and p3 tied at two votes to be executed, got %+v", result.Executed)
	}
}

func TestFinishVoteVillageWinsWhenNoWerewolvesAndNobodyDies(t *testing.T) {
	state := fixedState()
	state.Players[1].CurrentRole = RoleVillager
	state.Phase = PhaseDiscussion
	state.Votes = map[string]string{"p1": "p2", "p2": "p3", "p3": "p1"}

	result := finishVote(state)
	if result.WinningTeam != "village" {
		t.Fatalf("expected village to win when no werewolves exist and nobody dies, got %q", result.WinningTeam)
	}
	if len(result.Executed) != 0 || len(result.WinningPlayerIDs) != 3 {
		t.Fatalf("expected no execution and all villagers to win, got executed=%+v winners=%+v", result.Executed, result.WinningPlayerIDs)
	}
}

func TestFinishVoteHasNoWinnersWhenNoWerewolvesAndVillagerDies(t *testing.T) {
	state := fixedState()
	state.Players[1].CurrentRole = RoleVillager
	state.Phase = PhaseDiscussion
	state.Votes = map[string]string{"p1": "p3", "p2": "p3", "p3": "p1"}

	result := finishVote(state)
	if result.WinningTeam != "none" || len(result.WinningPlayerIDs) != 0 {
		t.Fatalf("expected no winners when villagers execute someone with no werewolves, got team=%q winners=%+v", result.WinningTeam, result.WinningPlayerIDs)
	}
}

func TestTannerWinsWhenExecuted(t *testing.T) {
	state := fixedState()
	state.Players[2].CurrentRole = RoleTanner
	state.Phase = PhaseDiscussion
	state.Votes = map[string]string{"p1": "p3", "p2": "p3", "p3": "p2"}

	result := finishVote(state)
	if result.WinningTeam != "tanner" || !contains(result.WinningPlayerIDs, "p3") || len(result.WinningPlayerIDs) != 1 {
		t.Fatalf("expected executed tanner alone to win, got team=%q winners=%+v", result.WinningTeam, result.WinningPlayerIDs)
	}
}

func TestHunterExecutionAlsoExecutesVotedTarget(t *testing.T) {
	state := fixedState()
	state.Players[0].CurrentRole = RoleHunter
	state.Phase = PhaseDiscussion
	state.Votes = map[string]string{"p1": "p2", "p2": "p1", "p3": "p1"}

	result := finishVote(state)
	if !contains(result.Executed, "p1") || !contains(result.Executed, "p2") {
		t.Fatalf("expected hunter and hunter target to be executed, got %+v", result.Executed)
	}
	if result.WinningTeam != "village" || !contains(result.WinningPlayerIDs, "p1") || contains(result.WinningPlayerIDs, "p2") {
		t.Fatalf("expected village to win after hunter takes werewolf down, got team=%q winners=%+v", result.WinningTeam, result.WinningPlayerIDs)
	}
}

func TestMinionWinsWithWerewolvesEvenIfExecuted(t *testing.T) {
	state := fixedState()
	state.Players[0].CurrentRole = RoleMinion
	state.Phase = PhaseDiscussion
	state.Votes = map[string]string{"p1": "p1", "p2": "p1", "p3": "p1"}

	result := finishVote(state)
	if result.WinningTeam != "werewolf" || !contains(result.WinningPlayerIDs, "p1") || !contains(result.WinningPlayerIDs, "p2") {
		t.Fatalf("expected minion and surviving werewolf to win, got team=%q winners=%+v", result.WinningTeam, result.WinningPlayerIDs)
	}
}

func TestMinionWinsWithoutWerewolvesIfVillagerDiesAndMinionSurvives(t *testing.T) {
	state := fixedState()
	state.Players[0].CurrentRole = RoleMinion
	state.Players[1].CurrentRole = RoleVillager
	state.Phase = PhaseDiscussion
	state.Votes = map[string]string{"p1": "p3", "p2": "p3", "p3": "p2"}

	result := finishVote(state)
	if result.WinningTeam != "werewolf" || len(result.WinningPlayerIDs) != 1 || result.WinningPlayerIDs[0] != "p1" {
		t.Fatalf("expected lone minion to win when a villager dies, got team=%q winners=%+v", result.WinningTeam, result.WinningPlayerIDs)
	}
}

func TestPublicStateHidesVotesBeforeAllSubmitted(t *testing.T) {
	module := NewModule()
	state := fixedState()
	state.Phase = PhaseDiscussion
	state.Votes = map[string]string{"p1": "p2"}
	state.Players[0].VotedFor = "p2"

	public := module.PublicState(state, "p3").(State)
	if len(public.Votes) != 0 {
		t.Fatalf("bystander should not see votes before all submitted, got %+v", public.Votes)
	}
	if public.Players[0].VotedFor != "" {
		t.Fatal("bystander should not see another player's vote target")
	}

	own := module.PublicState(state, "p1").(State)
	if own.Votes["p1"] != "p2" {
		t.Fatal("voter should see their own submitted vote")
	}
}

func TestApplyTimeoutOnlySkipsTimedOutRoleDuringNight(t *testing.T) {
	module := NewModule()
	state := fixedState()

	result, err := module.ApplyTimeout(context.Background(), state, "p2", testContext())
	if err != nil {
		t.Fatal(err)
	}
	next := result.State.(State)
	if next.Phase != PhaseNight {
		t.Fatal("night should stay open while the robber has not acted yet")
	}

	result, err = module.ApplyTimeout(context.Background(), next, "p1", testContext())
	if err != nil {
		t.Fatal(err)
	}
	next = result.State.(State)
	if next.Phase != PhaseDiscussion {
		t.Fatal("night should end once the last required role times out")
	}
}

func TestCenterIndexesAcceptIntegerPayload(t *testing.T) {
	indexes, err := centerIndexes(map[string]any{"centerIndexes": []any{0, 1}}, 2)
	if err != nil {
		t.Fatal(err)
	}
	if indexes[0] != 0 || indexes[1] != 1 {
		t.Fatalf("expected integer center indexes, got %+v", indexes)
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
