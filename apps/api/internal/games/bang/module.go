package bang

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"math/rand"

	"board-game-platform/apps/api/internal/gamecore"
)

const (
	ActionDraw      = "bang.draw"
	ActionPlay      = "bang.play"
	ActionEndTurn   = "bang.end_turn"
	ActionUseMissed = "bang.use_missed"
	ActionTakeHit   = "bang.take_hit"
	ActionDiscard   = "bang.discard"
	CardBang        = "bang"
	CardMissed      = "missed"
	CardBeer        = "beer"
	CardGatling     = "gatling"
	CardScope       = "scope"
	CardMustang     = "mustang"
	CardVolcanic    = "volcanic"
	CardSchofield   = "schofield"
	CardRemington   = "remington"
	CardCarabine    = "carabine"
	CardWinchester  = "winchester"
	RoleSheriff     = "sheriff"
	RoleDeputy      = "deputy"
	RoleOutlaw      = "outlaw"
	RoleRenegade    = "renegade"
)

type Card struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

type PlayerState struct {
	PlayerID  string `json:"playerId"`
	Role      string `json:"role,omitempty"`
	HP        int    `json:"hp"`
	MaxHP     int    `json:"maxHp"`
	Hand      []Card `json:"hand"`
	Equipment []Card `json:"equipment"`
	HandSize  int    `json:"handSize"`
	Alive     bool   `json:"alive"`
	Drawn     bool   `json:"drawn"`
	BangUsed  bool   `json:"bangUsed"`
	Active    bool   `json:"active"`
}

type PendingAttack struct {
	SourcePlayerID     string   `json:"sourcePlayerId"`
	TargetPlayerID     string   `json:"targetPlayerId"`
	CardType           string   `json:"cardType"`
	Damage             int      `json:"damage"`
	RemainingTargetIDs []string `json:"remainingTargetIds"`
}

type State struct {
	CurrentPlayerIndex  int            `json:"currentPlayerIndex"`
	Round               int            `json:"round"`
	Players             []PlayerState  `json:"players"`
	Deck                []Card         `json:"deck"`
	Discard             []Card         `json:"discard"`
	PendingAttack       *PendingAttack `json:"pendingAttack,omitempty"`
	PendingDiscardID    string         `json:"pendingDiscardPlayerId,omitempty"`
	PendingDiscardCount int            `json:"pendingDiscardCount,omitempty"`
	Winner              string         `json:"winner,omitempty"`
	Log                 []string       `json:"log"`
	Finished            bool           `json:"finished"`
}

type PlayPayload struct {
	CardID         string `json:"cardId"`
	TargetPlayerID string `json:"targetPlayerId"`
}

type DiscardPayload struct {
	CardIDs []string `json:"cardIds"`
}

type Module struct{}

func NewModule() Module {
	return Module{}
}

func (m Module) ID() gamecore.GameID {
	return "bang"
}

func (m Module) Name() string {
	return "뱅!"
}

func (m Module) MinPlayers() int {
	return 4
}

func (m Module) MaxPlayers() int {
	return 7
}

func (m Module) CreateInitialState(ctx gamecore.Context) any {
	roles := rolesFor(len(ctx.Players))
	deck := shuffledDeck(ctx.RandomSeed)
	players := make([]PlayerState, 0, len(ctx.Players))
	for index, player := range ctx.Players {
		maxHP := 4
		if roles[index] == RoleSheriff {
			maxHP = 5
		}
		hand := append([]Card(nil), deck[:maxHP]...)
		deck = deck[maxHP:]
		players = append(players, PlayerState{
			PlayerID: string(player.ID),
			Role:     roles[index],
			HP:       maxHP,
			MaxHP:    maxHP,
			Hand:     hand,
			HandSize: len(hand),
			Alive:    true,
			Active:   true,
		})
	}
	return State{
		Players: players,
		Deck:    deck,
		Log:     []string{"뱅! 결투가 시작되었습니다."},
	}
}

func (m Module) PublicState(state any, viewerID gamecore.PlayerID) any {
	current := asState(state)
	current.Deck = make([]Card, len(current.Deck))
	current.Players = append([]PlayerState(nil), current.Players...)
	revealed := current.Finished
	for index := range current.Players {
		player := &current.Players[index]
		player.HandSize = len(player.Hand)
		if player.PlayerID != string(viewerID) {
			player.Hand = []Card{}
		}
		if !revealed && player.Role != RoleSheriff && player.PlayerID != string(viewerID) && player.Alive {
			player.Role = ""
		}
	}
	return current
}

