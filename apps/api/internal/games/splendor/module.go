package splendor

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
	ActionTakeToken = "splendor.take_token"
	ActionBuyCard   = "splendor.buy_card"
)

var colors = []string{"white", "blue", "green", "red", "black"}

type Card struct {
	ID     string         `json:"id"`
	Color  string         `json:"color"`
	Points int            `json:"points"`
	Cost   map[string]int `json:"cost"`
}

type PlayerState struct {
	PlayerID string         `json:"playerId"`
	Tokens   map[string]int `json:"tokens"`
	Bonuses  map[string]int `json:"bonuses"`
	Cards    []Card         `json:"cards"`
	Score    int            `json:"score"`
	Active   bool           `json:"active"`
}

type State struct {
	CurrentPlayerIndex int            `json:"currentPlayerIndex"`
	Round              int            `json:"round"`
	Bank               map[string]int `json:"bank"`
	Market             []Card         `json:"market"`
	Deck               []Card         `json:"deck"`
	Players            []PlayerState  `json:"players"`
	Log                []string       `json:"log"`
	Finished           bool           `json:"finished"`
}

type Module struct{}

func NewModule() Module {
	return Module{}
}

func (m Module) ID() gamecore.GameID {
	return "splendor"
}

func (m Module) Name() string {
	return "스플랜더"
}

func (m Module) MinPlayers() int {
	return 2
}

func (m Module) MaxPlayers() int {
	return 4
}

func (m Module) CreateInitialState(ctx gamecore.Context) any {
	deck := shuffledDeck(ctx.RandomSeed)
	market := append([]Card(nil), deck[:4]...)
	deck = deck[4:]
	players := make([]PlayerState, 0, len(ctx.Players))
	for _, player := range ctx.Players {
		players = append(players, PlayerState{
			PlayerID: string(player.ID),
			Tokens:   emptyCounter(),
			Bonuses:  emptyCounter(),
			Cards:    []Card{},
			Active:   true,
		})
	}
	return State{
		CurrentPlayerIndex: 0,
		Round:              1,
		Bank:               startingBank(len(ctx.Players)),
		Market:             market,
		Deck:               deck,
		Players:            players,
		Log:                []string{"스플랜더가 시작되었습니다."},
	}
}

