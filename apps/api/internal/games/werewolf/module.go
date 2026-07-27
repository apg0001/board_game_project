package werewolf

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"math/rand"
	"sort"

	"board-game-platform/apps/api/internal/gamecore"
	"board-game-platform/apps/api/internal/games/internal/gameutil"
)

const (
	ActionSeeWerewolves  = "werewolf.see_werewolves"
	ActionLoneWolfCenter = "werewolf.lone_wolf_center"
	ActionDoppelganger   = "werewolf.doppelganger"
	ActionSeeMinion      = "werewolf.see_minion"
	ActionSeeMasons      = "werewolf.see_masons"
	ActionSeePlayer      = "werewolf.see_player"
	ActionSeeCenter      = "werewolf.see_center"
	ActionRob            = "werewolf.rob"
	ActionTroublemake    = "werewolf.troublemake"
	ActionDrunkSwap      = "werewolf.drunk_swap"
	ActionFinishNight    = "werewolf.finish_night"
	ActionVote           = "werewolf.vote"
	PhaseNight           = "NIGHT"
	PhaseDiscussion      = "DISCUSSION"
	PhaseFinished        = "FINISHED"
	RoleDoppelganger     = "doppelganger"
	RoleWerewolf         = "werewolf"
	RoleMinion           = "minion"
	RoleMason            = "mason"
	RoleSeer             = "seer"
	RoleRobber           = "robber"
	RoleTroublemaker     = "troublemaker"
	RoleDrunk            = "drunk"
	RoleInsomniac        = "insomniac"
	RoleHunter           = "hunter"
	RoleTanner           = "tanner"
	RoleVillager         = "villager"
)

type PlayerState struct {
	PlayerID     string            `json:"playerId"`
	OriginalRole string            `json:"originalRole,omitempty"`
	CurrentRole  string            `json:"currentRole,omitempty"`
	SeenRoles    map[string]string `json:"seenRoles,omitempty"`
	VotedFor     string            `json:"votedFor,omitempty"`
	Active       bool              `json:"active"`
}

type State struct {
	Phase                  string            `json:"phase"`
	CurrentPlayerIndex     int               `json:"currentPlayerIndex"`
	Round                  int               `json:"round"`
	Players                []PlayerState     `json:"players"`
	Center                 []string          `json:"center,omitempty"`
	CompletedActions       map[string]bool   `json:"completedActions"`
	Votes                  map[string]string `json:"votes,omitempty"`
	DoppelgangerPlayerID   string            `json:"doppelgangerPlayerId,omitempty"`
	DoppelgangerCopiedRole string            `json:"doppelgangerCopiedRole,omitempty"`
	Executed               []string          `json:"executed,omitempty"`
	WinningTeam            string            `json:"winningTeam,omitempty"`
	WinningPlayerIDs       []string          `json:"winningPlayerIds,omitempty"`
	Log                    []string          `json:"log"`
	Finished               bool              `json:"finished"`
}

type Module struct{}

func NewModule() Module {
	return Module{}
}

func (m Module) ID() gamecore.GameID {
	return "werewolf"
}

func (m Module) Name() string {
	return "한 밤의 늑대인간"
}

func (m Module) MinPlayers() int {
	return 3
}

func (m Module) MaxPlayers() int {
	return 10
}

func (m Module) CreateInitialState(ctx gamecore.Context) any {
	roles := shuffledRoles(ctx.RandomSeed, len(ctx.Players)+3)
	players := make([]PlayerState, 0, len(ctx.Players))
	for index, player := range ctx.Players {
		players = append(players, PlayerState{
			PlayerID:     string(player.ID),
			OriginalRole: roles[index],
			CurrentRole:  roles[index],
			SeenRoles:    map[string]string{},
			Active:       true,
		})
	}
	return State{
		Phase:            PhaseNight,
		Players:          players,
		Center:           append([]string(nil), roles[len(ctx.Players):]...),
		CompletedActions: map[string]bool{},
		Votes:            map[string]string{},
		Log:              []string{"한 밤의 늑대인간 밤 단계가 시작되었습니다."},
	}
}

