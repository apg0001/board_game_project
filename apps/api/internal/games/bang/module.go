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
	ActionDraw           = "bang.draw"
	ActionPlay           = "bang.play"
	ActionEndTurn        = "bang.end_turn"
	ActionUseBang        = "bang.use_bang"
	ActionUseMissed      = "bang.use_missed"
	ActionTakeHit        = "bang.take_hit"
	ActionDiscard        = "bang.discard"
	ActionChoose         = "bang.choose_general_store"
	CardBang             = "bang"
	CardMissed           = "missed"
	CardBeer             = "beer"
	CardGatling          = "gatling"
	CardBarrel           = "barrel"
	CardJail             = "jail"
	CardDynamite         = "dynamite"
	CardStagecoach       = "stagecoach"
	CardWellsFargo       = "wells_fargo"
	CardSaloon           = "saloon"
	CardCatBalou         = "cat_balou"
	CardPanic            = "panic"
	CardDuel             = "duel"
	CardIndians          = "indians"
	CardGeneralStore     = "general_store"
	CardScope            = "scope"
	CardMustang          = "mustang"
	CardVolcanic         = "volcanic"
	CardSchofield        = "schofield"
	CardRemington        = "remington"
	CardCarabine         = "carabine"
	CardWinchester       = "winchester"
	RoleSheriff          = "sheriff"
	RoleDeputy           = "deputy"
	RoleOutlaw           = "outlaw"
	RoleRenegade         = "renegade"
	CharacterBart        = "bart_cassidy"
	CharacterBlackJack   = "black_jack"
	CharacterCalamity    = "calamity_janet"
	CharacterElGringo    = "el_gringo"
	CharacterJesse       = "jesse_jones"
	CharacterJourdonnais = "jourdonnais"
	CharacterKit         = "kit_carlson"
	CharacterLucky       = "lucky_duke"
	CharacterPaul        = "paul_regret"
	CharacterPedro       = "pedro_ramirez"
	CharacterRose        = "rose_doolan"
	CharacterSid         = "sid_ketchum"
	CharacterSlab        = "slab_the_killer"
	CharacterSuzy        = "suzy_lafayette"
	CharacterVulture     = "vulture_sam"
	CharacterWilly       = "willy_the_kid"
	SuitSpade            = "spade"
	SuitHeart            = "heart"
	SuitDiamond          = "diamond"
	SuitClub             = "club"
)

type Card struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	Suit string `json:"suit,omitempty"`
	Rank int    `json:"rank,omitempty"`
}

type PlayerState struct {
	PlayerID      string `json:"playerId"`
	Role          string `json:"role,omitempty"`
	CharacterID   string `json:"characterId,omitempty"`
	CharacterName string `json:"characterName,omitempty"`
	HP            int    `json:"hp"`
	MaxHP         int    `json:"maxHp"`
	Hand          []Card `json:"hand"`
	Equipment     []Card `json:"equipment"`
	HandSize      int    `json:"handSize"`
	Alive         bool   `json:"alive"`
	Drawn         bool   `json:"drawn"`
	BangUsed      bool   `json:"bangUsed"`
	Active        bool   `json:"active"`
}

type PendingAttack struct {
	SourcePlayerID     string   `json:"sourcePlayerId"`
	TargetPlayerID     string   `json:"targetPlayerId"`
	CardType           string   `json:"cardType"`
	Damage             int      `json:"damage"`
	RemainingTargetIDs []string `json:"remainingTargetIds"`
}

type PendingGeneralStore struct {
	Offer               []Card   `json:"offer"`
	CurrentChooserID    string   `json:"currentChooserId"`
	RemainingChooserIDs []string `json:"remainingChooserIds"`
}

type State struct {
	CurrentPlayerIndex  int                  `json:"currentPlayerIndex"`
	Round               int                  `json:"round"`
	Players             []PlayerState        `json:"players"`
	Deck                []Card               `json:"deck"`
	Discard             []Card               `json:"discard"`
	PendingAttack       *PendingAttack       `json:"pendingAttack,omitempty"`
	PendingGeneralStore *PendingGeneralStore `json:"pendingGeneralStore,omitempty"`
	PendingDiscardID    string               `json:"pendingDiscardPlayerId,omitempty"`
	PendingDiscardCount int                  `json:"pendingDiscardCount,omitempty"`
	Winner              string               `json:"winner,omitempty"`
	Log                 []string             `json:"log"`
	Finished            bool                 `json:"finished"`
}

