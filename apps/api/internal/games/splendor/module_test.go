package splendor

import (
	"context"
	"testing"
	"time"

	"board-game-platform/apps/api/internal/gamecore"
)

func TestCreateInitialStateBuildsMarket(t *testing.T) {
	module := NewModule()
	state := module.CreateInitialState(testContext()).(State)
	if len(state.Market) != 12 {
		t.Fatalf("expected 12 visible market cards, got %d", len(state.Market))
	}
	for _, tier := range splendorTiers {
		if len(state.Markets[tier]) != 4 {
			t.Fatalf("expected 4 tier %d market cards, got %d", tier, len(state.Markets[tier]))
		}
	}
	if len(state.Decks[1]) != 36 || len(state.Decks[2]) != 26 || len(state.Decks[3]) != 16 {
		t.Fatalf("expected official tier deck remainders 36/26/16, got %d/%d/%d", len(state.Decks[1]), len(state.Decks[2]), len(state.Decks[3]))
	}
	if state.Bank["white"] != 4 {
		t.Fatalf("expected two-player bank amount 4, got %d", state.Bank["white"])
	}
	if len(state.Nobles) != 3 {
		t.Fatalf("expected player count plus one noble tiles, got %d", len(state.Nobles))
	}
}

func TestTakeThreeDifferentTokensAdvancesTurn(t *testing.T) {
	module := NewModule()
	state := module.CreateInitialState(testContext()).(State)
	result, err := module.ApplyAction(context.Background(), state, gamecore.Action{
		Type:     ActionTakeToken,
		PlayerID: "p1",
		Payload:  map[string]any{"colors": []any{"white", "blue", "green"}},
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}
	next := result.State.(State)
	if next.Players[0].Tokens["white"] != 1 || next.Players[0].Tokens["blue"] != 1 || next.Players[0].Tokens["green"] != 1 {
		t.Fatalf("expected one token of each color, got %+v", next.Players[0].Tokens)
	}
	if next.Bank["white"] != 3 {
		t.Fatalf("expected bank decremented, got %d", next.Bank["white"])
	}
	if next.CurrentPlayerIndex != 1 {
		t.Fatalf("expected next player, got %d", next.CurrentPlayerIndex)
	}
}

func TestTakeTwoSameTokensRequiresFourInBank(t *testing.T) {
	module := NewModule()
	state := module.CreateInitialState(testContext()).(State)
	state.Bank["white"] = 3

	err := module.ValidateAction(context.Background(), state, gamecore.Action{
		Type:     ActionTakeToken,
		PlayerID: "p1",
		Payload:  map[string]any{"colors": []any{"white", "white"}},
	}, testContext())
	if err == nil {
		t.Fatal("expected two-same-color take to require at least 4 tokens in bank")
	}

	state.Bank["white"] = 4
	result, err := module.ApplyAction(context.Background(), state, gamecore.Action{
		Type:     ActionTakeToken,
		PlayerID: "p1",
		Payload:  map[string]any{"colors": []any{"white", "white"}},
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}
	next := result.State.(State)
	if next.Players[0].Tokens["white"] != 2 {
		t.Fatalf("expected two white tokens, got %d", next.Players[0].Tokens["white"])
	}
}

func TestSingleTokenTakeIsRejected(t *testing.T) {
	module := NewModule()
	state := module.CreateInitialState(testContext()).(State)
	err := module.ValidateAction(context.Background(), state, gamecore.Action{
		Type:     ActionTakeToken,
		PlayerID: "p1",
		Payload:  map[string]any{"color": "white"},
	}, testContext())
	if err == nil {
		t.Fatal("expected single-token take to be rejected: must take 3 different or 2 same color")
	}
}

func TestTakingTokensOverLimitRequiresReturnBeforeTurnAdvances(t *testing.T) {
	module := NewModule()
	state := module.CreateInitialState(testContext()).(State)
	state.Players[0].Tokens["white"] = 4
	state.Players[0].Tokens["blue"] = 3
	state.Players[0].Tokens["green"] = 1
	state.Bank["white"] -= 4
	state.Bank["blue"] -= 3
	state.Bank["green"]--

	result, err := module.ApplyAction(context.Background(), state, gamecore.Action{
		Type:     ActionTakeToken,
		PlayerID: "p1",
		Payload:  map[string]any{"colors": []any{"red", "green", "black"}},
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}

	pending := result.State.(State)
	if pending.PendingReturnID != "p1" || pending.PendingReturnCount != 1 {
		t.Fatalf("expected p1 to return one token, got id=%q count=%d", pending.PendingReturnID, pending.PendingReturnCount)
	}
	if pending.CurrentPlayerIndex != 0 || totalTokens(pending.Players[0]) != 11 {
		t.Fatalf("expected turn to wait with 11 tokens, got turn=%d tokens=%d", pending.CurrentPlayerIndex, totalTokens(pending.Players[0]))
	}
	err = module.ValidateAction(context.Background(), pending, gamecore.Action{
		Type:     ActionTakeToken,
		PlayerID: "p2",
		Payload:  map[string]any{"colors": []any{"white", "blue", "green"}},
	}, testContext())
	if err == nil {
		t.Fatal("expected other actions to be blocked while token return is pending")
	}

	returned, err := module.ApplyAction(context.Background(), pending, gamecore.Action{
		Type:     ActionReturnToken,
		PlayerID: "p1",
		Payload:  map[string]any{"tokens": map[string]any{"red": 1}},
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}
	next := returned.State.(State)
	if next.PendingReturnID != "" || next.PendingReturnCount != 0 {
		t.Fatalf("expected pending return cleared, got id=%q count=%d", next.PendingReturnID, next.PendingReturnCount)
	}
	if totalTokens(next.Players[0]) != 10 || next.Bank["red"] != state.Bank["red"] {
		t.Fatalf("expected one red returned, got tokens=%d bankRed=%d", totalTokens(next.Players[0]), next.Bank["red"])
	}
	if next.CurrentPlayerIndex != 1 {
		t.Fatalf("expected turn to advance after return, got %d", next.CurrentPlayerIndex)
	}
}

func TestBuyCardUsesTokens(t *testing.T) {
	module := NewModule()
	state := module.CreateInitialState(testContext()).(State)
	card := Card{ID: "test", Tier: 1, Color: "white", Points: 1, Cost: map[string]int{"blue": 1}}
	setVisibleCard(&state, 1, 0, card)
	state.Players[0].Tokens["blue"] = 1
	result, err := module.ApplyAction(context.Background(), state, gamecore.Action{
		Type:     ActionBuyCard,
		PlayerID: "p1",
		Payload:  map[string]any{"marketTier": 1, "marketIndex": float64(0)},
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}
	next := result.State.(State)
	if next.Players[0].Score != 1 || next.Players[0].Bonuses["white"] != 1 {
		t.Fatalf("expected bought card effects, got %+v", next.Players[0])
	}
}

func TestBuyPayloadAcceptsIntegerMarketIndex(t *testing.T) {
	payload, err := buyPayload(map[string]any{"marketTier": 2, "marketIndex": 0})
	if err != nil {
		t.Fatal(err)
	}
	if payload.MarketTier != 2 || payload.MarketIndex != 0 || payload.ReservedIndex != -1 {
		t.Fatalf("expected market index 0, got %+v", payload)
	}
}

func TestReserveCardTakesGoldAndRefillsMarket(t *testing.T) {
	module := NewModule()
	state := module.CreateInitialState(testContext()).(State)
	reserved := state.Markets[1][0]
	nextDeckID := state.Decks[1][0].ID

	result, err := module.ApplyAction(context.Background(), state, gamecore.Action{
		Type:     ActionReserveCard,
		PlayerID: "p1",
		Payload:  map[string]any{"marketTier": 1, "marketIndex": 0},
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}

	next := result.State.(State)
	if len(next.Players[0].Reserved) != 1 || next.Players[0].Reserved[0].ID != reserved.ID {
		t.Fatalf("expected reserved card %+v, got %+v", reserved, next.Players[0].Reserved)
	}
	if next.Players[0].Tokens["gold"] != 1 || next.Bank["gold"] != 4 {
		t.Fatalf("expected one gold token taken, got player=%d bank=%d", next.Players[0].Tokens["gold"], next.Bank["gold"])
	}
	if len(next.Market) != 12 || len(next.Markets[1]) != 4 || next.Markets[1][3].ID != nextDeckID {
		t.Fatalf("expected tier 1 market refill with next deck card %s, got %+v", nextDeckID, next.Markets[1])
	}
}

func TestReserveTopDeckCardDoesNotChangeVisibleMarket(t *testing.T) {
	module := NewModule()
	state := module.CreateInitialState(testContext()).(State)
	reserved := state.Decks[2][0]
	beforeDeckLen := len(state.Decks[2])
	beforeMarketIDs := cardIDs(state.Markets[2])

	result, err := module.ApplyAction(context.Background(), state, gamecore.Action{
		Type:     ActionReserveCard,
		PlayerID: "p1",
		Payload:  map[string]any{"marketTier": 2, "fromDeck": true},
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}

	next := result.State.(State)
	if len(next.Players[0].Reserved) != 1 || next.Players[0].Reserved[0].ID != reserved.ID {
		t.Fatalf("expected top tier 2 deck card reserved, got %+v", next.Players[0].Reserved)
	}
	if len(next.Decks[2]) != beforeDeckLen-1 {
		t.Fatalf("expected tier 2 deck to shrink by one, got %d want %d", len(next.Decks[2]), beforeDeckLen-1)
	}
	if got := cardIDs(next.Markets[2]); !sameStrings(got, beforeMarketIDs) {
		t.Fatalf("visible tier 2 market should not change, got %+v want %+v", got, beforeMarketIDs)
	}
	if next.Players[0].Tokens["gold"] != 1 || next.Bank["gold"] != 4 {
		t.Fatalf("expected one gold token after deck reservation, got player=%d bank=%d", next.Players[0].Tokens["gold"], next.Bank["gold"])
	}
}

func TestReserveTopDeckCardRequiresAvailableTierDeck(t *testing.T) {
	module := NewModule()
	state := module.CreateInitialState(testContext()).(State)
	state.Decks[2] = nil
	syncLegacyMarket(&state)

	err := module.ValidateAction(context.Background(), state, gamecore.Action{
		Type:     ActionReserveCard,
		PlayerID: "p1",
		Payload:  map[string]any{"marketTier": 2, "fromDeck": true},
	}, testContext())
	if err == nil {
		t.Fatal("expected empty tier deck reservation to be rejected")
	}
}

func TestReserveAtTokenLimitTakesGoldThenRequiresReturn(t *testing.T) {
	module := NewModule()
	state := module.CreateInitialState(testContext()).(State)
	state.Players[0].Tokens["white"] = 4
	state.Players[0].Tokens["blue"] = 3
	state.Players[0].Tokens["green"] = 3

	result, err := module.ApplyAction(context.Background(), state, gamecore.Action{
		Type:     ActionReserveCard,
		PlayerID: "p1",
		Payload:  map[string]any{"marketTier": 1, "marketIndex": 0},
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}

	pending := result.State.(State)
	if pending.Players[0].Tokens["gold"] != 1 || pending.Bank["gold"] != 4 {
		t.Fatalf("expected gold to be taken before excess return, got player=%d bank=%d", pending.Players[0].Tokens["gold"], pending.Bank["gold"])
	}
	if pending.PendingReturnID != "p1" || pending.PendingReturnCount != 1 {
		t.Fatalf("expected one-token return after reservation, got id=%q count=%d", pending.PendingReturnID, pending.PendingReturnCount)
	}

	returned, err := module.ApplyAction(context.Background(), pending, gamecore.Action{
		Type:     ActionReturnToken,
		PlayerID: "p1",
		Payload:  map[string]any{"tokens": map[string]any{"gold": 1}},
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}
	next := returned.State.(State)
	if next.Players[0].Tokens["gold"] != 0 || next.Bank["gold"] != 5 || next.CurrentPlayerIndex != 1 {
		t.Fatalf("expected returned gold and turn advance, got playerGold=%d bankGold=%d turn=%d", next.Players[0].Tokens["gold"], next.Bank["gold"], next.CurrentPlayerIndex)
	}
}

func TestReserveCardLimitIsThree(t *testing.T) {
	module := NewModule()
	state := module.CreateInitialState(testContext()).(State)
	state.Players[0].Reserved = []Card{{ID: "r1"}, {ID: "r2"}, {ID: "r3"}}

	err := module.ValidateAction(context.Background(), state, gamecore.Action{
		Type:     ActionReserveCard,
		PlayerID: "p1",
		Payload:  map[string]any{"marketTier": 1, "marketIndex": 0},
	}, testContext())
	if err == nil {
		t.Fatal("expected fourth reserved card to be rejected")
	}
}

func TestBuyReservedCardCanSpendGold(t *testing.T) {
	module := NewModule()
	state := module.CreateInitialState(testContext()).(State)
	card := Card{ID: "reserved", Color: "white", Points: 2, Cost: map[string]int{"blue": 2, "red": 1}}
	state.Players[0].Reserved = []Card{card}
	state.Players[0].Tokens["blue"] = 1
	state.Players[0].Tokens["gold"] = 2
	state.Bank["blue"]--
	state.Bank["gold"] -= 2

	result, err := module.ApplyAction(context.Background(), state, gamecore.Action{
		Type:     ActionBuyCard,
		PlayerID: "p1",
		Payload:  map[string]any{"reservedIndex": 0},
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}

	next := result.State.(State)
	if len(next.Players[0].Reserved) != 0 || len(next.Players[0].Cards) != 1 {
		t.Fatalf("expected reserved card to move to purchased cards, got reserved=%+v cards=%+v", next.Players[0].Reserved, next.Players[0].Cards)
	}
	if next.Players[0].Tokens["gold"] != 0 || next.Bank["gold"] != 5 {
		t.Fatalf("expected gold to be spent and returned, got player=%d bank=%d", next.Players[0].Tokens["gold"], next.Bank["gold"])
	}
	if next.Players[0].Score != 2 || next.Players[0].Bonuses["white"] != 1 {
		t.Fatalf("expected score and bonus from reserved purchase, got %+v", next.Players[0])
	}
}

func TestNobleVisitsQualifiedPlayerAfterAction(t *testing.T) {
	module := NewModule()
	state := module.CreateInitialState(testContext()).(State)
	state.Nobles = []Noble{{ID: "noble-test", Points: 3, Cost: map[string]int{"white": 1}}}
	setVisibleCard(&state, 1, 0, Card{ID: "white-free", Tier: 1, Color: "white", Points: 0, Cost: map[string]int{}})

	result, err := module.ApplyAction(context.Background(), state, gamecore.Action{
		Type:     ActionBuyCard,
		PlayerID: "p1",
		Payload:  map[string]any{"marketTier": 1, "marketIndex": 0},
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}

	next := result.State.(State)
	if len(next.Players[0].Nobles) != 1 || next.Players[0].Nobles[0].ID != "noble-test" {
		t.Fatalf("expected noble visit, got %+v", next.Players[0].Nobles)
	}
	if next.Players[0].Score != 3 {
		t.Fatalf("expected noble points added to score, got %d", next.Players[0].Score)
	}
	if len(next.Nobles) != 0 {
		t.Fatalf("expected visited noble removed from board, got %+v", next.Nobles)
	}
}

func TestNobleDoesNotVisitAfterNonBuyAction(t *testing.T) {
	module := NewModule()
	state := module.CreateInitialState(testContext()).(State)
	state.Nobles = []Noble{{ID: "noble-test", Points: 3, Cost: map[string]int{"white": 1}}}
	state.Players[0].Bonuses["white"] = 1

	result, err := module.ApplyAction(context.Background(), state, gamecore.Action{
		Type:     ActionTakeToken,
		PlayerID: "p1",
		Payload:  map[string]any{"colors": []any{"white", "blue", "green"}},
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}

	next := result.State.(State)
	if len(next.Players[0].Nobles) != 0 || next.Players[0].Score != 0 {
		t.Fatalf("noble should only visit after buying a card, got player=%+v", next.Players[0])
	}
	if len(next.Nobles) != 1 {
		t.Fatalf("expected noble to remain on board, got %+v", next.Nobles)
	}
}

func TestPublicStateMasksDeckAndDoesNotMutatePrivateState(t *testing.T) {
	module := NewModule()
	state := module.CreateInitialState(testContext()).(State)
	public := module.PublicState(state, "p1").(State)

	if len(public.Deck) != len(state.Deck) {
		t.Fatalf("expected masked deck length %d, got %d", len(state.Deck), len(public.Deck))
	}
	if len(public.Decks[1]) != len(state.Decks[1]) || len(public.Decks[2]) != len(state.Decks[2]) || len(public.Decks[3]) != len(state.Decks[3]) {
		t.Fatalf("expected masked tier deck lengths to match private state, got %+v", public.Decks)
	}
	if len(public.Deck) > 0 && public.Deck[0].ID != "" {
		t.Fatalf("expected hidden deck card, got %+v", public.Deck[0])
	}
	if len(public.Decks[1]) > 0 && public.Decks[1][0].ID != "" {
		t.Fatalf("expected hidden tier deck card, got %+v", public.Decks[1][0])
	}
	if len(state.Deck) > 0 && state.Deck[0].ID == "" {
		t.Fatal("public state must not mutate private deck")
	}

	public.Bank["white"] = 99
	public.Market[0].Cost["blue"] = 99
	public.Markets[1][0].Cost["blue"] = 99
	public.Nobles[0].Cost["white"] = 99
	if state.Bank["white"] == 99 || state.Market[0].Cost["blue"] == 99 || state.Markets[1][0].Cost["blue"] == 99 || state.Nobles[0].Cost["white"] == 99 {
		t.Fatal("public state must not share mutable maps with private state")
	}
}

func TestPublicStateMasksOtherPlayersReservedCards(t *testing.T) {
	module := NewModule()
	state := module.CreateInitialState(testContext()).(State)
	state.Players[0].Reserved = []Card{{ID: "p1-secret", Tier: 1, Color: "white", Points: 1, Cost: map[string]int{"blue": 1}}}
	state.Players[1].Reserved = []Card{{ID: "p2-secret", Tier: 2, Color: "red", Points: 2, Cost: map[string]int{"black": 3}}}

	p1View := module.PublicState(state, "p1").(State)
	if p1View.Players[0].Reserved[0].ID != "p1-secret" || p1View.Players[0].Reserved[0].Hidden {
		t.Fatalf("expected viewer's own reserved card to stay visible, got %+v", p1View.Players[0].Reserved[0])
	}
	if !p1View.Players[1].Reserved[0].Hidden || p1View.Players[1].Reserved[0].ID == "p2-secret" {
		t.Fatalf("expected opponent reserved card to be hidden, got %+v", p1View.Players[1].Reserved[0])
	}

	broadcastView := module.PublicState(state, "").(State)
	if !broadcastView.Players[0].Reserved[0].Hidden || !broadcastView.Players[1].Reserved[0].Hidden {
		t.Fatalf("expected broadcast view to hide all reserved cards, got %+v", broadcastView.Players)
	}
	if state.Players[1].Reserved[0].Hidden || state.Players[1].Reserved[0].ID != "p2-secret" {
		t.Fatalf("public state must not mutate private reserved cards, got %+v", state.Players[1].Reserved[0])
	}
}

func TestReaching15PointsTriggersFinalRoundInsteadOfInstantEnd(t *testing.T) {
	module := NewModule()
	state := module.CreateInitialState(testContext()).(State)
	card := Card{ID: "winning", Tier: 1, Color: "white", Points: 15, Cost: map[string]int{}}
	setVisibleCard(&state, 1, 0, card)

	result, err := module.ApplyAction(context.Background(), state, gamecore.Action{
		Type:     ActionBuyCard,
		PlayerID: "p1",
		Payload:  map[string]any{"marketTier": 1, "marketIndex": 0},
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}
	next := result.State.(State)
	if next.Finished {
		t.Fatal("game should not end instantly; other players deserve an equal final turn")
	}
	if !next.EndTriggered {
		t.Fatal("expected final round to be triggered once a player reaches 15 points")
	}
	if next.CurrentPlayerIndex != 1 {
		t.Fatalf("expected turn to pass to the other player for their final turn, got %d", next.CurrentPlayerIndex)
	}

	final, err := module.ApplyAction(context.Background(), next, gamecore.Action{
		Type:     ActionTakeToken,
		PlayerID: "p2",
		Payload:  map[string]any{"colors": []any{"white", "blue", "green"}},
	}, testContext())
	if err != nil {
		t.Fatal(err)
	}
	if !final.State.(State).Finished {
		t.Fatal("expected game to finish once every other player has had their final turn")
	}
}

func TestTimeoutReturnsTokensToBank(t *testing.T) {
	module := NewModule()
	state := module.CreateInitialState(testContext()).(State)
	state.Players[0].Tokens["white"] = 2
	state.Bank["white"] -= 2

	result, err := module.ApplyTimeout(context.Background(), state, "p1", testContext())
	if err != nil {
		t.Fatal(err)
	}

	next := result.State.(State)
	if next.Players[0].Tokens["white"] != 0 || next.Bank["white"] != 4 {
		t.Fatalf("expected timed out player's tokens returned, got tokens=%d bank=%d", next.Players[0].Tokens["white"], next.Bank["white"])
	}
}

func setVisibleCard(state *State, tier int, index int, card Card) {
	if card.Tier == 0 {
		card.Tier = tier
	}
	state.Markets[tier][index] = card
	syncLegacyMarket(state)
}

func cardIDs(cards []Card) []string {
	ids := make([]string, len(cards))
	for index, card := range cards {
		ids[index] = card.ID
	}
	return ids
}

func sameStrings(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func testContext() gamecore.Context {
	return gamecore.Context{
		GameID: "splendor",
		Players: []gamecore.Player{
			{ID: "p1", SeatIndex: 0, DisplayName: "P1", Connected: true},
			{ID: "p2", SeatIndex: 1, DisplayName: "P2", Connected: true},
		},
		Now:        time.Date(2026, 7, 20, 1, 0, 0, 0, time.UTC),
		RandomSeed: "seed",
	}
}
