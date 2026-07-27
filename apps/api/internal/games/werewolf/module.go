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
	RoleWerewolf         = "werewolf"
	RoleSeer             = "seer"
	RoleRobber           = "robber"
	RoleTroublemaker     = "troublemaker"
	RoleDrunk            = "drunk"
	RoleInsomniac        = "insomniac"
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
	Phase              string            `json:"phase"`
	CurrentPlayerIndex int               `json:"currentPlayerIndex"`
	Round              int               `json:"round"`
	Players            []PlayerState     `json:"players"`
	Center             []string          `json:"center,omitempty"`
	CompletedActions   map[string]bool   `json:"completedActions"`
	Votes              map[string]string `json:"votes,omitempty"`
	Executed           []string          `json:"executed,omitempty"`
	WinningTeam        string            `json:"winningTeam,omitempty"`
	Log                []string          `json:"log"`
	Finished           bool              `json:"finished"`
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
	case ActionSeeWerewolves:
		if err := requireNightRole(current, player, RoleWerewolf); err != nil {
			return err
		}
		if originalWerewolfCount(current) < 2 {
			return errors.New("lone werewolf may view one center card instead")
		}
	case ActionLoneWolfCenter:
		if err := requireNightRole(current, player, RoleWerewolf); err != nil {
			return err
		}
		if originalWerewolfCount(current) != 1 {
			return errors.New("only lone werewolf may view center")
		}
		if _, err := centerIndexes(action.Payload, 1); err != nil {
			return err
		}
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
	case ActionSeeWerewolves:
		for _, other := range current.Players {
			if other.PlayerID == player.PlayerID || other.OriginalRole != RoleWerewolf {
				continue
			}
			player.SeenRoles["player:"+other.PlayerID] = other.OriginalRole
		}
		current.CompletedActions[player.PlayerID] = true
		current.Log = append(current.Log, player.PlayerID+" 님이 늑대인간 동료를 확인했습니다.")
	case ActionLoneWolfCenter:
		indexes, err := centerIndexes(action.Payload, 1)
		if err != nil {
			return gamecore.ActionResult{}, err
		}
		player.SeenRoles[fmt.Sprintf("center:%d", indexes[0])] = current.Center[indexes[0]]
		current.CompletedActions[player.PlayerID] = true
		current.Log = append(current.Log, player.PlayerID+" 님이 외로운 늑대로 중앙 카드 1장을 확인했습니다.")
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
		current.CompletedActions[player.PlayerID] = true
		current.Log = append(current.Log, player.PlayerID+" 님이 한 플레이어를 확인했습니다.")
	case ActionSeeCenter:
		indexes, err := centerIndexes(action.Payload, 2)
		if err != nil {
			return gamecore.ActionResult{}, err
		}
		for _, index := range indexes {
			player.SeenRoles[fmt.Sprintf("center:%d", index)] = current.Center[index]
		}
		current.CompletedActions[player.PlayerID] = true
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
		current.CompletedActions[player.PlayerID] = true
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
		current.CompletedActions[player.PlayerID] = true
		current.Log = append(current.Log, player.PlayerID+" 님이 두 플레이어의 카드를 바꿨습니다.")
	case ActionDrunkSwap:
		indexes, err := centerIndexes(action.Payload, 1)
		if err != nil {
			return gamecore.ActionResult{}, err
		}
		player.CurrentRole, current.Center[indexes[0]] = current.Center[indexes[0]], player.CurrentRole
		current.CompletedActions[player.PlayerID] = true
		current.Log = append(current.Log, player.PlayerID+" 님이 중앙 카드와 바꿨습니다.")
	case ActionFinishNight:
		for index := range current.Players {
			if current.Players[index].OriginalRole == RoleInsomniac {
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
		if index >= 0 && isRequiredNightRole(current.Players[index].OriginalRole) {
			current.CompletedActions[current.Players[index].PlayerID] = true
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
	for _, player := range current.Players {
		team := teamFor(player.CurrentRole)
		outcome := gamecore.OutcomeLose
		if team == current.WinningTeam {
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
	if player.OriginalRole != role {
		return errors.New("role cannot perform this action")
	}
	if state.CompletedActions[player.PlayerID] {
		return errors.New("night action already completed")
	}
	return nil
}

func requiredNightActionsComplete(state State) bool {
	for _, player := range state.Players {
		if !player.Active || !isRequiredNightRole(player.OriginalRole) {
			continue
		}
		if !state.CompletedActions[player.PlayerID] {
			return false
		}
	}
	return true
}

func isRequiredNightRole(role string) bool {
	switch role {
	case RoleWerewolf, RoleSeer, RoleRobber, RoleTroublemaker, RoleDrunk:
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

	werewolves := currentWerewolves(state)
	werewolfKilled := false
	for _, playerID := range executed {
		if contains(werewolves, playerID) {
			werewolfKilled = true
		}
	}
	if len(werewolves) == 0 {
		state.WinningTeam = "village"
	} else if werewolfKilled {
		state.WinningTeam = "village"
	} else {
		state.WinningTeam = "werewolf"
	}
	state.Executed = executed
	state.Phase = PhaseFinished
	state.Finished = true
	state.Log = append(state.Log, "투표가 종료되었습니다.")
	return state
}

func currentWerewolves(state State) []string {
	werewolves := []string{}
	for _, player := range state.Players {
		if player.CurrentRole == RoleWerewolf {
			werewolves = append(werewolves, player.PlayerID)
		}
	}
	return werewolves
}

func teamFor(role string) string {
	if role == RoleWerewolf {
		return "werewolf"
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
		RoleWerewolf,
		RoleWerewolf,
		RoleSeer,
		RoleRobber,
		RoleTroublemaker,
		RoleDrunk,
		RoleInsomniac,
		RoleVillager,
		RoleVillager,
		RoleVillager,
		RoleVillager,
		RoleVillager,
		RoleVillager,
	}
	if count > len(roles) {
		count = len(roles)
	}
	selected := append([]string(nil), roles[:count]...)
	random := rand.New(rand.NewSource(seedToInt(seed)))
	random.Shuffle(len(selected), func(i, j int) {
		selected[i], selected[j] = selected[j], selected[i]
	})
	return selected
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
