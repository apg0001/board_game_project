package dalmuti

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
	ActionPlay = "dalmuti.play"
	ActionPass = "dalmuti.pass"
)

type Card struct {
	ID   string `json:"id"`
	Rank int    `json:"rank"`
}

type PlayerState struct {
	PlayerID string `json:"playerId"`
	Hand     []Card `json:"hand"`
	HandSize int    `json:"handSize"`
	Passed   bool   `json:"passed"`
	Out      bool   `json:"out"`
	Active   bool   `json:"active"`
}

type Trick struct {
	Rank     int    `json:"rank,omitempty"`
	Count    int    `json:"count,omitempty"`
	PlayerID string `json:"playerId,omitempty"`
}

type State struct {
	CurrentPlayerIndex int           `json:"currentPlayerIndex"`
	Round              int           `json:"round"`
	Players            []PlayerState `json:"players"`
	CurrentTrick       Trick         `json:"currentTrick"`
	FinishOrder        []string      `json:"finishOrder"`
	TaxApplied         bool          `json:"taxApplied"`
	Revolution         bool          `json:"revolution"`
	GreaterRevolution  bool          `json:"greaterRevolution"`
	Log                []string      `json:"log"`
	Finished           bool          `json:"finished"`
}

type PlayPayload struct {
	Rank  int `json:"rank"`
	Count int `json:"count"`
}

type Module struct{}

func NewModule() Module {
	return Module{}
}

func (m Module) ID() gamecore.GameID {
	return "dalmuti"
}

func (m Module) Name() string {
	return "위대한 달무티"
}

func (m Module) MinPlayers() int {
	return 4
}

func (m Module) MaxPlayers() int {
	return 8
}

func (m Module) CreateInitialState(ctx gamecore.Context) any {
	deck := shuffledDeck(ctx.RandomSeed)
	players := make([]PlayerState, 0, len(ctx.Players))
	for index, player := range ctx.Players {
		hand := []Card{}
		for cardIndex := index; cardIndex < len(deck); cardIndex += len(ctx.Players) {
			hand = append(hand, deck[cardIndex])
		}
		sortHand(hand)
		players = append(players, PlayerState{
			PlayerID: string(player.ID),
			Hand:     hand,
			HandSize: len(hand),
			Active:   true,
		})
	}
	state := State{
		CurrentPlayerIndex: 0,
		Round:              1,
		Players:            players,
		Log:                []string{"위대한 달무티가 시작되었습니다."},
	}
	return applyOpeningTaxAndRevolution(state)
}

