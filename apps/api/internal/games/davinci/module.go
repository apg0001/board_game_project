package davinci

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
	ActionGuess  = "davinci.guess"
	ActionPass   = "davinci.pass"
	ActionFinish = "davinci.finish"
)

type Tile struct {
	Color    string `json:"color"`
	Value    int    `json:"value"`
	Joker    bool   `json:"joker"`
	Revealed bool   `json:"revealed"`
}

type PlayerState struct {
	PlayerID string `json:"playerId"`
	Tiles    []Tile `json:"tiles"`
	Active   bool   `json:"active"`
}

type State struct {
	CurrentPlayerIndex int           `json:"currentPlayerIndex"`
	Round              int           `json:"round"`
	Players            []PlayerState `json:"players"`
	Deck               []Tile        `json:"deck"`
	PendingTile        *Tile         `json:"pendingTile,omitempty"`
	PendingOwnerID     string        `json:"pendingOwnerId,omitempty"`
	CanEndTurn         bool          `json:"canEndTurn"`
	Log                []string      `json:"log"`
	Finished           bool          `json:"finished"`
}

type GuessPayload struct {
	TargetPlayerID string `json:"targetPlayerId"`
	TileIndex      int    `json:"tileIndex"`
	Color          string `json:"color"`
	Value          int    `json:"value"`
	Joker          bool   `json:"joker"`
	InsertIndex    int    `json:"insertIndex"`
}

type EndTurnPayload struct {
	InsertIndex int `json:"insertIndex"`
}

type Module struct{}

func NewModule() Module {
	return Module{}
}

func (m Module) ID() gamecore.GameID {
	return "davinci"
}

func (m Module) Name() string {
	return "다빈치 코드"
}

func (m Module) MinPlayers() int {
	return 2
}

func (m Module) MaxPlayers() int {
	return 4
}

func (m Module) CreateInitialState(ctx gamecore.Context) any {
	deck := shuffledDeck(ctx.RandomSeed)
	handSize := 4
	if len(ctx.Players) == 4 {
		handSize = 3
	}

	players := make([]PlayerState, 0, len(ctx.Players))
	for playerIndex, player := range ctx.Players {
		hand := append([]Tile(nil), deck[:handSize]...)
		deck = deck[handSize:]
		hand = arrangeInitialTiles(hand, fmt.Sprintf("%s:%s:%d", ctx.RandomSeed, player.ID, playerIndex))
		players = append(players, PlayerState{
			PlayerID: string(player.ID),
			Tiles:    hand,
			Active:   true,
		})
	}

	state := State{
		CurrentPlayerIndex: 0,
		Round:              1,
		Players:            players,
		Deck:               deck,
		Log:                []string{"다빈치 코드가 시작되었습니다."},
		Finished:           false,
	}
	return beginTurn(state)
}

func (m Module) PublicState(state any, viewerID gamecore.PlayerID) any {
	current := cloneState(asState(state))
	for playerIndex := range current.Players {
		for tileIndex := range current.Players[playerIndex].Tiles {
			tile := &current.Players[playerIndex].Tiles[tileIndex]
			if current.Players[playerIndex].PlayerID == string(viewerID) || tile.Revealed {
				continue
			}
			tile.Value = -1
			tile.Color = "hidden"
			tile.Joker = false
		}
	}
	if current.PendingTile != nil && current.PendingOwnerID != string(viewerID) {
		hidden := *current.PendingTile
		hidden.Value = -1
		hidden.Color = "hidden"
		hidden.Joker = false
		hidden.Revealed = false
		current.PendingTile = &hidden
	}
	return current
}

