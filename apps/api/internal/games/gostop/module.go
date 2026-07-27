package gostop

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"math/rand"

	"board-game-platform/apps/api/internal/gamecore"
)

const (
	ActionPlay = "gostop.play"
	ActionGo   = "gostop.go"
	ActionStop = "gostop.stop"
)

type Card struct {
	ID        string   `json:"id"`
	Month     int      `json:"month"`
	Kind      string   `json:"kind"`
	Tags      []string `json:"tags,omitempty"`
	JunkValue int      `json:"junkValue,omitempty"`
}

type PlayerState struct {
	PlayerID    string   `json:"playerId"`
	Hand        []Card   `json:"hand"`
	HandSize    int      `json:"handSize"`
	Captured    []Card   `json:"captured"`
	Score       int      `json:"score"`
	FinalScore  int      `json:"finalScore,omitempty"`
	PenaltyTags []string `json:"penaltyTags,omitempty"`
	GoCount     int      `json:"goCount"`
	Active      bool     `json:"active"`
}

type State struct {
	CurrentPlayerIndex int           `json:"currentPlayerIndex"`
	Round              int           `json:"round"`
	Players            []PlayerState `json:"players"`
	Field              []Card        `json:"field"`
	Deck               []Card        `json:"deck"`
	WinnerID           string        `json:"winnerId,omitempty"`
	AwaitingDecision   bool          `json:"awaitingDecision"`
	Log                []string      `json:"log"`
	Finished           bool          `json:"finished"`
}

type PlayPayload struct {
	CardID string `json:"cardId"`
}

type Module struct{}

func NewModule() Module {
	return Module{}
}

func (m Module) ID() gamecore.GameID {
	return "gostop"
}

func (m Module) Name() string {
	return "고스톱"
}

func (m Module) MinPlayers() int {
	return 2
}

func (m Module) MaxPlayers() int {
	return 3
}

func (m Module) CreateInitialState(ctx gamecore.Context) any {
	deck := shuffledDeck(ctx.RandomSeed)
	handSize := 7
	fieldSize := 6
	if len(ctx.Players) == 2 {
		handSize = 10
		fieldSize = 8
	}
	players := make([]PlayerState, 0, len(ctx.Players))
	for _, player := range ctx.Players {
		hand := append([]Card(nil), deck[:handSize]...)
		deck = deck[handSize:]
		players = append(players, PlayerState{PlayerID: string(player.ID), Hand: hand, HandSize: len(hand), Active: true})
	}
	field := append([]Card(nil), deck[:fieldSize]...)
	deck = deck[fieldSize:]
	return State{Players: players, Field: field, Deck: deck, Log: []string{"고스톱이 시작되었습니다."}}
}

