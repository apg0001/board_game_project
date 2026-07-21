package onecard

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"math/rand"

	"board-game-platform/apps/api/internal/gamecore"
)

const (
	ActionPlay = "onecard.play"
	ActionDraw = "onecard.draw"
)

type Card struct {
	ID    string `json:"id"`
	Suit  string `json:"suit"`
	Rank  string `json:"rank"`
	Value int    `json:"value"`
}

type PlayerState struct {
	PlayerID string `json:"playerId"`
	Hand     []Card `json:"hand"`
	HandSize int    `json:"handSize"`
	Active   bool   `json:"active"`
}

type State struct {
	CurrentPlayerIndex int           `json:"currentPlayerIndex"`
	Round              int           `json:"round"`
	Direction          int           `json:"direction"`
	Players            []PlayerState `json:"players"`
	DrawPile           []Card        `json:"drawPile"`
	DiscardPile        []Card        `json:"discardPile"`
	WinnerID           string        `json:"winnerId,omitempty"`
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
	return "onecard"
}

func (m Module) Name() string {
	return "원카드"
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
	for _, player := range ctx.Players {
		hand := append([]Card(nil), deck[:7]...)
		deck = deck[7:]
		players = append(players, PlayerState{PlayerID: string(player.ID), Hand: hand, HandSize: len(hand), Active: true})
	}
	discard := []Card{deck[0]}
	deck = deck[1:]
	return State{Direction: 1, Players: players, DrawPile: deck, DiscardPile: discard, Log: []string{"원카드가 시작되었습니다."}}
}

func (m Module) PublicState(state any, viewerID gamecore.PlayerID) any {
	current := asState(state)
	current.DrawPile = make([]Card, len(current.DrawPile))
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
	case ActionDraw:
		if len(current.DrawPile) == 0 {
			return errors.New("draw pile is empty")
		}
	case ActionPlay:
		payload, err := playPayload(action.Payload)
		if err != nil {
			return err
		}
		card, ok := findCard(current.Players[current.CurrentPlayerIndex].Hand, payload.CardID)
		if !ok {
			return errors.New("card not found")
		}
		if !canPlay(card, topCard(current)) {
			return errors.New("card must match suit or rank")
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
	case ActionDraw:
		player.Hand = append(player.Hand, current.DrawPile[0])
		current.DrawPile = current.DrawPile[1:]
		player.HandSize = len(player.Hand)
		current.Log = append(current.Log, player.PlayerID+" 님이 카드 1장을 뽑았습니다.")
	case ActionPlay:
		payload, err := playPayload(action.Payload)
		if err != nil {
			return gamecore.ActionResult{}, err
		}
		card, _ := removeCard(player, payload.CardID)
		current.DiscardPile = append(current.DiscardPile, card)
		current.Log = append(current.Log, fmt.Sprintf("%s 님이 %s %s 카드를 냈습니다.", player.PlayerID, card.Suit, card.Rank))
		if card.Rank == "A" {
			current.Direction *= -1
		}
		if card.Rank == "2" {
			next := nextActiveIndex(current, current.CurrentPlayerIndex)
			if len(current.DrawPile) >= 2 {
				current.Players[next].Hand = append(current.Players[next].Hand, current.DrawPile[:2]...)
				current.DrawPile = current.DrawPile[2:]
				current.Players[next].HandSize = len(current.Players[next].Hand)
			}
		}
		if len(player.Hand) == 0 {
			current.WinnerID = player.PlayerID
			current.Finished = true
			current.Log = append(current.Log, "원카드가 종료되었습니다.")
		}
	}
	if !current.Finished {
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
		current.Players[index].Active = false
		if activeCount(current) <= 1 {
			current.WinnerID = current.Players[nextActiveIndex(current, index)].PlayerID
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
		outcome := gamecore.OutcomeLose
		if player.PlayerID == current.WinnerID {
			outcome = gamecore.OutcomeWin
		}
		results = append(results, gamecore.Result{PlayerID: gamecore.PlayerID(player.PlayerID), Rank: rankForOutcome(outcome), Score: -len(player.Hand), Outcome: outcome})
	}
	return results
}

func canPlay(card Card, top Card) bool {
	return card.Suit == top.Suit || card.Rank == top.Rank
}

func topCard(state State) Card {
	return state.DiscardPile[len(state.DiscardPile)-1]
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

func removeCard(player *PlayerState, cardID string) (Card, bool) {
	next := []Card{}
	removed := Card{}
	found := false
	for _, card := range player.Hand {
		if !found && card.ID == cardID {
			removed = card
			found = true
			continue
		}
		next = append(next, card)
	}
	player.Hand = next
	player.HandSize = len(next)
	return removed, found
}

func nextActiveIndex(state State, current int) int {
	for step := 1; step <= len(state.Players); step++ {
		next := (current + step*state.Direction + len(state.Players)*2) % len(state.Players)
		if state.Players[next].Active {
			return next
		}
	}
	return current
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
	suits := []string{"spade", "heart", "diamond", "club"}
	ranks := []string{"A", "2", "3", "4", "5", "6", "7", "8", "9", "10", "J", "Q", "K"}
	deck := make([]Card, 0, 52)
	for _, suit := range suits {
		for value, rank := range ranks {
			deck = append(deck, Card{ID: suit + "-" + rank, Suit: suit, Rank: rank, Value: value + 1})
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

func asState(state any) State {
	typed, ok := state.(State)
	if ok {
		return typed
	}
	return State{}
}