func (m Module) PublicState(state any, viewerID gamecore.PlayerID) any {
	current := asState(state)
	current.Center = append([]string(nil), current.Center...)
	current.Players = append([]PlayerState(nil), current.Players...)
	for index := range current.Players {
		current.Players[index].SeenRoles = copySeenRoles(current.Players[index].SeenRoles)
	}
	revealed := current.Finished
	for index := range current.Players {
		player := &current.Players[index]
		if revealed || player.PlayerID == string(viewerID) {
			continue
		}
		player.OriginalRole = ""
		player.CurrentRole = ""
		player.SeenRoles = nil
	}
	if !revealed {
		if current.DoppelgangerPlayerID != string(viewerID) {
			current.DoppelgangerPlayerID = ""
			current.DoppelgangerCopiedRole = ""
		}
		current.Center = []string{"hidden", "hidden", "hidden"}
	}
	if !revealed {
		votes := map[string]string{}
		if viewerVote, ok := current.Votes[string(viewerID)]; ok {
			votes[string(viewerID)] = viewerVote
		}
		current.Votes = votes
		for index := range current.Players {
			if current.Players[index].PlayerID != string(viewerID) {
				current.Players[index].VotedFor = ""
			}
		}
	}
	return current
}

func (m Module) ValidateAction(_ context.Context, state any, action gamecore.Action, _ gamecore.Context) error {
	current := asState(state)
	if current.Finished {
		return errors.New("game is already finished")
	}
	playerIndex := findPlayer(current, string(action.PlayerID))
	if playerIndex < 0 {
		return errors.New("player not found")
	}
	player := current.Players[playerIndex]
	switch action.Type {
	case ActionDoppelganger:
		if err := requireDoppelgangerAction(current, player); err != nil {
			return err
		}
		target, err := targetPayload(action.Payload)
		if err != nil {
			return err
		}
		if target == player.PlayerID || findPlayer(current, target) < 0 {
			return errors.New("invalid doppelganger target")
		}
	case ActionSeeWerewolves:
		if err := requireNightRole(current, player, RoleWerewolf); err != nil {
			return err
		}
		if nightWerewolfCount(current) < 2 {
			return errors.New("lone werewolf may view one center card instead")
		}
	case ActionLoneWolfCenter:
		if err := requireNightRole(current, player, RoleWerewolf); err != nil {
			return err
		}
		if nightWerewolfCount(current) != 1 {
			return errors.New("only lone werewolf may view center")
		}
		if _, err := centerIndexes(action.Payload, 1); err != nil {
			return err
		}
	case ActionSeeMinion:
		return requireNightRole(current, player, RoleMinion)
	case ActionSeeMasons:
		return requireNightRole(current, player, RoleMason)
	case ActionSeePlayer:
		return requireNightRole(current, player, RoleSeer)
	case ActionSeeCenter:
		return requireNightRole(current, player, RoleSeer)
	case ActionRob:
		return requireNightRole(current, player, RoleRobber)
	case ActionTroublemake:
		return requireNightRole(current, player, RoleTroublemaker)
	case ActionDrunkSwap:
		return requireNightRole(current, player, RoleDrunk)
	case ActionFinishNight:
		if current.Phase != PhaseNight {
			return errors.New("night is already finished")
		}
		if !requiredNightActionsComplete(current) {
			return errors.New("night actions are not completed")
		}
	case ActionVote:
		if current.Phase != PhaseDiscussion {
			return errors.New("voting is not open")
		}
		target, err := targetPayload(action.Payload)
		if err != nil {
			return err
		}
		if findPlayer(current, target) < 0 {
			return errors.New("vote target not found")
		}
	default:
		return errors.New("unsupported action")
	}
	return nil
}

