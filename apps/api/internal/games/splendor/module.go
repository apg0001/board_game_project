package splendor

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"math/rand"
	"sort"
	"strings"

	"board-game-platform/apps/api/internal/gamecore"
	"board-game-platform/apps/api/internal/games/internal/gameutil"
)

const (
	ActionTakeToken   = "splendor.take_token"
	ActionBuyCard     = "splendor.buy_card"
	ActionReserveCard = "splendor.reserve_card"
	ActionReturnToken = "splendor.return_tokens"
	ActionChooseNoble = "splendor.choose_noble"
)

var (
	colors        = []string{"white", "blue", "green", "red", "black"}
	splendorTiers = []int{1, 2, 3}
)

type Card struct {
	ID     string         `json:"id"`
	Tier   int            `json:"tier"`
	Color  string         `json:"color"`
	Points int            `json:"points"`
	Cost   map[string]int `json:"cost"`
	Hidden bool           `json:"hidden,omitempty"`
}

type Noble struct {
	ID     string         `json:"id"`
	Points int            `json:"points"`
	Cost   map[string]int `json:"cost"`
}

type PlayerState struct {
	PlayerID string         `json:"playerId"`
	Tokens   map[string]int `json:"tokens"`
	Bonuses  map[string]int `json:"bonuses"`
	Cards    []Card         `json:"cards"`
	Reserved []Card         `json:"reserved"`
	Nobles   []Noble        `json:"nobles"`
	Score    int            `json:"score"`
	Active   bool           `json:"active"`
}

type State struct {
	CurrentPlayerIndex int            `json:"currentPlayerIndex"`
	Round              int            `json:"round"`
	Bank               map[string]int `json:"bank"`
	Market             []Card         `json:"market"`
	Deck               []Card         `json:"deck"`
	Markets            map[int][]Card `json:"markets"`
	Decks              map[int][]Card `json:"decks"`
	Nobles             []Noble        `json:"nobles"`
	Players            []PlayerState  `json:"players"`
	Log                []string       `json:"log"`
	Finished           bool           `json:"finished"`
	EndTriggered       bool           `json:"endTriggered,omitempty"`
	EndTriggerIndex    int            `json:"-"`
	PendingReturnID    string         `json:"pendingReturnPlayerId,omitempty"`
	PendingReturnCount int            `json:"pendingReturnCount,omitempty"`
	PendingNobleID     string         `json:"pendingNoblePlayerId,omitempty"`
	PendingNobles      []Noble        `json:"pendingNobleChoices,omitempty"`
}

type BuyPayload struct {
	MarketTier    int
	MarketIndex   int
	ReservedIndex int
	FromDeck      bool
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
	decks := shuffledDecks(ctx.RandomSeed)
	markets := startingMarkets(decks)
	players := make([]PlayerState, 0, len(ctx.Players))
	for _, player := range ctx.Players {
		players = append(players, PlayerState{
			PlayerID: string(player.ID),
			Tokens:   emptyCounter(),
			Bonuses:  emptyCounter(),
			Cards:    []Card{},
			Reserved: []Card{},
			Nobles:   []Noble{},
			Active:   true,
		})
	}
	return State{
		CurrentPlayerIndex: 0,
		Round:              1,
		Bank:               startingBank(len(ctx.Players)),
		Market:             flattenMarkets(markets),
		Deck:               flattenDecks(decks),
		Markets:            markets,
		Decks:              decks,
		Nobles:             startingNobles(ctx.RandomSeed, len(ctx.Players)),
		Players:            players,
		Log:                []string{"스플랜더가 시작되었습니다."},
	}
}

func (m Module) PublicState(state any, viewerID gamecore.PlayerID) any {
	current := cloneState(asState(state))
	current.Deck = make([]Card, len(current.Deck))
	current.Decks = maskDecks(current.Decks)
	maskReservedCards(&current, string(viewerID))
	return current
}

func (m Module) ValidateAction(_ context.Context, state any, action gamecore.Action, _ gamecore.Context) error {
	current := asState(state)
	ensureTieredMarket(&current)
	if current.Finished {
		return errors.New("game is already finished")
	}
	if current.PendingReturnID != "" {
		if action.Type != ActionReturnToken {
			return errors.New("excess tokens must be returned first")
		}
		if current.PendingReturnID != string(action.PlayerID) {
			return errors.New("not your token return")
		}
		return validateReturnTokens(current, action)
	}
	if current.PendingNobleID != "" {
		if action.Type != ActionChooseNoble {
			return errors.New("noble must be chosen first")
		}
		if current.PendingNobleID != string(action.PlayerID) {
			return errors.New("not your noble choice")
		}
		return validateChooseNoble(current, action)
	}
	if len(current.Players) == 0 || current.Players[current.CurrentPlayerIndex].PlayerID != string(action.PlayerID) {
		return errors.New("not your turn")
	}
	if !current.Players[current.CurrentPlayerIndex].Active {
		return errors.New("player is not active")
	}
	switch action.Type {
	case ActionTakeToken:
		colorList, err := tokenPayload(action.Payload)
		if err != nil {
			return err
		}
		if err := validateTakeTokens(current, colorList); err != nil {
			return err
		}
	case ActionReserveCard:
		payload, err := reservePayload(action.Payload)
		if err != nil {
			return err
		}
		if payload.FromDeck {
			if !canReserveFromDeck(current, payload.MarketTier) {
				return errors.New("deck is not available")
			}
		} else if _, ok := selectedMarketCard(current, payload); !ok {
			return errors.New("card index out of range")
		}
		if len(current.Players[current.CurrentPlayerIndex].Reserved) >= 3 {
			return errors.New("reserved card limit is reached")
		}
	case ActionBuyCard:
		payload, err := buyPayload(action.Payload)
		if err != nil {
			return err
		}
		card, ok := selectedBuyCard(current.Players[current.CurrentPlayerIndex], current, payload)
		if !ok {
			return errors.New("card index out of range")
		}
		if !canAfford(current.Players[current.CurrentPlayerIndex], card) {
			return errors.New("not enough tokens")
		}
	default:
		return errors.New("unsupported action")
	}
	return nil
}