func (m Module) ValidateAction(_ context.Context, state any, action gamecore.Action, _ gamecore.Context) error {
	current := asState(state)
	if current.Finished {
		return errors.New("game is already finished")
	}
	if action.Type == ActionUseMissed || action.Type == ActionTakeHit {
		return validatePendingAttackAction(current, action)
	}
	if current.PendingAttack != nil {
		return errors.New("pending attack response")
	}
	if action.Type == ActionDiscard {
		return validatePendingDiscardAction(current, action)
	}
	if current.PendingDiscardID != "" {
		return errors.New("pending discard")
	}
	if len(current.Players) == 0 || current.Players[current.CurrentPlayerIndex].PlayerID != string(action.PlayerID) {
		return errors.New("not your turn")
	}
	player := current.Players[current.CurrentPlayerIndex]
	if !player.Alive {
		return errors.New("player is eliminated")
	}
	switch action.Type {
	case ActionDraw:
		if player.Drawn {
			return errors.New("already drawn this turn")
		}
		if availableDrawCount(current) == 0 {
			return errors.New("deck is empty")
		}
	case ActionEndTurn:
		if !player.Drawn && availableDrawCount(current) > 0 {
			return errors.New("must draw before ending turn")
		}
		return nil
	case ActionPlay:
		if !player.Drawn {
			return errors.New("must draw before playing cards")
		}
		payload, err := playPayload(action.Payload)
		if err != nil {
			return err
		}
		card, ok := findCard(player.Hand, payload.CardID)
		if !ok {
			return errors.New("card not found")
		}
		if card.Type == CardBang && player.BangUsed && !hasEquipment(player, CardVolcanic) {
			return errors.New("only one bang per turn")
		}
		switch card.Type {
		case CardBeer:
			if player.HP >= player.MaxHP {
				return errors.New("beer can only heal missing hp")
			}
			if alivePlayerCount(current) <= 2 {
				return errors.New("beer has no effect with two players left")
			}
		case CardBang:
			targetIndex := findPlayer(current, payload.TargetPlayerID)
			if targetIndex < 0 || !current.Players[targetIndex].Alive || payload.TargetPlayerID == player.PlayerID {
				return errors.New("invalid target")
			}
			if attackDistance(current, current.CurrentPlayerIndex, targetIndex) > attackRange(player) {
				return errors.New("target is out of range")
			}
		case CardGatling:
			return nil
		case CardScope, CardMustang, CardVolcanic, CardSchofield, CardRemington, CardCarabine, CardWinchester:
			return nil
		default:
			return errors.New("unsupported card")
		}
	default:
		return errors.New("unsupported action")
	}
	return nil
}

func validatePendingDiscardAction(state State, action gamecore.Action) error {
	if state.PendingDiscardID == "" || state.PendingDiscardCount <= 0 {
		return errors.New("no pending discard")
	}
	if state.PendingDiscardID != string(action.PlayerID) {
		return errors.New("not your discard")
	}
	playerIndex := findPlayer(state, string(action.PlayerID))
	if playerIndex < 0 || !state.Players[playerIndex].Alive {
		return errors.New("player is not active")
	}
	payload, err := discardPayload(action.Payload)
	if err != nil {
		return err
	}
	if len(payload.CardIDs) != state.PendingDiscardCount {
		return fmt.Errorf("must discard %d cards", state.PendingDiscardCount)
	}
	seen := map[string]bool{}
	for _, cardID := range payload.CardIDs {
		if seen[cardID] {
			return errors.New("duplicate discard card")
		}
		seen[cardID] = true
		if _, ok := findCard(state.Players[playerIndex].Hand, cardID); !ok {
			return errors.New("discard card not found")
		}
	}
	return nil
}