func (m Module) ApplyAction(_ context.Context, state any, action gamecore.Action, _ gamecore.Context) (gamecore.ActionResult, error) {
	current := asState(state)
	playerIndex := findPlayer(current, string(action.PlayerID))
	player := &current.Players[playerIndex]

	switch action.Type {
	case ActionDoppelganger:
		target, err := targetPayload(action.Payload)
		if err != nil {
			return gamecore.ActionResult{}, err
		}
		targetIndex := findPlayer(current, target)
		if targetIndex < 0 || target == player.PlayerID {
			return gamecore.ActionResult{}, errors.New("invalid doppelganger target")
		}
		copiedRole := current.Players[targetIndex].CurrentRole
		current.DoppelgangerPlayerID = player.PlayerID
		current.DoppelgangerCopiedRole = copiedRole
		player.SeenRoles["player:"+target] = copiedRole
		player.SeenRoles["self:doppelgangerRole"] = copiedRole
		current.CompletedActions[player.PlayerID] = true
		current.Log = append(current.Log, fmt.Sprintf("%s 님이 도플갱어로 역할을 복사했습니다.", player.PlayerID))
	case ActionSeeWerewolves:
		for _, otherID := range nightWerewolfIDs(current) {
			if otherID == player.PlayerID {
				continue
			}
			player.SeenRoles["player:"+otherID] = RoleWerewolf
		}
		markNightActionComplete(&current, *player, RoleWerewolf)
		current.Log = append(current.Log, player.PlayerID+" 님이 늑대인간 동료를 확인했습니다.")
	case ActionLoneWolfCenter:
		indexes, err := centerIndexes(action.Payload, 1)
		if err != nil {
			return gamecore.ActionResult{}, err
		}
		player.SeenRoles[fmt.Sprintf("center:%d", indexes[0])] = current.Center[indexes[0]]
		markNightActionComplete(&current, *player, RoleWerewolf)
		current.Log = append(current.Log, player.PlayerID+" 님이 외로운 늑대로 중앙 카드 1장을 확인했습니다.")
	case ActionSeeMinion:
		for _, otherID := range nightWerewolfIDs(current) {
			if otherID == player.PlayerID {
				continue
			}
			player.SeenRoles["player:"+otherID] = RoleWerewolf
		}
		markNightActionComplete(&current, *player, RoleMinion)
		current.Log = append(current.Log, player.PlayerID+" 님이 앞잡이로 늑대인간을 확인했습니다.")
	case ActionSeeMasons:
		for _, otherID := range nightMasonIDs(current) {
			if otherID == player.PlayerID {
				continue
			}
			player.SeenRoles["player:"+otherID] = RoleMason
		}
		markNightActionComplete(&current, *player, RoleMason)
		current.Log = append(current.Log, player.PlayerID+" 님이 석공 동료를 확인했습니다.")
	case ActionSeePlayer:
		target, err := targetPayload(action.Payload)
		if err != nil {
			return gamecore.ActionResult{}, err
		}
		targetIndex := findPlayer(current, target)
		if targetIndex < 0 {
			return gamecore.ActionResult{}, errors.New("target not found")
		}
		player.SeenRoles["player:"+target] = current.Players[targetIndex].CurrentRole
		markNightActionComplete(&current, *player, RoleSeer)
		current.Log = append(current.Log, player.PlayerID+" 님이 한 플레이어를 확인했습니다.")
	case ActionSeeCenter:
		indexes, err := centerIndexes(action.Payload, 2)
		if err != nil {
			return gamecore.ActionResult{}, err
		}
		for _, index := range indexes {
			player.SeenRoles[fmt.Sprintf("center:%d", index)] = current.Center[index]
		}
		markNightActionComplete(&current, *player, RoleSeer)
		current.Log = append(current.Log, player.PlayerID+" 님이 중앙 카드 2장을 확인했습니다.")
	case ActionRob:
		target, err := targetPayload(action.Payload)
		if err != nil {
			return gamecore.ActionResult{}, err
		}
		targetIndex := findPlayer(current, target)
		if targetIndex < 0 || target == player.PlayerID {
			return gamecore.ActionResult{}, errors.New("invalid robber target")
		}
		player.CurrentRole, current.Players[targetIndex].CurrentRole = current.Players[targetIndex].CurrentRole, player.CurrentRole
		player.SeenRoles["self:current"] = player.CurrentRole
		markNightActionComplete(&current, *player, RoleRobber)
		current.Log = append(current.Log, player.PlayerID+" 님이 역할을 강탈했습니다.")
	case ActionTroublemake:
		left, right, err := twoTargets(action.Payload)
		if err != nil {
			return gamecore.ActionResult{}, err
		}
		leftIndex := findPlayer(current, left)
		rightIndex := findPlayer(current, right)
		if leftIndex < 0 || rightIndex < 0 || left == right || left == player.PlayerID || right == player.PlayerID {
			return gamecore.ActionResult{}, errors.New("invalid troublemaker targets")
		}
		current.Players[leftIndex].CurrentRole, current.Players[rightIndex].CurrentRole = current.Players[rightIndex].CurrentRole, current.Players[leftIndex].CurrentRole
		markNightActionComplete(&current, *player, RoleTroublemaker)
		current.Log = append(current.Log, player.PlayerID+" 님이 두 플레이어의 카드를 바꿨습니다.")
	case ActionDrunkSwap:
		indexes, err := centerIndexes(action.Payload, 1)
		if err != nil {
			return gamecore.ActionResult{}, err
		}
		player.CurrentRole, current.Center[indexes[0]] = current.Center[indexes[0]], player.CurrentRole
		markNightActionComplete(&current, *player, RoleDrunk)
		current.Log = append(current.Log, player.PlayerID+" 님이 중앙 카드와 바꿨습니다.")
	case ActionFinishNight:
		for index := range current.Players {
			if current.Players[index].OriginalRole == RoleInsomniac || isDoppelgangerActingAs(current, current.Players[index], RoleInsomniac) {
				current.Players[index].SeenRoles["self:current"] = current.Players[index].CurrentRole
			}
		}
		current.Phase = PhaseDiscussion
		current.Log = append(current.Log, "낮 토론과 투표 단계가 시작되었습니다.")
	case ActionVote:
		target, err := targetPayload(action.Payload)
		if err != nil {
			return gamecore.ActionResult{}, err
		}
		player.VotedFor = target
		current.Votes[player.PlayerID] = target
		current.Log = append(current.Log, player.PlayerID+" 님이 투표했습니다.")
		if len(current.Votes) >= len(current.Players) {
			current = finishVote(current)
		}
	}
	return gamecore.ActionResult{
		State: current,
		Events: []gamecore.Event{{
			Type:       "game.state_updated",
			Visibility: gamecore.VisibilityPublic,
			Payload:    m.PublicState(current, ""),
		}},
	}, nil
}