type PlayPayload struct {
	CardID         string `json:"cardId"`
	TargetPlayerID string `json:"targetPlayerId"`
	TargetCardID   string `json:"targetCardId"`
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
	characters := shuffledCharacters(ctx.RandomSeed, len(ctx.Players))
	deck := shuffledDeck(ctx.RandomSeed)
	players := make([]PlayerState, 0, len(ctx.Players))
	for index, player := range ctx.Players {
		character := characters[index]
		maxHP := character.MaxHP
		if roles[index] == RoleSheriff {
			maxHP++
		}
		hand := append([]Card(nil), deck[:maxHP]...)
		deck = deck[maxHP:]
		players = append(players, PlayerState{
			PlayerID:      string(player.ID),
			Role:          roles[index],
			CharacterID:   character.ID,
			CharacterName: character.Name,
			HP:            maxHP,
			MaxHP:         maxHP,
			Hand:          hand,
			HandSize:      len(hand),
			Alive:         true,
			Active:        true,
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
	if action.Type == ActionUseBang || action.Type == ActionUseMissed || action.Type == ActionTakeHit {
		return validatePendingAttackAction(current, action)
	}
	if current.PendingAttack != nil {
		return errors.New("pending attack response")
	}
	if action.Type == ActionChoose {
		return validateGeneralStoreAction(current, action)
	}
	if current.PendingGeneralStore != nil {
		return errors.New("pending general store choice")
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
		playType := effectivePlayType(player, card, payload)
		if playType == CardBang && player.BangUsed && !hasEquipment(player, CardVolcanic) && !hasCharacter(player, CharacterWilly) {
			return errors.New("only one bang per turn")
		}
		switch playType {
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
		case CardGatling, CardIndians:
			return nil
		case CardStagecoach, CardWellsFargo, CardSaloon:
			return nil
		case CardGeneralStore:
			if availableDrawCount(current) == 0 {
				return errors.New("deck is empty")
			}
		case CardDuel:
			targetIndex := findPlayer(current, payload.TargetPlayerID)
			if targetIndex < 0 || !current.Players[targetIndex].Alive || payload.TargetPlayerID == player.PlayerID {
				return errors.New("invalid target")
			}
		case CardJail:
			targetIndex := findPlayer(current, payload.TargetPlayerID)
			if targetIndex < 0 || !current.Players[targetIndex].Alive {
				return errors.New("invalid target")
			}
			if current.Players[targetIndex].Role == RoleSheriff {
				return errors.New("jail cannot target sheriff")
			}
		case CardBarrel, CardDynamite:
			return nil
		case CardCatBalou:
			targetIndex := findPlayer(current, payload.TargetPlayerID)
			if targetIndex < 0 || !current.Players[targetIndex].Alive || payload.TargetPlayerID == player.PlayerID {
				return errors.New("invalid target")
			}
			if !targetHasRemovableCard(current.Players[targetIndex], payload.TargetCardID) {
				return errors.New("target has no removable card")
			}
		case CardPanic:
			targetIndex := findPlayer(current, payload.TargetPlayerID)
			if targetIndex < 0 || !current.Players[targetIndex].Alive || payload.TargetPlayerID == player.PlayerID {
				return errors.New("invalid target")
			}
			if attackDistance(current, current.CurrentPlayerIndex, targetIndex) > 1 {
				return errors.New("panic target is out of range")
			}
			if !targetHasRemovableCard(current.Players[targetIndex], payload.TargetCardID) {
				return errors.New("target has no removable card")
			}
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
	case ActionUseBang:
		if requiredPendingResponseCard(state.PendingAttack.CardType) != CardBang {
			return errors.New("bang response is not allowed")
		}
		if _, ok := findBangResponseCard(state.Players[targetIndex]); !ok {
			return errors.New("bang card not found")
		}
	case ActionUseMissed:
		if requiredPendingResponseCard(state.PendingAttack.CardType) != CardMissed {
			return errors.New("missed response is not allowed")
		}
		if _, ok := findMissedResponseCard(state.Players[targetIndex]); !ok {
			return errors.New("missed card not found")
		}
	case ActionTakeHit:
		return nil
	default:
		return errors.New("unsupported pending action")
	}
	return nil
}

func validateGeneralStoreAction(state State, action gamecore.Action) error {
	if state.PendingGeneralStore == nil || state.PendingGeneralStore.CurrentChooserID == "" {
		return errors.New("no pending general store choice")
	}
	if state.PendingGeneralStore.CurrentChooserID != string(action.PlayerID) {
		return errors.New("not your general store choice")
	}
	payload, err := playPayload(action.Payload)
	if err != nil {
		return err
	}
	if _, ok := findCard(state.PendingGeneralStore.Offer, payload.CardID); !ok {
		return errors.New("general store card not found")
	}
	return nil
}

func (m Module) ApplyAction(_ context.Context, state any, action gamecore.Action, _ gamecore.Context) (gamecore.ActionResult, error) {
	current := asState(state)
	player := &current.Players[current.CurrentPlayerIndex]
	switch action.Type {
	case ActionDraw:
		var shouldContinue bool
		current, shouldContinue = resolveStartOfTurnCards(current, current.CurrentPlayerIndex)
		if !shouldContinue || current.Finished {
			break
		}
		player = &current.Players[current.CurrentPlayerIndex]
		drawn := drawCards(&current, 2)
		if hasCharacter(*player, CharacterBlackJack) && len(drawn) >= 2 && isRedSuit(drawn[1].Suit) {
			extra := drawCards(&current, 1)
			drawn = append(drawn, extra...)
			current.Log = append(current.Log, player.PlayerID+" 님이 블랙 잭 능력으로 추가 카드를 확인했습니다.")
		}
		player.Hand = append(player.Hand, drawn...)
		player.HandSize = len(player.Hand)
		player.Drawn = true
		current.Log = append(current.Log, fmt.Sprintf("%s 님이 카드 %d장을 뽑았습니다.", player.PlayerID, len(drawn)))
	case ActionPlay:
		payload, err := playPayload(action.Payload)
		if err != nil {
			return gamecore.ActionResult{}, err
		}
		playerBefore := *player
		card, _ := removeCard(player, payload.CardID)
		triggerSuzyIfEmpty(&current, current.CurrentPlayerIndex)
		playType := effectivePlayType(playerBefore, card, payload)
		switch playType {
		case CardBeer:
			current.Discard = append(current.Discard, card)
			if player.HP < player.MaxHP {
				player.HP++
			}
			current.Log = append(current.Log, player.PlayerID+" 님이 맥주로 회복했습니다.")
		case CardBang:
			current.Discard = append(current.Discard, card)
			if !hasEquipment(*player, CardVolcanic) && !hasCharacter(*player, CharacterWilly) {
				player.BangUsed = true
			}
			current.Log = append(current.Log, player.PlayerID+" 님이 BANG!을 사용했습니다.")
			current = startPendingAttack(current, player.PlayerID, card.Type, []string{payload.TargetPlayerID}, 1)
		case CardGatling:
			current.Discard = append(current.Discard, card)
			current.Log = append(current.Log, player.PlayerID+" 님이 개틀링을 사용했습니다.")
			current = startPendingAttack(current, player.PlayerID, card.Type, attackTargets(current, player.PlayerID), 1)
		case CardIndians:
			current.Discard = append(current.Discard, card)
			current.Log = append(current.Log, player.PlayerID+" 님이 인디언!을 사용했습니다.")
			current = startPendingAttack(current, player.PlayerID, card.Type, attackTargets(current, player.PlayerID), 1)
		case CardDuel:
			current.Discard = append(current.Discard, card)
			current.Log = append(current.Log, player.PlayerID+" 님이 결투를 신청했습니다.")
			current = startPendingDuel(current, player.PlayerID, payload.TargetPlayerID)
		case CardGeneralStore:
			current.Discard = append(current.Discard, card)
			offer := drawCards(&current, alivePlayerCount(current))
			current.PendingGeneralStore = &PendingGeneralStore{
				Offer:               offer,
				RemainingChooserIDs: generalStoreChooserOrder(current, current.CurrentPlayerIndex),
			}
			current.Log = append(current.Log, fmt.Sprintf("%s 님이 잡화점 카드 %d장을 공개했습니다.", player.PlayerID, len(offer)))
			current = advanceGeneralStore(current)
		case CardBarrel, CardDynamite:
			current = equipCard(current, current.CurrentPlayerIndex, card)
		case CardJail:
			targetIndex := findPlayer(current, payload.TargetPlayerID)
			if targetIndex < 0 {
				return gamecore.ActionResult{}, errors.New("invalid target")
			}
			current = equipCard(current, targetIndex, card)
		case CardStagecoach:
			current.Discard = append(current.Discard, card)
			drawn := drawCards(&current, 2)
			player.Hand = append(player.Hand, drawn...)
			player.HandSize = len(player.Hand)
			current.Log = append(current.Log, fmt.Sprintf("%s 님이 역마차로 카드 %d장을 뽑았습니다.", player.PlayerID, len(drawn)))
		case CardWellsFargo:
			current.Discard = append(current.Discard, card)
			drawn := drawCards(&current, 3)
			player.Hand = append(player.Hand, drawn...)
			player.HandSize = len(player.Hand)
			current.Log = append(current.Log, fmt.Sprintf("%s 님이 웰스 파고로 카드 %d장을 뽑았습니다.", player.PlayerID, len(drawn)))
		case CardSaloon:
			current.Discard = append(current.Discard, card)
			healed := 0
			for index := range current.Players {
				if current.Players[index].Alive && current.Players[index].HP < current.Players[index].MaxHP {
					current.Players[index].HP++
					healed++
				}
			}
			current.Log = append(current.Log, fmt.Sprintf("%s 님이 살룬으로 %d명을 회복시켰습니다.", player.PlayerID, healed))
		case CardCatBalou:
			current.Discard = append(current.Discard, card)
			targetIndex := findPlayer(current, payload.TargetPlayerID)
			if targetIndex < 0 || targetIndex == current.CurrentPlayerIndex {
				return gamecore.ActionResult{}, errors.New("invalid target")
			}
			targetHandSize := len(current.Players[targetIndex].Hand)
			removed, ok := removeRemovableCard(&current.Players[targetIndex], payload.TargetCardID)
			if !ok {
				return gamecore.ActionResult{}, errors.New("target has no removable card")
			}
			if len(current.Players[targetIndex].Hand) < targetHandSize {
				triggerSuzyIfEmpty(&current, targetIndex)
			}
			current.Discard = append(current.Discard, removed)
			current.Log = append(current.Log, player.PlayerID+" 님이 캣 벌루로 카드를 버리게 했습니다.")
		case CardPanic:
			current.Discard = append(current.Discard, card)
			targetIndex := findPlayer(current, payload.TargetPlayerID)
			if targetIndex < 0 || targetIndex == current.CurrentPlayerIndex {
				return gamecore.ActionResult{}, errors.New("invalid target")
			}
			targetHandSize := len(current.Players[targetIndex].Hand)
			stolen, ok := removeRemovableCard(&current.Players[targetIndex], payload.TargetCardID)
			if !ok {
				return gamecore.ActionResult{}, errors.New("target has no removable card")
			}
			if len(current.Players[targetIndex].Hand) < targetHandSize {
				triggerSuzyIfEmpty(&current, targetIndex)
			}
			player.Hand = append(player.Hand, stolen)
			player.HandSize = len(player.Hand)
			current.Log = append(current.Log, player.PlayerID+" 님이 패닉으로 카드를 가져왔습니다.")
		case CardScope, CardMustang, CardVolcanic, CardSchofield, CardRemington, CardCarabine, CardWinchester:
			current = equipCard(current, current.CurrentPlayerIndex, card)
		}
		current = checkEnd(current)
	case ActionUseBang:
		current = resolvePendingAttackWithBang(current, string(action.PlayerID))
	case ActionUseMissed:
		current = resolvePendingAttackWithMissed(current, string(action.PlayerID))
	case ActionTakeHit:
		current = resolvePendingAttackWithDamage(current)
	case ActionChoose:
		payload, err := playPayload(action.Payload)
		if err != nil {
			return gamecore.ActionResult{}, err
		}
		current = applyGeneralStoreChoice(current, string(action.PlayerID), payload.CardID)
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
		if current.PendingGeneralStore != nil && current.PendingGeneralStore.CurrentChooserID == string(playerID) && !current.Finished {
			current.PendingGeneralStore.CurrentChooserID = ""
			current = advanceGeneralStore(current)
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
	triggerSuzyIfEmpty(&state, playerIndex)
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

func generalStoreChooserOrder(state State, startIndex int) []string {
	chooserIDs := []string{}
	for offset := 0; offset < len(state.Players); offset++ {
		index := (startIndex + offset) % len(state.Players)
		if state.Players[index].Alive {
			chooserIDs = append(chooserIDs, state.Players[index].PlayerID)
		}
	}
	return chooserIDs
}

func advanceGeneralStore(state State) State {
	for state.PendingGeneralStore != nil {
		if len(state.PendingGeneralStore.Offer) == 0 || len(state.PendingGeneralStore.RemainingChooserIDs) == 0 {
			state.PendingGeneralStore = nil
			state.Log = append(state.Log, "잡화점 선택이 끝났습니다.")
			return state
		}
		chooserID := state.PendingGeneralStore.RemainingChooserIDs[0]
		state.PendingGeneralStore.RemainingChooserIDs = state.PendingGeneralStore.RemainingChooserIDs[1:]
		chooserIndex := findPlayer(state, chooserID)
		if chooserIndex < 0 || !state.Players[chooserIndex].Alive {
			continue
		}
		state.PendingGeneralStore.CurrentChooserID = chooserID
		state.Log = append(state.Log, chooserID+" 님이 잡화점 카드를 선택해야 합니다.")
		return state
	}
	return state
}

func applyGeneralStoreChoice(state State, playerID string, cardID string) State {
	if state.PendingGeneralStore == nil || state.PendingGeneralStore.CurrentChooserID != playerID {
		return state
	}
	playerIndex := findPlayer(state, playerID)
	if playerIndex < 0 {
		return state
	}
	card, ok := removeOfferCard(state.PendingGeneralStore, cardID)
	if !ok {
		return state
	}
	player := &state.Players[playerIndex]
	player.Hand = append(player.Hand, card)
	player.HandSize = len(player.Hand)
	state.PendingGeneralStore.CurrentChooserID = ""
	state.Log = append(state.Log, playerID+" 님이 잡화점 카드를 가져갔습니다.")
	return advanceGeneralStore(state)
}

func removeOfferCard(pending *PendingGeneralStore, cardID string) (Card, bool) {
	next := []Card{}
	removed := Card{}
	found := false
	for _, card := range pending.Offer {
		if !found && card.ID == cardID {
			removed = card
			found = true
			continue
		}
		next = append(next, card)
	}
	pending.Offer = next
	return removed, found
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
	if hasCharacter(state.Players[sourceIndex], CharacterRose) {
		distance--
	}
	if hasEquipment(state.Players[targetIndex], CardMustang) {
		distance++
	}
	if hasCharacter(state.Players[targetIndex], CharacterPaul) {
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

func hasCharacter(player PlayerState, characterID string) bool {
	return player.CharacterID == characterID
}

func effectivePlayType(player PlayerState, card Card, payload PlayPayload) string {
	if hasCharacter(player, CharacterCalamity) && card.Type == CardMissed && payload.TargetPlayerID != "" {
		return CardBang
	}
	return card.Type
}

func responseAvailable(player PlayerState, cardType string) bool {
	if cardType == CardMissed {
		_, ok := findMissedResponseCard(player)
		return ok
	}
	if cardType == CardBang {
		_, ok := findBangResponseCard(player)
		return ok
	}
	_, ok := findCardByType(player.Hand, cardType)
	return ok
}

func findBangResponseCard(player PlayerState) (Card, bool) {
	if card, ok := findCardByType(player.Hand, CardBang); ok {
		return card, true
	}
	if hasCharacter(player, CharacterCalamity) {
		return findCardByType(player.Hand, CardMissed)
	}
	return Card{}, false
}

func removeBangResponseCard(player *PlayerState) (Card, bool) {
	if card, ok := removeFirstType(player, CardBang); ok {
		return card, true
	}
	if hasCharacter(*player, CharacterCalamity) {
		return removeFirstType(player, CardMissed)
	}
	return Card{}, false
}

func findMissedResponseCard(player PlayerState) (Card, bool) {
	if card, ok := findCardByType(player.Hand, CardMissed); ok {
		return card, true
	}
	if hasCharacter(player, CharacterCalamity) {
		return findCardByType(player.Hand, CardBang)
	}
	return Card{}, false
}

func removeMissedResponseCard(player *PlayerState) (Card, bool) {
	if card, ok := removeFirstType(player, CardMissed); ok {
		return card, true
	}
	if hasCharacter(*player, CharacterCalamity) {
		return removeFirstType(player, CardBang)
	}
	return Card{}, false
}

func barrelAttemptCount(player PlayerState) int {
	count := 0
	if hasEquipment(player, CardBarrel) {
		count++
	}
	if hasCharacter(player, CharacterJourdonnais) {
		count++
	}
	return count
}

func triggerSuzyIfEmpty(state *State, playerIndex int) {
	if playerIndex < 0 || playerIndex >= len(state.Players) {
		return
	}
	player := &state.Players[playerIndex]
	if !player.Alive || !hasCharacter(*player, CharacterSuzy) || len(player.Hand) != 0 {
		return
	}
	drawn := drawCards(state, 1)
	if len(drawn) == 0 {
		return
	}
	player.Hand = append(player.Hand, drawn...)
	player.HandSize = len(player.Hand)
	state.Log = append(state.Log, player.PlayerID+" 님이 수지 라파예트 능력으로 카드 1장을 뽑았습니다.")
}

func findAliveCharacter(state State, characterID string, excludePlayerID string) int {
	for index, player := range state.Players {
		if player.Alive && player.PlayerID != excludePlayerID && hasCharacter(player, characterID) {
			return index
		}
	}
	return -1
}

func firstEquipmentID(player PlayerState, cardType string) string {
	for _, equipment := range player.Equipment {
		if equipment.Type == cardType {
			return equipment.ID
		}
	}
	return ""
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

func resolveStartOfTurnCards(state State, playerIndex int) (State, bool) {
	if playerIndex < 0 || playerIndex >= len(state.Players) || !state.Players[playerIndex].Alive {
		return state, false
	}
	playerID := state.Players[playerIndex].PlayerID
	if hasEquipment(state.Players[playerIndex], CardDynamite) {
		dynamite, _ := removeEquipment(&state.Players[playerIndex], firstEquipmentID(state.Players[playerIndex], CardDynamite))
		check, ok := drawCheckForPlayer(&state, playerIndex, func(card Card) bool {
			return !dynamiteExplodes(card)
		})
		if ok && dynamiteExplodes(check) {
			state.Discard = append(state.Discard, dynamite)
			state.Log = append(state.Log, playerID+" 님의 다이너마이트가 폭발했습니다.")
			state = damageTargetWithoutMissed(state, playerID, 3, "")
			state = checkEnd(state)
			if state.Finished {
				return state, false
			}
			if playerIndex >= len(state.Players) || !state.Players[playerIndex].Alive {
				return finishTurn(state), false
			}
		} else {
			nextIndex := nextAliveIndex(state, playerIndex)
			state.Players[nextIndex].Equipment = append(state.Players[nextIndex].Equipment, dynamite)
			state.Log = append(state.Log, playerID+" 님의 다이너마이트가 넘어갔습니다.")
		}
	}
	if hasEquipment(state.Players[playerIndex], CardJail) {
		jail, _ := removeEquipment(&state.Players[playerIndex], firstEquipmentID(state.Players[playerIndex], CardJail))
		check, ok := drawCheckForPlayer(&state, playerIndex, drawCheckIsHeart)
		state.Discard = append(state.Discard, jail)
		if ok && drawCheckIsHeart(check) {
			state.Log = append(state.Log, playerID+" 님이 감옥에서 풀려났습니다.")
			return state, true
		}
		state.Log = append(state.Log, playerID+" 님이 감옥 때문에 턴을 넘깁니다.")
		return finishTurn(state), false
	}
	return state, true
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

func startPendingDuel(state State, challengerID string, challengedID string) State {
	state.PendingAttack = &PendingAttack{
		SourcePlayerID: challengerID,
		TargetPlayerID: challengedID,
		CardType:       CardDuel,
		Damage:         1,
	}
	return advancePendingAttack(state)
}

func advancePendingAttack(state State) State {
	for state.PendingAttack != nil {
		if state.PendingAttack.CardType == CardDuel {
			return advancePendingDuel(state)
		}
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
		responseCard := requiredPendingResponseCard(state.PendingAttack.CardType)
		if responseCard == CardMissed && canUseBarrelForAttack(state.PendingAttack.CardType) {
			barrelAttempts := barrelAttemptCount(state.Players[targetIndex])
			avoided := false
			for attempt := 0; attempt < barrelAttempts; attempt++ {
				check, ok := drawCheckForPlayer(&state, targetIndex, drawCheckIsHeart)
				if ok && drawCheckIsHeart(check) {
					state.Log = append(state.Log, targetPlayerID+" 님이 술통으로 공격을 피했습니다.")
					avoided = true
					break
				}
				state.Log = append(state.Log, targetPlayerID+" 님의 술통 판정이 실패했습니다.")
			}
			if avoided {
				continue
			}
		}
		if responseAvailable(state.Players[targetIndex], responseCard) {
			state.PendingAttack.TargetPlayerID = targetPlayerID
			state.Log = append(state.Log, fmt.Sprintf("%s 님의 %s 반응을 기다립니다.", targetPlayerID, bangCardName(responseCard)))
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

func canUseBarrelForAttack(cardType string) bool {
	return cardType == CardBang || cardType == CardGatling
}

func advancePendingDuel(state State) State {
	if state.PendingAttack == nil {
		return state
	}
	pending := *state.PendingAttack
	targetIndex := findPlayer(state, pending.TargetPlayerID)
	if targetIndex < 0 || !state.Players[targetIndex].Alive {
		state.PendingAttack = nil
		return state
	}
	if responseAvailable(state.Players[targetIndex], CardBang) {
		state.Log = append(state.Log, pending.TargetPlayerID+" 님의 결투 BANG! 반응을 기다립니다.")
		return state
	}
	state.PendingAttack = nil
	state = damageTargetWithoutMissed(state, pending.TargetPlayerID, pending.Damage, pending.SourcePlayerID)
	return checkEnd(state)
}

func requiredPendingResponseCard(cardType string) string {
	if cardType == CardDuel || cardType == CardIndians {
		return CardBang
	}
	return CardMissed
}

func bangCardName(cardType string) string {
	if cardType == CardBang {
		return "BANG!"
	}
	if cardType == CardMissed {
		return "빗나감"
	}
	return cardType
}

func resolvePendingAttackWithBang(state State, playerID string) State {
	if state.PendingAttack == nil || state.PendingAttack.TargetPlayerID != playerID {
		return state
	}
	targetIndex := findPlayer(state, playerID)
	if targetIndex < 0 {
		return state
	}
	bang, ok := removeBangResponseCard(&state.Players[targetIndex])
	if !ok {
		return state
	}
	pending := *state.PendingAttack
	state.Discard = append(state.Discard, bang)
	triggerSuzyIfEmpty(&state, targetIndex)
	if pending.CardType == CardDuel {
		state.Log = append(state.Log, playerID+" 님이 결투에 BANG!으로 응수했습니다.")
		state.PendingAttack = &PendingAttack{
			SourcePlayerID: playerID,
			TargetPlayerID: pending.SourcePlayerID,
			CardType:       CardDuel,
			Damage:         pending.Damage,
		}
		return advancePendingAttack(state)
	}
	state.Log = append(state.Log, playerID+" 님이 BANG!으로 인디언을 막았습니다.")
	state.PendingAttack = &PendingAttack{
		SourcePlayerID:     pending.SourcePlayerID,
		CardType:           pending.CardType,
		Damage:             pending.Damage,
		RemainingTargetIDs: append([]string(nil), pending.RemainingTargetIDs...),
	}
	return advancePendingAttack(state)
}

func resolvePendingAttackWithMissed(state State, playerID string) State {
	if state.PendingAttack == nil || state.PendingAttack.TargetPlayerID != playerID {
		return state
	}
	targetIndex := findPlayer(state, playerID)
	if targetIndex < 0 {
		return state
	}
	missed, ok := removeMissedResponseCard(&state.Players[targetIndex])
	if !ok {
		return state
	}
	triggerSuzyIfEmpty(&state, targetIndex)
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
			triggerSuzyIfEmpty(&state, targetIndex)
			state.Log = append(state.Log, target.PlayerID+" 님이 빗맞음으로 피했습니다.")
			return state
		}
	}
	oldHP := target.HP
	target.HP -= amount
	if target.HP <= 0 {
		if alivePlayerCount(state) > 2 {
			if beer, ok := removeFirstType(target, CardBeer); ok {
				state.Discard = append(state.Discard, beer)
				triggerSuzyIfEmpty(&state, targetIndex)
				target.HP = 1
				state.Log = append(state.Log, target.PlayerID+" 님이 맥주로 버텼습니다.")
				return state
			}
		}
		targetRole := target.Role
		target.HP = 0
		target.Alive = false
		target.Active = false
		eliminatedCards := append([]Card{}, target.Hand...)
		eliminatedCards = append(eliminatedCards, target.Equipment...)
		target.Hand = []Card{}
		target.Equipment = []Card{}
		target.HandSize = 0
		if vultureIndex := findAliveCharacter(state, CharacterVulture, target.PlayerID); vultureIndex >= 0 {
			state.Players[vultureIndex].Hand = append(state.Players[vultureIndex].Hand, eliminatedCards...)
			state.Players[vultureIndex].HandSize = len(state.Players[vultureIndex].Hand)
			state.Log = append(state.Log, state.Players[vultureIndex].PlayerID+" 님이 벌처 샘 능력으로 탈락자의 카드를 가져갔습니다.")
		} else {
			state.Discard = append(state.Discard, eliminatedCards...)
		}
		state.Log = append(state.Log, target.PlayerID+" 님이 탈락했습니다.")
		state = applyEliminationReward(state, sourcePlayerID, targetRole)
		return state
	}
	lostHP := oldHP - target.HP
	if lostHP > 0 {
		state = applyLostHPAbilities(state, targetIndex, sourcePlayerID, lostHP)
	}
	return state
}

func applyLostHPAbilities(state State, targetIndex int, sourcePlayerID string, lostHP int) State {
	target := &state.Players[targetIndex]
	if hasCharacter(*target, CharacterBart) {
		drawn := drawCards(&state, lostHP)
		target.Hand = append(target.Hand, drawn...)
		target.HandSize = len(target.Hand)
		state.Log = append(state.Log, fmt.Sprintf("%s 님이 바트 캐시디 능력으로 카드 %d장을 뽑았습니다.", target.PlayerID, len(drawn)))
	}
	if hasCharacter(*target, CharacterElGringo) && sourcePlayerID != "" {
		sourceIndex := findPlayer(state, sourcePlayerID)
		for stolenCount := 0; sourceIndex >= 0 && sourceIndex != targetIndex && stolenCount < lostHP && len(state.Players[sourceIndex].Hand) > 0; stolenCount++ {
			stolen, _ := removeCard(&state.Players[sourceIndex], state.Players[sourceIndex].Hand[0].ID)
			target.Hand = append(target.Hand, stolen)
			target.HandSize = len(target.Hand)
			triggerSuzyIfEmpty(&state, sourceIndex)
		}
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

func drawCheckForPlayer(state *State, playerIndex int, prefer func(Card) bool) (Card, bool) {
	count := 1
	if playerIndex >= 0 && playerIndex < len(state.Players) && hasCharacter(state.Players[playerIndex], CharacterLucky) {
		count = 2
	}
	drawn := drawCards(state, count)
	if len(drawn) == 0 {
		return Card{}, false
	}
	state.Discard = append(state.Discard, drawn...)
	if len(drawn) > 1 {
		for _, card := range drawn {
			if prefer(card) {
				state.Log = append(state.Log, state.Players[playerIndex].PlayerID+" 님이 럭키 듀크 능력으로 유리한 Draw! 결과를 선택했습니다.")
				return card, true
			}
		}
	}
	return drawn[0], true
}

func drawCheckIsHeart(card Card) bool {
	return card.Suit == SuitHeart
}

func isRedSuit(suit string) bool {
	return suit == SuitHeart || suit == SuitDiamond
}

func dynamiteExplodes(card Card) bool {
	return card.Suit == SuitSpade && card.Rank >= 2 && card.Rank <= 9
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
	targetCardID, _ := raw["targetCardId"].(string)
	if cardID == "" {
		return PlayPayload{}, errors.New("card is required")
	}
	return PlayPayload{CardID: cardID, TargetPlayerID: targetID, TargetCardID: targetCardID}, nil
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

func targetHasRemovableCard(player PlayerState, cardID string) bool {
	if cardID == "" {
		return len(player.Hand) > 0 || len(player.Equipment) > 0
	}
	if _, ok := findCard(player.Hand, cardID); ok {
		return true
	}
	_, ok := findCard(player.Equipment, cardID)
	return ok
}

func removeRemovableCard(player *PlayerState, cardID string) (Card, bool) {
	if cardID != "" {
		if card, ok := removeCard(player, cardID); ok {
			return card, true
		}
		return removeEquipment(player, cardID)
	}
	if len(player.Hand) > 0 {
		return removeCard(player, player.Hand[0].ID)
	}
	if len(player.Equipment) > 0 {
		return removeEquipment(player, player.Equipment[0].ID)
	}
	return Card{}, false
}

func removeEquipment(player *PlayerState, cardID string) (Card, bool) {
	next := []Card{}
	removed := Card{}
	found := false
	for _, card := range player.Equipment {
		if !found && card.ID == cardID {
			removed = card
			found = true
			continue
		}
		next = append(next, card)
	}
	player.Equipment = next
	return removed, found
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
	deck := officialBaseDeck()
	for index := range deck {
		deck[index].ID = fmt.Sprintf("%s-%d", deck[index].Type, index)
	}
	random := rand.New(rand.NewSource(seedToInt(seed)))
	random.Shuffle(len(deck), func(i, j int) {
		deck[i], deck[j] = deck[j], deck[i]
	})
	return deck
}

func officialBaseDeck() []Card {
	deck := []Card{}
	addCards(&deck, CardBarrel, SuitSpade, 12, 13)
	addCards(&deck, CardDynamite, SuitHeart, 2)
	addCards(&deck, CardScope, SuitSpade, 14)
	addCards(&deck, CardMustang, SuitHeart, 8, 9)
	addCards(&deck, CardJail, SuitSpade, 11)
	addCards(&deck, CardJail, SuitHeart, 4)
	addCards(&deck, CardJail, SuitSpade, 10)
	addCards(&deck, CardRemington, SuitClub, 13)
	addCards(&deck, CardCarabine, SuitClub, 14)
	addCards(&deck, CardSchofield, SuitClub, 11, 12)
	addCards(&deck, CardSchofield, SuitSpade, 13)
	addCards(&deck, CardVolcanic, SuitSpade, 10)
	addCards(&deck, CardVolcanic, SuitClub, 10)
	addCards(&deck, CardWinchester, SuitSpade, 8)
	addCards(&deck, CardBang, SuitSpade, 14)
	addCardRange(&deck, CardBang, SuitDiamond, 2, 14)
	addCardRange(&deck, CardBang, SuitClub, 2, 9)
	addCardRange(&deck, CardBang, SuitHeart, 12, 14)
	addCardRange(&deck, CardBeer, SuitHeart, 6, 11)
	addCards(&deck, CardCatBalou, SuitHeart, 13)
	addCardRange(&deck, CardCatBalou, SuitDiamond, 9, 11)
	addCards(&deck, CardStagecoach, SuitSpade, 9, 9)
	addCards(&deck, CardDuel, SuitDiamond, 12)
	addCards(&deck, CardDuel, SuitSpade, 11)
	addCards(&deck, CardDuel, SuitClub, 8)
	addCards(&deck, CardGeneralStore, SuitClub, 9)
	addCards(&deck, CardGeneralStore, SuitSpade, 12)
	addCards(&deck, CardGatling, SuitHeart, 10)
	addCards(&deck, CardIndians, SuitDiamond, 13, 14)
	addCardRange(&deck, CardMissed, SuitClub, 10, 14)
	addCardRange(&deck, CardMissed, SuitSpade, 2, 8)
	addCards(&deck, CardPanic, SuitHeart, 11, 12, 14)
	addCards(&deck, CardPanic, SuitDiamond, 8)
	addCards(&deck, CardSaloon, SuitHeart, 5)
	addCards(&deck, CardWellsFargo, SuitHeart, 3)
	return deck
}

type characterDefinition struct {
	ID    string
	Name  string
	MaxHP int
}

func shuffledCharacters(seed string, count int) []characterDefinition {
	characters := officialCharacters()
	random := rand.New(rand.NewSource(seedToInt("characters:" + seed)))
	random.Shuffle(len(characters), func(i, j int) {
		characters[i], characters[j] = characters[j], characters[i]
	})
	if count > len(characters) {
		count = len(characters)
	}
	return characters[:count]
}

func officialCharacters() []characterDefinition {
	return []characterDefinition{
		{ID: CharacterBart, Name: "Bart Cassidy", MaxHP: 4},
		{ID: CharacterBlackJack, Name: "Black Jack", MaxHP: 4},
		{ID: CharacterCalamity, Name: "Calamity Janet", MaxHP: 4},
		{ID: CharacterElGringo, Name: "El Gringo", MaxHP: 3},
		{ID: CharacterJesse, Name: "Jesse Jones", MaxHP: 4},
		{ID: CharacterJourdonnais, Name: "Jourdonnais", MaxHP: 4},
		{ID: CharacterKit, Name: "Kit Carlson", MaxHP: 4},
		{ID: CharacterLucky, Name: "Lucky Duke", MaxHP: 4},
		{ID: CharacterPaul, Name: "Paul Regret", MaxHP: 3},
		{ID: CharacterPedro, Name: "Pedro Ramirez", MaxHP: 4},
		{ID: CharacterRose, Name: "Rose Doolan", MaxHP: 4},
		{ID: CharacterSid, Name: "Sid Ketchum", MaxHP: 4},
		{ID: CharacterSlab, Name: "Slab the Killer", MaxHP: 4},
		{ID: CharacterSuzy, Name: "Suzy Lafayette", MaxHP: 4},
		{ID: CharacterVulture, Name: "Vulture Sam", MaxHP: 4},
		{ID: CharacterWilly, Name: "Willy the Kid", MaxHP: 4},
	}
}

func addCardRange(deck *[]Card, cardType string, suit string, start int, end int) {
	for rank := start; rank <= end; rank++ {
		addCards(deck, cardType, suit, rank)
	}
}

func addCards(deck *[]Card, cardType string, suit string, ranks ...int) {
	for _, rank := range ranks {
		*deck = append(*deck, Card{Type: cardType, Suit: suit, Rank: rank})
	}
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
