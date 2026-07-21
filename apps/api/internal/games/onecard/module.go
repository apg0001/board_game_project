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
	Joker bool   `json:"joker"`
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
	Rules              RuleConfig    `json:"rules"`
	RuleMessages       []string      `json:"ruleMessages,omitempty"`
	PendingDraw        int           `json:"pendingDraw"`
	PendingAttackRank  string        `json:"pendingAttackRank,omitempty"`
	WinnerID           string        `json:"winnerId,omitempty"`
	Log                []string      `json:"log"`
	Finished           bool          `json:"finished"`
}

type PlayPayload struct {
	CardID string `json:"cardId"`
}

type RuleConfig struct {
	AttackCards    []string `json:"attackCards"`
	DefenseMode    string   `json:"defenseMode"`
	JokerDrawCount int      `json:"jokerDrawCount"`
	TwoDrawCount   int      `json:"twoDrawCount"`
	Stacking       bool     `json:"stacking"`
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

	config := RuleConfig{
		AttackCards:    attackCardsForChoice(attackCards),
		DefenseMode:    defenseMode,
		JokerDrawCount: intChoice(jokerDraw, 5),
		TwoDrawCount:   2,
		Stacking:       stacking != "off",
	}
	messages := []string{
		fmt.Sprintf("원카드 룰 확정: 공격 %s, 방어 %s, 조커 %d장, 공격 누적 %s", attackChoiceLabel(attackCards), defenseChoiceLabel(defenseMode), config.JokerDrawCount, onOffLabel(config.Stacking)),
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

	return gamecore.RuleResolution{
		Options: map[string]any{
			"attackCards":    config.AttackCards,
			"defenseMode":    config.DefenseMode,
			"jokerDrawCount": config.JokerDrawCount,
			"twoDrawCount":   config.TwoDrawCount,
			"stacking":       config.Stacking,
			"ruleMessages":   messages,
		},
		Announcements: messages,
	}
}

func (m Module) CreateInitialState(ctx gamecore.Context) any {
	rules := ruleConfigFromOptions(ctx.Options)
	deck := shuffledDeck(ctx.RandomSeed)
	players := make([]PlayerState, 0, len(ctx.Players))
	for _, player := range ctx.Players {
		hand := append([]Card(nil), deck[:7]...)
		deck = deck[7:]
		players = append(players, PlayerState{PlayerID: string(player.ID), Hand: hand, HandSize: len(hand), Active: true})
	}
	firstCardIndex := firstDiscardIndex(deck)
	discard := []Card{deck[firstCardIndex]}
	deck = append(deck[:firstCardIndex], deck[firstCardIndex+1:]...)
	log := append([]string{"원카드가 시작되었습니다."}, ruleMessagesFromOptions(ctx.Options)...)
	return State{Direction: 1, Players: players, DrawPile: deck, DiscardPile: discard, Rules: rules, RuleMessages: ruleMessagesFromOptions(ctx.Options), Log: log}
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
		drawCount := current.PendingDraw
		if drawCount <= 0 {
			drawCount = 1
		}
		drawn := drawCards(&current, drawCount)
		player.Hand = append(player.Hand, drawn...)
		player.HandSize = len(player.Hand)
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
		current.Log = append(current.Log, fmt.Sprintf("%s 님이 %s %s 카드를 냈습니다.", player.PlayerID, card.Suit, card.Rank))
		if card.Rank == "A" {
			current.Direction *= -1
		}
		if card.Rank == "J" {
			current.CurrentPlayerIndex = nextActiveIndex(current, current.CurrentPlayerIndex)
			current.Round++
			current.Log = append(current.Log, "다음 순서를 건너뜁니다.")
		}
		if amount := attackAmount(card, current.Rules); amount > 0 {
			if current.Rules.Stacking {
				current.PendingDraw += amount
				current.PendingAttackRank = card.Rank
				current.Log = append(current.Log, fmt.Sprintf("공격이 %d장으로 누적되었습니다.", current.PendingDraw))
			} else {
				next := nextActiveIndex(current, current.CurrentPlayerIndex)
				drawn := drawCards(&current, amount)
				current.Players[next].Hand = append(current.Players[next].Hand, drawn...)
				current.Players[next].HandSize = len(current.Players[next].Hand)
				current.CurrentPlayerIndex = next
				current.Round++
				current.Log = append(current.Log, fmt.Sprintf("%s 님이 공격으로 카드 %d장을 받았습니다.", current.Players[next].PlayerID, len(drawn)))
			}
		} else {
			current.PendingDraw = 0
			current.PendingAttackRank = ""
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

func canPlay(card Card, state State) bool {
	if state.PendingDraw > 0 {
		return canDefend(card, topCard(state), state)
	}
	if card.Joker {
		return true
	}
	top := topCard(state)
	if top.Joker {
		return true
	}
	return card.Suit == top.Suit || card.Rank == top.Rank
}

func canDefend(card Card, top Card, state State) bool {
	if attackAmount(card, state.Rules) <= 0 {
		return false
	}
	switch state.Rules.DefenseMode {
	case "same-rank":
		return card.Joker && top.Joker || card.Rank == top.Rank
	case "any-attack":
		return !card.Joker
	default:
		return true
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
		if !card.Joker && card.Rank != "2" && card.Rank != "A" && card.Rank != "J" {
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
		AttackCards:    []string{"2", "A", "JOKER"},
		DefenseMode:    "attack-or-joker",
		JokerDrawCount: 5,
		TwoDrawCount:   2,
		Stacking:       true,
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

func intChoice(choice string, fallback int) int {
	value, err := strconv.Atoi(choice)
	if err != nil {
		return fallback
	}
	return value
}

func intOption(value any, fallback int) int {
	switch typed := value.(type) {
	case int:
		return typed
	case float64:
		return int(typed)
	default:
		return fallback
	}
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