func (m Module) ApplyTimeout(_ context.Context, state any, playerID gamecore.PlayerID, _ gamecore.Context) (gamecore.ActionResult, error) {
	current := asState(state)
	if current.Phase == PhaseNight {
		index := findPlayer(current, string(playerID))
		if index >= 0 && skipPendingNightAction(&current, current.Players[index]) {
			current.Log = append(current.Log, current.Players[index].PlayerID+" 님의 밤 행동이 시간 초과로 건너뛰어졌습니다.")
		}
		if requiredNightActionsComplete(current) {
			current.Phase = PhaseDiscussion
			current.Log = append(current.Log, "낮 토론과 투표 단계가 시작되었습니다.")
		}
		return gamecore.ActionResult{State: current}, nil
	}
	if current.Phase == PhaseDiscussion {
		index := findPlayer(current, string(playerID))
		if index >= 0 {
			current.Players[index].VotedFor = current.Players[index].PlayerID
			current.Votes[current.Players[index].PlayerID] = current.Players[index].PlayerID
		}
		if len(current.Votes) >= len(current.Players) {
			current = finishVote(current)
		}
	}
	return gamecore.ActionResult{State: current}, nil
}

func (m Module) IsFinished(state any, _ gamecore.Context) bool {
	return asState(state).Finished
}

