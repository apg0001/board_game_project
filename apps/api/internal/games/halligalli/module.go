package halligalli

import (
	"context"
	"errors"
	"hash/fnv"
	"math/rand"

	"board-game-platform/apps/api/internal/gamecore"
)

const (
	ActionFlip = "halli-galli.flip"
	ActionRing = "halli-galli.ring"
)

type Card struct {
	Fruit string `json:"fruit"`
	Count int    `json:"count"`
}

type PlayerState struct {
	PlayerID  string `json:"playerId"`
	Deck      []Card `json:"deck"`
	FaceUp    []Card `json:"faceUp"`
	Score     int    `json:"score"`
	Active    bool   `json:"active"`
	Forfeited bool   `json:"forfeited"`
}

type State struct {
	CurrentPlayerIndex int             `json:"currentPlayerIndex"`
	Round              int             `json:"round"`
	Players            []PlayerState   `json:"players"`
	BellSettled        bool            `json:"bellSettled"`
	LastBellWinnerID   string          `json:"lastBellWinnerId,omitempty"`
	RingAttempts       map[string]bool `json:"ringAttempts,omitempty"`
	Log                []string        `json:"log"`
	Finished           bool            `json:"finished"`
}

type Module struct{}

func NewModule() Module {
	return Module{}
}

func (m Module) ID() gamecore.GameID {
	return "halli-galli"
}

func (m Module) Name() string {
	return "할리갈리"
}

func (m Module) MinPlayers() int {
	return 2
}

func (m Module) MaxPlayers() int {
	return 6
}

func (m Module) CreateInitialState(ctx gamecore.Context) any {
	deck := shuffledDeck(ctx.RandomSeed)
	players := make([]PlayerState, 0, len(ctx.Players))
	for index, player := range ctx.Players {
		hand := make([]Card, 0)
		for cardIndex := index; cardIndex < len(deck); cardIndex += len(ctx.Players) {
			hand = append(hand, deck[cardIndex])
		}
		players = append(players, PlayerState{
			PlayerID: string(player.ID),
			Deck:     hand,
			FaceUp:   []Card{},
			Score:    0,
			Active:   len(hand) > 0,
		})
	}
	return State{
		CurrentPlayerIndex: 0,
		Round:              1,
		Players:            players,
		RingAttempts:       map[string]bool{},
		Log:                []string{"할리갈리가 시작되었습니다."},
	}
}

func (m Module) PublicState(state any, _ gamecore.PlayerID) any {
	current := cloneState(asState(state))
	for index := range current.Players {
		current.Players[index].Deck = make([]Card, len(current.Players[index].Deck))
	}
	return current
}

func (m Module) ValidateAction(_ context.Context, state any, action gamecore.Action, _ gamecore.Context) error {
	current := asState(state)
	if current.Finished {
		return errors.New("game is already finished")
	}
	switch action.Type {
	case ActionFlip:
		if len(current.Players) == 0 || current.Players[current.CurrentPlayerIndex].PlayerID != string(action.PlayerID) {
			return errors.New("not your turn")
		}
		if len(current.Players[current.CurrentPlayerIndex].Deck) == 0 {
			return errors.New("player has no cards to flip")
		}
	case ActionRing:
		if current.BellSettled {
			return errors.New("bell is already resolved for the current table")
		}
		index := findPlayer(current, string(action.PlayerID))
		if index < 0 {
			return errors.New("player not found")
		}
		if current.Players[index].Forfeited || !current.Players[index].Active || totalCards(current.Players[index]) == 0 {
			return errors.New("player cannot ring")
		}
		if current.RingAttempts[string(action.PlayerID)] {
			return errors.New("player already rang for the current table")
		}
		return nil
	default:
		return errors.New("unsupported action")
	}
	return nil
}

func (m Module) ApplyAction(_ context.Context, state any, action gamecore.Action, _ gamecore.Context) (gamecore.ActionResult, error) {
	current := asState(state)
	switch action.Type {
	case ActionFlip:
		current = flip(current, string(action.PlayerID))
	case ActionRing:
		current = ring(current, string(action.PlayerID))
	}
	current = refresh(current)
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

	current.Players[index].Deck = []Card{}
	current.Players[index].Active = false
	current.Players[index].Forfeited = true
	current.Log = append(current.Log, string(playerID)+" 님의 재접속 시간이 만료되어 자동 기권 처리되었습니다.")
	if current.CurrentPlayerIndex == index {
		current.CurrentPlayerIndex = nextActiveIndex(current, current.CurrentPlayerIndex)
		current.Round++
	}
	current = refresh(current)

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
	for index, player := range sortedPlayers(current.Players) {
		outcome := gamecore.OutcomeLose
		if index == 0 {
			outcome = gamecore.OutcomeWin
		}
		results = append(results, gamecore.Result{
			PlayerID: gamecore.PlayerID(player.PlayerID),
			Rank:     index + 1,
			Score:    totalCards(player),
			Outcome:  outcome,
		})
	}
	return results
}

func flip(state State, playerID string) State {
	index := findPlayer(state, playerID)
	if index < 0 || len(state.Players[index].Deck) == 0 {
		return state
	}
	card := state.Players[index].Deck[0]
	state.Players[index].Deck = state.Players[index].Deck[1:]
	state.Players[index].FaceUp = append(state.Players[index].FaceUp, card)
	state.BellSettled = false
	state.LastBellWinnerID = ""
	state.RingAttempts = map[string]bool{}
	state.Log = append(state.Log, playerID+" 님이 카드를 펼쳤습니다.")
	state.CurrentPlayerIndex = nextActiveIndex(state, state.CurrentPlayerIndex)
	state.Round++
	return state
}