func (m Module) PublicState(state any, _ gamecore.PlayerID) any {
	current := cloneState(asState(state))
	current.Deck = make([]Card, len(current.Deck))
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
	case ActionTakeToken:
		color, err := tokenPayload(action.Payload)
		if err != nil {
			return err
		}
		if current.Bank[color] <= 0 {
			return errors.New("token is not available")
		}
		if totalTokens(current.Players[current.CurrentPlayerIndex]) >= 10 {
			return errors.New("token limit is reached")
		}
	case ActionBuyCard:
		index, err := buyPayload(action.Payload)
		if err != nil {
			return err
		}
		if index < 0 || index >= len(current.Market) {
			return errors.New("card index out of range")
		}
		if !canAfford(current.Players[current.CurrentPlayerIndex], current.Market[index]) {
			return errors.New("not enough tokens")
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
	case ActionTakeToken:
		color, err := tokenPayload(action.Payload)
		if err != nil {
			return gamecore.ActionResult{}, err
		}
		player.Tokens[color]++
		current.Bank[color]--
		current.Log = append(current.Log, fmt.Sprintf("%s 님이 %s 보석을 가져갔습니다.", player.PlayerID, color))
	case ActionBuyCard:
		index, err := buyPayload(action.Payload)
		if err != nil {
			return gamecore.ActionResult{}, err
		}
		card := current.Market[index]
		payCost(player, &current, card)
		player.Cards = append(player.Cards, card)
		player.Bonuses[card.Color]++
		player.Score += card.Points
		current.Market = append(current.Market[:index], current.Market[index+1:]...)
		if len(current.Deck) > 0 {
			current.Market = append(current.Market, current.Deck[0])
			current.Deck = current.Deck[1:]
		}
		current.Log = append(current.Log, fmt.Sprintf("%s 님이 %d점 카드를 구매했습니다.", player.PlayerID, card.Points))
	}
	if player.Score >= 15 {
		current.Finished = true
		current.Log = append(current.Log, "스플랜더가 종료되었습니다.")
	}
	if !current.Finished {
		current = advanceTurn(current)
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
	if index < 0 || current.Finished {
		return gamecore.ActionResult{State: current}, nil
	}
	current.Players[index].Active = false
	returnTokensToBank(&current, index)
	current.Log = append(current.Log, string(playerID)+" 님의 재접속 시간이 만료되어 자동 기권 처리되었습니다.")
	if current.CurrentPlayerIndex == index {
		current = advanceTurn(current)
	}
	if activePlayers(current) <= 1 {
		current.Finished = true
		current.Log = append(current.Log, "스플랜더가 종료되었습니다.")
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
	players := append([]PlayerState(nil), current.Players...)
	sort.SliceStable(players, func(i, j int) bool {
		if players[i].Score != players[j].Score {
			return players[i].Score > players[j].Score
		}
		return len(players[i].Cards) < len(players[j].Cards)
	})
	results := make([]gamecore.Result, 0, len(players))
	for index, player := range players {
		outcome := gamecore.OutcomeLose
		if index == 0 {
			outcome = gamecore.OutcomeWin
		}
		results = append(results, gamecore.Result{
			PlayerID: gamecore.PlayerID(player.PlayerID),
			Rank:     index + 1,
			Score:    player.Score,
			Outcome:  outcome,
		})
	}
	return results
}

func advanceTurn(state State) State {
	for step := 1; step <= len(state.Players); step++ {
		next := (state.CurrentPlayerIndex + step) % len(state.Players)
		if state.Players[next].Active {
			state.CurrentPlayerIndex = next
			state.Round++
			return state
		}
	}
	return state
}

func canAfford(player PlayerState, card Card) bool {
	for color, cost := range card.Cost {
		required := cost - player.Bonuses[color]
		if required < 0 {
			required = 0
		}
		if player.Tokens[color] < required {
			return false
		}
	}
	return true
}

func payCost(player *PlayerState, state *State, card Card) {
	for color, cost := range card.Cost {
		required := cost - player.Bonuses[color]
		if required < 0 {
			required = 0
		}
		player.Tokens[color] -= required
		state.Bank[color] += required
	}
}

func returnTokensToBank(state *State, playerIndex int) {
	for color, count := range state.Players[playerIndex].Tokens {
		if count <= 0 {
			continue
		}
		state.Bank[color] += count
		state.Players[playerIndex].Tokens[color] = 0
	}
}

func totalTokens(player PlayerState) int {
	total := 0
	for _, count := range player.Tokens {
		total += count
	}
	return total
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

func findPlayer(state State, playerID string) int {
	for index, player := range state.Players {
		if player.PlayerID == playerID {
			return index
		}
	}
	return -1
}

func tokenPayload(payload any) (string, error) {
	raw, ok := payload.(map[string]any)
	if !ok {
		return "", errors.New("invalid token payload")
	}
	color, _ := raw["color"].(string)
	if !validColor(color) {
		return "", errors.New("invalid token color")
	}
	return color, nil
}

func buyPayload(payload any) (int, error) {
	raw, ok := payload.(map[string]any)
	if !ok {
		return 0, errors.New("invalid buy payload")
	}
	value, ok := gameutil.Int(raw["marketIndex"])
	if !ok {
		return 0, errors.New("market index is required")
	}
	return value, nil
}

func validColor(color string) bool {
	for _, item := range colors {
		if item == color {
			return true
		}
	}
	return false
}

func emptyCounter() map[string]int {
	counter := map[string]int{}
	for _, color := range colors {
		counter[color] = 0
	}
	return counter
}

func startingBank(playerCount int) map[string]int {
	amount := 7
	if playerCount == 2 {
		amount = 4
	}
	if playerCount == 3 {
		amount = 5
	}
	bank := emptyCounter()
	for _, color := range colors {
		bank[color] = amount
	}
	return bank
}

func cloneState(state State) State {
	clone := state
	clone.Bank = copyCounter(state.Bank)
	clone.Market = cloneCards(state.Market)
	clone.Deck = cloneCards(state.Deck)
	clone.Players = make([]PlayerState, len(state.Players))
	for index := range state.Players {
		clone.Players[index] = state.Players[index]
		clone.Players[index].Tokens = copyCounter(state.Players[index].Tokens)
		clone.Players[index].Bonuses = copyCounter(state.Players[index].Bonuses)
		clone.Players[index].Cards = cloneCards(state.Players[index].Cards)
	}
	clone.Log = append([]string(nil), state.Log...)
	return clone
}

func cloneCards(cards []Card) []Card {
	result := make([]Card, len(cards))
	for index := range cards {
		result[index] = cards[index]
		result[index].Cost = copyCounter(cards[index].Cost)
	}
	return result
}

func copyCounter(source map[string]int) map[string]int {
	result := map[string]int{}
	for key, value := range source {
		result[key] = value
	}
	return result
}

func shuffledDeck(seed string) []Card {
	deck := []Card{
		{ID: "w1", Color: "white", Points: 0, Cost: map[string]int{"blue": 1, "green": 1}},
		{ID: "u1", Color: "blue", Points: 0, Cost: map[string]int{"red": 1, "black": 1}},
		{ID: "g1", Color: "green", Points: 0, Cost: map[string]int{"white": 1, "blue": 1}},
		{ID: "r1", Color: "red", Points: 0, Cost: map[string]int{"green": 1, "black": 1}},
		{ID: "b1", Color: "black", Points: 0, Cost: map[string]int{"white": 1, "red": 1}},
		{ID: "w2", Color: "white", Points: 1, Cost: map[string]int{"blue": 2, "green": 2}},
		{ID: "u2", Color: "blue", Points: 1, Cost: map[string]int{"red": 2, "black": 2}},
		{ID: "g2", Color: "green", Points: 1, Cost: map[string]int{"white": 2, "blue": 2}},
		{ID: "r2", Color: "red", Points: 1, Cost: map[string]int{"green": 2, "black": 2}},
		{ID: "b2", Color: "black", Points: 1, Cost: map[string]int{"white": 2, "red": 2}},
		{ID: "w3", Color: "white", Points: 3, Cost: map[string]int{"blue": 3, "green": 3, "red": 1}},
		{ID: "u3", Color: "blue", Points: 3, Cost: map[string]int{"red": 3, "black": 3, "white": 1}},
		{ID: "g3", Color: "green", Points: 3, Cost: map[string]int{"white": 3, "blue": 3, "black": 1}},
		{ID: "r3", Color: "red", Points: 3, Cost: map[string]int{"green": 3, "black": 3, "blue": 1}},
		{ID: "b3", Color: "black", Points: 3, Cost: map[string]int{"white": 3, "red": 3, "green": 1}},
	}
	random := rand.New(rand.NewSource(seedToInt(seed)))
	random.Shuffle(len(deck), func(i, j int) {
		deck[i], deck[j] = deck[j], deck[i]
	})
	return deck
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