func (m Module) CalculateResult(state any, _ gamecore.Context) []gamecore.Result {
	current := asState(state)
	results := make([]gamecore.Result, 0, len(current.Players))
	winners := map[string]bool{}
	for _, playerID := range current.WinningPlayerIDs {
		winners[playerID] = true
	}
	for _, player := range current.Players {
		outcome := gamecore.OutcomeLose
		if winners[player.PlayerID] {
			outcome = gamecore.OutcomeWin
		}
		results = append(results, gamecore.Result{
			PlayerID: gamecore.PlayerID(player.PlayerID),
			Rank:     rankForOutcome(outcome),
			Score:    scoreForOutcome(outcome),
			Outcome:  outcome,
		})
	}
	return results
}

func requireNightRole(state State, player PlayerState, role string) error {
	if state.Phase != PhaseNight {
		return errors.New("night action is closed")
	}
	if !canPerformNightRole(state, player, role) {
		return errors.New("role cannot perform this action")
	}
	if nightActionCompleted(state, player, role) {
		return errors.New("night action already completed")
	}
	if err := requireNightOrderForAction(state, player, role); err != nil {
		return err
	}
	return nil
}

func requireDoppelgangerAction(state State, player PlayerState) error {
	if state.Phase != PhaseNight {
		return errors.New("night action is closed")
	}
	if player.OriginalRole != RoleDoppelganger {
		return errors.New("role cannot perform this action")
	}
	if state.CompletedActions[player.PlayerID] {
		return errors.New("night action already completed")
	}
	return requireNightOrder(state, RoleDoppelganger)
}

func canPerformNightRole(state State, player PlayerState, role string) bool {
	return player.OriginalRole == role || isDoppelgangerActingAs(state, player, role)
}

func isDoppelgangerActingAs(state State, player PlayerState, role string) bool {
	return player.OriginalRole == RoleDoppelganger &&
		state.DoppelgangerPlayerID == player.PlayerID &&
		state.DoppelgangerCopiedRole == role &&
		doppelgangerCanActAs(role)
}

func nightActionCompleted(state State, player PlayerState, role string) bool {
	return state.CompletedActions[nightActionKey(state, player, role)]
}

func markNightActionComplete(state *State, player PlayerState, role string) {
	state.CompletedActions[nightActionKey(*state, player, role)] = true
}

func nightActionKey(state State, player PlayerState, role string) string {
	if isDoppelgangerActingAs(state, player, role) {
		return doppelgangerActionKey(player.PlayerID, role)
	}
	return player.PlayerID
}

func doppelgangerActionKey(playerID string, role string) string {
	return playerID + ":doppelganger:" + role
}

func requireNightOrderForAction(state State, player PlayerState, role string) error {
	if isDoppelgangerActingAs(state, player, role) && doppelgangerImmediateAction(role) {
		return nil
	}
	return requireNightOrder(state, role)
}

func requireNightOrder(state State, role string) error {
	roleIndex := nightRoleIndex(role)
	if roleIndex < 0 {
		return nil
	}
	for _, earlierRole := range nightRoleOrder()[:roleIndex] {
		if !nightRoleComplete(state, earlierRole) {
			return errors.New("previous night role has not completed")
		}
	}
	return nil
}

func nightRoleComplete(state State, role string) bool {
	if role == RoleDoppelganger {
		for _, player := range state.Players {
			if !player.Active || player.OriginalRole != RoleDoppelganger {
				continue
			}
			if !state.CompletedActions[player.PlayerID] {
				return false
			}
			if state.DoppelgangerPlayerID == player.PlayerID &&
				doppelgangerImmediateAction(state.DoppelgangerCopiedRole) &&
				!state.CompletedActions[doppelgangerActionKey(player.PlayerID, state.DoppelgangerCopiedRole)] {
				return false
			}
		}
		return true
	}
	for _, player := range state.Players {
		if !player.Active {
			continue
		}
		if player.OriginalRole == role {
			if !state.CompletedActions[player.PlayerID] {
				return false
			}
			continue
		}
		if isDoppelgangerActingAs(state, player, role) && doppelgangerNormalOrderAction(role) {
			if !state.CompletedActions[doppelgangerActionKey(player.PlayerID, role)] {
				return false
			}
		}
	}
	return true
}

