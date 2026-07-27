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
	ActionRaise    = "sutda.raise"
	ActionShowdown = "sutda.showdown"
	maxRaiseAmount = 3
	maxRaisesRound = 3
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
	Bet      int    `json:"bet"`
	Folded   bool   `json:"folded"`
	Ready    bool   `json:"ready"`
	Active   bool   `json:"active"`
}

type State struct {
	CurrentPlayerIndex int           `json:"currentPlayerIndex"`
	Round              int           `json:"round"`
	Players            []PlayerState `json:"players"`
	Pot                int           `json:"pot"`
	CurrentBet         int           `json:"currentBet"`
	RaisesThisRound    int           `json:"raisesThisRound"`
	WinnerID           string        `json:"winnerId,omitempty"`
	WinnerIDs          []string      `json:"winnerIds,omitempty"`
	Draw               bool          `json:"draw,omitempty"`
	Log                []string      `json:"log"`
	Finished           bool          `json:"finished"`
}

type RaisePayload struct {
	Amount int `json:"amount"`
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
			Bet:      1,
			Active:   true,
		})
	}
	return State{Players: players, Pot: len(players), CurrentBet: 1, Log: []string{"섯다 한 판이 시작되었습니다."}}
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
	if current.Players[current.CurrentPlayerIndex].Folded || !current.Players[current.CurrentPlayerIndex].Active {
		return errors.New("player is not active")
	}
	switch action.Type {
	case ActionCall, ActionFold, ActionShowdown:
		return nil
	case ActionRaise:
		payload, err := raisePayload(action.Payload)
		if err != nil {
			return err
		}
		if payload.Amount < 1 || payload.Amount > maxRaiseAmount {
			return fmt.Errorf("raise amount must be between 1 and %d", maxRaiseAmount)
		}
		if current.RaisesThisRound >= maxRaisesRound {
			return errors.New("raise limit reached")
		}
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
		payment := current.CurrentBet - player.Bet
		if payment < 0 {
			payment = 0
		}
		player.Bet += payment
		player.Ready = true
		current.Pot += payment
		current.Log = append(current.Log, fmt.Sprintf("%s 님이 콜했습니다.", player.PlayerID))
	case ActionRaise:
		payload, err := raisePayload(action.Payload)
		if err != nil {
			return gamecore.ActionResult{}, err
		}
		nextBet := current.CurrentBet + payload.Amount
		payment := nextBet - player.Bet
		player.Bet = nextBet
		player.Ready = true
		current.Pot += payment
		current.CurrentBet = nextBet
		current.RaisesThisRound++
		resetOtherReady(&current, current.CurrentPlayerIndex)
		current.Log = append(current.Log, fmt.Sprintf("%s 님이 %d만큼 레이즈했습니다.", player.PlayerID, payload.Amount))
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
		} else if current.CurrentPlayerIndex == index {
			current.CurrentPlayerIndex = nextActiveIndex(current, index)
			current.Round++
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
	for _, playerID := range current.WinnerIDs {
		winners[playerID] = true
	}
	for _, player := range current.Players {
		outcome := gamecore.OutcomeLose
		if current.Draw && winners[player.PlayerID] {
			outcome = gamecore.OutcomeDraw
		} else if winners[player.PlayerID] || player.PlayerID == current.WinnerID {
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
	bestIndexes := []int{}
	for index, player := range state.Players {
		if player.Folded {
			continue
		}
		if len(bestIndexes) == 0 {
			bestIndexes = append(bestIndexes, index)
			continue
		}
		comparison := compare(player.Hand, state.Players[bestIndexes[0]].Hand)
		if comparison > 0 {
			bestIndexes = []int{index}
		} else if comparison == 0 {
			bestIndexes = append(bestIndexes, index)
		}
	}
	state.WinnerIDs = []string{}
	for _, index := range bestIndexes {
		state.WinnerIDs = append(state.WinnerIDs, state.Players[index].PlayerID)
	}
	if len(bestIndexes) == 1 {
		state.WinnerID = state.Players[bestIndexes[0]].PlayerID
		state.Draw = false
	} else if len(bestIndexes) > 1 {
		state.WinnerID = ""
		state.Draw = true
	}
	state.Finished = true
	if state.Draw {
		state.Log = append(state.Log, "섯다 승부가 무승부로 종료되었습니다.")
	} else {
		state.Log = append(state.Log, "섯다 승부가 종료되었습니다.")
	}
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
		{4, 7}:  {501, "암행어사"},
		{3, 7}:  {500, "땡잡이"},
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
	if isAmhaengEosa(left) {
		if gwang, thirtyEight := isGwangTtaeng(right); gwang && !thirtyEight {
			return 1
		}
	}
	if isAmhaengEosa(right) {
		if gwang, thirtyEight := isGwangTtaeng(left); gwang && !thirtyEight {
			return -1
		}
	}
	if isTtaengJabi(left) && isOrdinaryTtaeng(right) {
		return 1
	}
	if isTtaengJabi(right) && isOrdinaryTtaeng(left) {
		return -1
	}
	leftRank, _ := evaluate(left)
	rightRank, _ := evaluate(right)
	return leftRank - rightRank
}

func isAmhaengEosa(hand []Card) bool {
	return hasMonths(hand, 4, 7)
}

func isTtaengJabi(hand []Card) bool {
	return hasMonths(hand, 3, 7)
}

func isGwangTtaeng(hand []Card) (bool, bool) {
	if len(hand) != 2 || !hand[0].Gwang || !hand[1].Gwang {
		return false, false
	}
	return true, hasMonths(hand, 3, 8)
}

func isOrdinaryTtaeng(hand []Card) bool {
	if len(hand) != 2 || hand[0].Month != hand[1].Month {
		return false
	}
	return !(hand[0].Gwang && hand[1].Gwang)
}

func hasMonths(hand []Card, first int, second int) bool {
	if len(hand) != 2 {
		return false
	}
	months := []int{hand[0].Month, hand[1].Month}
	sort.Ints(months)
	return months[0] == first && months[1] == second
}

func activeCount(state State) int {
	count := 0
	for _, player := range state.Players {
		if !player.Folded && player.Active {
			count++
		}
	}
	return count
}

func allActiveReady(state State) bool {
	for _, player := range state.Players {
		if !player.Folded && player.Active && (!player.Ready || player.Bet < state.CurrentBet) {
			return false
		}
	}
	return true
}

func resetOtherReady(state *State, raiserIndex int) {
	for index := range state.Players {
		if index == raiserIndex || state.Players[index].Folded || !state.Players[index].Active {
			continue
		}
		state.Players[index].Ready = false
	}
}

func nextActiveIndex(state State, current int) int {
	for step := 1; step <= len(state.Players); step++ {
		next := (current + step) % len(state.Players)
		if !state.Players[next].Folded && state.Players[next].Active {
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
	if outcome == gamecore.OutcomeWin || outcome == gamecore.OutcomeDraw {
		return 1
	}
	return 2
}

func raisePayload(payload any) (RaisePayload, error) {
	raw, ok := payload.(map[string]any)
	if !ok {
		return RaisePayload{}, errors.New("invalid raise payload")
	}
	amount, ok := intFromAny(raw["amount"])
	if !ok {
		return RaisePayload{}, errors.New("raise amount is required")
	}
	return RaisePayload{Amount: amount}, nil
}

func intFromAny(value any) (int, bool) {
	switch typed := value.(type) {
	case int:
		return typed, true
	case int32:
		return int(typed), true
	case int64:
		return int(typed), true
	case float64:
		if typed != float64(int(typed)) {
			return 0, false
		}
		return int(typed), true
	default:
		return 0, false
	}
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
