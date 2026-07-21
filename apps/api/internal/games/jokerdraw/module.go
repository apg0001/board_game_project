package jokerdraw

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"math/rand"

	"board-game-platform/apps/api/internal/gamecore"
)

const ActionDraw = "jokerdraw.draw"

type Card struct {
	ID    string `json:"id"`
	Rank  string `json:"rank"`
	Joker bool   `json:"joker"`
}

type PlayerState struct {
	PlayerID string `json:"playerId"`
	Hand     []Card `json:"hand"`
	HandSize int    `json:"handSize"`
	Out      bool   `json:"out"`
	Active   bool   `json:"active"`
}

type State struct {
	CurrentPlayerIndex int           `json:"currentPlayerIndex"`
	Round              int           `json:"round"`
	Players            []PlayerState `json:"players"`
	LoserID            string        `json:"loserId,omitempty"`
	Log                []string      `json:"log"`
	Finished           bool          `json:"finished"`
}

type DrawPayload struct {
	TargetPlayerID string `json:"targetPlayerId"`
	CardIndex      int    `json:"cardIndex"`
}

type Module struct{}

func NewModule() Module {
	return Module{}
}

func (m Module) ID() gamecore.GameID {
	return "jokerdraw"
}

func (m Module) Name() string {
	return "조커뽑기"
}

func (m Module) MinPlayers() int {
	return 2
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
		hand = removePairs(hand)
		players = append(players, PlayerState{PlayerID: string(player.ID), Hand: hand, HandSize: len(hand), Active: len(hand) > 0})
	}
	state := State{Players: players, Log: []string{"조커뽑기가 시작되었습니다."}}
	state.CurrentPlayerIndex = nextActiveIndex(state, len(players)-1)
	return state
}

func (m Module) PublicState(state any, viewerID gamecore.PlayerID) any {
	current := asState(state)
	current.Players = append([]PlayerState(nil), current.Players...)
	for index := range current.Players {
		current.Players[index].HandSize = len(current.Players[index].Hand)
		if current.Players[index].PlayerID != string(viewerID) && !current.Finished {
			current.Players[index].Hand = make([]Card, len(current.Players[index].Hand))
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
	if action.Type != ActionDraw {
		return errors.New("unsupported action")
	}
	payload, err := drawPayload(action.Payload)
	if err != nil {
		return err
	}
	targetIndex := findPlayer(current, payload.TargetPlayerID)
	if targetIndex < 0 || !current.Players[targetIndex].Active || targetIndex == current.CurrentPlayerIndex {
		return errors.New("invalid draw target")
	}
	if payload.CardIndex < 0 || payload.CardIndex >= len(current.Players[targetIndex].Hand) {
		return errors.New("card index out of range")
	}
	return nil
}

func (m Module) ApplyAction(_ context.Context, state any, action gamecore.Action, _ gamecore.Context) (gamecore.ActionResult, error) {
	current := asState(state)
	payload, err := drawPayload(action.Payload)
	if err != nil {
		return gamecore.ActionResult{}, err
	}
	player := &current.Players[current.CurrentPlayerIndex]
	targetIndex := findPlayer(current, payload.TargetPlayerID)
	target := &current.Players[targetIndex]
	card := target.Hand[payload.CardIndex]
	target.Hand = append(target.Hand[:payload.CardIndex], target.Hand[payload.CardIndex+1:]...)
	target.HandSize = len(target.Hand)
	player.Hand = append(player.Hand, card)
	player.Hand = removePairs(player.Hand)
	player.HandSize = len(player.Hand)
	markOuts(&current)
	current.Log = append(current.Log, fmt.Sprintf("%s 님이 %s 님에게서 카드를 뽑았습니다.", player.PlayerID, target.PlayerID))
	if activeCount(current) <= 1 {
		current.Finished = true
		for _, item := range current.Players {
			if item.Active {
				current.LoserID = item.PlayerID
			}
		}
		current.Log = append(current.Log, "조커뽑기가 종료되었습니다.")
	} else {
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
	if index := findPlayer(current, string(playerID)); index >= 0 && !current.Finished {
		current.Players[index].Out = true
		current.Players[index].Active = false
		if activeCount(current) <= 1 {
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
	results := make([]gamecore.Result, 0, len(current.Players))
	for _, player := range current.Players {
		outcome := gamecore.OutcomeWin
		if player.PlayerID == current.LoserID {
			outcome = gamecore.OutcomeLose
		}
		results = append(results, gamecore.Result{PlayerID: gamecore.PlayerID(player.PlayerID), Rank: rankForOutcome(outcome), Score: -len(player.Hand), Outcome: outcome})
	}
	return results
}

func removePairs(hand []Card) []Card {
	buckets := map[string][]Card{}
	for _, card := range hand {
		if card.Joker {
			buckets[card.ID] = append(buckets[card.ID], card)
			continue
		}
		buckets[card.Rank] = append(buckets[card.Rank], card)
	}
	result := []Card{}
	for key, cards := range buckets {
		if key == "joker" || len(cards)%2 == 1 {
			result = append(result, cards[len(cards)-1])
		}
	}
	return result
}

func markOuts(state *State) {
	for index := range state.Players {
		if len(state.Players[index].Hand) == 0 {
			state.Players[index].Out = true
			state.Players[index].Active = false
		}
	}
}

func drawPayload(payload any) (DrawPayload, error) {
	raw, ok := payload.(map[string]any)
	if !ok {
		return DrawPayload{}, errors.New("invalid draw payload")
	}
	targetID, _ := raw["targetPlayerId"].(string)
	value, ok := raw["cardIndex"].(float64)
	if targetID == "" || !ok {
		return DrawPayload{}, errors.New("target and card index are required")
	}
	return DrawPayload{TargetPlayerID: targetID, CardIndex: int(value)}, nil
}

func activeCount(state State) int {
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

func rankForOutcome(outcome gamecore.Outcome) int {
	if outcome == gamecore.OutcomeWin {
		return 1
	}
	return 2
}

func shuffledDeck(seed string) []Card {
	ranks := []string{"A", "2", "3", "4", "5", "6", "7", "8", "9", "10", "J", "Q", "K"}
	deck := make([]Card, 0, 53)
	for copyIndex := 1; copyIndex <= 4; copyIndex++ {
		for _, rank := range ranks {
			deck = append(deck, Card{ID: fmt.Sprintf("%s-%d", rank, copyIndex), Rank: rank})
		}
	}
	deck = append(deck, Card{ID: "joker", Rank: "joker", Joker: true})
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