func (m Module) ValidateAction(_ context.Context, state any, action gamecore.Action, _ gamecore.Context) error {
	current := asState(state)
	if current.Finished {
		return errors.New("game is already finished")
	}
	if len(current.Players) == 0 {
		return errors.New("game has no players")
	}
	if action.Type == ActionFinish {
		return errors.New("manual finish is not allowed")
	}
	if current.Players[current.CurrentPlayerIndex].PlayerID != string(action.PlayerID) {
		return errors.New("not your turn")
	}
	if action.Type == ActionPass {
		if !current.CanEndTurn {
			return errors.New("must make a correct guess before ending turn")
		}
		return nil
	}
	if action.Type != ActionGuess {
		return errors.New("unsupported action")
	}
	if !current.Players[current.CurrentPlayerIndex].Active {
		return errors.New("player is not active")
	}

	payload, err := guessPayload(action.Payload)
	if err != nil {
		return err
	}
	targetIndex := findPlayerIndex(current, payload.TargetPlayerID)
	if targetIndex < 0 {
		return errors.New("target player not found")
	}
	if targetIndex == current.CurrentPlayerIndex {
		return errors.New("cannot guess own tile")
	}
	if payload.TileIndex < 0 || payload.TileIndex >= len(current.Players[targetIndex].Tiles) {
		return errors.New("tile index out of range")
	}
	if current.Players[targetIndex].Tiles[payload.TileIndex].Revealed {
		return errors.New("tile is already revealed")
	}
	if payload.Color != "black" && payload.Color != "white" {
		return errors.New("tile color must be black or white")
	}
	if !payload.Joker && (payload.Value < 0 || payload.Value > 11) {
		return errors.New("tile value must be between 0 and 11")
	}
	return nil
}