func (m Module) PublicState(state any, viewerID gamecore.PlayerID) any {
	current := cloneState(asState(state))
	for index := range current.Players {
		current.Players[index].HandSize = len(current.Players[index].Hand)
		if current.Players[index].PlayerID != string(viewerID) {
			current.Players[index].Hand = []Card{}
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
	if current.Players[current.CurrentPlayerIndex].Out {
		return errors.New("player is already out")
	}

	switch action.Type {
	case ActionPass:
		if current.CurrentTrick.Count == 0 {
			return errors.New("leader cannot pass")
		}
	case ActionPlay:
		payload, err := playPayload(action.Payload)
		if err != nil {
			return err
		}
		if err := validatePlay(current.Players[current.CurrentPlayerIndex], current.CurrentTrick, payload); err != nil {
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
	case ActionPass:
		player.Passed = true
		current.Log = append(current.Log, player.PlayerID+" 님이 패스했습니다.")
	case ActionPlay:
		payload, err := playPayload(action.Payload)
		if err != nil {
			return gamecore.ActionResult{}, err
		}
		playCards(player, payload)
		resetPasses(&current)
		current.CurrentTrick = Trick{Rank: payload.Rank, Count: payload.Count, PlayerID: player.PlayerID}
		current.Log = append(current.Log, fmt.Sprintf("%s 님이 %d 계급 %d장을 냈습니다.", player.PlayerID, payload.Rank, payload.Count))
		if len(player.Hand) == 0 {
			player.Out = true
			player.Active = false
			current.FinishOrder = append(current.FinishOrder, player.PlayerID)
			current.Log = append(current.Log, player.PlayerID+" 님이 모든 카드를 털었습니다.")
		}
	}

	if remainingPlayers(current) <= 1 {
		for _, player := range current.Players {
			if !player.Out {
				current.FinishOrder = append(current.FinishOrder, player.PlayerID)
			}
		}
		current.Finished = true
		current.Log = append(current.Log, "위대한 달무티 한 판이 종료되었습니다.")
		return m.result(current)
	}

	if trickComplete(current) {
		current = resetTrick(current)
	} else {
		current.CurrentPlayerIndex = nextPlayableIndex(current, current.CurrentPlayerIndex)
		current.Round++
	}
	return m.result(current)
}

func (m Module) ApplyTimeout(_ context.Context, state any, playerID gamecore.PlayerID, _ gamecore.Context) (gamecore.ActionResult, error) {
	current := asState(state)
	index := findPlayer(current, string(playerID))
	if index < 0 || current.Finished {
		return gamecore.ActionResult{State: current}, nil
	}
	current.Players[index].Out = true
	current.Players[index].Active = false
	current.Players[index].Hand = []Card{}
	current.Log = append(current.Log, string(playerID)+" 님의 재접속 시간이 만료되어 자동 기권 처리되었습니다.")
	if remainingPlayers(current) <= 1 {
		for _, player := range current.Players {
			if !contains(current.FinishOrder, player.PlayerID) {
				current.FinishOrder = append(current.FinishOrder, player.PlayerID)
			}
		}
		current.Finished = true
	} else if current.CurrentPlayerIndex == index {
		current.CurrentPlayerIndex = nextPlayableIndex(current, index)
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
	results := make([]gamecore.Result, 0, len(current.FinishOrder))
	for index, playerID := range current.FinishOrder {
		outcome := gamecore.OutcomeLose
		if index == 0 {
			outcome = gamecore.OutcomeWin
		}
		results = append(results, gamecore.Result{
			PlayerID: gamecore.PlayerID(playerID),
			Rank:     index + 1,
			Score:    len(current.FinishOrder) - index,
			Outcome:  outcome,
		})
	}
	return results
}

func (m Module) result(current State) (gamecore.ActionResult, error) {
	return gamecore.ActionResult{
		State: current,
		Events: []gamecore.Event{{
			Type:       "game.state_updated",
			Visibility: gamecore.VisibilityPublic,
			Payload:    m.PublicState(current, ""),
		}},
	}, nil
}

func validatePlay(player PlayerState, trick Trick, payload PlayPayload) error {
	if payload.Count <= 0 {
		return errors.New("card count is required")
	}
	if payload.Rank < 1 || payload.Rank > 13 {
		return errors.New("rank out of range")
	}
	if payload.Rank == 13 && payload.Count != 1 {
		return errors.New("jester can only be played alone")
	}
	if trick.Count > 0 {
		if payload.Count != trick.Count {
			return errors.New("must play the same number of cards")
		}
		if payload.Rank >= trick.Rank {
			return errors.New("must play a stronger rank")
		}
	}
	if !hasPlayableCards(player.Hand, payload) {
		return errors.New("selected cards are not available")
	}
	return nil
}

func hasPlayableCards(hand []Card, payload PlayPayload) bool {
	if payload.Rank == 13 {
		return countRank(hand, 13) >= 1
	}
	return countRank(hand, payload.Rank)+countRank(hand, 13) >= payload.Count
}

func playCards(player *PlayerState, payload PlayPayload) {
	remaining := payload.Count
	next := make([]Card, 0, len(player.Hand)-payload.Count)
	for _, card := range player.Hand {
		if remaining > 0 && card.Rank == payload.Rank {
			remaining--
			continue
		}
		next = append(next, card)
	}
	if payload.Rank != 13 {
		filtered := next[:0]
		for _, card := range next {
			if remaining > 0 && card.Rank == 13 {
				remaining--
				continue
			}
			filtered = append(filtered, card)
		}
		next = filtered
	}
	player.Hand = next
	player.HandSize = len(next)
}

func applyOpeningTaxAndRevolution(state State) State {
	if len(state.Players) < 4 {
		return state
	}
	greaterPeonIndex := len(state.Players) - 1
	jesterHolder := playerWithBothJesters(state.Players)
	if jesterHolder >= 0 {
		state.Revolution = true
		if jesterHolder == greaterPeonIndex {
			state.GreaterRevolution = true
			reversePlayers(state.Players)
			state.Log = append(state.Log, "큰 하인이 대혁명을 선언해 계급 순서가 뒤집혔습니다.")
		} else {
			state.Log = append(state.Log, "혁명이 선언되어 세금이 취소되었습니다.")
		}
		return state
	}

	exchangeTax(&state, len(state.Players)-1, 0, 2)
	exchangeTax(&state, len(state.Players)-2, 1, 1)
	state.TaxApplied = true
	state.Log = append(state.Log, "세금 교환이 적용되었습니다.")
	return state
}

func exchangeTax(state *State, peonIndex int, dalmutiIndex int, count int) {
	if peonIndex < 0 || peonIndex >= len(state.Players) || dalmutiIndex < 0 || dalmutiIndex >= len(state.Players) {
		return
	}
	fromPeon := takeBestCards(&state.Players[peonIndex], count)
	fromDalmuti := takeWorstCards(&state.Players[dalmutiIndex], count)
	state.Players[peonIndex].Hand = append(state.Players[peonIndex].Hand, fromDalmuti...)
	state.Players[dalmutiIndex].Hand = append(state.Players[dalmutiIndex].Hand, fromPeon...)
	sortHand(state.Players[peonIndex].Hand)
	sortHand(state.Players[dalmutiIndex].Hand)
	state.Players[peonIndex].HandSize = len(state.Players[peonIndex].Hand)
	state.Players[dalmutiIndex].HandSize = len(state.Players[dalmutiIndex].Hand)
}

func takeBestCards(player *PlayerState, count int) []Card {
	sortHand(player.Hand)
	return takeCardsAt(player, count, func(index int) int { return index })
}

func takeWorstCards(player *PlayerState, count int) []Card {
	sortHand(player.Hand)
	return takeCardsAt(player, count, func(index int) int { return len(player.Hand) - 1 - index })
}

func takeCardsAt(player *PlayerState, count int, pickIndex func(int) int) []Card {
	if count <= 0 || len(player.Hand) == 0 {
		return nil
	}
	selected := make([]Card, 0, count)
	remove := map[int]bool{}
	for step := 0; step < count && step < len(player.Hand); step++ {
		index := pickIndex(step)
		selected = append(selected, player.Hand[index])
		remove[index] = true
	}
	next := make([]Card, 0, len(player.Hand)-len(remove))
	for index, card := range player.Hand {
		if !remove[index] {
			next = append(next, card)
		}
	}
	player.Hand = next
	player.HandSize = len(next)
	return selected
}

func playerWithBothJesters(players []PlayerState) int {
	for index, player := range players {
		if countRank(player.Hand, 13) >= 2 {
			return index
		}
	}
	return -1
}

func reversePlayers(players []PlayerState) {
	for left, right := 0, len(players)-1; left < right; left, right = left+1, right-1 {
		players[left], players[right] = players[right], players[left]
	}
}

func trickComplete(state State) bool {
	needed := 0
	passed := 0
	for _, player := range state.Players {
		if player.Out {
			continue
		}
		needed++
		if player.Passed || player.PlayerID == state.CurrentTrick.PlayerID {
			passed++
		}
	}
	return needed > 1 && passed >= needed
}

func resetTrick(state State) State {
	leader := findPlayer(state, state.CurrentTrick.PlayerID)
	state.CurrentTrick = Trick{}
	for index := range state.Players {
		state.Players[index].Passed = false
	}
	if leader < 0 || state.Players[leader].Out {
		state.CurrentPlayerIndex = nextPlayableIndex(state, state.CurrentPlayerIndex)
	} else {
		state.CurrentPlayerIndex = leader
	}
	state.Round++
	state.Log = append(state.Log, "새 트릭이 시작되었습니다.")
	return state
}

func resetPasses(state *State) {
	for index := range state.Players {
		state.Players[index].Passed = false
	}
}

func remainingPlayers(state State) int {
	count := 0
	for _, player := range state.Players {
		if !player.Out {
			count++
		}
	}
	return count
}

func nextPlayableIndex(state State, current int) int {
	for step := 1; step <= len(state.Players); step++ {
		next := (current + step) % len(state.Players)
		if !state.Players[next].Out {
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

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func countRank(hand []Card, rank int) int {
	count := 0
	for _, card := range hand {
		if card.Rank == rank {
			count++
		}
	}
	return count
}

func playPayload(payload any) (PlayPayload, error) {
	raw, ok := payload.(map[string]any)
	if !ok {
		return PlayPayload{}, errors.New("invalid play payload")
	}
	payloadRank, ok := gameutil.Int(raw["rank"])
	if !ok {
		return PlayPayload{}, errors.New("rank is required")
	}
	payloadCount, ok := gameutil.Int(raw["count"])
	if !ok {
		return PlayPayload{}, errors.New("count is required")
	}
	return PlayPayload{Rank: payloadRank, Count: payloadCount}, nil
}

func cloneState(state State) State {
	clone := state
	clone.Players = make([]PlayerState, len(state.Players))
	for index := range state.Players {
		clone.Players[index] = state.Players[index]
		clone.Players[index].Hand = append([]Card(nil), state.Players[index].Hand...)
	}
	clone.FinishOrder = append([]string(nil), state.FinishOrder...)
	clone.Log = append([]string(nil), state.Log...)
	return clone
}

func shuffledDeck(seed string) []Card {
	deck := make([]Card, 0, 80)
	for rank := 1; rank <= 12; rank++ {
		for copyIndex := 1; copyIndex <= rank; copyIndex++ {
			deck = append(deck, Card{ID: fmt.Sprintf("%d-%d", rank, copyIndex), Rank: rank})
		}
	}
	deck = append(deck, Card{ID: "jester-1", Rank: 13}, Card{ID: "jester-2", Rank: 13})
	random := rand.New(rand.NewSource(seedToInt(seed)))
	random.Shuffle(len(deck), func(i, j int) {
		deck[i], deck[j] = deck[j], deck[i]
	})
	return deck
}

func sortHand(hand []Card) {
	sort.SliceStable(hand, func(i, j int) bool {
		return hand[i].Rank < hand[j].Rank
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
