package rummikub

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"math/rand"
	"sort"

	"board-game-platform/apps/api/internal/gamecore"
)

const (
	ActionMeld      = "rummikub.meld"
	ActionDraw      = "rummikub.draw"
	ActionRearrange = "rummikub.rearrange"
)

var colors = []string{"black", "blue", "red", "orange"}

type Tile struct {
	ID     string `json:"id"`
	Color  string `json:"color"`
	Number int    `json:"number"`
	Joker  bool   `json:"joker"`
}

type PlayerState struct {
	PlayerID      string `json:"playerId"`
	Rack          []Tile `json:"rack"`
	RackSize      int    `json:"rackSize"`
	InitialMelded bool   `json:"initialMelded"`
	Active        bool   `json:"active"`
}

type State struct {
	CurrentPlayerIndex int           `json:"currentPlayerIndex"`
	Round              int           `json:"round"`
	Players            []PlayerState `json:"players"`
	Pool               []Tile        `json:"pool"`
	Table              [][]Tile      `json:"table"`
	Log                []string      `json:"log"`
	Finished           bool          `json:"finished"`
}

type MeldPayload struct {
	TileIDs []string   `json:"tileIds"`
	Groups  [][]string `json:"groups"`
}

type RearrangePayload struct {
	Groups [][]string `json:"groups"`
}

type Module struct{}

func NewModule() Module {
	return Module{}
}

func (m Module) ID() gamecore.GameID {
	return "rummikub"
}

func (m Module) Name() string {
	return "루미큐브"
}

func (m Module) MinPlayers() int {
	return 2
}

func (m Module) MaxPlayers() int {
	return 4
}

func (m Module) CreateInitialState(ctx gamecore.Context) any {
	pool := shuffledTiles(ctx.RandomSeed)
	players := make([]PlayerState, 0, len(ctx.Players))
	for _, player := range ctx.Players {
		rack := append([]Tile(nil), pool[:14]...)
		pool = pool[14:]
		sortRack(rack)
		players = append(players, PlayerState{
			PlayerID: string(player.ID),
			Rack:     rack,
			RackSize: len(rack),
			Active:   true,
		})
	}
	return State{
		Players: players,
		Pool:    pool,
		Table:   [][]Tile{},
		Log:     []string{"루미큐브가 시작되었습니다."},
	}
}

func (m Module) PublicState(state any, viewerID gamecore.PlayerID) any {
	current := asState(state)
	current.Pool = make([]Tile, len(current.Pool))
	current.Players = append([]PlayerState(nil), current.Players...)
	for index := range current.Players {
		current.Players[index].RackSize = len(current.Players[index].Rack)
		if current.Players[index].PlayerID != string(viewerID) {
			current.Players[index].Rack = []Tile{}
		}
	}
	return current
}

func (m Module) ValidateAction(_ context.Context, state any, action gamecore.Action, _ gamecore.Context) error {
	current := asState(state)
	if current.Finished {
		return errors.New("game is already finished")
	}
	if len(current.Players) == 0 || current.Players[current.CurrentPlayerIndex].PlayerID != string(action.PlayerID) {
		return errors.New("not your turn")
	}
	if !current.Players[current.CurrentPlayerIndex].Active {
		return errors.New("player is not active")
	}
	switch action.Type {
	case ActionDraw:
		if len(current.Pool) == 0 {
			return errors.New("pool is empty")
		}
	case ActionMeld:
		payload, err := meldPayload(action.Payload)
		if err != nil {
			return err
		}
		groups, err := selectTileGroups(current.Players[current.CurrentPlayerIndex].Rack, payload.Groups)
		if err != nil {
			return err
		}
		for _, tiles := range groups {
			if !validSet(tiles) {
				return errors.New("tiles must form a valid group or run")
			}
		}
		if !current.Players[current.CurrentPlayerIndex].InitialMelded && meldGroupsValue(groups) < 30 {
			return errors.New("initial meld must be at least 30 points")
		}
	case ActionRearrange:
		payload, err := rearrangePayload(action.Payload)
		if err != nil {
			return err
		}
		if _, _, err := rearrangedTable(current, current.CurrentPlayerIndex, payload.Groups); err != nil {
			return err
		}
	default:
		return errors.New("unsupported action")
	}
	return nil
}