func nightRoleIndex(role string) int {
	for index, item := range nightRoleOrder() {
		if item == role {
			return index
		}
	}
	return -1
}

func nightRoleOrder() []string {
	return []string{RoleDoppelganger, RoleWerewolf, RoleMinion, RoleMason, RoleSeer, RoleRobber, RoleTroublemaker, RoleDrunk}
}

func requiredNightActionsComplete(state State) bool {
	for _, role := range nightRoleOrder() {
		if !nightRoleComplete(state, role) {
			return false
		}
	}
	return true
}

func isRequiredNightRole(role string) bool {
	switch role {
	case RoleDoppelganger, RoleWerewolf, RoleMinion, RoleMason, RoleSeer, RoleRobber, RoleTroublemaker, RoleDrunk:
		return true
	default:
		return false
	}
}

func doppelgangerCanActAs(role string) bool {
	switch role {
	case RoleWerewolf, RoleMinion, RoleMason, RoleSeer, RoleRobber, RoleTroublemaker, RoleDrunk, RoleInsomniac:
		return true
	default:
		return false
	}
}

func doppelgangerImmediateAction(role string) bool {
	switch role {
	case RoleSeer, RoleRobber, RoleTroublemaker, RoleDrunk:
		return true
	default:
		return false
	}
}

func doppelgangerNormalOrderAction(role string) bool {
	switch role {
	case RoleWerewolf, RoleMinion, RoleMason:
		return true
	default:
		return false
	}
}

func originalWerewolfCount(state State) int {
	count := 0
	for _, player := range state.Players {
		if player.OriginalRole == RoleWerewolf {
			count++
		}
	}
	return count
}

func nightWerewolfCount(state State) int {
	return len(nightWerewolfIDs(state))
}

func nightWerewolfIDs(state State) []string {
	ids := []string{}
	for _, player := range state.Players {
		if !player.Active {
			continue
		}
		if player.OriginalRole == RoleWerewolf || isDoppelgangerActingAs(state, player, RoleWerewolf) {
			ids = append(ids, player.PlayerID)
		}
	}
	sort.Strings(ids)
	return ids
}

func nightMasonIDs(state State) []string {
	ids := []string{}
	for _, player := range state.Players {
		if !player.Active {
			continue
		}
		if player.OriginalRole == RoleMason || isDoppelgangerActingAs(state, player, RoleMason) {
			ids = append(ids, player.PlayerID)
		}
	}
	sort.Strings(ids)
	return ids
}

func skipPendingNightAction(state *State, player PlayerState) bool {
	if state.Phase != PhaseNight || !player.Active {
		return false
	}
	if player.OriginalRole == RoleDoppelganger && !state.CompletedActions[player.PlayerID] {
		state.CompletedActions[player.PlayerID] = true
		return true
	}
	if player.OriginalRole == RoleDoppelganger &&
		state.DoppelgangerPlayerID == player.PlayerID &&
		doppelgangerCanActAs(state.DoppelgangerCopiedRole) &&
		!state.CompletedActions[doppelgangerActionKey(player.PlayerID, state.DoppelgangerCopiedRole)] {
		state.CompletedActions[doppelgangerActionKey(player.PlayerID, state.DoppelgangerCopiedRole)] = true
		return true
	}
	if isRequiredNightRole(player.OriginalRole) && !state.CompletedActions[player.PlayerID] {
		state.CompletedActions[player.PlayerID] = true
		return true
	}
	return false
}