func ring(state State, playerID string) State {
	if state.BellSettled {
		return state
	}
	if state.RingAttempts == nil {
		state.RingAttempts = map[string]bool{}
	}
	if state.RingAttempts[playerID] {
		return state
	}
	state.RingAttempts[playerID] = true
	if hasFiveFruit(state) {
		index := findPlayer(state, playerID)
		if index >= 0 {
			won := collectFaceUp(&state)
			state.Players[index].Deck = append(state.Players[index].Deck, won...)
			state.Players[index].Score += len(won)
			state.BellSettled = true
			state.LastBellWinnerID = playerID
			state.Log = append(state.Log, playerID+" 님이 종을 맞게 눌렀습니다.")
		}
	} else {
		index := findPlayer(state, playerID)
		if index >= 0 {
			penalty := payWrongRingPenalty(&state, index)
			if penalty > 0 {
				state.Log = append(state.Log, playerID+" 님이 종을 잘못 눌러 벌칙 카드를 냈습니다.")
			} else {
				state.Log = append(state.Log, playerID+" 님이 종을 잘못 눌렀지만 낼 카드가 없습니다.")
			}
		} else {
			state.Log = append(state.Log, playerID+" 님이 종을 잘못 눌렀습니다.")
		}
	}
	return state
}

func refresh(state State) State {
	active := 0
	for index := range state.Players {
		if state.Players[index].Forfeited {
			state.Players[index].Active = false
			continue
		}
		state.Players[index].Active = totalCards(state.Players[index]) > 0
		if state.Players[index].Active {
			active++
		}
	}
	if active <= 1 {
		state.Finished = true
		state.Log = append(state.Log, "할리갈리가 종료되었습니다.")
		return state
	}
	if len(state.Players[state.CurrentPlayerIndex].Deck) == 0 {
		state.CurrentPlayerIndex = nextCanFlipIndex(state, state.CurrentPlayerIndex)
	}
	return state
}

func hasFiveFruit(state State) bool {
	counts := map[string]int{}
	for _, player := range state.Players {
		if len(player.FaceUp) == 0 {
			continue
		}
		card := player.FaceUp[len(player.FaceUp)-1]
		counts[card.Fruit] += card.Count
	}
	for _, count := range counts {
		if count == 5 {
			return true
		}
	}
	return false
}

func collectFaceUp(state *State) []Card {
	won := []Card{}
	for playerIndex := range state.Players {
		won = append(won, state.Players[playerIndex].FaceUp...)
		state.Players[playerIndex].FaceUp = []Card{}
	}
	return won
}

func payWrongRingPenalty(state *State, payerIndex int) int {
	paid := 0
	for playerIndex := range state.Players {
		if playerIndex == payerIndex || !state.Players[playerIndex].Active || len(state.Players[payerIndex].Deck) == 0 {
			continue
		}
		card := state.Players[payerIndex].Deck[0]
		state.Players[payerIndex].Deck = state.Players[payerIndex].Deck[1:]
		state.Players[playerIndex].Deck = append(state.Players[playerIndex].Deck, card)
		paid++
	}
	return paid
}

func nextActiveIndex(state State, current int) int {
	for step := 1; step <= len(state.Players); step++ {
		next := (current + step) % len(state.Players)
		if state.Players[next].Active && len(state.Players[next].Deck) > 0 {
			return next
		}
	}
	return current
}

func nextCanFlipIndex(state State, current int) int {
	for step := 1; step <= len(state.Players); step++ {
		next := (current + step) % len(state.Players)
		if len(state.Players[next].Deck) > 0 {
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

func shuffledDeck(seed string) []Card {
	fruits := []string{"banana", "strawberry", "lime", "plum"}
	copiesByCount := map[int]int{1: 5, 2: 3, 3: 3, 4: 2, 5: 1}
	deck := make([]Card, 0, 56)
	for _, fruit := range fruits {
		for count := 1; count <= 5; count++ {
			for copy := 0; copy < copiesByCount[count]; copy++ {
				deck = append(deck, Card{Fruit: fruit, Count: count})
			}
		}
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

func sortedPlayers(players []PlayerState) []PlayerState {
	copyPlayers := append([]PlayerState(nil), players...)
	for i := range copyPlayers {
		for j := i + 1; j < len(copyPlayers); j++ {
			if totalCards(copyPlayers[j]) > totalCards(copyPlayers[i]) {
				copyPlayers[i], copyPlayers[j] = copyPlayers[j], copyPlayers[i]
			}
		}
	}
	return copyPlayers
}

func totalCards(player PlayerState) int {
	return len(player.Deck) + len(player.FaceUp)
}

func cloneState(state State) State {
	clone := state
	clone.Players = make([]PlayerState, len(state.Players))
	for index := range state.Players {
		clone.Players[index] = state.Players[index]
		clone.Players[index].Deck = append([]Card(nil), state.Players[index].Deck...)
		clone.Players[index].FaceUp = append([]Card(nil), state.Players[index].FaceUp...)
	}
	clone.Log = append([]string(nil), state.Log...)
	if state.RingAttempts != nil {
		clone.RingAttempts = map[string]bool{}
		for playerID, attempted := range state.RingAttempts {
			clone.RingAttempts[playerID] = attempted
		}
	}
	return clone
}

func asState(state any) State {
	typed, ok := state.(State)
	if ok {
		return typed
	}
	return State{}
}