func (m Module) ApplyAction(_ context.Context, state any, action gamecore.Action, _ gamecore.Context) (gamecore.ActionResult, error) {
	current := asState(state)
	player := &current.Players[current.CurrentPlayerIndex]
	switch action.Type {
	case ActionDraw:
		drawn := current.Pool[0]
		current.Pool = current.Pool[1:]
		player.Rack = append(player.Rack, drawn)
		sortRack(player.Rack)
		player.RackSize = len(player.Rack)
		current.Log = append(current.Log, player.PlayerID+" 님이 타일을 1개 뽑았습니다.")
	case ActionMeld:
		payload, err := meldPayload(action.Payload)
		if err != nil {
			return gamecore.ActionResult{}, err
		}
		groups, err := selectTileGroups(player.Rack, payload.Groups)
		if err != nil {
			return gamecore.ActionResult{}, err
		}
		if _, err := removeTiles(player, flattenIDs(payload.Groups)); err != nil {
			return gamecore.ActionResult{}, err
		}
		player.InitialMelded = true
		current.Table = append(current.Table, groups...)
		current.Log = append(current.Log, fmt.Sprintf("%s 님이 %d개 조합을 등록했습니다.", player.PlayerID, len(groups)))
		if len(player.Rack) == 0 {
			current.Finished = true
			current.Log = append(current.Log, "루미큐브가 종료되었습니다.")
		}
	case ActionRearrange:
		payload, err := rearrangePayload(action.Payload)
		if err != nil {
			return gamecore.ActionResult{}, err
		}
		groups, rackTileIDs, err := rearrangedTable(current, current.CurrentPlayerIndex, payload.Groups)
		if err != nil {
			return gamecore.ActionResult{}, err
		}
		if _, err := removeTiles(player, rackTileIDs); err != nil {
			return gamecore.ActionResult{}, err
		}
		current.Table = groups
		current.Log = append(current.Log, fmt.Sprintf("%s 님이 테이블을 %d개 조합으로 재배열했습니다.", player.PlayerID, len(groups)))
		if len(player.Rack) == 0 {
			current.Finished = true
			current.Log = append(current.Log, "루미큐브가 종료되었습니다.")
		}
	}
	if !current.Finished {
		current.CurrentPlayerIndex = nextActiveIndex(current, current.CurrentPlayerIndex)
		current.Round++
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
	index := findPlayer(current, string(playerID))
	if index >= 0 && !current.Finished {
		current.Players[index].Active = false
		current.Log = append(current.Log, string(playerID)+" 님의 재접속 시간이 만료되어 자동 기권 처리되었습니다.")
		if activePlayers(current) <= 1 {
			current.Finished = true
		} else if current.CurrentPlayerIndex == index {
			current.CurrentPlayerIndex = nextActiveIndex(current, index)
		}
	}
	return gamecore.ActionResult{State: current}, nil
}

func (m Module) IsFinished(state any, _ gamecore.Context) bool {
	return asState(state).Finished
}

func (m Module) CalculateResult(state any, _ gamecore.Context) []gamecore.Result {
	current := asState(state)
	players := append([]PlayerState(nil), current.Players...)
	sort.SliceStable(players, func(i, j int) bool {
		if players[i].Active != players[j].Active {
			return players[i].Active
		}
		return rackPenalty(players[i].Rack) < rackPenalty(players[j].Rack)
	})
	results := make([]gamecore.Result, 0, len(players))
	winnerScore := 0
	if len(players) > 0 && players[0].Active && len(players[0].Rack) == 0 {
		for _, player := range players[1:] {
			winnerScore += rackPenalty(player.Rack)
		}
	}
	for index, player := range players {
		outcome := gamecore.OutcomeLose
		score := -rackPenalty(player.Rack)
		if index == 0 {
			outcome = gamecore.OutcomeWin
			if winnerScore > 0 {
				score = winnerScore
			}
		}
		results = append(results, gamecore.Result{
			PlayerID: gamecore.PlayerID(player.PlayerID),
			Rank:     index + 1,
			Score:    score,
			Outcome:  outcome,
		})
	}
	return results
}

func validSet(tiles []Tile) bool {
	if len(tiles) < 3 {
		return false
	}
	return validGroup(tiles) || validRun(tiles)
}

func validGroup(tiles []Tile) bool {
	number := 0
	seenColors := map[string]bool{}
	for _, tile := range tiles {
		if tile.Joker {
			continue
		}
		if number == 0 {
			number = tile.Number
		}
		if tile.Number != number || seenColors[tile.Color] {
			return false
		}
		seenColors[tile.Color] = true
	}
	return len(tiles) <= 4 && number > 0
}

func validRun(tiles []Tile) bool {
	nonJokers := []Tile{}
	jokers := 0
	for _, tile := range tiles {
		if tile.Joker {
			jokers++
			continue
		}
		nonJokers = append(nonJokers, tile)
	}
	if len(nonJokers) == 0 {
		return false
	}
	color := nonJokers[0].Color
	for _, tile := range nonJokers {
		if tile.Color != color {
			return false
		}
	}
	sort.SliceStable(nonJokers, func(i, j int) bool {
		return nonJokers[i].Number < nonJokers[j].Number
	})
	gaps := 0
	for index := 1; index < len(nonJokers); index++ {
		diff := nonJokers[index].Number - nonJokers[index-1].Number
		if diff <= 0 {
			return false
		}
		gaps += diff - 1
	}
	return len(tiles) <= 13 && gaps <= jokers && nonJokers[len(nonJokers)-1].Number <= 13
}

func meldValue(tiles []Tile) int {
	total := 0
	for _, tile := range tiles {
		if tile.Joker {
			total += 30
		} else {
			total += tile.Number
		}
	}
	return total
}

func meldGroupsValue(groups [][]Tile) int {
	total := 0
	for _, group := range groups {
		total += meldValue(group)
	}
	return total
}

func rackPenalty(tiles []Tile) int {
	total := 0
	for _, tile := range tiles {
		if tile.Joker {
			total += 30
		} else {
			total += tile.Number
		}
	}
	return total
}

func rearrangedTable(state State, playerIndex int, groups [][]string) ([][]Tile, []string, error) {
	if !state.Players[playerIndex].InitialMelded {
		return nil, nil, errors.New("initial meld is required before rearranging table")
	}
	if len(state.Table) == 0 {
		return nil, nil, errors.New("table is empty")
	}
	tableTiles := tileMap(flattenTiles(state.Table))
	rackTiles := tileMap(state.Players[playerIndex].Rack)
	usedTableTiles := map[string]bool{}
	rackTileIDs := []string{}
	seen := map[string]bool{}
	result := make([][]Tile, 0, len(groups))
	for _, ids := range groups {
		if len(ids) < 3 {
			return nil, nil, errors.New("at least three tiles are required")
		}
		group := make([]Tile, 0, len(ids))
		for _, id := range ids {
			if seen[id] {
				return nil, nil, errors.New("tile cannot be used twice")
			}
			seen[id] = true
			if tile, ok := tableTiles[id]; ok {
				usedTableTiles[id] = true
				group = append(group, tile)
				continue
			}
			if tile, ok := rackTiles[id]; ok {
				rackTileIDs = append(rackTileIDs, id)
				group = append(group, tile)
				continue
			}
			return nil, nil, errors.New("tile not found on table or rack")
		}
		if !validSet(group) {
			return nil, nil, errors.New("rearranged groups must be valid")
		}
		result = append(result, group)
	}
	if len(rackTileIDs) == 0 {
		return nil, nil, errors.New("rearrange must add at least one rack tile")
	}
	if len(usedTableTiles) != len(tableTiles) {
		return nil, nil, errors.New("all table tiles must remain on the table")
	}
	return result, rackTileIDs, nil
}

func flattenTiles(groups [][]Tile) []Tile {
	tiles := []Tile{}
	for _, group := range groups {
		tiles = append(tiles, group...)
	}
	return tiles
}

func tileMap(tiles []Tile) map[string]Tile {
	mapped := map[string]Tile{}
	for _, tile := range tiles {
		mapped[tile.ID] = tile
	}
	return mapped
}

func selectTileGroups(rack []Tile, groups [][]string) ([][]Tile, error) {
	if len(groups) == 0 {
		return nil, errors.New("at least one tile group is required")
	}
	seen := map[string]bool{}
	selectedGroups := make([][]Tile, 0, len(groups))
	for _, ids := range groups {
		if len(ids) < 3 {
			return nil, errors.New("at least three tiles are required")
		}
		for _, id := range ids {
			if seen[id] {
				return nil, errors.New("tile cannot be used twice")
			}
			seen[id] = true
		}
		tiles, err := selectTiles(rack, ids)
		if err != nil {
			return nil, err
		}
		selectedGroups = append(selectedGroups, tiles)
	}
	return selectedGroups, nil
}

func selectTiles(rack []Tile, ids []string) ([]Tile, error) {
	available := map[string]Tile{}
	for _, tile := range rack {
		available[tile.ID] = tile
	}
	tiles := make([]Tile, 0, len(ids))
	for _, id := range ids {
		tile, ok := available[id]
		if !ok {
			return nil, errors.New("tile not found in rack")
		}
		tiles = append(tiles, tile)
	}
	return tiles, nil
}

func flattenIDs(groups [][]string) []string {
	ids := []string{}
	for _, group := range groups {
		ids = append(ids, group...)
	}
	return ids
}

func removeTiles(player *PlayerState, ids []string) ([]Tile, error) {
	selected := map[string]bool{}
	for _, id := range ids {
		selected[id] = true
	}
	tiles := []Tile{}
	next := []Tile{}
	for _, tile := range player.Rack {
		if selected[tile.ID] {
			tiles = append(tiles, tile)
			continue
		}
		next = append(next, tile)
	}
	if len(tiles) != len(ids) {
		return nil, errors.New("tile not found in rack")
	}
	player.Rack = next
	player.RackSize = len(next)
	return tiles, nil
}

func meldPayload(payload any) (MeldPayload, error) {
	raw, ok := payload.(map[string]any)
	if !ok {
		return MeldPayload{}, errors.New("invalid meld payload")
	}
	if rawGroups, ok := raw["groups"]; ok {
		groups, err := tileIDGroupsFromAny(rawGroups)
		if err != nil {
			return MeldPayload{}, err
		}
		return MeldPayload{Groups: groups}, nil
	}
	values, ok := raw["tileIds"].([]any)
	if !ok {
		return MeldPayload{}, errors.New("at least three tiles are required")
	}
	ids, err := tileIDsFromAny(values)
	if err != nil {
		return MeldPayload{}, err
	}
	return MeldPayload{TileIDs: ids, Groups: [][]string{ids}}, nil
}

func rearrangePayload(payload any) (RearrangePayload, error) {
	raw, ok := payload.(map[string]any)
	if !ok {
		return RearrangePayload{}, errors.New("invalid rearrange payload")
	}
	rawGroups, ok := raw["groups"]
	if !ok {
		return RearrangePayload{}, errors.New("at least one tile group is required")
	}
	groups, err := tileIDGroupsFromAny(rawGroups)
	if err != nil {
		return RearrangePayload{}, err
	}
	return RearrangePayload{Groups: groups}, nil
}

func tileIDGroupsFromAny(value any) ([][]string, error) {
	values, ok := value.([]any)
	if !ok || len(values) == 0 {
		return nil, errors.New("at least one tile group is required")
	}
	groups := make([][]string, 0, len(values))
	for _, item := range values {
		rawIDs, ok := item.([]any)
		if !ok {
			return nil, errors.New("invalid tile group")
		}
		ids, err := tileIDsFromAny(rawIDs)
		if err != nil {
			return nil, err
		}
		groups = append(groups, ids)
	}
	return groups, nil
}

func tileIDsFromAny(values []any) ([]string, error) {
	if len(values) < 3 {
		return nil, errors.New("at least three tiles are required")
	}
	ids := make([]string, 0, len(values))
	for _, value := range values {
		id, ok := value.(string)
		if !ok || id == "" {
			return nil, errors.New("invalid tile id")
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func activePlayers(state State) int {
	count := 0
	for _, player := range state.Players {
		if player.Active {
			count++
		}
	}
	return count
}

func nextActiveIndex(state State, current int) int {
	for step := 1; step <= len(state.Players); step++ {
		next := (current + step) % len(state.Players)
		if state.Players[next].Active {
			return next
		}
	}
	return current
}

func findPlayer(state State, playerID string) int {
	for index, player := range state.Players {
		if player.PlayerID == playerID {
			return index
		}
	}
	return -1
}

func shuffledTiles(seed string) []Tile {
	tiles := make([]Tile, 0, 106)
	for copyIndex := 1; copyIndex <= 2; copyIndex++ {
		for _, color := range colors {
			for number := 1; number <= 13; number++ {
				tiles = append(tiles, Tile{ID: fmt.Sprintf("%s-%d-%d", color, number, copyIndex), Color: color, Number: number})
			}
		}
	}
	tiles = append(tiles, Tile{ID: "joker-1", Joker: true}, Tile{ID: "joker-2", Joker: true})
	random := rand.New(rand.NewSource(seedToInt(seed)))
	random.Shuffle(len(tiles), func(i, j int) {
		tiles[i], tiles[j] = tiles[j], tiles[i]
	})
	return tiles
}

func sortRack(rack []Tile) {
	sort.SliceStable(rack, func(i, j int) bool {
		if rack[i].Joker != rack[j].Joker {
			return !rack[i].Joker
		}
		if rack[i].Color != rack[j].Color {
			return rack[i].Color < rack[j].Color
		}
		return rack[i].Number < rack[j].Number
	})
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