func finishVote(state State) State {
	counts := map[string]int{}
	for _, target := range state.Votes {
		counts[target]++
	}
	highest := 0
	for _, count := range counts {
		if count > highest {
			highest = count
		}
	}
	executed := []string{}
	if highest == 1 && len(counts) == len(state.Players) {
		executed = []string{}
	} else if highest > 0 {
		for playerID, count := range counts {
			if count == highest {
				executed = append(executed, playerID)
			}
		}
		sort.Strings(executed)
	}
	executed = applyHunterExecution(state, executed)

	werewolves := currentWerewolves(state)
	werewolfKilled := false
	for _, playerID := range executed {
		if contains(werewolves, playerID) {
			werewolfKilled = true
		}
	}
	state.WinningPlayerIDs = winningPlayers(state, executed, werewolfKilled, len(werewolves) > 0)
	if tannerExecuted(state, executed) && werewolfKilled {
		state.WinningTeam = "mixed"
	} else if tannerExecuted(state, executed) {
		state.WinningTeam = "tanner"
	} else if len(werewolves) == 0 && len(executed) == 0 {
		state.WinningTeam = "village"
	} else if len(werewolves) > 0 && werewolfKilled {
		state.WinningTeam = "village"
	} else if len(state.WinningPlayerIDs) == 0 {
		state.WinningTeam = "none"
	} else {
		state.WinningTeam = "werewolf"
	}
	state.Executed = executed
	state.Phase = PhaseFinished
	state.Finished = true
	state.Log = append(state.Log, "투표가 종료되었습니다.")
	return state
}

func applyHunterExecution(state State, executed []string) []string {
	next := append([]string(nil), executed...)
	for _, playerID := range executed {
		playerIndex := findPlayer(state, playerID)
		if playerIndex < 0 || effectiveRole(state, state.Players[playerIndex]) != RoleHunter {
			continue
		}
		target := state.Votes[playerID]
		if target != "" && !contains(next, target) {
			next = append(next, target)
		}
	}
	sort.Strings(next)
	return next
}

func winningPlayers(state State, executed []string, werewolfKilled bool, hasWerewolf bool) []string {
	winners := []string{}
	if tannerExecuted(state, executed) {
		winners = appendRoleWinners(state, winners, RoleTanner)
		if werewolfKilled {
			winners = appendVillageWinners(state, winners)
		}
		sort.Strings(winners)
		return uniqueStrings(winners)
	}
	if hasWerewolf {
		if werewolfKilled {
			winners = appendVillageWinners(state, winners)
		} else {
			winners = appendWerewolfTeamWinners(state, winners)
		}
	} else if len(executed) == 0 {
		winners = appendVillageWinners(state, winners)
	} else {
		winners = appendNoWerewolfMinionWinners(state, winners, executed)
	}
	sort.Strings(winners)
	return uniqueStrings(winners)
}

func appendVillageWinners(state State, winners []string) []string {
	for _, player := range state.Players {
		if teamFor(effectiveRole(state, player)) == "village" {
			winners = append(winners, player.PlayerID)
		}
	}
	return winners
}

func appendWerewolfTeamWinners(state State, winners []string) []string {
	for _, player := range state.Players {
		if teamFor(effectiveRole(state, player)) == "werewolf" {
			winners = append(winners, player.PlayerID)
		}
	}
	return winners
}

func appendRoleWinners(state State, winners []string, role string) []string {
	for _, player := range state.Players {
		if effectiveRole(state, player) == role {
			winners = append(winners, player.PlayerID)
		}
	}
	return winners
}

func appendNoWerewolfMinionWinners(state State, winners []string, executed []string) []string {
	for _, player := range state.Players {
		if player.CurrentRole == RoleMinion && !contains(executed, player.PlayerID) && nonTannerNonWerewolfTeamExecuted(state, executed) {
			winners = append(winners, player.PlayerID)
		}
	}
	return winners
}

func tannerExecuted(state State, executed []string) bool {
	for _, playerID := range executed {
		playerIndex := findPlayer(state, playerID)
		if playerIndex >= 0 && effectiveRole(state, state.Players[playerIndex]) == RoleTanner {
			return true
		}
	}
	return false
}

func nonTannerNonWerewolfTeamExecuted(state State, executed []string) bool {
	for _, playerID := range executed {
		playerIndex := findPlayer(state, playerID)
		if playerIndex < 0 {
			continue
		}
		role := effectiveRole(state, state.Players[playerIndex])
		if role != RoleTanner && teamFor(role) != "werewolf" {
			return true
		}
	}
	return false
}