func (m Module) PublicState(state any, viewerID gamecore.PlayerID) any {
	current := asState(state)
	current.Deck = make([]Card, len(current.Deck))
	current.Players = append([]PlayerState(nil), current.Players...)
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
	switch action.Type {
	case ActionPlay:
		if current.AwaitingDecision {
			return errors.New("must choose go or stop")
		}
		payload, err := playPayload(action.Payload)
		if err != nil {
			return err
		}
		if _, ok := findCard(current.Players[current.CurrentPlayerIndex].Hand, payload.CardID); !ok {
			return errors.New("card not found")
		}
	case ActionGo:
		if current.Players[current.CurrentPlayerIndex].Score < 3 {
			return errors.New("go requires at least 3 points")
		}
		if !current.AwaitingDecision {
			return errors.New("no go decision is pending")
		}
	case ActionStop:
		if current.Players[current.CurrentPlayerIndex].Score < 3 {
			return errors.New("stop requires at least 3 points")
		}
		if !current.AwaitingDecision {
			return errors.New("no stop decision is pending")
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
	case ActionPlay:
		previousScore := player.Score
		payload, err := playPayload(action.Payload)
		if err != nil {
			return gamecore.ActionResult{}, err
		}
		card, _ := removeCard(&player.Hand, payload.CardID)
		captureLogs := captureTurn(&current, current.CurrentPlayerIndex, card)
		player.HandSize = len(player.Hand)
		refreshScores(&current)
		current.Log = append(current.Log, fmt.Sprintf("%s 님이 패를 냈습니다.", player.PlayerID))
		current.Log = append(current.Log, captureLogs...)
		if len(player.Hand) == 0 || len(current.Deck) == 0 {
			current = finishWithWinner(current, bestPlayer(current).PlayerID)
		} else if player.Score >= 3 && player.Score > previousScore {
			current.AwaitingDecision = true
			current.Log = append(current.Log, player.PlayerID+" 님이 고/스톱을 선택해야 합니다.")
		}
	case ActionGo:
		player.GoCount++
		current.AwaitingDecision = false
		current.Log = append(current.Log, player.PlayerID+" 님이 고를 외쳤습니다.")
	case ActionStop:
		current.AwaitingDecision = false
		current = finishWithWinner(current, player.PlayerID)
		current.Log = append(current.Log, player.PlayerID+" 님이 스톱했습니다.")
	}
	if !current.Finished && !current.AwaitingDecision {
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
	if index < 0 || current.Finished {
		return gamecore.ActionResult{State: current}, nil
	}
	current.Players[index].Active = false
	current.Log = append(current.Log, current.Players[index].PlayerID+" 님의 재접속 시간이 만료되어 자동 기권 처리되었습니다.")

	if activePlayers(current) <= 1 {
		current = finishWithWinner(current, bestPlayer(current).PlayerID)
		return gamecore.ActionResult{State: current}, nil
	}

	if current.CurrentPlayerIndex == index {
		if current.AwaitingDecision {
			current.AwaitingDecision = false
			current = finishWithWinner(current, current.Players[index].PlayerID)
			current.Log = append(current.Log, current.Players[index].PlayerID+" 님이 시간 초과로 자동 스톱 처리되었습니다.")
		} else {
			current.CurrentPlayerIndex = nextActiveIndex(current, index)
			current.Round++
		}
	}
	return gamecore.ActionResult{State: current}, nil
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

func (m Module) IsFinished(state any, _ gamecore.Context) bool {
	return asState(state).Finished
}

func (m Module) CalculateResult(state any, _ gamecore.Context) []gamecore.Result {
	current := asState(state)
	results := make([]gamecore.Result, 0, len(current.Players))
	for _, player := range current.Players {
		outcome := gamecore.OutcomeLose
		resultScore := player.Score
		if player.PlayerID == current.WinnerID {
			outcome = gamecore.OutcomeWin
			if player.FinalScore > 0 {
				resultScore = player.FinalScore
			}
		}
		results = append(results, gamecore.Result{PlayerID: gamecore.PlayerID(player.PlayerID), Rank: rankForOutcome(outcome), Score: resultScore, Outcome: outcome})
	}
	return results
}

func finishWithWinner(state State, winnerID string) State {
	state.WinnerID = winnerID
	state.Finished = true
	winnerIndex := findPlayer(state, winnerID)
	if winnerIndex < 0 {
		return state
	}
	finalScore, tags := finalScore(state, winnerIndex)
	state.Players[winnerIndex].FinalScore = finalScore
	state.Players[winnerIndex].PenaltyTags = tags
	if len(tags) > 0 {
		state.Log = append(state.Log, fmt.Sprintf("%s 님에게 최종 배율 %s 이 적용되었습니다.", winnerID, joinTags(tags)))
	}
	return state
}

func finalScore(state State, winnerIndex int) (int, []string) {
	winner := state.Players[winnerIndex]
	base := winner.Score + goBonus(winner.GoCount)
	if base < 1 {
		base = winner.Score
	}
	multiplier := goMultiplier(winner.GoCount)
	tags := []string{}
	if multiplier > 1 {
		tags = append(tags, fmt.Sprintf("%d고", winner.GoCount))
	}
	for index, loser := range state.Players {
		if index == winnerIndex {
			continue
		}
		if junkScore(winner.Captured) > 0 && junkCount(loser.Captured) < 6 {
			multiplier *= 2
			tags = append(tags, "피박")
		}
		if brightCount(winner.Captured) >= 3 && brightCount(loser.Captured) == 0 {
			multiplier *= 2
			tags = append(tags, "광박")
		}
		if animalScore(winner.Captured) > 0 && animalCount(loser.Captured) == 0 {
			multiplier *= 2
			tags = append(tags, "멍박")
		}
		if loser.GoCount > 0 {
			multiplier *= 2
			tags = append(tags, "고박")
		}
	}
	if multiplier < 1 {
		multiplier = 1
	}
	return base * multiplier, tags
}

func goBonus(goCount int) int {
	if goCount <= 0 {
		return 0
	}
	if goCount == 1 {
		return 1
	}
	return 2
}

func goMultiplier(goCount int) int {
	if goCount < 3 {
		return 1
	}
	multiplier := 1
	for count := 3; count <= goCount; count++ {
		multiplier *= 2
	}
	return multiplier
}

func captureMonth(player *PlayerState, field *[]Card, card Card) {
	matches := takeMonthCards(field, card.Month)
	if len(matches) == 0 {
		*field = append(*field, card)
		return
	}
	player.Captured = append(player.Captured, card)
	player.Captured = append(player.Captured, matches...)
}

func captureTurn(state *State, playerIndex int, card Card) []string {
	player := &state.Players[playerIndex]
	logs := []string{}
	capturedThisTurn := 0
	firstMatches := takeMonthCards(&state.Field, card.Month)
	if len(firstMatches) > 0 {
		player.Captured = append(player.Captured, card)
		player.Captured = append(player.Captured, firstMatches...)
		capturedThisTurn += 1 + len(firstMatches)
	} else {
		state.Field = append(state.Field, card)
	}

	if len(state.Deck) > 0 {
		drawn := state.Deck[0]
		state.Deck = state.Deck[1:]
		switch {
		case len(firstMatches) == 1 && drawn.Month == card.Month:
			player.Captured = player.Captured[:len(player.Captured)-2]
			capturedThisTurn -= 2
			state.Field = append(state.Field, card, firstMatches[0], drawn)
			logs = append(logs, player.PlayerID+" 님이 뻑을 만들었습니다.")
		case len(firstMatches) == 2 && drawn.Month == card.Month:
			player.Captured = append(player.Captured, drawn)
			capturedThisTurn++
			if stolen := stealJunkFromOpponents(state, playerIndex); stolen > 0 {
				logs = append(logs, fmt.Sprintf("%s 님이 따닥으로 피 %d장을 가져왔습니다.", player.PlayerID, stolen))
			} else {
				logs = append(logs, player.PlayerID+" 님이 따닥을 했습니다.")
			}
		default:
			before := len(player.Captured)
			drawMatches := takeMonthCards(&state.Field, drawn.Month)
			if len(drawMatches) > 0 {
				player.Captured = append(player.Captured, drawn)
				player.Captured = append(player.Captured, drawMatches...)
				capturedThisTurn += len(player.Captured) - before
				if len(firstMatches) == 0 && drawn.Month == card.Month {
					if stolen := stealJunkFromOpponents(state, playerIndex); stolen > 0 {
						logs = append(logs, fmt.Sprintf("%s 님이 쪽으로 피 %d장을 가져왔습니다.", player.PlayerID, stolen))
					} else {
						logs = append(logs, player.PlayerID+" 님이 쪽을 했습니다.")
					}
				}
			} else {
				state.Field = append(state.Field, drawn)
			}
		}
	}

	if capturedThisTurn > 0 && len(state.Field) == 0 {
		if stolen := stealJunkFromOpponents(state, playerIndex); stolen > 0 {
			logs = append(logs, fmt.Sprintf("%s 님이 싹쓸이로 피 %d장을 가져왔습니다.", player.PlayerID, stolen))
		} else {
			logs = append(logs, player.PlayerID+" 님이 싹쓸이를 했습니다.")
		}
	}
	return logs
}

func takeMonthCards(field *[]Card, month int) []Card {
	matches := []Card{}
	rest := []Card{}
	for _, item := range *field {
		if item.Month == month {
			matches = append(matches, item)
		} else {
			rest = append(rest, item)
		}
	}
	*field = rest
	return matches
}

func stealJunkFromOpponents(state *State, playerIndex int) int {
	stolen := 0
	for index := range state.Players {
		if index == playerIndex {
			continue
		}
		card, ok := takeStealableJunk(&state.Players[index])
		if !ok {
			continue
		}
		state.Players[playerIndex].Captured = append(state.Players[playerIndex].Captured, card)
		stolen++
	}
	return stolen
}

func takeStealableJunk(player *PlayerState) (Card, bool) {
	selected := -1
	selectedValue := 99
	for index, card := range player.Captured {
		if card.Kind != "junk" {
			continue
		}
		value := card.JunkValue
		if value <= 0 {
			value = 1
		}
		if value < selectedValue {
			selected = index
			selectedValue = value
		}
	}
	if selected < 0 {
		return Card{}, false
	}
	card := player.Captured[selected]
	player.Captured = append(player.Captured[:selected], player.Captured[selected+1:]...)
	return card, true
}

func refreshScores(state *State) {
	for index := range state.Players {
		state.Players[index].Score = score(state.Players[index].Captured)
	}
}

func score(cards []Card) int {
	bright, animal, ribbon, junk := 0, 0, 0, 0
	rainBright := false
	birds := map[int]bool{}
	ribbonSets := map[string]int{}
	for _, card := range cards {
		switch card.Kind {
		case "bright":
			bright++
			if hasTag(card, "rain") {
				rainBright = true
			}
		case "animal":
			animal++
			if hasTag(card, "bird") || card.Month == 2 || card.Month == 4 || card.Month == 8 {
				birds[card.Month] = true
			}
		case "ribbon":
			ribbon++
			for _, tag := range card.Tags {
				if tag == "red" || tag == "blue" || tag == "poetry" {
					ribbonSets[tag]++
				}
			}
		default:
			if card.JunkValue > 0 {
				junk += card.JunkValue
			} else {
				junk++
			}
		}
	}
	total := 0
	if bright == 3 {
		if rainBright {
			total += 2
		} else {
			total += 3
		}
	}
	if bright == 4 {
		total += 4
	}
	if bright >= 5 {
		total += 15
	}
	if animal >= 5 {
		total += animal - 4
	}
	if len(birds) == 3 {
		total += 5
	}
	if ribbon >= 5 {
		total += ribbon - 4
	}
	for _, tag := range []string{"red", "blue", "poetry"} {
		if ribbonSets[tag] >= 3 {
			total += 3
		}
	}
	if junk >= 10 {
		total += junk - 9
	}
	return total
}

func brightCount(cards []Card) int {
	count := 0
	for _, card := range cards {
		if card.Kind == "bright" {
			count++
		}
	}
	return count
}

func animalCount(cards []Card) int {
	count := 0
	for _, card := range cards {
		if card.Kind == "animal" {
			count++
		}
	}
	return count
}

func animalScore(cards []Card) int {
	count := animalCount(cards)
	if count < 5 {
		return 0
	}
	return count - 4
}

func junkCount(cards []Card) int {
	count := 0
	for _, card := range cards {
		if card.Kind != "junk" {
			continue
		}
		if card.JunkValue > 0 {
			count += card.JunkValue
		} else {
			count++
		}
	}
	return count
}

func junkScore(cards []Card) int {
	count := junkCount(cards)
	if count < 10 {
		return 0
	}
	return count - 9
}

func bestPlayer(state State) PlayerState {
	best := state.Players[0]
	for _, player := range state.Players[1:] {
		if player.Score > best.Score {
			best = player
		}
	}
	return best
}

func joinTags(tags []string) string {
	if len(tags) == 0 {
		return ""
	}
	result := tags[0]
	for _, tag := range tags[1:] {
		result += "/" + tag
	}
	return result
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

func playPayload(payload any) (PlayPayload, error) {
	raw, ok := payload.(map[string]any)
	if !ok {
		return PlayPayload{}, errors.New("invalid play payload")
	}
	cardID, _ := raw["cardId"].(string)
	if cardID == "" {
		return PlayPayload{}, errors.New("card is required")
	}
	return PlayPayload{CardID: cardID}, nil
}

func findCard(cards []Card, cardID string) (Card, bool) {
	for _, card := range cards {
		if card.ID == cardID {
			return card, true
		}
	}
	return Card{}, false
}

func removeCard(cards *[]Card, cardID string) (Card, bool) {
	next := []Card{}
	removed := Card{}
	found := false
	for _, card := range *cards {
		if !found && card.ID == cardID {
			removed = card
			found = true
			continue
		}
		next = append(next, card)
	}
	*cards = next
	return removed, found
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
	deck := standardDeck()
	random := rand.New(rand.NewSource(seedToInt(seed)))
	random.Shuffle(len(deck), func(i, j int) {
		deck[i], deck[j] = deck[j], deck[i]
	})
	return deck
}

func standardDeck() []Card {
	return []Card{
		{ID: "1-bright", Month: 1, Kind: "bright"},
		{ID: "1-ribbon", Month: 1, Kind: "ribbon", Tags: []string{"red"}},
		{ID: "1-junk-a", Month: 1, Kind: "junk"},
		{ID: "1-junk-b", Month: 1, Kind: "junk"},
		{ID: "2-animal", Month: 2, Kind: "animal", Tags: []string{"bird"}},
		{ID: "2-ribbon", Month: 2, Kind: "ribbon", Tags: []string{"red"}},
		{ID: "2-junk-a", Month: 2, Kind: "junk"},
		{ID: "2-junk-b", Month: 2, Kind: "junk"},
		{ID: "3-bright", Month: 3, Kind: "bright"},
		{ID: "3-ribbon", Month: 3, Kind: "ribbon", Tags: []string{"red"}},
		{ID: "3-junk-a", Month: 3, Kind: "junk"},
		{ID: "3-junk-b", Month: 3, Kind: "junk"},
		{ID: "4-animal", Month: 4, Kind: "animal", Tags: []string{"bird"}},
		{ID: "4-ribbon", Month: 4, Kind: "ribbon", Tags: []string{"poetry"}},
		{ID: "4-junk-a", Month: 4, Kind: "junk"},
		{ID: "4-junk-b", Month: 4, Kind: "junk"},
		{ID: "5-animal", Month: 5, Kind: "animal"},
		{ID: "5-ribbon", Month: 5, Kind: "ribbon", Tags: []string{"poetry"}},
		{ID: "5-junk-a", Month: 5, Kind: "junk"},
		{ID: "5-junk-b", Month: 5, Kind: "junk"},
		{ID: "6-animal", Month: 6, Kind: "animal"},
		{ID: "6-ribbon", Month: 6, Kind: "ribbon", Tags: []string{"blue"}},
		{ID: "6-junk-a", Month: 6, Kind: "junk"},
		{ID: "6-junk-b", Month: 6, Kind: "junk"},
		{ID: "7-animal", Month: 7, Kind: "animal"},
		{ID: "7-ribbon", Month: 7, Kind: "ribbon", Tags: []string{"poetry"}},
		{ID: "7-junk-a", Month: 7, Kind: "junk"},
		{ID: "7-junk-b", Month: 7, Kind: "junk"},
		{ID: "8-bright", Month: 8, Kind: "bright"},
		{ID: "8-animal", Month: 8, Kind: "animal", Tags: []string{"bird"}},
		{ID: "8-junk-a", Month: 8, Kind: "junk"},
		{ID: "8-junk-b", Month: 8, Kind: "junk"},
		{ID: "9-animal", Month: 9, Kind: "animal"},
		{ID: "9-ribbon", Month: 9, Kind: "ribbon", Tags: []string{"blue"}},
		{ID: "9-junk-a", Month: 9, Kind: "junk"},
		{ID: "9-junk-b", Month: 9, Kind: "junk"},
		{ID: "10-animal", Month: 10, Kind: "animal"},
		{ID: "10-ribbon", Month: 10, Kind: "ribbon", Tags: []string{"blue"}},
		{ID: "10-junk-a", Month: 10, Kind: "junk"},
		{ID: "10-junk-b", Month: 10, Kind: "junk"},
		{ID: "11-bright", Month: 11, Kind: "bright"},
		{ID: "11-animal", Month: 11, Kind: "animal"},
		{ID: "11-junk-double", Month: 11, Kind: "junk", JunkValue: 2},
		{ID: "11-junk", Month: 11, Kind: "junk"},
		{ID: "12-bright", Month: 12, Kind: "bright", Tags: []string{"rain"}},
		{ID: "12-animal", Month: 12, Kind: "animal"},
		{ID: "12-ribbon", Month: 12, Kind: "ribbon"},
		{ID: "12-junk-double", Month: 12, Kind: "junk", JunkValue: 2},
	}
}

func hasTag(card Card, tag string) bool {
	for _, item := range card.Tags {
		if item == tag {
			return true
		}
	}
	return false
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
