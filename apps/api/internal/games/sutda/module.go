package sutda

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
	ActionCall     = "sutda.call"
	ActionFold     = "sutda.fold"
	ActionShowdown = "sutda.showdown"
)

type Card struct {
	ID    string `json:"id"`
	Month int    `json:"month"`
	Gwang bool   `json:"gwang"`
}

type PlayerState struct {
	PlayerID string `json:"playerId"`
	Hand     []Card `json:"hand"`
	RankName string `json:"rankName,omitempty"`
	Rank     int    `json:"rank,omitempty"`
	Folded   bool   `json:"folded"`
	Ready    bool   `json:"ready"`
	Active   bool   `json:"active"`
}

type State struct {
	CurrentPlayerIndex int           `json:"currentPlayerIndex"`
	Round              int           `json:"round"`
	Players            []PlayerState `json:"players"`
	Pot                int           `json:"pot"`
	WinnerID           string        `json:"winnerId,omitempty"`
	Log                []string      `json:"log"`
	Finished           bool          `json:"finished"`
}

type Module struct{}

func NewModule() Module {
	return Module{}
}

func (m Module) ID() gamecore.GameID {
	return "sutda"
}

func (m Module) Name() string {
	return "섯다"
}

func (m Module) MinPlayers() int {
	return 2
}

func (m Module) MaxPlayers() int {
	return 10
}

func (m Module) CreateInitialState(ctx gamecore.Context) any {
	deck := shuffledDeck(ctx.RandomSeed)
	players := make([]PlayerState, 0, len(ctx.Players))
	for _, player := range ctx.Players {
		hand := append([]Card(nil), deck[:2]...)
		deck = deck[2:]
		rank, name := evaluate(hand)
		players = append(players, PlayerState{
			PlayerID: string(player.ID),
			Hand:     hand,
			Rank:     rank,
			RankName: name,
			Active:   true,
		})
	}
	return State{Players: players, Pot: len(players), Log: []string{"섯다 한 판이 시작되었습니다."}}
}

func (m Module) PublicState(state any, viewerID gamecore.PlayerID) any {
	current := asState(state)
	current.Players = append([]PlayerState(nil), current.Players...)
	for index := range current.Players {
		if current.Finished || current.Players[index].PlayerID == string(viewerID) {
			continue
		}
		current.Players[index].Hand = []Card{}
		current.Players[index].Rank = 0
		current.Players[index].RankName = ""
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
	switch action.Type {
	case ActionCall, ActionFold, ActionShowdown:
		return nil
	default:
		return errors.New("unsupported action")
	}
}

func (m Module) ApplyAction(_ context.Context, state any, action gamecore.Action, _ gamecore.Context) (gamecore.ActionResult, error) {
	current := asState(state)
	player := &current.Players[current.CurrentPlayerIndex]
	switch action.Type {
	case ActionCall:
		player.Ready = true
		current.Pot++
		current.Log = append(current.Log, player.PlayerID+" 님이 콜했습니다.")
	case ActionFold:
		player.Folded = true
		player.Active = false
		current.Log = append(current.Log, player.PlayerID+" 님이 다이했습니다.")
	case ActionShowdown:
		current = finish(current)
	}
	if !current.Finished {
		if activeCount(current) <= 1 || allActiveReady(current) {
			current = finish(current)
		} else {
			current.CurrentPlayerIndex = nextActiveIndex(current, current.CurrentPlayerIndex)
			current.Round++
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
	index := findPlayer(current, string(playerID))
	if index >= 0 && !current.Finished {
		current.Players[index].Folded = true
		current.Players[index].Active = false
		if activeCount(current) <= 1 {
			current = finish(current)
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
		outcome := gamecore.OutcomeLose
		if player.PlayerID == current.WinnerID {
			outcome = gamecore.OutcomeWin
		}
		results = append(results, gamecore.Result{
			PlayerID: gamecore.PlayerID(player.PlayerID),
			Rank:     rankForOutcome(outcome),
			Score:    player.Rank,
			Outcome:  outcome,
		})
	}
	return results
}

func finish(state State) State {
	bestIndex := -1
	for index, player := range state.Players {
		if player.Folded {
			continue
		}
		if bestIndex < 0 || compare(player.Hand, state.Players[bestIndex].Hand) > 0 {
			bestIndex = index
		}
	}
	if bestIndex >= 0 {
		state.WinnerID = state.Players[bestIndex].PlayerID
	}
	state.Finished = true
	state.Log = append(state.Log, "섯다 승부가 종료되었습니다.")
	return state
}

func evaluate(hand []Card) (int, string) {
	if len(hand) != 2 {
		return 0, "무효"
	}
	a, b := hand[0], hand[1]
	months := []int{a.Month, b.Month}
	sort.Ints(months)
	if a.Gwang && b.Gwang {
		if months[0] == 3 && months[1] == 8 {
			return 1000, "38광땡"
		}
		return 950 + months[1], "광땡"
	}
	if a.Month == b.Month {
		return 800 + a.Month, fmt.Sprintf("%d땡", a.Month)
	}
	specials := map[[2]int]struct {
		score int
		name  string
	}{
		{1, 2}:  {700, "알리"},
		{1, 4}:  {690, "독사"},
		{1, 9}:  {680, "구삥"},
		{1, 10}: {670, "장삥"},
		{4, 10}: {660, "장사"},
		{4, 6}:  {650, "세륙"},
	}
	if found, ok := specials[[2]int{months[0], months[1]}]; ok {
		return found.score, found.name
	}
	keut := (a.Month + b.Month) % 10
	if keut == 9 {
		return 609, "갑오"
	}
	if keut == 0 {
		return 500, "망통"
	}
	return 500 + keut, fmt.Sprintf("%d끗", keut)
}

func compare(left []Card, right []Card) int {
	leftRank, _ := evaluate(left)
	rightRank, _ := evaluate(right)
	return leftRank - rightRank
}

func activeCount(state State) int {
	count := 0
	for _, player := range state.Players {
		if !player.Folded {
			count++
		}
	}
	return count
}

func allActiveReady(state State) bool {
	for _, player := range state.Players {
		if !player.Folded && !player.Ready {
			return false
		}
	}
	return true
}

func nextActiveIndex(state State, current int) int {
	for step := 1; step <= len(state.Players); step++ {
		next := (current + step) % len(state.Players)
		if !state.Players[next].Folded {
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

func rankForOutcome(outcome gamecore.Outcome) int {
	if outcome == gamecore.OutcomeWin {
		return 1
	}
	return 2
}

func shuffledDeck(seed string) []Card {
	deck := []Card{}
	for month := 1; month <= 10; month++ {
		deck = append(deck, Card{ID: fmt.Sprintf("%d-a", month), Month: month, Gwang: month == 1 || month == 3 || month == 8})
		deck = append(deck, Card{ID: fmt.Sprintf("%d-b", month), Month: month})
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
