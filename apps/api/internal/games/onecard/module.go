package onecard

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"math/rand"
	"sort"
	"strconv"

	"board-game-platform/apps/api/internal/gamecore"
	"board-game-platform/apps/api/internal/games/internal/gameutil"
)

const (
	ActionPlay       = "onecard.play"
	ActionDraw       = "onecard.draw"
	ActionDeclareOne = "onecard.declare_one"
	ActionCalloutOne = "onecard.callout_one"
)

type Card struct {
	ID    string `json:"id"`
	Suit  string `json:"suit"`
	Rank  string `json:"rank"`
	Value int    `json:"value"`
	Joker bool   `json:"joker"`
}

type PlayerState struct {
	PlayerID string `json:"playerId"`
	Hand     []Card `json:"hand"`
	HandSize int    `json:"handSize"`
	Active   bool   `json:"active"`
}

type State struct {
	CurrentPlayerIndex int             `json:"currentPlayerIndex"`
	Round              int             `json:"round"`
	Direction          int             `json:"direction"`
	Players            []PlayerState   `json:"players"`
	DrawPile           []Card          `json:"drawPile"`
	DiscardPile        []Card          `json:"discardPile"`
	Rules              RuleConfig      `json:"rules"`
	RuleMessages       []string        `json:"ruleMessages,omitempty"`
	PendingDraw        int             `json:"pendingDraw"`
	PendingAttackRank  string          `json:"pendingAttackRank,omitempty"`
	DeclaredSuit       string          `json:"declaredSuit,omitempty"`
	DeclaredOne        map[string]bool `json:"declaredOne,omitempty"`
	WinnerID           string          `json:"winnerId,omitempty"`
	Log                []string        `json:"log"`
	Finished           bool            `json:"finished"`
}

type PlayPayload struct {
	CardID       string `json:"cardId"`
	DeclaredSuit string `json:"declaredSuit,omitempty"`
	DeclareOne   bool   `json:"declareOne,omitempty"`
}

type CalloutOnePayload struct {
	TargetPlayerID string `json:"targetPlayerId"`
}