func validatePendingAttackAction(state State, action gamecore.Action) error {
	if state.PendingAttack == nil || state.PendingAttack.TargetPlayerID == "" {
		return errors.New("no pending attack")
	}
	if state.PendingAttack.TargetPlayerID != string(action.PlayerID) {
		return errors.New("not your attack response")
	}
	targetIndex := findPlayer(state, string(action.PlayerID))
	if targetIndex < 0 || !state.Players[targetIndex].Alive {
		return errors.New("target is not active")
	}
	switch action.Type {
	case ActionUseMissed:
		if _, ok := findCardByType(state.Players[targetIndex].Hand, CardMissed); !ok {
			return errors.New("missed card not found")
		}
	case ActionTakeHit:
		return nil
	default:
		return errors.New("unsupported pending action")
	}
	return nil
}

func (m Module) ApplyAction(_ context.Context, state any, action gamecore.Action, _ gamecore.Context) (gamecore.ActionResult, error) {
	current := asState(state)
	player := &current.Players[current.CurrentPlayerIndex]
	switch action.Type {
	case ActionDraw:
		drawn := drawCards(&current, 2)
		player.Hand = append(player.Hand, drawn...)
		player.HandSize = len(player.Hand)
		player.Drawn = true
		current.Log = append(current.Log, fmt.Sprintf("%s 님이 카드 %d장을 뽑았습니다.", player.PlayerID, len(drawn)))
	case ActionPlay:
		payload, err := playPayload(action.Payload)
		if err != nil {
			return gamecore.ActionResult{}, err
		}
		card, _ := removeCard(player, payload.CardID)
		switch card.Type {
		case CardBeer:
			current.Discard = append(current.Discard, card)
			if player.HP < player.MaxHP {
				player.HP++
			}
			current.Log = append(current.Log, player.PlayerID+" 님이 맥주로 회복했습니다.")
		case CardBang:
			current.Discard = append(current.Discard, card)
			if !hasEquipment(*player, CardVolcanic) {
				player.BangUsed = true
			}
			current.Log = append(current.Log, player.PlayerID+" 님이 BANG!을 사용했습니다.")
			current = startPendingAttack(current, player.PlayerID, card.Type, []string{payload.TargetPlayerID}, 1)
		case CardGatling:
			current.Discard = append(current.Discard, card)
			current.Log = append(current.Log, player.PlayerID+" 님이 개틀링을 사용했습니다.")
			current = startPendingAttack(current, player.PlayerID, card.Type, attackTargets(current, player.PlayerID), 1)
		case CardScope, CardMustang, CardVolcanic, CardSchofield, CardRemington, CardCarabine, CardWinchester:
			current = equipCard(current, current.CurrentPlayerIndex, card)
		}
		current = checkEnd(current)
	case ActionUseMissed:
		current = resolvePendingAttackWithMissed(current, string(action.PlayerID))
	case ActionTakeHit:
		current = resolvePendingAttackWithDamage(current)
	case ActionDiscard:
		payload, err := discardPayload(action.Payload)
		if err != nil {
			return gamecore.ActionResult{}, err
		}
		current = applyPendingDiscard(current, string(action.PlayerID), payload.CardIDs)
	case ActionEndTurn:
		if excess := handLimitExcess(*player); excess > 0 {
			current.PendingDiscardID = player.PlayerID
			current.PendingDiscardCount = excess
			current.Log = append(current.Log, fmt.Sprintf("%s 님이 손패 제한으로 카드 %d장을 버려야 합니다.", player.PlayerID, excess))
		} else {
			current = finishTurn(current)
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
		wasPendingAttackTarget := current.PendingAttack != nil && current.PendingAttack.TargetPlayerID == string(playerID)
		current.Players[index].HP = 0
		current.Players[index].Alive = false
		current.Players[index].Active = false
		current.Discard = append(current.Discard, current.Players[index].Hand...)
		current.Discard = append(current.Discard, current.Players[index].Equipment...)
		current.Players[index].Hand = []Card{}
		current.Players[index].Equipment = []Card{}
		current.Players[index].HandSize = 0
		if current.PendingDiscardID == string(playerID) {
			current.PendingDiscardID = ""
			current.PendingDiscardCount = 0
		}
		current = checkEnd(current)
		if wasPendingAttackTarget && !current.Finished {
			current.PendingAttack.TargetPlayerID = ""
			current = advancePendingAttack(current)
		}
		if current.CurrentPlayerIndex == index && !current.Finished {
			current.CurrentPlayerIndex = nextAliveIndex(current, index)
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
		if winningRole(player.Role, current.Winner) {
			outcome = gamecore.OutcomeWin
		}
		results = append(results, gamecore.Result{
			PlayerID: gamecore.PlayerID(player.PlayerID),
			Rank:     rankForOutcome(outcome),
			Score:    player.HP,
			Outcome:  outcome,
		})
	}
	return results
}

func applyPendingDiscard(state State, playerID string, cardIDs []string) State {
	playerIndex := findPlayer(state, playerID)
	if playerIndex < 0 {
		return state
	}
	player := &state.Players[playerIndex]
	for _, cardID := range cardIDs {
		if card, ok := removeCard(player, cardID); ok {
			state.Discard = append(state.Discard, card)
		}
	}
	if excess := handLimitExcess(*player); excess > 0 {
		state.PendingDiscardID = player.PlayerID
		state.PendingDiscardCount = excess
		state.Log = append(state.Log, fmt.Sprintf("%s 님이 카드 %d장을 더 버려야 합니다.", player.PlayerID, excess))
		return state
	}
	state.PendingDiscardID = ""
	state.PendingDiscardCount = 0
	state.Log = append(state.Log, player.PlayerID+" 님이 손패 제한을 맞췄습니다.")
	return finishTurn(state)
}

func equipCard(state State, playerIndex int, card Card) State {
	player := &state.Players[playerIndex]
	if isWeapon(card.Type) {
		kept := make([]Card, 0, len(player.Equipment))
		for _, equipment := range player.Equipment {
			if isWeapon(equipment.Type) {
				state.Discard = append(state.Discard, equipment)
				continue
			}
			kept = append(kept, equipment)
		}
		player.Equipment = append(kept, card)
		state.Log = append(state.Log, player.PlayerID+" 님이 무기를 장착했습니다.")
		return state
	}
	kept := make([]Card, 0, len(player.Equipment))
	for _, equipment := range player.Equipment {
		if equipment.Type == card.Type {
			state.Discard = append(state.Discard, equipment)
			continue
		}
		kept = append(kept, equipment)
	}
	player.Equipment = append(kept, card)
	state.Log = append(state.Log, player.PlayerID+" 님이 장비를 장착했습니다.")
	return state
}

func attackRange(player PlayerState) int {
	for _, equipment := range player.Equipment {
		if rangeValue := weaponRange(equipment.Type); rangeValue > 0 {
			return rangeValue
		}
	}
	return 1
}

func attackDistance(state State, sourceIndex int, targetIndex int) int {
	distance := seatDistance(state, sourceIndex, targetIndex)
	if hasEquipment(state.Players[sourceIndex], CardScope) {
		distance--
	}
	if hasEquipment(state.Players[targetIndex], CardMustang) {
		distance++
	}
	if distance < 1 {
		return 1
	}
	return distance
}

func seatDistance(state State, sourceIndex int, targetIndex int) int {
	clockwise := aliveDistance(state, sourceIndex, targetIndex, 1)
	counterClockwise := aliveDistance(state, sourceIndex, targetIndex, -1)
	if clockwise < counterClockwise {
		return clockwise
	}
	return counterClockwise
}

func aliveDistance(state State, sourceIndex int, targetIndex int, step int) int {
	if sourceIndex == targetIndex {
		return 0
	}
	distance := 0
	for offset := step; ; offset += step {
		index := (sourceIndex + offset + len(state.Players)*len(state.Players)) % len(state.Players)
		if state.Players[index].Alive {
			distance++
		}
		if index == targetIndex {
			return distance
		}
	}
}

func hasEquipment(player PlayerState, cardType string) bool {
	for _, equipment := range player.Equipment {
		if equipment.Type == cardType {
			return true
		}
	}
	return false
}

func isWeapon(cardType string) bool {
	return weaponRange(cardType) > 0
}

func weaponRange(cardType string) int {
	switch cardType {
	case CardVolcanic:
		return 1
	case CardSchofield:
		return 2
	case CardRemington:
		return 3
	case CardCarabine:
		return 4
	case CardWinchester:
		return 5
	default:
		return 0
	}
}

func finishTurn(state State) State {
	if len(state.Players) == 0 {
		return state
	}
	player := &state.Players[state.CurrentPlayerIndex]
	player.Drawn = false
	player.BangUsed = false
	state.CurrentPlayerIndex = nextAliveIndex(state, state.CurrentPlayerIndex)
	state.Round++
	state.Log = append(state.Log, "턴이 넘어갔습니다.")
	return state
}

func handLimitExcess(player PlayerState) int {
	if len(player.Hand) <= player.HP {
		return 0
	}
	return len(player.Hand) - player.HP
}

func startPendingAttack(state State, sourcePlayerID string, cardType string, targetPlayerIDs []string, damage int) State {
	state.PendingAttack = &PendingAttack{
		SourcePlayerID:     sourcePlayerID,
		CardType:           cardType,
		Damage:             damage,
		RemainingTargetIDs: append([]string(nil), targetPlayerIDs...),
	}
	return advancePendingAttack(state)
}

func advancePendingAttack(state State) State {
	for state.PendingAttack != nil {
		if len(state.PendingAttack.RemainingTargetIDs) == 0 {
			state.PendingAttack = nil
			return state
		}
		targetPlayerID := state.PendingAttack.RemainingTargetIDs[0]
		state.PendingAttack.RemainingTargetIDs = state.PendingAttack.RemainingTargetIDs[1:]
		targetIndex := findPlayer(state, targetPlayerID)
		if targetIndex < 0 || !state.Players[targetIndex].Alive {
			continue
		}
		if _, ok := findCardByType(state.Players[targetIndex].Hand, CardMissed); ok {
			state.PendingAttack.TargetPlayerID = targetPlayerID
			state.Log = append(state.Log, targetPlayerID+" 님의 빗맞음 반응을 기다립니다.")
			return state
		}
		pending := *state.PendingAttack
		state = damageTargetWithoutMissed(state, targetPlayerID, pending.Damage, pending.SourcePlayerID)
		state = checkEnd(state)
		if state.Finished {
			state.PendingAttack = nil
			return state
		}
	}
	return state
}

func resolvePendingAttackWithMissed(state State, playerID string) State {
	if state.PendingAttack == nil || state.PendingAttack.TargetPlayerID != playerID {
		return state
	}
	targetIndex := findPlayer(state, playerID)
	if targetIndex < 0 {
		return state
	}
	missed, ok := removeFirstType(&state.Players[targetIndex], CardMissed)
	if !ok {
		return state
	}
	pending := *state.PendingAttack
	state.Discard = append(state.Discard, missed)
	state.Log = append(state.Log, playerID+" 님이 빗맞음으로 피했습니다.")
	state.PendingAttack = &PendingAttack{
		SourcePlayerID:     pending.SourcePlayerID,
		CardType:           pending.CardType,
		Damage:             pending.Damage,
		RemainingTargetIDs: append([]string(nil), pending.RemainingTargetIDs...),
	}
	return advancePendingAttack(state)
}

func resolvePendingAttackWithDamage(state State) State {
	if state.PendingAttack == nil {
		return state
	}
	pending := *state.PendingAttack
	targetPlayerID := pending.TargetPlayerID
	state.PendingAttack = &PendingAttack{
		SourcePlayerID:     pending.SourcePlayerID,
		CardType:           pending.CardType,
		Damage:             pending.Damage,
		RemainingTargetIDs: append([]string(nil), pending.RemainingTargetIDs...),
	}
	state = damageTargetWithoutMissed(state, targetPlayerID, pending.Damage, pending.SourcePlayerID)
	state = checkEnd(state)
	if state.Finished {
		state.PendingAttack = nil
		return state
	}
	return advancePendingAttack(state)
}

func damageTarget(state State, targetPlayerID string, amount int) State {
	return damageTargetBy(state, targetPlayerID, amount, "")
}

func damageTargetBy(state State, targetPlayerID string, amount int, sourcePlayerID string) State {
	return damageTargetInternal(state, targetPlayerID, amount, sourcePlayerID, true)
}

func damageTargetWithoutMissed(state State, targetPlayerID string, amount int, sourcePlayerID string) State {
	return damageTargetInternal(state, targetPlayerID, amount, sourcePlayerID, false)
}

func damageTargetInternal(state State, targetPlayerID string, amount int, sourcePlayerID string, allowMissed bool) State {
	targetIndex := findPlayer(state, targetPlayerID)
	if targetIndex < 0 {
		return state
	}
	target := &state.Players[targetIndex]
	if allowMissed {
		if missed, ok := removeFirstType(target, CardMissed); ok {
			state.Discard = append(state.Discard, missed)
			state.Log = append(state.Log, target.PlayerID+" 님이 빗맞음으로 피했습니다.")
			return state
		}
	}
	target.HP -= amount
	if target.HP <= 0 {
		if alivePlayerCount(state) > 2 {
			if beer, ok := removeFirstType(target, CardBeer); ok {
				state.Discard = append(state.Discard, beer)
				target.HP = 1
				state.Log = append(state.Log, target.PlayerID+" 님이 맥주로 버텼습니다.")
				return state
			}
		}
		targetRole := target.Role
		target.HP = 0
		target.Alive = false
		target.Active = false
		state.Discard = append(state.Discard, target.Hand...)
		state.Discard = append(state.Discard, target.Equipment...)
		target.Hand = []Card{}
		target.Equipment = []Card{}
		target.HandSize = 0
		state.Log = append(state.Log, target.PlayerID+" 님이 탈락했습니다.")
		state = applyEliminationReward(state, sourcePlayerID, targetRole)
	}
	return state
}

func applyEliminationReward(state State, sourcePlayerID string, eliminatedRole string) State {
	if sourcePlayerID == "" {
		return state
	}
	sourceIndex := findPlayer(state, sourcePlayerID)
	if sourceIndex < 0 {
		return state
	}
	source := &state.Players[sourceIndex]
	if !source.Alive {
		return state
	}
	if eliminatedRole == RoleOutlaw {
		drawn := drawCards(&state, 3)
		source.Hand = append(source.Hand, drawn...)
		source.HandSize = len(source.Hand)
		state.Log = append(state.Log, fmt.Sprintf("%s 님이 무법자 처치 보상으로 카드 %d장을 뽑았습니다.", source.PlayerID, len(drawn)))
	}
	if source.Role == RoleSheriff && eliminatedRole == RoleDeputy {
		state.Discard = append(state.Discard, source.Hand...)
		state.Discard = append(state.Discard, source.Equipment...)
		source.Hand = []Card{}
		source.Equipment = []Card{}
		source.HandSize = 0
		state.Log = append(state.Log, source.PlayerID+" 보안관이 부관을 제거해 손패를 모두 버렸습니다.")
	}
	return state
}

func checkEnd(state State) State {
	sheriffAlive := false
	threatsAlive := false
	aliveCount := 0
	lastRole := ""
	for _, player := range state.Players {
		if !player.Alive {
			continue
		}
		aliveCount++
		lastRole = player.Role
		if player.Role == RoleSheriff {
			sheriffAlive = true
		}
		if player.Role == RoleOutlaw || player.Role == RoleRenegade {
			threatsAlive = true
		}
	}
	if !sheriffAlive {
		state.Finished = true
		if aliveCount == 1 && lastRole == RoleRenegade {
			state.Winner = "renegade"
		} else {
			state.Winner = "outlaw"
		}
	}
	if sheriffAlive && !threatsAlive {
		state.Finished = true
		state.Winner = "law"
	}
	return state
}

func winningRole(role string, winner string) bool {
	switch winner {
	case "law":
		return role == RoleSheriff || role == RoleDeputy
	case "outlaw":
		return role == RoleOutlaw
	case "renegade":
		return role == RoleRenegade
	default:
		return false
	}
}

func drawCards(state *State, count int) []Card {
	drawn := make([]Card, 0, count)
	for len(drawn) < count && availableDrawCount(*state) > 0 {
		if len(state.Deck) == 0 {
			recycleDiscard(state)
		}
		if len(state.Deck) == 0 {
			break
		}
		drawn = append(drawn, state.Deck[0])
		state.Deck = state.Deck[1:]
	}
	return drawn
}

func availableDrawCount(state State) int {
	return len(state.Deck) + len(state.Discard)
}

func recycleDiscard(state *State) {
	if len(state.Discard) == 0 {
		return
	}
	recycled := append([]Card(nil), state.Discard...)
	random := rand.New(rand.NewSource(seedToInt(fmt.Sprintf("discard:%d:%d", len(recycled), len(state.Deck)))))
	random.Shuffle(len(recycled), func(i, j int) {
		recycled[i], recycled[j] = recycled[j], recycled[i]
	})
	state.Deck = recycled
	state.Discard = []Card{}
}

func rolesFor(count int) []string {
	switch count {
	case 4:
		return []string{RoleSheriff, RoleRenegade, RoleOutlaw, RoleOutlaw}
	case 5:
		return []string{RoleSheriff, RoleRenegade, RoleDeputy, RoleOutlaw, RoleOutlaw}
	case 6:
		return []string{RoleSheriff, RoleRenegade, RoleDeputy, RoleOutlaw, RoleOutlaw, RoleOutlaw}
	default:
		return []string{RoleSheriff, RoleRenegade, RoleDeputy, RoleDeputy, RoleOutlaw, RoleOutlaw, RoleOutlaw}
	}
}

func playPayload(payload any) (PlayPayload, error) {
	raw, ok := payload.(map[string]any)
	if !ok {
		return PlayPayload{}, errors.New("invalid play payload")
	}
	cardID, _ := raw["cardId"].(string)
	targetID, _ := raw["targetPlayerId"].(string)
	if cardID == "" {
		return PlayPayload{}, errors.New("card is required")
	}
	return PlayPayload{CardID: cardID, TargetPlayerID: targetID}, nil
}

func discardPayload(payload any) (DiscardPayload, error) {
	raw, ok := payload.(map[string]any)
	if !ok {
		return DiscardPayload{}, errors.New("invalid discard payload")
	}
	value, ok := raw["cardIds"]
	if !ok {
		return DiscardPayload{}, errors.New("discard cards are required")
	}
	switch typed := value.(type) {
	case []string:
		return DiscardPayload{CardIDs: append([]string(nil), typed...)}, nil
	case []any:
		cardIDs := make([]string, 0, len(typed))
		for _, item := range typed {
			cardID, _ := item.(string)
			if cardID == "" {
				return DiscardPayload{}, errors.New("invalid discard card")
			}
			cardIDs = append(cardIDs, cardID)
		}
		return DiscardPayload{CardIDs: cardIDs}, nil
	default:
		return DiscardPayload{}, errors.New("invalid discard cards")
	}
}

func findCard(hand []Card, cardID string) (Card, bool) {
	for _, card := range hand {
		if card.ID == cardID {
			return card, true
		}
	}
	return Card{}, false
}

func findCardByType(hand []Card, cardType string) (Card, bool) {
	for _, card := range hand {
		if card.Type == cardType {
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

func removeFirstType(player *PlayerState, cardType string) (Card, bool) {
	for _, card := range player.Hand {
		if card.Type == cardType {
			removeCard(player, card.ID)
			return card, true
		}
	}
	return Card{}, false
}

func nextAliveIndex(state State, current int) int {
	for step := 1; step <= len(state.Players); step++ {
		next := (current + step) % len(state.Players)
		if state.Players[next].Alive {
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

func attackTargets(state State, sourcePlayerID string) []string {
	targets := []string{}
	for _, player := range state.Players {
		if player.Alive && player.PlayerID != sourcePlayerID {
			targets = append(targets, player.PlayerID)
		}
	}
	return targets
}

func alivePlayerCount(state State) int {
	count := 0
	for _, player := range state.Players {
		if player.Alive {
			count++
		}
	}
	return count
}

func rankForOutcome(outcome gamecore.Outcome) int {
	if outcome == gamecore.OutcomeWin {
		return 1
	}
	return 2
}

func shuffledDeck(seed string) []Card {
	cardTypes := []string{}
	for range 28 {
		cardTypes = append(cardTypes, CardBang)
	}
	for range 14 {
		cardTypes = append(cardTypes, CardMissed)
	}
	for range 8 {
		cardTypes = append(cardTypes, CardBeer)
	}
	for range 4 {
		cardTypes = append(cardTypes, CardGatling)
	}
	for range 2 {
		cardTypes = append(cardTypes, CardScope, CardMustang, CardVolcanic, CardSchofield)
	}
	cardTypes = append(cardTypes, CardRemington, CardCarabine, CardWinchester)
	deck := make([]Card, 0, len(cardTypes))
	for index, cardType := range cardTypes {
		deck = append(deck, Card{ID: fmt.Sprintf("%s-%d", cardType, index), Type: cardType})
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
