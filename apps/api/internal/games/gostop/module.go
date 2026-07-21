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
	ID    string `json:"id"`
	Month int    `json:"month"`
	Kind  string `json:"kind"`
}

type PlayerState struct {
	PlayerID string `json:"playerId"`
	Hand     []Card `json:"hand"`
	HandSize int    `json:"handSize"`
	Captured []Card `json:"captured"`
	Score    int    `json:"score"`
	GoCount  int    `json:"goCount"`
	Active   bool   `json:"active"`
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
		captureMonth(player, &current.Field, card)
		if len(current.Deck) > 0 {
			drawn := current.Deck[0]
			current.Deck = current.Deck[1:]
			captureMonth(player, &current.Field, drawn)
		}
		player.HandSize = len(player.Hand)
		player.Score = score(player.Captured)
		current.Log = append(current.Log, fmt.Sprintf("%s 님이 패를 냈습니다.", player.PlayerID))
		if len(player.Hand) == 0 || len(current.Deck) == 0 {
			current.WinnerID = bestPlayer(current).PlayerID
			current.Finished = true
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
		current.WinnerID = player.PlayerID
		current.Finished = true
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
		current.WinnerID = bestPlayer(current).PlayerID
		current.Finished = true
		return gamecore.ActionResult{State: current}, nil
	}

	if current.CurrentPlayerIndex == index {
		if current.AwaitingDecision {
			current.AwaitingDecision = false
			current.WinnerID = current.Players[index].PlayerID
			current.Finished = true
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
		if player.PlayerID == current.WinnerID {
			outcome = gamecore.OutcomeWin
		}
		results = append(results, gamecore.Result{PlayerID: gamecore.PlayerID(player.PlayerID), Rank: rankForOutcome(outcome), Score: player.Score, Outcome: outcome})
	}
	return results
}

func captureMonth(player *PlayerState, field *[]Card, card Card) {
	matches := []Card{}
	rest := []Card{}
	for _, item := range *field {
		if item.Month == card.Month {
			matches = append(matches, item)
		} else {
			rest = append(rest, item)
		}
	}
	if len(matches) > 0 {
		player.Captured = append(player.Captured, card)
		player.Captured = append(player.Captured, matches...)
		*field = rest
		return
	}
	*field = append(*field, card)
}

func score(cards []Card) int {
	bright, animal, ribbon, junk := 0, 0, 0, 0
	birds := map[int]bool{}
	for _, card := range cards {
		switch card.Kind {
		case "bright":
			bright++
		case "animal":
			animal++
			if card.Month == 2 || card.Month == 4 || card.Month == 8 {
				birds[card.Month] = true
			}
		case "ribbon":
			ribbon++
		default:
			junk++
		}
	}
	total := 0
	if bright == 3 {
		total += 3
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
	if junk >= 10 {
		total += junk - 9
	}
	return total
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
	deck := []Card{}
	for month := 1; month <= 12; month++ {
		deck = append(deck,
			Card{ID: fmt.Sprintf("%d-bright", month), Month: month, Kind: brightKind(month)},
			Card{ID: fmt.Sprintf("%d-animal", month), Month: month, Kind: "animal"},
			Card{ID: fmt.Sprintf("%d-ribbon", month), Month: month, Kind: "ribbon"},
			Card{ID: fmt.Sprintf("%d-junk", month), Month: month, Kind: "junk"},
		)
	}
	random := rand.New(rand.NewSource(seedToInt(seed)))
	random.Shuffle(len(deck), func(i, j int) {
		deck[i], deck[j] = deck[j], deck[i]
	})
	return deck
}

func brightKind(month int) string {
	if month == 1 || month == 3 || month == 8 || month == 11 || month == 12 {
		return "bright"
	}
	return "junk"
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