type RuleConfig struct {
	AttackCards        []string `json:"attackCards"`
	DefenseMode        string   `json:"defenseMode"`
	JokerDrawCount     int      `json:"jokerDrawCount"`
	TwoDrawCount       int      `json:"twoDrawCount"`
	Stacking           bool     `json:"stacking"`
	ChangeSuitCards    []string `json:"changeSuitCards"`
	OneCardPenalty     bool     `json:"oneCardPenalty"`
	OneCardPenaltyDraw int      `json:"oneCardPenaltyDraw"`
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

func (m Module) ResolveRules(votes []gamecore.RuleVote, seed string) gamecore.RuleResolution {
	attackCards, attackTied := resolveChoice(votes, "attackCards", "two-ace-joker", seed)
	defenseMode, defenseTied := resolveChoice(votes, "defenseMode", "attack-or-joker", seed)
	jokerDraw, jokerTied := resolveChoice(votes, "jokerDrawCount", "5", seed)
	stacking, stackingTied := resolveChoice(votes, "stacking", "on", seed)
	changeSuit, changeSuitTied := resolveChoice(votes, "changeSuitCards", "seven-joker", seed)
	oneCardPenalty, oneCardPenaltyTied := resolveChoice(votes, "oneCardPenalty", "on", seed)

	config := RuleConfig{
		AttackCards:        attackCardsForChoice(attackCards),
		DefenseMode:        defenseMode,
		JokerDrawCount:     intChoice(jokerDraw, 5),
		TwoDrawCount:       2,
		Stacking:           stacking != "off",
		ChangeSuitCards:    changeSuitCardsForChoice(changeSuit),
		OneCardPenalty:     oneCardPenalty != "off",
		OneCardPenaltyDraw: 2,
	}
	messages := []string{
		fmt.Sprintf(
			"원카드 룰 확정: 공격 %s, 방어 %s, 조커 %d장, 공격 누적 %s, 문양 변경 %s, 원카드 벌칙 %s",
			attackChoiceLabel(attackCards),
			defenseChoiceLabel(defenseMode),
			config.JokerDrawCount,
			onOffLabel(config.Stacking),
			changeSuitChoiceLabel(changeSuit),
			onOffLabel(config.OneCardPenalty),
		),
	}
	if attackTied {
		messages = append(messages, "공격카드 투표가 동률이라 랜덤으로 결정했습니다.")
	}
	if defenseTied {
		messages = append(messages, "방어카드 투표가 동률이라 랜덤으로 결정했습니다.")
	}
	if jokerTied {
		messages = append(messages, "조커 공격 개수 투표가 동률이라 랜덤으로 결정했습니다.")
	}
	if stackingTied {
		messages = append(messages, "공격 누적 투표가 동률이라 랜덤으로 결정했습니다.")
	}
	if changeSuitTied {
		messages = append(messages, "문양 변경 투표가 동률이라 랜덤으로 결정했습니다.")
	}
	if oneCardPenaltyTied {
		messages = append(messages, "원카드 선언 벌칙 투표가 동률이라 랜덤으로 결정했습니다.")
	}

	return gamecore.RuleResolution{
		Options: map[string]any{
			"attackCards":        config.AttackCards,
			"defenseMode":        config.DefenseMode,
			"jokerDrawCount":     config.JokerDrawCount,
			"twoDrawCount":       config.TwoDrawCount,
			"stacking":           config.Stacking,
			"changeSuitCards":    config.ChangeSuitCards,
			"oneCardPenalty":     config.OneCardPenalty,
			"oneCardPenaltyDraw": config.OneCardPenaltyDraw,
			"ruleMessages":       messages,
		},
		Announcements: messages,
	}
}

func (m Module) CreateInitialState(ctx gamecore.Context) any {
	rules := ruleConfigFromOptions(ctx.Options)
	deck := shuffledDeck(ctx.RandomSeed)
	handSize := 5
	if len(ctx.Players) == 2 {
		handSize = 7
	}
	players := make([]PlayerState, 0, len(ctx.Players))
	for _, player := range ctx.Players {
		hand := append([]Card(nil), deck[:handSize]...)
		deck = deck[handSize:]
		players = append(players, PlayerState{PlayerID: string(player.ID), Hand: hand, HandSize: len(hand), Active: true})
	}
	firstCardIndex := firstDiscardIndex(deck)
	discard := []Card{deck[firstCardIndex]}
	deck = append(deck[:firstCardIndex], deck[firstCardIndex+1:]...)
	log := append([]string{"원카드가 시작되었습니다."}, ruleMessagesFromOptions(ctx.Options)...)
	return State{Direction: 1, Players: players, DrawPile: deck, DiscardPile: discard, Rules: rules, RuleMessages: ruleMessagesFromOptions(ctx.Options), DeclaredOne: map[string]bool{}, Log: log}
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
	switch action.Type {
	case ActionDeclareOne:
		return validateDeclareOne(current, string(action.PlayerID))
	case ActionCalloutOne:
		payload, err := calloutOnePayload(action.Payload)
		if err != nil {
			return err
		}
		return validateCalloutOne(current, string(action.PlayerID), payload.TargetPlayerID)
	}
	if len(current.Players) == 0 || current.Players[current.CurrentPlayerIndex].PlayerID != string(action.PlayerID) {
		return errors.New("not your turn")
	}
	if !current.Players[current.CurrentPlayerIndex].Active {
		return errors.New("player is not active")
	}
	switch action.Type {
	case ActionDraw:
		if availableDrawCount(current) == 0 {
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
		if !canPlay(card, current) {
			return errors.New("card cannot be played now")
		}
		if payload.DeclaredSuit != "" {
			if !isStandardSuit(payload.DeclaredSuit) {
				return errors.New("declared suit must be spade, heart, diamond or club")
			}
			if !canChangeSuit(card, current.Rules) {
				return errors.New("this card cannot change suit")
			}
		}
		if canChangeSuit(card, current.Rules) && payload.DeclaredSuit == "" {
			return errors.New("declared suit is required for this card")
		}
	default:
		return errors.New("unsupported action")
	}
	return nil
}

func (m Module) ApplyAction(_ context.Context, state any, action gamecore.Action, _ gamecore.Context) (gamecore.ActionResult, error) {
	current := asState(state)
	switch action.Type {
	case ActionDeclareOne:
		playerIndex := findPlayer(current, string(action.PlayerID))
		if playerIndex < 0 {
			return gamecore.ActionResult{}, errors.New("player not found")
		}
		ensureDeclaredOne(&current)
		current.DeclaredOne[current.Players[playerIndex].PlayerID] = true
		current.Log = append(current.Log, fmt.Sprintf("%s 님이 원카드를 선언했습니다.", current.Players[playerIndex].PlayerID))
		return stateUpdatedResult(m, current), nil
	case ActionCalloutOne:
		payload, err := calloutOnePayload(action.Payload)
		if err != nil {
			return gamecore.ActionResult{}, err
		}
		targetIndex := findPlayer(current, payload.TargetPlayerID)
		if targetIndex < 0 {
			return gamecore.ActionResult{}, errors.New("target player not found")
		}
		drawn := drawCards(&current, current.Rules.OneCardPenaltyDraw)
		current.Players[targetIndex].Hand = append(current.Players[targetIndex].Hand, drawn...)
		current.Players[targetIndex].HandSize = len(current.Players[targetIndex].Hand)
		if current.DeclaredOne != nil {
			delete(current.DeclaredOne, current.Players[targetIndex].PlayerID)
		}
		current.Log = append(current.Log, fmt.Sprintf("%s 님이 원카드 미선언으로 카드 %d장을 받았습니다.", current.Players[targetIndex].PlayerID, len(drawn)))
		return stateUpdatedResult(m, current), nil
	}

	player := &current.Players[current.CurrentPlayerIndex]
	advance := true
	switch action.Type {
	case ActionDraw:
		drawCount := current.PendingDraw
		if drawCount <= 0 {
			drawCount = 1
		}
		drawn := drawCards(&current, drawCount)
		player.Hand = append(player.Hand, drawn...)
		player.HandSize = len(player.Hand)
		clearDeclaredOne(&current, player.PlayerID)
		current.Log = append(current.Log, fmt.Sprintf("%s 님이 카드 %d장을 뽑았습니다.", player.PlayerID, len(drawn)))
		current.PendingDraw = 0
		current.PendingAttackRank = ""
	case ActionPlay:
		payload, err := playPayload(action.Payload)
		if err != nil {
			return gamecore.ActionResult{}, err
		}
		card, _ := removeCard(player, payload.CardID)
		current.DiscardPile = append(current.DiscardPile, card)
		current.DeclaredSuit = ""
		current.Log = append(current.Log, fmt.Sprintf("%s 님이 %s %s 카드를 냈습니다.", player.PlayerID, card.Suit, card.Rank))
		if payload.DeclaredSuit != "" {
			current.DeclaredSuit = payload.DeclaredSuit
			current.Log = append(current.Log, fmt.Sprintf("%s 문양을 선언했습니다.", suitLabel(payload.DeclaredSuit)))
		}
		if card.Rank == "Q" {
			current.Direction *= -1
		}
		if card.Rank == "J" {
			current.CurrentPlayerIndex = nextActiveIndex(current, current.CurrentPlayerIndex)
			current.Round++
			current.Log = append(current.Log, "다음 순서를 건너뜁니다.")
		}
		if card.Rank == "K" {
			advance = false
			current.Log = append(current.Log, fmt.Sprintf("%s 님이 추가 턴을 얻었습니다.", player.PlayerID))
		}
		if amount := attackAmount(card, current.Rules); amount > 0 {
			if current.Rules.Stacking {
				current.PendingDraw += amount
			} else {
				current.PendingDraw = amount
			}
			current.PendingAttackRank = card.Rank
			current.Log = append(current.Log, fmt.Sprintf("공격 카드로 다음 차례에 %d장의 페널티가 걸렸습니다.", current.PendingDraw))
		} else {
			current.PendingDraw = 0
			current.PendingAttackRank = ""
		}
		if len(player.Hand) == 0 {
			clearDeclaredOne(&current, player.PlayerID)
			current.WinnerID = player.PlayerID
			current.Finished = true
			current.Log = append(current.Log, "원카드가 종료되었습니다.")
		} else if len(player.Hand) == 1 && current.Rules.OneCardPenalty {
			ensureDeclaredOne(&current)
			current.DeclaredOne[player.PlayerID] = payload.DeclareOne
			if payload.DeclareOne {
				current.Log = append(current.Log, fmt.Sprintf("%s 님이 원카드를 선언했습니다.", player.PlayerID))
			} else {
				current.Log = append(current.Log, fmt.Sprintf("%s 님의 손패가 한 장 남았습니다.", player.PlayerID))
			}
		} else {
			clearDeclaredOne(&current, player.PlayerID)
		}
	}
	if !current.Finished && advance {
		current.CurrentPlayerIndex = nextActiveIndex(current, current.CurrentPlayerIndex)
		current.Round++
	}
	return stateUpdatedResult(m, current), nil
}

func (m Module) ApplyTimeout(_ context.Context, state any, playerID gamecore.PlayerID, _ gamecore.Context) (gamecore.ActionResult, error) {
	current := asState(state)
	if index := findPlayer(current, string(playerID)); index >= 0 && !current.Finished {
		if current.CurrentPlayerIndex == index && current.PendingDraw > 0 {
			drawn := drawCards(&current, current.PendingDraw)
			current.Players[index].Hand = append(current.Players[index].Hand, drawn...)
			current.Players[index].HandSize = len(current.Players[index].Hand)
			clearDeclaredOne(&current, current.Players[index].PlayerID)
			current.Log = append(current.Log, fmt.Sprintf("%s 님이 시간 초과로 공격 카드 %d장을 받았습니다.", current.Players[index].PlayerID, len(drawn)))
			current.PendingDraw = 0
			current.PendingAttackRank = ""
		}
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

func canPlay(card Card, state State) bool {
	if state.PendingDraw > 0 {
		return canDefend(card, topCard(state), state)
	}
	if card.Joker {
		return true
	}
	top := topCard(state)
	topSuit := activeTopSuit(state)
	if top.Joker && topSuit == "" {
		return true
	}
	return card.Suit == topSuit || card.Rank == top.Rank
}

func canDefend(card Card, top Card, state State) bool {
	isAttackCard := attackAmount(card, state.Rules) > 0
	switch state.Rules.DefenseMode {
	case "same-rank":
		if card.Joker || top.Joker {
			return card.Joker && top.Joker
		}
		return isAttackCard && card.Rank == top.Rank
	case "any-attack":
		return isAttackCard
	default:
		return isAttackCard || card.Joker
	}
}

func attackAmount(card Card, rules RuleConfig) int {
	if card.Joker && containsRuleCard(rules.AttackCards, "JOKER") {
		return rules.JokerDrawCount
	}
	if card.Rank == "2" && containsRuleCard(rules.AttackCards, "2") {
		return rules.TwoDrawCount
	}
	if card.Rank == "A" && containsRuleCard(rules.AttackCards, "A") {
		return 3
	}
	return 0
}

func topCard(state State) Card {
	return state.DiscardPile[len(state.DiscardPile)-1]
}

func activeTopSuit(state State) string {
	if state.DeclaredSuit != "" {
		return state.DeclaredSuit
	}
	return topCard(state).Suit
}

func canChangeSuit(card Card, rules RuleConfig) bool {
	if len(rules.ChangeSuitCards) == 0 {
		return false
	}
	if card.Joker {
		return containsRuleCard(rules.ChangeSuitCards, "JOKER")
	}
	return containsRuleCard(rules.ChangeSuitCards, card.Rank)
}

func isStandardSuit(suit string) bool {
	return suit == "spade" || suit == "heart" || suit == "diamond" || suit == "club"
}

func validateDeclareOne(state State, playerID string) error {
	if !state.Rules.OneCardPenalty {
		return errors.New("one-card penalty rule is disabled")
	}
	index := findPlayer(state, playerID)
	if index < 0 {
		return errors.New("player not found")
	}
	if !state.Players[index].Active {
		return errors.New("player is not active")
	}
	if len(state.Players[index].Hand) != 1 {
		return errors.New("one-card declaration requires exactly one card")
	}
	return nil
}

func validateCalloutOne(state State, actorID string, targetPlayerID string) error {
	if !state.Rules.OneCardPenalty {
		return errors.New("one-card penalty rule is disabled")
	}
	if actorID == targetPlayerID {
		return errors.New("cannot call out yourself")
	}
	actorIndex := findPlayer(state, actorID)
	if actorIndex < 0 || !state.Players[actorIndex].Active {
		return errors.New("player is not active")
	}
	targetIndex := findPlayer(state, targetPlayerID)
	if targetIndex < 0 {
		return errors.New("target player not found")
	}
	if len(state.Players[targetIndex].Hand) != 1 {
		return errors.New("target does not have exactly one card")
	}
	if state.DeclaredOne != nil && state.DeclaredOne[targetPlayerID] {
		return errors.New("target already declared one-card")
	}
	if availableDrawCount(state) == 0 {
		return errors.New("draw pile is empty")
	}
	return nil
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
	result := PlayPayload{CardID: cardID}
	if declaredSuit, ok := raw["declaredSuit"].(string); ok {
		result.DeclaredSuit = declaredSuit
	}
	if declareOne, ok := raw["declareOne"].(bool); ok {
		result.DeclareOne = declareOne
	}
	return result, nil
}

func calloutOnePayload(payload any) (CalloutOnePayload, error) {
	raw, ok := payload.(map[string]any)
	if !ok {
		return CalloutOnePayload{}, errors.New("invalid callout payload")
	}
	targetPlayerID, _ := raw["targetPlayerId"].(string)
	if targetPlayerID == "" {
		return CalloutOnePayload{}, errors.New("target player is required")
	}
	return CalloutOnePayload{TargetPlayerID: targetPlayerID}, nil
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

func ensureDeclaredOne(state *State) {
	if state.DeclaredOne == nil {
		state.DeclaredOne = map[string]bool{}
	}
}

func clearDeclaredOne(state *State, playerID string) {
	if state.DeclaredOne == nil {
		return
	}
	delete(state.DeclaredOne, playerID)
}

func stateUpdatedResult(module Module, state State) gamecore.ActionResult {
	return gamecore.ActionResult{
		State: state,
		Events: []gamecore.Event{{
			Type:       "game.state_updated",
			Visibility: gamecore.VisibilityPublic,
			Payload:    module.PublicState(state, ""),
		}},
	}
}

func nextActiveIndex(state State, current int) int {
	direction := state.Direction
	if direction == 0 {
		direction = 1
	}
	for step := 1; step <= len(state.Players); step++ {
		next := (current + step*direction + len(state.Players)*2) % len(state.Players)
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
	deck := make([]Card, 0, 54)
	for _, suit := range suits {
		for value, rank := range ranks {
			deck = append(deck, Card{ID: suit + "-" + rank, Suit: suit, Rank: rank, Value: value + 1})
		}
	}
	deck = append(deck, Card{ID: "joker-black", Suit: "joker", Rank: "JOKER", Value: 15, Joker: true})
	deck = append(deck, Card{ID: "joker-color", Suit: "joker", Rank: "JOKER", Value: 15, Joker: true})
	random := rand.New(rand.NewSource(seedToInt(seed)))
	random.Shuffle(len(deck), func(i, j int) {
		deck[i], deck[j] = deck[j], deck[i]
	})
	return deck
}

func firstDiscardIndex(deck []Card) int {
	for index, card := range deck {
		if !card.Joker && card.Rank != "2" && card.Rank != "A" && card.Rank != "J" && card.Rank != "Q" && card.Rank != "K" && card.Rank != "7" {
			return index
		}
	}
	return 0
}

func drawCards(state *State, count int) []Card {
	drawn := make([]Card, 0, count)
	for len(drawn) < count && availableDrawCount(*state) > 0 {
		if len(state.DrawPile) == 0 {
			recycleDiscardPile(state)
		}
		if len(state.DrawPile) == 0 {
			break
		}
		drawn = append(drawn, state.DrawPile[0])
		state.DrawPile = state.DrawPile[1:]
	}
	return drawn
}

func availableDrawCount(state State) int {
	count := len(state.DrawPile)
	if len(state.DiscardPile) > 1 {
		count += len(state.DiscardPile) - 1
	}
	return count
}

func recycleDiscardPile(state *State) {
	if len(state.DiscardPile) <= 1 {
		return
	}
	top := state.DiscardPile[len(state.DiscardPile)-1]
	recycled := append([]Card(nil), state.DiscardPile[:len(state.DiscardPile)-1]...)
	random := rand.New(rand.NewSource(seedToInt(fmt.Sprintf("%s:%d", top.ID, len(recycled)))))
	random.Shuffle(len(recycled), func(i, j int) {
		recycled[i], recycled[j] = recycled[j], recycled[i]
	})
	state.DrawPile = recycled
	state.DiscardPile = []Card{top}
}

func ruleConfigFromOptions(options map[string]any) RuleConfig {
	config := RuleConfig{
		AttackCards:        []string{"2", "A", "JOKER"},
		DefenseMode:        "attack-or-joker",
		JokerDrawCount:     5,
		TwoDrawCount:       2,
		Stacking:           true,
		ChangeSuitCards:    []string{"7", "JOKER"},
		OneCardPenalty:     true,
		OneCardPenaltyDraw: 2,
	}
	if raw, ok := options["attackCards"]; ok {
		config.AttackCards = stringList(raw, config.AttackCards)
	}
	if raw, ok := options["defenseMode"].(string); ok && raw != "" {
		config.DefenseMode = raw
	}
	config.JokerDrawCount = intOption(options["jokerDrawCount"], config.JokerDrawCount)
	config.TwoDrawCount = intOption(options["twoDrawCount"], config.TwoDrawCount)
	if raw, ok := options["stacking"].(bool); ok {
		config.Stacking = raw
	}
	if raw, ok := options["changeSuitCards"]; ok {
		config.ChangeSuitCards = stringList(raw, config.ChangeSuitCards)
	}
	if raw, ok := options["oneCardPenalty"].(bool); ok {
		config.OneCardPenalty = raw
	}
	config.OneCardPenaltyDraw = intOption(options["oneCardPenaltyDraw"], config.OneCardPenaltyDraw)
	return config
}

func ruleMessagesFromOptions(options map[string]any) []string {
	return stringList(options["ruleMessages"], nil)
}

func resolveChoice(votes []gamecore.RuleVote, key string, fallback string, seed string) (string, bool) {
	counts := map[string]int{}
	for _, vote := range votes {
		if choice := vote.Choices[key]; choice != "" {
			counts[choice]++
		}
	}
	if len(counts) == 0 {
		return fallback, false
	}
	choices := make([]string, 0, len(counts))
	best := 0
	for choice, count := range counts {
		if count > best {
			best = count
			choices = choices[:0]
		}
		if count == best {
			choices = append(choices, choice)
		}
	}
	sort.Strings(choices)
	if len(choices) == 1 {
		return choices[0], false
	}
	random := rand.New(rand.NewSource(seedToInt(seed + ":" + key)))
	return choices[random.Intn(len(choices))], true
}

func attackCardsForChoice(choice string) []string {
	switch choice {
	case "two":
		return []string{"2"}
	case "two-ace":
		return []string{"2", "A"}
	default:
		return []string{"2", "A", "JOKER"}
	}
}

func changeSuitCardsForChoice(choice string) []string {
	switch choice {
	case "off":
		return []string{}
	case "seven":
		return []string{"7"}
	case "joker":
		return []string{"JOKER"}
	case "queen-joker":
		return []string{"Q", "JOKER"}
	default:
		return []string{"7", "JOKER"}
	}
}

func intChoice(choice string, fallback int) int {
	value, err := strconv.Atoi(choice)
	if err != nil {
		return fallback
	}
	return value
}

func intOption(value any, fallback int) int {
	if typed, ok := gameutil.Int(value); ok {
		return typed
	}
	return fallback
}

func stringList(value any, fallback []string) []string {
	switch typed := value.(type) {
	case []string:
		return append([]string(nil), typed...)
	case []any:
		result := make([]string, 0, len(typed))
		for _, item := range typed {
			if value, ok := item.(string); ok {
				result = append(result, value)
			}
		}
		if len(result) > 0 {
			return result
		}
	}
	return append([]string(nil), fallback...)
}

func containsRuleCard(cards []string, target string) bool {
	for _, card := range cards {
		if card == target {
			return true
		}
	}
	return false
}

func attackChoiceLabel(choice string) string {
	labels := map[string]string{
		"two":           "2만",
		"two-ace":       "2/A",
		"two-ace-joker": "2/A/조커",
	}
	return labels[choice]
}

func defenseChoiceLabel(choice string) string {
	labels := map[string]string{
		"same-rank":       "같은 공격카드만",
		"any-attack":      "공격카드",
		"attack-or-joker": "공격카드/조커",
	}
	return labels[choice]
}

func changeSuitChoiceLabel(choice string) string {
	labels := map[string]string{
		"off":         "없음",
		"seven":       "7",
		"joker":       "조커",
		"queen-joker": "Q/조커",
		"seven-joker": "7/조커",
	}
	return labels[choice]
}

func suitLabel(suit string) string {
	labels := map[string]string{
		"spade":   "스페이드",
		"heart":   "하트",
		"diamond": "다이아",
		"club":    "클럽",
	}
	if label, ok := labels[suit]; ok {
		return label
	}
	return suit
}

func onOffLabel(enabled bool) string {
	if enabled {
		return "허용"
	}
	return "없음"
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