func (m Module) ApplyAction(_ context.Context, state any, action gamecore.Action, gameCtx gamecore.Context) (gamecore.ActionResult, error) {
	current := asState(state)

	switch action.Type {
	case ActionGuess:
		payload, err := guessPayload(action.Payload)
		if err != nil {
			return gamecore.ActionResult{}, err
		}
		current = applyGuess(current, string(action.PlayerID), payload)
	case ActionPass:
		payload := endTurnPayload(action.Payload)
		insertPendingTile(&current, false, payload.InsertIndex)
		current.Log = append(current.Log, string(action.PlayerID)+" 님이 턴을 종료했습니다.")
		current = advanceTurn(current)
	case ActionFinish:
		return gamecore.ActionResult{}, errors.New("manual finish is not allowed")
	default:
		return gamecore.ActionResult{}, errors.New("unsupported action")
	}

	current = refreshActivePlayers(current)
	if remainingActivePlayers(current) <= 1 {
		current.Finished = true
		current.Log = append(current.Log, "게임이 종료되었습니다.")
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
	index := findPlayerIndex(current, string(playerID))
	if index < 0 || current.Finished {
		return gamecore.ActionResult{State: current}, nil
	}

	if current.PendingOwnerID == string(playerID) {
		insertPendingTile(&current, true, -1)
	}
	for tileIndex := range current.Players[index].Tiles {
		current.Players[index].Tiles[tileIndex].Revealed = true
	}
	current.Players[index].Active = false
	current.Log = append(current.Log, string(playerID)+" 님의 재접속 시간이 만료되어 자동 기권 처리되었습니다.")
	if current.CurrentPlayerIndex == index {
		current = advanceTurn(current)
	}
	current = refreshActivePlayers(current)
	if remainingActivePlayers(current) <= 1 {
		current.Finished = true
		current.Log = append(current.Log, "게임이 종료되었습니다.")
	}

	return gamecore.ActionResult{
		State: current,
		Events: []gamecore.Event{{
			Type:       "game.player_timed_out",
			Visibility: gamecore.VisibilityPublic,
			Payload: map[string]any{
				"playerId": string(playerID),
				"policy":   "forfeit",
			},
		}},
	}, nil
}

func (m Module) IsFinished(state any, _ gamecore.Context) bool {
	return asState(state).Finished
}

func (m Module) CalculateResult(state any, _ gamecore.Context) []gamecore.Result {
	current := asState(state)
	results := make([]gamecore.Result, 0, len(current.Players))

	sort.SliceStable(current.Players, func(i, j int) bool {
		return hiddenCount(current.Players[i]) > hiddenCount(current.Players[j])
	})

	for index, player := range current.Players {
		outcome := gamecore.OutcomeLose
		if index == 0 {
			outcome = gamecore.OutcomeWin
		}
		if len(current.Players) > 1 && hiddenCount(player) == hiddenCount(current.Players[0]) {
			outcome = gamecore.OutcomeDraw
		}
		results = append(results, gamecore.Result{
			PlayerID: gamecore.PlayerID(player.PlayerID),
			Rank:     index + 1,
			Score:    hiddenCount(player),
			Outcome:  outcome,
		})
	}
	return results
}

func applyGuess(state State, playerID string, payload GuessPayload) State {
	targetIndex := findPlayerIndex(state, payload.TargetPlayerID)
	targetTile := &state.Players[targetIndex].Tiles[payload.TileIndex]
	correct := matchesGuess(*targetTile, payload)

	if correct {
		targetTile.Revealed = true
		state.CanEndTurn = true
		state.Log = append(state.Log, fmt.Sprintf("%s 님의 추측이 성공했습니다.", playerID))
		state = refreshActivePlayers(state)
		if remainingActivePlayers(state) <= 1 {
			state.Finished = true
			state.Log = append(state.Log, "게임이 종료되었습니다.")
		}
		return state
	} else {
		if !insertPendingTile(&state, true, payload.InsertIndex) {
			revealFirstHiddenOwnTile(&state, playerID)
		}
		state.Log = append(state.Log, fmt.Sprintf("%s 님의 추측이 실패했습니다.", playerID))
	}

	return advanceTurn(state)
}

func matchesGuess(tile Tile, payload GuessPayload) bool {
	if tile.Color != payload.Color {
		return false
	}
	if tile.Joker {
		return payload.Joker
	}
	return !payload.Joker && tile.Value == payload.Value
}

func beginTurn(state State) State {
	state.CanEndTurn = false
	if len(state.Players) == 0 || state.Finished {
		return state
	}
	if state.PendingTile != nil {
		return state
	}
	currentPlayer := state.Players[state.CurrentPlayerIndex]
	if !currentPlayer.Active || len(state.Deck) == 0 {
		return state
	}
	pending := state.Deck[0]
	state.Deck = state.Deck[1:]
	state.PendingTile = &pending
	state.PendingOwnerID = currentPlayer.PlayerID
	return state
}

func insertPendingTile(state *State, revealed bool, insertIndex int) bool {
	if state.PendingTile == nil {
		return false
	}
	playerIndex := findPlayerIndex(*state, state.PendingOwnerID)
	if playerIndex < 0 {
		state.PendingTile = nil
		state.PendingOwnerID = ""
		state.CanEndTurn = false
		return false
	}
	tile := *state.PendingTile
	tile.Revealed = revealed
	if tile.Joker {
		state.Players[playerIndex].Tiles = insertTileAt(state.Players[playerIndex].Tiles, tile, insertIndex)
	} else {
		state.Players[playerIndex].Tiles = insertNumberTile(state.Players[playerIndex].Tiles, tile)
	}
	state.PendingTile = nil
	state.PendingOwnerID = ""
	state.CanEndTurn = false
	return true
}

func revealFirstHiddenOwnTile(state *State, playerID string) {
	playerIndex := findPlayerIndex(*state, playerID)
	if playerIndex < 0 {
		return
	}
	for tileIndex := range state.Players[playerIndex].Tiles {
		if !state.Players[playerIndex].Tiles[tileIndex].Revealed {
			state.Players[playerIndex].Tiles[tileIndex].Revealed = true
			return
		}
	}
}

func advanceTurn(state State) State {
	if len(state.Players) == 0 {
		return state
	}
	state.PendingTile = nil
	state.PendingOwnerID = ""
	state.CanEndTurn = false
	for step := 1; step <= len(state.Players); step++ {
		next := (state.CurrentPlayerIndex + step) % len(state.Players)
		if state.Players[next].Active {
			state.CurrentPlayerIndex = next
			state.Round++
			return beginTurn(state)
		}
	}
	return state
}

func refreshActivePlayers(state State) State {
	for index := range state.Players {
		state.Players[index].Active = hiddenCount(state.Players[index]) > 0
	}
	return state
}

func remainingActivePlayers(state State) int {
	count := 0
	for _, player := range state.Players {
		if player.Active {
			count++
		}
	}
	return count
}

func hiddenCount(player PlayerState) int {
	count := 0
	for _, tile := range player.Tiles {
		if !tile.Revealed {
			count++
		}
	}
	return count
}

func findPlayerIndex(state State, playerID string) int {
	for index, player := range state.Players {
		if player.PlayerID == playerID {
			return index
		}
	}
	return -1
}

func shuffledDeck(seed string) []Tile {
	deck := make([]Tile, 0, 26)
	for _, color := range []string{"black", "white"} {
		for value := 0; value <= 11; value++ {
			deck = append(deck, Tile{Color: color, Value: value})
		}
		deck = append(deck, Tile{Color: color, Value: -1, Joker: true})
	}

	random := rand.New(rand.NewSource(seedToInt(seed)))
	random.Shuffle(len(deck), func(i, j int) {
		deck[i], deck[j] = deck[j], deck[i]
	})
	return deck
}

func sortTiles(tiles []Tile) {
	sort.SliceStable(tiles, func(i, j int) bool {
		if tiles[i].Value != tiles[j].Value {
			return tiles[i].Value < tiles[j].Value
		}
		return tiles[i].Color < tiles[j].Color
	})
}

func arrangeInitialTiles(tiles []Tile, seed string) []Tile {
	numbered := make([]Tile, 0, len(tiles))
	jokers := make([]Tile, 0, len(tiles))
	for _, tile := range tiles {
		if tile.Joker {
			jokers = append(jokers, tile)
			continue
		}
		numbered = append(numbered, tile)
	}
	sortTiles(numbered)
	if len(jokers) == 0 {
		return numbered
	}

	random := rand.New(rand.NewSource(seedToInt(seed)))
	arranged := append([]Tile(nil), numbered...)
	for _, joker := range jokers {
		insertIndex := 0
		if len(arranged) > 0 {
			insertIndex = random.Intn(len(arranged) + 1)
		}
		arranged = insertTileAt(arranged, joker, insertIndex)
	}
	return arranged
}

func insertNumberTile(tiles []Tile, tile Tile) []Tile {
	insertIndex := len(tiles)
	for index, existing := range tiles {
		if existing.Joker {
			continue
		}
		if tileComesBefore(tile, existing) {
			insertIndex = index
			break
		}
	}
	return insertTileAt(tiles, tile, insertIndex)
}

func insertTileAt(tiles []Tile, tile Tile, index int) []Tile {
	if index < 0 || index > len(tiles) {
		index = len(tiles)
	}
	next := make([]Tile, 0, len(tiles)+1)
	next = append(next, tiles[:index]...)
	next = append(next, tile)
	next = append(next, tiles[index:]...)
	return next
}

func tileComesBefore(left Tile, right Tile) bool {
	if left.Value != right.Value {
		return left.Value < right.Value
	}
	return left.Color < right.Color
}

func cloneState(state State) State {
	clone := state
	clone.Players = make([]PlayerState, len(state.Players))
	for index := range state.Players {
		clone.Players[index] = state.Players[index]
		clone.Players[index].Tiles = append([]Tile(nil), state.Players[index].Tiles...)
	}
	clone.Deck = append([]Tile(nil), state.Deck...)
	clone.Log = append([]string(nil), state.Log...)
	if state.PendingTile != nil {
		pending := *state.PendingTile
		clone.PendingTile = &pending
	}
	return clone
}

func seedToInt(seed string) int64 {
	hash := fnv.New64a()
	_, _ = hash.Write([]byte(seed))
	return int64(hash.Sum64())
}

func guessPayload(payload any) (GuessPayload, error) {
	raw, ok := payload.(map[string]any)
	if !ok {
		return GuessPayload{}, errors.New("invalid guess payload")
	}

	result := GuessPayload{TileIndex: -1, InsertIndex: -1}
	if value, ok := raw["targetPlayerId"].(string); ok {
		result.TargetPlayerID = value
	}
	if value, ok := gameutil.Int(raw["tileIndex"]); ok {
		result.TileIndex = value
	}
	if value, ok := raw["color"].(string); ok {
		result.Color = value
	}
	if value, ok := gameutil.Int(raw["value"]); ok {
		result.Value = value
	}
	if value, ok := raw["joker"].(bool); ok {
		result.Joker = value
	}
	if value, ok := gameutil.Int(raw["insertIndex"]); ok {
		result.InsertIndex = value
	}
	if result.TargetPlayerID == "" || result.Color == "" {
		return GuessPayload{}, errors.New("guess target, color and value are required")
	}
	return result, nil
}

func endTurnPayload(payload any) EndTurnPayload {
	raw, ok := payload.(map[string]any)
	if !ok {
		return EndTurnPayload{InsertIndex: -1}
	}
	result := EndTurnPayload{InsertIndex: -1}
	if value, ok := gameutil.Int(raw["insertIndex"]); ok {
		result.InsertIndex = value
	}
	return result
}

func asState(state any) State {
	typed, ok := state.(State)
	if ok {
		return typed
	}
	return State{}
}