func uniqueStrings(values []string) []string {
	seen := map[string]bool{}
	unique := []string{}
	for _, value := range values {
		if seen[value] {
			continue
		}
		seen[value] = true
		unique = append(unique, value)
	}
	return unique
}

func currentWerewolves(state State) []string {
	werewolves := []string{}
	for _, player := range state.Players {
		if effectiveRole(state, player) == RoleWerewolf {
			werewolves = append(werewolves, player.PlayerID)
		}
	}
	return werewolves
}

func effectiveRole(state State, player PlayerState) string {
	if player.CurrentRole == RoleDoppelganger && state.DoppelgangerCopiedRole != "" {
		return state.DoppelgangerCopiedRole
	}
	return player.CurrentRole
}

func teamFor(role string) string {
	if role == RoleWerewolf || role == RoleMinion {
		return "werewolf"
	}
	if role == RoleTanner {
		return "tanner"
	}
	return "village"
}

func rankForOutcome(outcome gamecore.Outcome) int {
	if outcome == gamecore.OutcomeWin {
		return 1
	}
	return 2
}

func scoreForOutcome(outcome gamecore.Outcome) int {
	if outcome == gamecore.OutcomeWin {
		return 1
	}
	return 0
}

func targetPayload(payload any) (string, error) {
	raw, ok := payload.(map[string]any)
	if !ok {
		return "", errors.New("invalid target payload")
	}
	target, _ := raw["targetPlayerId"].(string)
	if target == "" {
		return "", errors.New("target player is required")
	}
	return target, nil
}

func twoTargets(payload any) (string, string, error) {
	raw, ok := payload.(map[string]any)
	if !ok {
		return "", "", errors.New("invalid target payload")
	}
	left, _ := raw["leftPlayerId"].(string)
	right, _ := raw["rightPlayerId"].(string)
	if left == "" || right == "" {
		return "", "", errors.New("two targets are required")
	}
	return left, right, nil
}

func centerIndexes(payload any, expected int) ([]int, error) {
	raw, ok := payload.(map[string]any)
	if !ok {
		return nil, errors.New("invalid center payload")
	}
	values, ok := raw["centerIndexes"].([]any)
	if !ok || len(values) != expected {
		return nil, errors.New("center indexes are required")
	}
	indexes := make([]int, 0, expected)
	seen := map[int]bool{}
	for _, value := range values {
		index, ok := gameutil.Int(value)
		if !ok {
			return nil, errors.New("invalid center index")
		}
		if index < 0 || index > 2 || seen[index] {
			return nil, errors.New("center index out of range")
		}
		seen[index] = true
		indexes = append(indexes, index)
	}
	return indexes, nil
}

func shuffledRoles(seed string, count int) []string {
	roles := []string{
		RoleDoppelganger,
		RoleWerewolf,
		RoleWerewolf,
		RoleMinion,
		RoleMason,
		RoleMason,
		RoleSeer,
		RoleRobber,
		RoleTroublemaker,
		RoleDrunk,
		RoleInsomniac,
		RoleHunter,
		RoleTanner,
		RoleVillager,
		RoleVillager,
		RoleVillager,
	}
	if count > len(roles) {
		count = len(roles)
	}
	selected := append([]string(nil), roles...)
	random := rand.New(rand.NewSource(seedToInt(seed)))
	random.Shuffle(len(selected), func(i, j int) {
		selected[i], selected[j] = selected[j], selected[i]
	})
	return selected[:count]
}

func findPlayer(state State, playerID string) int {
	for index, player := range state.Players {
		if player.PlayerID == playerID {
			return index
		}
	}
	return -1
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func copySeenRoles(source map[string]string) map[string]string {
	if source == nil {
		return nil
	}
	result := make(map[string]string, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}

func seedToInt(seed string) int64 {
	hash := fnv.New64a()
	_, _ = hash.Write([]byte(seed))
	return int64(hash.Sum64())
}

func asState(state any) State {
	typed, ok := state.(State)
	if ok {
		return typed
	}
	return State{}
}