func (m Module) ApplyAction(_ context.Context, state any, action gamecore.Action, _ gamecore.Context) (gamecore.ActionResult, error) {
	current := asState(state)
	ensureTieredMarket(&current)
	if current.PendingReturnID != "" {
		next, err := applyReturnTokens(current, action)
		if err != nil {
			return gamecore.ActionResult{}, err
		}
		return gamecore.ActionResult{
			State: next,
			Events: []gamecore.Event{{
				Type:       "game.state_updated",
				Visibility: gamecore.VisibilityPublic,
				Payload:    m.PublicState(next, ""),
			}},
		}, nil
	}
	if current.PendingNobleID != "" {
		next, err := applyChooseNoble(current, action)
		if err != nil {
			return gamecore.ActionResult{}, err
		}
		return gamecore.ActionResult{
			State: next,
			Events: []gamecore.Event{{
				Type:       "game.state_updated",
				Visibility: gamecore.VisibilityPublic,
				Payload:    m.PublicState(next, ""),
			}},
		}, nil
	}
	player := &current.Players[current.CurrentPlayerIndex]
	switch action.Type {
	case ActionTakeToken:
		colorList, err := tokenPayload(action.Payload)
		if err != nil {
			return gamecore.ActionResult{}, err
		}
		for _, color := range colorList {
			player.Tokens[color]++
			current.Bank[color]--
		}
		current.Log = append(current.Log, fmt.Sprintf("%s 님이 %s 보석을 가져갔습니다.", player.PlayerID, strings.Join(colorList, ", ")))
	case ActionReserveCard:
		payload, err := reservePayload(action.Payload)
		if err != nil {
			return gamecore.ActionResult{}, err
		}
		var card Card
		var ok bool
		if payload.FromDeck {
			card, ok = removeDeckTopCard(&current, payload.MarketTier)
		} else {
			card, ok = removeAndRefillMarketCard(&current, payload)
		}
		if !ok {
			return gamecore.ActionResult{}, errors.New("card index out of range")
		}
		player.Reserved = append(player.Reserved, card)
		if current.Bank["gold"] > 0 {
			player.Tokens["gold"]++
			current.Bank["gold"]--
			current.Log = append(current.Log, fmt.Sprintf("%s 님이 카드를 예약하고 금 토큰을 받았습니다.", player.PlayerID))
		} else {
			current.Log = append(current.Log, fmt.Sprintf("%s 님이 카드를 예약했습니다.", player.PlayerID))
		}
	case ActionBuyCard:
		payload, err := buyPayload(action.Payload)
		if err != nil {
			return gamecore.ActionResult{}, err
		}
		card, ok := selectedBuyCard(*player, current, payload)
		if !ok {
			return gamecore.ActionResult{}, errors.New("card index out of range")
		}
		payCost(player, &current, card)
		player.Cards = append(player.Cards, card)
		player.Bonuses[card.Color]++
		player.Score += card.Points
		if payload.ReservedIndex >= 0 {
			player.Reserved = append(player.Reserved[:payload.ReservedIndex], player.Reserved[payload.ReservedIndex+1:]...)
		} else {
			if _, ok := removeAndRefillMarketCard(&current, payload); !ok {
				return gamecore.ActionResult{}, errors.New("card index out of range")
			}
		}
		current.Log = append(current.Log, fmt.Sprintf("%s 님이 %d점 카드를 구매했습니다.", player.PlayerID, card.Points))
		resolveNobleVisit(&current, current.CurrentPlayerIndex)
	}
	if overflow := totalTokens(*player) - 10; overflow > 0 {
		current.PendingReturnID = player.PlayerID
		current.PendingReturnCount = overflow
		current.Log = append(current.Log, fmt.Sprintf("%s 님이 토큰 %d개를 반납해야 합니다.", player.PlayerID, overflow))
	} else if current.PendingNobleID != "" {
		current.Log = append(current.Log, fmt.Sprintf("%s 님이 방문할 귀족을 선택해야 합니다.", player.PlayerID))
	} else {
		current = completeTurn(current, current.CurrentPlayerIndex)
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
	if current.PendingReturnID == string(playerID) {
		current.PendingReturnID = ""
		current.PendingReturnCount = 0
	}
	if current.PendingNobleID == string(playerID) {
		current.PendingNobleID = ""
		current.PendingNobles = nil
	}
	current.Log = append(current.Log, string(playerID)+" 님의 재접속 시간이 만료되어 자동 기권 처리되었습니다.")
	if current.CurrentPlayerIndex == index {
		current = advanceTurn(current)
		if current.EndTriggered && current.CurrentPlayerIndex == current.EndTriggerIndex {
			current.Finished = true
			current.Log = append(current.Log, "스플랜더가 종료되었습니다.")
		}
	}
	if current.EndTriggered && index == current.EndTriggerIndex && !current.Finished {
		current.Finished = true
		current.Log = append(current.Log, "스플랜더가 종료되었습니다.")
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

func completeTurn(state State, playerIndex int) State {
	if playerIndex >= 0 && playerIndex < len(state.Players) && state.Players[playerIndex].Score >= 15 && !state.EndTriggered {
		state.EndTriggered = true
		state.EndTriggerIndex = playerIndex
		state.Log = append(state.Log, fmt.Sprintf("%s 님이 15점을 달성해 이번 라운드가 마지막 라운드가 됩니다.", state.Players[playerIndex].PlayerID))
	}
	if !state.Finished {
		state = advanceTurn(state)
		if state.EndTriggered && state.CurrentPlayerIndex == state.EndTriggerIndex {
			state.Finished = true
			state.Log = append(state.Log, "스플랜더가 종료되었습니다.")
		}
	}
	return state
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

func validateReturnTokens(state State, action gamecore.Action) error {
	index := findPlayer(state, string(action.PlayerID))
	if index < 0 {
		return errors.New("player not found")
	}
	tokens, err := returnTokensPayload(action.Payload)
	if err != nil {
		return err
	}
	total := 0
	for color, count := range tokens {
		if !validTokenColor(color) {
			return errors.New("invalid token color")
		}
		if count <= 0 {
			return errors.New("return token count must be positive")
		}
		if state.Players[index].Tokens[color] < count {
			return errors.New("not enough tokens to return")
		}
		total += count
	}
	if total != state.PendingReturnCount {
		return errors.New("must return exact excess token count")
	}
	return nil
}

func applyReturnTokens(state State, action gamecore.Action) (State, error) {
	if err := validateReturnTokens(state, action); err != nil {
		return state, err
	}
	index := findPlayer(state, string(action.PlayerID))
	tokens, err := returnTokensPayload(action.Payload)
	if err != nil {
		return state, err
	}
	for color, count := range tokens {
		state.Players[index].Tokens[color] -= count
		state.Bank[color] += count
	}
	state.PendingReturnID = ""
	state.PendingReturnCount = 0
	state.Log = append(state.Log, fmt.Sprintf("%s 님이 초과 토큰을 반납했습니다.", action.PlayerID))
	if state.PendingNobleID != "" {
		return state, nil
	}
	return completeTurn(state, state.CurrentPlayerIndex), nil
}

func validateChooseNoble(state State, action gamecore.Action) error {
	nobleID, err := chooseNoblePayload(action.Payload)
	if err != nil {
		return err
	}
	for _, noble := range state.PendingNobles {
		if noble.ID == nobleID {
			return nil
		}
	}
	return errors.New("invalid noble choice")
}

func applyChooseNoble(state State, action gamecore.Action) (State, error) {
	if err := validateChooseNoble(state, action); err != nil {
		return state, err
	}
	index := findPlayer(state, string(action.PlayerID))
	if index < 0 {
		return state, errors.New("player not found")
	}
	nobleID, err := chooseNoblePayload(action.Payload)
	if err != nil {
		return state, err
	}
	if !awardNoble(&state, index, nobleID) {
		return state, errors.New("noble choice cannot be awarded")
	}
	return completeTurn(state, state.CurrentPlayerIndex), nil
}

func canAfford(player PlayerState, card Card) bool {
	gold := player.Tokens["gold"]
	for color, cost := range card.Cost {
		required := cost - player.Bonuses[color]
		if required < 0 {
			required = 0
		}
		colorTokens := player.Tokens[color]
		if colorTokens >= required {
			continue
		}
		gold -= required - colorTokens
		if gold < 0 {
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
		colorPayment := required
		if player.Tokens[color] < colorPayment {
			colorPayment = player.Tokens[color]
		}
		player.Tokens[color] -= colorPayment
		state.Bank[color] += colorPayment
		goldPayment := required - colorPayment
		if goldPayment > 0 {
			player.Tokens["gold"] -= goldPayment
			state.Bank["gold"] += goldPayment
		}
	}
}

func resolveNobleVisit(state *State, playerIndex int) {
	eligible := eligibleNobles(*state, playerIndex)
	switch len(eligible) {
	case 0:
		return
	case 1:
		awardNoble(state, playerIndex, eligible[0].ID)
	default:
		state.PendingNobleID = state.Players[playerIndex].PlayerID
		state.PendingNobles = cloneNobles(eligible)
	}
}

func eligibleNobles(state State, playerIndex int) []Noble {
	if playerIndex < 0 || playerIndex >= len(state.Players) {
		return nil
	}
	player := state.Players[playerIndex]
	eligible := []Noble{}
	for _, noble := range state.Nobles {
		if !canReceiveNoble(player, noble) {
			continue
		}
		eligible = append(eligible, noble)
	}
	return eligible
}

func awardNoble(state *State, playerIndex int, nobleID string) bool {
	if playerIndex < 0 || playerIndex >= len(state.Players) {
		return false
	}
	for nobleIndex, noble := range state.Nobles {
		if noble.ID != nobleID || !canReceiveNoble(state.Players[playerIndex], noble) {
			continue
		}
		state.Players[playerIndex].Nobles = append(state.Players[playerIndex].Nobles, noble)
		state.Players[playerIndex].Score += noble.Points
		state.Nobles = append(state.Nobles[:nobleIndex], state.Nobles[nobleIndex+1:]...)
		state.PendingNobleID = ""
		state.PendingNobles = nil
		state.Log = append(state.Log, fmt.Sprintf("%s 님에게 귀족이 방문했습니다.", state.Players[playerIndex].PlayerID))
		return true
	}
	return false
}

func canReceiveNoble(player PlayerState, noble Noble) bool {
	for color, required := range noble.Cost {
		if player.Bonuses[color] < required {
			return false
		}
	}
	return true
}

func selectedBuyCard(player PlayerState, state State, payload BuyPayload) (Card, bool) {
	if payload.ReservedIndex >= 0 {
		if payload.ReservedIndex >= len(player.Reserved) {
			return Card{}, false
		}
		return player.Reserved[payload.ReservedIndex], true
	}
	return selectedMarketCard(state, payload)
}

func selectedMarketCard(state State, payload BuyPayload) (Card, bool) {
	markets := state.Markets
	if len(markets) == 0 {
		markets = map[int][]Card{1: state.Market}
	}
	tier, index, ok := resolveMarketSelection(markets, payload)
	if !ok {
		return Card{}, false
	}
	return markets[tier][index], true
}

func removeAndRefillMarketCard(state *State, payload BuyPayload) (Card, bool) {
	ensureTieredMarket(state)
	tier, index, ok := resolveMarketSelection(state.Markets, payload)
	if !ok {
		return Card{}, false
	}
	card := state.Markets[tier][index]
	state.Markets[tier] = append(state.Markets[tier][:index], state.Markets[tier][index+1:]...)
	if len(state.Decks[tier]) > 0 {
		state.Markets[tier] = append(state.Markets[tier], state.Decks[tier][0])
		state.Decks[tier] = state.Decks[tier][1:]
	}
	syncLegacyMarket(state)
	return card, true
}

func canReserveFromDeck(state State, tier int) bool {
	return tier > 0 && len(state.Decks[tier]) > 0
}

func removeDeckTopCard(state *State, tier int) (Card, bool) {
	ensureTieredMarket(state)
	if !canReserveFromDeck(*state, tier) {
		return Card{}, false
	}
	card := state.Decks[tier][0]
	state.Decks[tier] = state.Decks[tier][1:]
	syncLegacyMarket(state)
	return card, true
}

func resolveMarketSelection(markets map[int][]Card, payload BuyPayload) (int, int, bool) {
	if payload.MarketIndex < 0 {
		return 0, 0, false
	}
	if payload.MarketTier > 0 {
		market := markets[payload.MarketTier]
		if payload.MarketIndex < len(market) {
			return payload.MarketTier, payload.MarketIndex, true
		}
		return 0, 0, false
	}
	return locateFlatMarketIndex(markets, payload.MarketIndex)
}

func locateFlatMarketIndex(markets map[int][]Card, flatIndex int) (int, int, bool) {
	if flatIndex < 0 {
		return 0, 0, false
	}
	offset := 0
	for _, tier := range splendorTiers {
		market := markets[tier]
		if flatIndex < offset+len(market) {
			return tier, flatIndex - offset, true
		}
		offset += len(market)
	}
	return 0, 0, false
}

func ensureTieredMarket(state *State) {
	if len(state.Markets) == 0 {
		state.Markets = map[int][]Card{1: cloneCards(state.Market)}
	}
	if len(state.Decks) == 0 {
		state.Decks = map[int][]Card{1: cloneCards(state.Deck)}
	}
	syncLegacyMarket(state)
}

func syncLegacyMarket(state *State) {
	state.Market = flattenMarkets(state.Markets)
	state.Deck = flattenDecks(state.Decks)
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

func tokenPayload(payload any) ([]string, error) {
	raw, ok := payload.(map[string]any)
	if !ok {
		return nil, errors.New("invalid token payload")
	}
	if rawColors, ok := raw["colors"]; ok {
		colors := colorListFromAny(rawColors)
		if len(colors) == 0 {
			return nil, errors.New("invalid token colors")
		}
		return colors, nil
	}
	color, _ := raw["color"].(string)
	if !validColor(color) {
		return nil, errors.New("invalid token color")
	}
	return []string{color}, nil
}

func colorListFromAny(value any) []string {
	switch typed := value.(type) {
	case []string:
		return append([]string(nil), typed...)
	case []any:
		colors := make([]string, 0, len(typed))
		for _, item := range typed {
			color, ok := item.(string)
			if !ok {
				return nil
			}
			colors = append(colors, color)
		}
		return colors
	default:
		return nil
	}
}

func validateTakeTokens(state State, colors []string) error {
	switch len(colors) {
	case 3:
		seen := map[string]bool{}
		for _, color := range colors {
			if !validColor(color) {
				return errors.New("invalid token color")
			}
			if seen[color] {
				return errors.New("must take three different colors")
			}
			seen[color] = true
			if state.Bank[color] <= 0 {
				return errors.New("token is not available")
			}
		}
	case 2:
		if !validColor(colors[0]) || colors[0] != colors[1] {
			return errors.New("taking two tokens requires the same color")
		}
		if state.Bank[colors[0]] < 4 {
			return errors.New("token is not available")
		}
	default:
		return errors.New("must take three different colors or two of the same color")
	}
	return nil
}

func reservePayload(payload any) (BuyPayload, error) {
	raw, ok := payload.(map[string]any)
	if !ok {
		return BuyPayload{}, errors.New("invalid reserve payload")
	}
	result := BuyPayload{MarketIndex: -1, ReservedIndex: -1}
	if fromDeck, ok := raw["fromDeck"].(bool); ok {
		result.FromDeck = fromDeck
	}
	if tier, ok := gameutil.Int(raw["marketTier"]); ok {
		result.MarketTier = tier
	}
	if result.FromDeck {
		if result.MarketTier <= 0 {
			return BuyPayload{}, errors.New("market tier is required")
		}
		return result, nil
	}
	value, ok := gameutil.Int(raw["marketIndex"])
	if !ok {
		return BuyPayload{}, errors.New("market index is required")
	}
	result.MarketIndex = value
	return result, nil
}

func buyPayload(payload any) (BuyPayload, error) {
	raw, ok := payload.(map[string]any)
	if !ok {
		return BuyPayload{}, errors.New("invalid buy payload")
	}
	result := BuyPayload{MarketIndex: -1, ReservedIndex: -1}
	if value, ok := gameutil.Int(raw["marketIndex"]); ok {
		result.MarketIndex = value
	}
	if value, ok := gameutil.Int(raw["marketTier"]); ok {
		result.MarketTier = value
	}
	if value, ok := gameutil.Int(raw["reservedIndex"]); ok {
		result.ReservedIndex = value
	}
	if (result.MarketIndex < 0 && result.ReservedIndex < 0) || (result.MarketIndex >= 0 && result.ReservedIndex >= 0) {
		return BuyPayload{}, errors.New("exactly one buy index is required")
	}
	return result, nil
}

func returnTokensPayload(payload any) (map[string]int, error) {
	raw, ok := payload.(map[string]any)
	if !ok {
		return nil, errors.New("invalid return payload")
	}
	rawTokens, ok := raw["tokens"].(map[string]any)
	if !ok {
		return nil, errors.New("return tokens are required")
	}
	tokens := map[string]int{}
	for color, value := range rawTokens {
		count, ok := gameutil.Int(value)
		if !ok {
			return nil, errors.New("invalid return token count")
		}
		if count > 0 {
			tokens[color] = count
		}
	}
	if len(tokens) == 0 {
		return nil, errors.New("return tokens are required")
	}
	return tokens, nil
}

func chooseNoblePayload(payload any) (string, error) {
	raw, ok := payload.(map[string]any)
	if !ok {
		return "", errors.New("invalid noble payload")
	}
	nobleID, _ := raw["nobleId"].(string)
	if nobleID == "" {
		return "", errors.New("noble id is required")
	}
	return nobleID, nil
}

func validColor(color string) bool {
	for _, item := range colors {
		if item == color {
			return true
		}
	}
	return false
}

func validTokenColor(color string) bool {
	return color == "gold" || validColor(color)
}

func emptyCounter() map[string]int {
	counter := map[string]int{}
	for _, color := range colors {
		counter[color] = 0
	}
	counter["gold"] = 0
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
	bank["gold"] = 5
	return bank
}

func startingNobles(seed string, playerCount int) []Noble {
	nobles := []Noble{
		{ID: "noble-1", Points: 3, Cost: map[string]int{"white": 4, "blue": 4}},
		{ID: "noble-2", Points: 3, Cost: map[string]int{"blue": 4, "green": 4}},
		{ID: "noble-3", Points: 3, Cost: map[string]int{"green": 4, "red": 4}},
		{ID: "noble-4", Points: 3, Cost: map[string]int{"red": 4, "black": 4}},
		{ID: "noble-5", Points: 3, Cost: map[string]int{"black": 4, "white": 4}},
		{ID: "noble-6", Points: 3, Cost: map[string]int{"white": 3, "blue": 3, "green": 3}},
		{ID: "noble-7", Points: 3, Cost: map[string]int{"blue": 3, "green": 3, "red": 3}},
		{ID: "noble-8", Points: 3, Cost: map[string]int{"green": 3, "red": 3, "black": 3}},
		{ID: "noble-9", Points: 3, Cost: map[string]int{"red": 3, "black": 3, "white": 3}},
		{ID: "noble-10", Points: 3, Cost: map[string]int{"black": 3, "white": 3, "blue": 3}},
	}
	random := rand.New(rand.NewSource(seedToInt(seed + ":nobles")))
	random.Shuffle(len(nobles), func(i, j int) {
		nobles[i], nobles[j] = nobles[j], nobles[i]
	})
	count := playerCount + 1
	if count > len(nobles) {
		count = len(nobles)
	}
	return cloneNobles(nobles[:count])
}

func cloneState(state State) State {
	clone := state
	clone.Bank = copyCounter(state.Bank)
	clone.Market = cloneCards(state.Market)
	clone.Deck = cloneCards(state.Deck)
	clone.Markets = cloneCardMap(state.Markets)
	clone.Decks = cloneCardMap(state.Decks)
	clone.Nobles = cloneNobles(state.Nobles)
	clone.PendingNobles = cloneNobles(state.PendingNobles)
	clone.Players = make([]PlayerState, len(state.Players))
	for index := range state.Players {
		clone.Players[index] = state.Players[index]
		clone.Players[index].Tokens = copyCounter(state.Players[index].Tokens)
		clone.Players[index].Bonuses = copyCounter(state.Players[index].Bonuses)
		clone.Players[index].Cards = cloneCards(state.Players[index].Cards)
		clone.Players[index].Reserved = cloneCards(state.Players[index].Reserved)
		clone.Players[index].Nobles = cloneNobles(state.Players[index].Nobles)
	}
	clone.Log = append([]string(nil), state.Log...)
	return clone
}

func cloneCardMap(cardsByTier map[int][]Card) map[int][]Card {
	if cardsByTier == nil {
		return nil
	}
	result := make(map[int][]Card, len(cardsByTier))
	for tier, cards := range cardsByTier {
		result[tier] = cloneCards(cards)
	}
	return result
}

func cloneCards(cards []Card) []Card {
	result := make([]Card, len(cards))
	for index := range cards {
		result[index] = cards[index]
		result[index].Cost = copyCounter(cards[index].Cost)
	}
	return result
}

func maskDecks(decks map[int][]Card) map[int][]Card {
	masked := make(map[int][]Card, len(decks))
	for tier, deck := range decks {
		masked[tier] = make([]Card, len(deck))
	}
	return masked
}

func maskReservedCards(state *State, viewerID string) {
	for playerIndex := range state.Players {
		player := &state.Players[playerIndex]
		if viewerID != "" && player.PlayerID == viewerID {
			continue
		}
		player.Reserved = make([]Card, len(player.Reserved))
		for cardIndex := range player.Reserved {
			player.Reserved[cardIndex] = Card{
				ID:     fmt.Sprintf("hidden-reserved-%s-%d", player.PlayerID, cardIndex),
				Hidden: true,
			}
		}
	}
}

func cloneNobles(nobles []Noble) []Noble {
	result := make([]Noble, len(nobles))
	for index := range nobles {
		result[index] = nobles[index]
		result[index].Cost = copyCounter(nobles[index].Cost)
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

func shuffledDecks(seed string) map[int][]Card {
	decks := map[int][]Card{}
	for _, card := range fullDeck() {
		decks[card.Tier] = append(decks[card.Tier], card)
	}
	for _, tier := range splendorTiers {
		deck := decks[tier]
		random := rand.New(rand.NewSource(seedToInt(fmt.Sprintf("%s:tier:%d", seed, tier))))
		random.Shuffle(len(deck), func(i, j int) {
			deck[i], deck[j] = deck[j], deck[i]
		})
		decks[tier] = deck
	}
	return decks
}

func startingMarkets(decks map[int][]Card) map[int][]Card {
	markets := map[int][]Card{}
	for _, tier := range splendorTiers {
		count := 4
		if len(decks[tier]) < count {
			count = len(decks[tier])
		}
		markets[tier] = cloneCards(decks[tier][:count])
		decks[tier] = decks[tier][count:]
	}
	return markets
}

func flattenMarkets(markets map[int][]Card) []Card {
	return flattenCardMap(markets)
}

func flattenDecks(decks map[int][]Card) []Card {
	return flattenCardMap(decks)
}

func flattenCardMap(cardsByTier map[int][]Card) []Card {
	total := 0
	for _, cards := range cardsByTier {
		total += len(cards)
	}
	result := make([]Card, 0, total)
	for _, tier := range splendorTiers {
		result = append(result, cardsByTier[tier]...)
	}
	return cloneCards(result)
}

func fullDeck() []Card {
	return []Card{
		{ID: "splendor-1", Tier: 1, Color: "white", Points: 0, Cost: map[string]int{"blue": 3}},
		{ID: "splendor-2", Tier: 1, Color: "white", Points: 0, Cost: map[string]int{"red": 2, "black": 1}},
		{ID: "splendor-3", Tier: 1, Color: "white", Points: 0, Cost: map[string]int{"blue": 1, "green": 1, "red": 1, "black": 1}},
		{ID: "splendor-4", Tier: 1, Color: "white", Points: 0, Cost: map[string]int{"blue": 2, "black": 2}},
		{ID: "splendor-5", Tier: 1, Color: "white", Points: 1, Cost: map[string]int{"green": 4}},
		{ID: "splendor-6", Tier: 1, Color: "white", Points: 0, Cost: map[string]int{"blue": 1, "green": 2, "red": 1, "black": 1}},
		{ID: "splendor-7", Tier: 1, Color: "white", Points: 0, Cost: map[string]int{"blue": 2, "green": 2, "black": 1}},
		{ID: "splendor-8", Tier: 1, Color: "white", Points: 0, Cost: map[string]int{"white": 3, "blue": 1, "black": 1}},
		{ID: "splendor-9", Tier: 1, Color: "blue", Points: 0, Cost: map[string]int{"white": 1, "black": 2}},
		{ID: "splendor-10", Tier: 1, Color: "blue", Points: 0, Cost: map[string]int{"black": 3}},
		{ID: "splendor-11", Tier: 1, Color: "blue", Points: 0, Cost: map[string]int{"white": 1, "green": 1, "red": 1, "black": 1}},
		{ID: "splendor-12", Tier: 1, Color: "blue", Points: 0, Cost: map[string]int{"green": 2, "black": 2}},
		{ID: "splendor-13", Tier: 1, Color: "blue", Points: 1, Cost: map[string]int{"red": 4}},
		{ID: "splendor-14", Tier: 1, Color: "blue", Points: 0, Cost: map[string]int{"white": 1, "green": 1, "red": 2, "black": 1}},
		{ID: "splendor-15", Tier: 1, Color: "blue", Points: 0, Cost: map[string]int{"white": 1, "green": 2, "red": 2}},
		{ID: "splendor-16", Tier: 1, Color: "blue", Points: 0, Cost: map[string]int{"blue": 1, "green": 3, "red": 1}},
		{ID: "splendor-17", Tier: 1, Color: "green", Points: 0, Cost: map[string]int{"white": 2, "blue": 1}},
		{ID: "splendor-18", Tier: 1, Color: "green", Points: 0, Cost: map[string]int{"red": 3}},
		{ID: "splendor-19", Tier: 1, Color: "green", Points: 0, Cost: map[string]int{"white": 1, "blue": 1, "red": 1, "black": 1}},
		{ID: "splendor-20", Tier: 1, Color: "green", Points: 0, Cost: map[string]int{"blue": 2, "red": 2}},
		{ID: "splendor-21", Tier: 1, Color: "green", Points: 1, Cost: map[string]int{"black": 4}},
		{ID: "splendor-22", Tier: 1, Color: "green", Points: 0, Cost: map[string]int{"white": 1, "blue": 1, "red": 1, "black": 2}},
		{ID: "splendor-23", Tier: 1, Color: "green", Points: 0, Cost: map[string]int{"blue": 1, "red": 2, "black": 2}},
		{ID: "splendor-24", Tier: 1, Color: "green", Points: 0, Cost: map[string]int{"white": 1, "blue": 3, "green": 1}},
		{ID: "splendor-25", Tier: 1, Color: "red", Points: 0, Cost: map[string]int{"blue": 2, "green": 1}},
		{ID: "splendor-26", Tier: 1, Color: "red", Points: 0, Cost: map[string]int{"white": 3}},
		{ID: "splendor-27", Tier: 1, Color: "red", Points: 0, Cost: map[string]int{"white": 1, "blue": 1, "green": 1, "black": 1}},
		{ID: "splendor-28", Tier: 1, Color: "red", Points: 0, Cost: map[string]int{"white": 2, "red": 2}},
		{ID: "splendor-29", Tier: 1, Color: "red", Points: 1, Cost: map[string]int{"white": 4}},
		{ID: "splendor-30", Tier: 1, Color: "red", Points: 0, Cost: map[string]int{"white": 2, "blue": 1, "green": 1, "black": 1}},
		{ID: "splendor-31", Tier: 1, Color: "red", Points: 0, Cost: map[string]int{"white": 2, "green": 1, "black": 2}},
		{ID: "splendor-32", Tier: 1, Color: "red", Points: 0, Cost: map[string]int{"white": 1, "red": 1, "black": 3}},
		{ID: "splendor-33", Tier: 1, Color: "black", Points: 0, Cost: map[string]int{"green": 2, "red": 1}},
		{ID: "splendor-34", Tier: 1, Color: "black", Points: 0, Cost: map[string]int{"green": 3}},
		{ID: "splendor-35", Tier: 1, Color: "black", Points: 0, Cost: map[string]int{"white": 1, "blue": 1, "green": 1, "red": 1}},
		{ID: "splendor-36", Tier: 1, Color: "black", Points: 0, Cost: map[string]int{"white": 2, "green": 2}},
		{ID: "splendor-37", Tier: 1, Color: "black", Points: 1, Cost: map[string]int{"blue": 4}},
		{ID: "splendor-38", Tier: 1, Color: "black", Points: 0, Cost: map[string]int{"white": 1, "blue": 2, "green": 1, "red": 1}},
		{ID: "splendor-39", Tier: 1, Color: "black", Points: 0, Cost: map[string]int{"white": 2, "blue": 2, "red": 1}},
		{ID: "splendor-40", Tier: 1, Color: "black", Points: 0, Cost: map[string]int{"green": 1, "red": 3, "black": 1}},
		{ID: "splendor-41", Tier: 2, Color: "white", Points: 2, Cost: map[string]int{"red": 5}},
		{ID: "splendor-42", Tier: 2, Color: "white", Points: 3, Cost: map[string]int{"white": 6}},
		{ID: "splendor-43", Tier: 2, Color: "white", Points: 1, Cost: map[string]int{"green": 3, "red": 2, "black": 2}},
		{ID: "splendor-44", Tier: 2, Color: "white", Points: 2, Cost: map[string]int{"green": 1, "red": 4, "black": 2}},
		{ID: "splendor-45", Tier: 2, Color: "white", Points: 1, Cost: map[string]int{"white": 2, "blue": 3, "red": 3}},
		{ID: "splendor-46", Tier: 2, Color: "white", Points: 2, Cost: map[string]int{"red": 5, "black": 3}},
		{ID: "splendor-47", Tier: 2, Color: "blue", Points: 2, Cost: map[string]int{"blue": 5}},
		{ID: "splendor-48", Tier: 2, Color: "blue", Points: 3, Cost: map[string]int{"blue": 6}},
		{ID: "splendor-49", Tier: 2, Color: "blue", Points: 1, Cost: map[string]int{"blue": 2, "green": 2, "red": 3}},
		{ID: "splendor-50", Tier: 2, Color: "blue", Points: 2, Cost: map[string]int{"white": 2, "red": 1, "black": 4}},
		{ID: "splendor-51", Tier: 2, Color: "blue", Points: 1, Cost: map[string]int{"blue": 2, "green": 3, "black": 3}},
		{ID: "splendor-52", Tier: 2, Color: "blue", Points: 2, Cost: map[string]int{"white": 5, "blue": 3}},
		{ID: "splendor-53", Tier: 2, Color: "green", Points: 2, Cost: map[string]int{"green": 5}},
		{ID: "splendor-54", Tier: 2, Color: "green", Points: 3, Cost: map[string]int{"green": 6}},
		{ID: "splendor-55", Tier: 2, Color: "green", Points: 1, Cost: map[string]int{"white": 2, "blue": 3, "black": 2}},
		{ID: "splendor-56", Tier: 2, Color: "green", Points: 1, Cost: map[string]int{"white": 3, "green": 2, "red": 3}},
		{ID: "splendor-57", Tier: 2, Color: "green", Points: 2, Cost: map[string]int{"white": 4, "blue": 2, "black": 1}},
		{ID: "splendor-58", Tier: 2, Color: "green", Points: 2, Cost: map[string]int{"blue": 5, "green": 3}},
		{ID: "splendor-59", Tier: 2, Color: "red", Points: 2, Cost: map[string]int{"black": 5}},
		{ID: "splendor-60", Tier: 2, Color: "red", Points: 3, Cost: map[string]int{"red": 6}},
		{ID: "splendor-61", Tier: 2, Color: "red", Points: 1, Cost: map[string]int{"white": 2, "red": 2, "black": 3}},
		{ID: "splendor-62", Tier: 2, Color: "red", Points: 2, Cost: map[string]int{"white": 1, "blue": 4, "green": 2}},
		{ID: "splendor-63", Tier: 2, Color: "red", Points: 1, Cost: map[string]int{"blue": 3, "red": 2, "black": 3}},
		{ID: "splendor-64", Tier: 2, Color: "red", Points: 2, Cost: map[string]int{"white": 3, "black": 5}},
		{ID: "splendor-65", Tier: 2, Color: "black", Points: 2, Cost: map[string]int{"white": 5}},
		{ID: "splendor-66", Tier: 2, Color: "black", Points: 3, Cost: map[string]int{"black": 6}},
		{ID: "splendor-67", Tier: 2, Color: "black", Points: 1, Cost: map[string]int{"white": 3, "blue": 2, "green": 2}},
		{ID: "splendor-68", Tier: 2, Color: "black", Points: 2, Cost: map[string]int{"blue": 1, "green": 4, "red": 2}},
		{ID: "splendor-69", Tier: 2, Color: "black", Points: 1, Cost: map[string]int{"white": 3, "green": 3, "black": 2}},
		{ID: "splendor-70", Tier: 2, Color: "black", Points: 2, Cost: map[string]int{"green": 5, "red": 3}},
		{ID: "splendor-71", Tier: 3, Color: "white", Points: 4, Cost: map[string]int{"black": 7}},
		{ID: "splendor-72", Tier: 3, Color: "white", Points: 5, Cost: map[string]int{"white": 3, "black": 7}},
		{ID: "splendor-73", Tier: 3, Color: "white", Points: 4, Cost: map[string]int{"white": 3, "red": 3, "black": 6}},
		{ID: "splendor-74", Tier: 3, Color: "white", Points: 3, Cost: map[string]int{"blue": 3, "green": 3, "red": 5, "black": 3}},
		{ID: "splendor-75", Tier: 3, Color: "blue", Points: 4, Cost: map[string]int{"white": 7}},
		{ID: "splendor-76", Tier: 3, Color: "blue", Points: 5, Cost: map[string]int{"white": 7, "blue": 3}},
		{ID: "splendor-77", Tier: 3, Color: "blue", Points: 4, Cost: map[string]int{"white": 6, "blue": 3, "black": 3}},
		{ID: "splendor-78", Tier: 3, Color: "blue", Points: 3, Cost: map[string]int{"white": 3, "green": 3, "red": 3, "black": 5}},
		{ID: "splendor-79", Tier: 3, Color: "green", Points: 4, Cost: map[string]int{"blue": 7}},
		{ID: "splendor-80", Tier: 3, Color: "green", Points: 5, Cost: map[string]int{"blue": 7, "green": 3}},
		{ID: "splendor-81", Tier: 3, Color: "green", Points: 4, Cost: map[string]int{"white": 3, "blue": 6, "green": 3}},
		{ID: "splendor-82", Tier: 3, Color: "green", Points: 3, Cost: map[string]int{"white": 5, "blue": 3, "red": 3, "black": 3}},
		{ID: "splendor-83", Tier: 3, Color: "red", Points: 4, Cost: map[string]int{"green": 7}},
		{ID: "splendor-84", Tier: 3, Color: "red", Points: 5, Cost: map[string]int{"green": 7, "red": 3}},
		{ID: "splendor-85", Tier: 3, Color: "red", Points: 4, Cost: map[string]int{"blue": 3, "green": 6, "red": 3}},
		{ID: "splendor-86", Tier: 3, Color: "red", Points: 3, Cost: map[string]int{"white": 3, "blue": 5, "green": 3, "black": 3}},
		{ID: "splendor-87", Tier: 3, Color: "black", Points: 4, Cost: map[string]int{"red": 7}},
		{ID: "splendor-88", Tier: 3, Color: "black", Points: 5, Cost: map[string]int{"red": 7, "black": 3}},
		{ID: "splendor-89", Tier: 3, Color: "black", Points: 4, Cost: map[string]int{"green": 3, "red": 6, "black": 3}},
		{ID: "splendor-90", Tier: 3, Color: "black", Points: 3, Cost: map[string]int{"white": 3, "blue": 3, "green": 5, "red": 3}},
	}
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
