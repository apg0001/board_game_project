package session

import (
	"context"
	"fmt"
	"testing"
	"time"

	"board-game-platform/apps/api/internal/gamecore"
	"board-game-platform/apps/api/internal/games/bang"
	"board-game-platform/apps/api/internal/games/dalmuti"
	"board-game-platform/apps/api/internal/games/davinci"
	"board-game-platform/apps/api/internal/games/gostop"
	"board-game-platform/apps/api/internal/games/halligalli"
	"board-game-platform/apps/api/internal/games/jokerdraw"
	"board-game-platform/apps/api/internal/games/onecard"
	"board-game-platform/apps/api/internal/games/rummikub"
	"board-game-platform/apps/api/internal/games/splendor"
	"board-game-platform/apps/api/internal/games/sutda"
	"board-game-platform/apps/api/internal/games/werewolf"
	"board-game-platform/apps/api/internal/guest"
	"board-game-platform/apps/api/internal/room"
)

func TestAllGamesStartAndAcceptRepresentativeSessionActions(t *testing.T) {
	cases := []struct {
		gameID      string
		playerCount int
		options     map[string]any
		play        func(t *testing.T, service *Service, testRoom room.Room, started Session)
	}{
		{gameID: "davinci", playerCount: 2, play: playDavinciTurn},
		{gameID: "halli-galli", playerCount: 2, play: playHalliGalliTurn},
		{gameID: "splendor", playerCount: 2, play: playSplendorTurn},
		{gameID: "dalmuti", playerCount: 4, play: playDalmutiTurn},
		{gameID: "rummikub", playerCount: 2, play: playRummikubTurn},
		{gameID: "bang", playerCount: 4, play: playBangTurn},
		{gameID: "sutda", playerCount: 2, play: playSutdaRound},
		{gameID: "gostop", playerCount: 3, play: playGoStopTurn},
		{
			gameID:      "onecard",
			playerCount: 2,
			options: map[string]any{
				"attackCards":    []string{"2", "A", "JOKER"},
				"defenseMode":    "attack-or-joker",
				"jokerDrawCount": 7,
				"stacking":       true,
			},
			play: playOneCardTurn,
		},
		{gameID: "jokerdraw", playerCount: 3, play: playJokerDrawTurn},
		{gameID: "werewolf", playerCount: 3, play: playWerewolfRound},
	}

	for _, tc := range cases {
		t.Run(tc.gameID, func(t *testing.T) {
			service := NewService(NewMemoryStore(), gameplayRegistry(), gameplayClock)
			testRoom := gameplayRoom(tc.gameID, tc.playerCount, tc.options)

			started, err := service.Start(testRoom)
			if err != nil {
				t.Fatalf("start %s: %v", tc.gameID, err)
			}
			if started.GameID != tc.gameID {
				t.Fatalf("expected game %s, got %s", tc.gameID, started.GameID)
			}

			tc.play(t, service, testRoom, started)
		})
	}
}

func playDavinciTurn(t *testing.T, service *Service, testRoom room.Room, started Session) {
	state := started.State.(davinci.State)
	actor := state.Players[state.CurrentPlayerIndex]
	target := state.Players[(state.CurrentPlayerIndex+1)%len(state.Players)]
	tile := firstHiddenDavinciTile(t, target)
	payload := map[string]any{
		"targetPlayerId": target.PlayerID,
		"tileIndex":      tile.index,
		"color":          tile.tile.Color,
		"value":          tile.tile.Value,
		"joker":          tile.tile.Joker,
		"insertIndex":    0,
	}

	updated := applyGameplayAction(t, service, testRoom, started, actor.PlayerID, davinci.ActionGuess, payload)
	next := updated.State.(davinci.State)
	if !next.CanEndTurn {
		t.Fatalf("expected correct guess to allow ending turn, got %+v", next)
	}

	updated = applyGameplayAction(t, service, testRoom, updated, actor.PlayerID, davinci.ActionPass, map[string]any{"insertIndex": 0})
	next = updated.State.(davinci.State)
	if next.CurrentPlayerIndex == state.CurrentPlayerIndex {
		t.Fatalf("expected turn to advance after pass, got %+v", next)
	}
}

func playHalliGalliTurn(t *testing.T, service *Service, testRoom room.Room, started Session) {
	state := started.State.(halligalli.State)
	actor := state.Players[state.CurrentPlayerIndex]
	updated := applyGameplayAction(t, service, testRoom, started, actor.PlayerID, halligalli.ActionFlip, nil)
	next := updated.State.(halligalli.State)
	if len(next.Players[state.CurrentPlayerIndex].FaceUp) == 0 {
		t.Fatalf("expected flipped card on table, got %+v", next)
	}
}

func playSplendorTurn(t *testing.T, service *Service, testRoom room.Room, started Session) {
	state := started.State.(splendor.State)
	currentIndex := state.CurrentPlayerIndex
	actor := state.Players[currentIndex]
	selection := threeAvailableGems(t, state.Bank)
	updated := applyGameplayAction(t, service, testRoom, started, actor.PlayerID, splendor.ActionTakeToken, map[string]any{"colors": selection})
	next := updated.State.(splendor.State)
	for _, color := range selection {
		if next.Players[currentIndex].Tokens[color] != 1 {
			t.Fatalf("expected selected splendor token %s, got %+v", color, next.Players[currentIndex].Tokens)
		}
	}
	if next.CurrentPlayerIndex == currentIndex {
		t.Fatalf("expected token taken and turn advanced, got %+v", next)
	}
}

func playDalmutiTurn(t *testing.T, service *Service, testRoom room.Room, started Session) {
	current := started
	state := current.State.(dalmuti.State)
	for state.PendingTax != nil {
		chooser := findDalmutiPlayerForTest(t, state, state.PendingTax.CurrentChooserID)
		exchange := state.PendingTax.Exchanges[0]
		current = applyGameplayAction(t, service, testRoom, current, chooser.PlayerID, dalmuti.ActionTax, map[string]any{
			"cardIds": firstDalmutiCardIDsForTest(t, chooser.Hand, exchange.Count),
		})
		state = current.State.(dalmuti.State)
	}
	actor := state.Players[state.CurrentPlayerIndex]
	if len(actor.Hand) == 0 {
		t.Fatal("expected current dalmuti player to have cards")
	}
	card := actor.Hand[0]
	updated := applyGameplayAction(t, service, testRoom, current, actor.PlayerID, dalmuti.ActionPlay, map[string]any{"rank": card.Rank, "count": 1})
	next := updated.State.(dalmuti.State)
	if next.CurrentTrick.PlayerID != actor.PlayerID || next.CurrentTrick.Count != 1 {
		t.Fatalf("expected played trick, got %+v", next.CurrentTrick)
	}
}

func playRummikubTurn(t *testing.T, service *Service, testRoom room.Room, started Session) {
	state := started.State.(rummikub.State)
	currentIndex := state.CurrentPlayerIndex
	actor := state.Players[currentIndex]
	beforeRackSize := actor.RackSize
	updated := applyGameplayAction(t, service, testRoom, started, actor.PlayerID, rummikub.ActionDraw, nil)
	next := updated.State.(rummikub.State)
	if next.Players[currentIndex].RackSize != beforeRackSize+1 || next.CurrentPlayerIndex == currentIndex {
		t.Fatalf("expected tile draw and turn advance, got %+v", next)
	}
}

func playBangTurn(t *testing.T, service *Service, testRoom room.Room, started Session) {
	state := started.State.(bang.State)
	actor := state.Players[state.CurrentPlayerIndex]
	updated := applyGameplayAction(t, service, testRoom, started, actor.PlayerID, bang.ActionDraw, nil)
	next := updated.State.(bang.State)
	if !next.Players[state.CurrentPlayerIndex].Drawn {
		t.Fatalf("expected bang draw phase, got %+v", next.Players[state.CurrentPlayerIndex])
	}

	updated = applyGameplayAction(t, service, testRoom, updated, actor.PlayerID, bang.ActionEndTurn, nil)
	next = updated.State.(bang.State)
	if next.PendingDiscardID == actor.PlayerID {
		cardIDs := make([]any, 0, next.PendingDiscardCount)
		for _, card := range next.Players[state.CurrentPlayerIndex].Hand[:next.PendingDiscardCount] {
			cardIDs = append(cardIDs, card.ID)
		}
		updated = applyGameplayAction(t, service, testRoom, updated, actor.PlayerID, bang.ActionDiscard, map[string]any{"cardIds": cardIDs})
		next = updated.State.(bang.State)
	}
	if next.CurrentPlayerIndex == state.CurrentPlayerIndex {
		t.Fatalf("expected bang turn to advance, got %+v", next)
	}
}

func playSutdaRound(t *testing.T, service *Service, testRoom room.Room, started Session) {
	first := started.State.(sutda.State)
	updated := applyGameplayAction(t, service, testRoom, started, first.Players[first.CurrentPlayerIndex].PlayerID, sutda.ActionCall, nil)
	second := updated.State.(sutda.State)
	if second.CurrentPlayerIndex == first.CurrentPlayerIndex {
		t.Fatalf("expected sutda turn to advance after first call, got %+v", second)
	}

	updated = applyGameplayAction(t, service, testRoom, updated, second.Players[second.CurrentPlayerIndex].PlayerID, sutda.ActionCall, nil)
	if updated.Status != StatusFinished {
		t.Fatalf("expected two calls to finish sutda round, got %+v", updated)
	}
}

func playGoStopTurn(t *testing.T, service *Service, testRoom room.Room, started Session) {
	state := started.State.(gostop.State)
	currentIndex := state.CurrentPlayerIndex
	actor := state.Players[currentIndex]
	if len(actor.Hand) == 0 {
		t.Fatal("expected gostop player to have cards")
	}
	beforeHandSize := len(actor.Hand)
	updated := applyGameplayAction(t, service, testRoom, started, actor.PlayerID, gostop.ActionPlay, map[string]any{"cardId": actor.Hand[0].ID})
	next := updated.State.(gostop.State)
	if len(next.Players[currentIndex].Hand) >= beforeHandSize {
		t.Fatalf("expected gostop hand to shrink after play, got before=%d after=%d", beforeHandSize, len(next.Players[currentIndex].Hand))
	}
}

func playOneCardTurn(t *testing.T, service *Service, testRoom room.Room, started Session) {
	state := started.State.(onecard.State)
	currentIndex := state.CurrentPlayerIndex
	actor := state.Players[currentIndex]
	beforeHandSize := actor.HandSize
	updated := applyGameplayAction(t, service, testRoom, started, actor.PlayerID, onecard.ActionDraw, nil)
	next := updated.State.(onecard.State)
	if next.Players[currentIndex].HandSize != beforeHandSize+1 || next.CurrentPlayerIndex == currentIndex {
		t.Fatalf("expected onecard draw and turn advance, got %+v", next)
	}
}

func playJokerDrawTurn(t *testing.T, service *Service, testRoom room.Room, started Session) {
	state := started.State.(jokerdraw.State)
	if state.Finished {
		if started.Status != StatusFinished {
			t.Fatalf("finished initial jokerdraw state should create a finished session, got %+v", started)
		}
		return
	}

	targetIndex := nextActiveJokerPlayer(state, state.CurrentPlayerIndex)
	if targetIndex < 0 || len(state.Players[targetIndex].Hand) == 0 {
		t.Fatalf("expected active joker draw target, got %+v", state)
	}
	actor := state.Players[state.CurrentPlayerIndex]
	target := state.Players[targetIndex]
	updated := applyGameplayAction(t, service, testRoom, started, actor.PlayerID, jokerdraw.ActionDraw, map[string]any{
		"targetPlayerId": target.PlayerID,
		"cardIndex":      0,
	})
	next := updated.State.(jokerdraw.State)
	if next.Round <= state.Round && !next.Finished {
		t.Fatalf("expected joker draw turn to progress, got %+v", next)
	}
}

func playWerewolfRound(t *testing.T, service *Service, testRoom room.Room, started Session) {
	current := started
	state := current.State.(werewolf.State)
	for _, role := range []string{werewolf.RoleWerewolf, werewolf.RoleSeer, werewolf.RoleRobber, werewolf.RoleTroublemaker, werewolf.RoleDrunk} {
		for _, player := range state.Players {
			if player.OriginalRole != role {
				continue
			}
			switch player.OriginalRole {
			case werewolf.RoleWerewolf:
				if countOriginalWerewolves(state) == 1 {
					current = applyGameplayAction(t, service, testRoom, current, player.PlayerID, werewolf.ActionLoneWolfCenter, map[string]any{"centerIndexes": []any{0}})
				} else {
					current = applyGameplayAction(t, service, testRoom, current, player.PlayerID, werewolf.ActionSeeWerewolves, nil)
				}
			case werewolf.RoleSeer:
				current = applyGameplayAction(t, service, testRoom, current, player.PlayerID, werewolf.ActionSeeCenter, map[string]any{"centerIndexes": []any{0, 1}})
			case werewolf.RoleRobber:
				current = applyGameplayAction(t, service, testRoom, current, player.PlayerID, werewolf.ActionRob, map[string]any{"targetPlayerId": firstOtherWerewolfPlayer(state, player.PlayerID)})
			case werewolf.RoleTroublemaker:
				left, right := twoOtherWerewolfPlayers(t, state, player.PlayerID)
				current = applyGameplayAction(t, service, testRoom, current, player.PlayerID, werewolf.ActionTroublemake, map[string]any{
					"leftPlayerId":  left,
					"rightPlayerId": right,
				})
			case werewolf.RoleDrunk:
				current = applyGameplayAction(t, service, testRoom, current, player.PlayerID, werewolf.ActionDrunkSwap, map[string]any{"centerIndexes": []any{0}})
			}
			state = current.State.(werewolf.State)
		}
	}

	current = applyGameplayAction(t, service, testRoom, current, state.Players[0].PlayerID, werewolf.ActionFinishNight, nil)
	state = current.State.(werewolf.State)
	if state.Phase != werewolf.PhaseDiscussion {
		t.Fatalf("expected werewolf discussion phase, got %+v", state)
	}

	targetID := state.Players[0].PlayerID
	for _, player := range state.Players {
		current = applyGameplayAction(t, service, testRoom, current, player.PlayerID, werewolf.ActionVote, map[string]any{"targetPlayerId": targetID})
	}
	if current.Status != StatusFinished {
		t.Fatalf("expected werewolf vote to finish, got %+v", current)
	}
}

func countOriginalWerewolves(state werewolf.State) int {
	count := 0
	for _, player := range state.Players {
		if player.OriginalRole == werewolf.RoleWerewolf {
			count++
		}
	}
	return count
}

type indexedDavinciTile struct {
	index int
	tile  davinci.Tile
}

func firstHiddenDavinciTile(t *testing.T, player davinci.PlayerState) indexedDavinciTile {
	t.Helper()
	for index, tile := range player.Tiles {
		if !tile.Revealed {
			return indexedDavinciTile{index: index, tile: tile}
		}
	}
	t.Fatalf("expected hidden tile for %+v", player)
	return indexedDavinciTile{}
}

func threeAvailableGems(t *testing.T, bank map[string]int) []string {
	t.Helper()
	selection := []string{}
	for _, color := range []string{"white", "blue", "green", "red", "black"} {
		if bank[color] > 0 {
			selection = append(selection, color)
			if len(selection) == 3 {
				return selection
			}
		}
	}
	t.Fatalf("expected three available splendor gems, got %+v", bank)
	return nil
}

func findDalmutiPlayerForTest(t *testing.T, state dalmuti.State, playerID string) dalmuti.PlayerState {
	t.Helper()
	for _, player := range state.Players {
		if player.PlayerID == playerID {
			return player
		}
	}
	t.Fatalf("expected dalmuti player %s in %+v", playerID, state.Players)
	return dalmuti.PlayerState{}
}

func firstDalmutiCardIDsForTest(t *testing.T, hand []dalmuti.Card, count int) []any {
	t.Helper()
	if len(hand) < count {
		t.Fatalf("expected at least %d dalmuti cards, got %+v", count, hand)
	}
	cardIDs := make([]any, 0, count)
	for index := 0; index < count; index++ {
		cardIDs = append(cardIDs, hand[index].ID)
	}
	return cardIDs
}

func nextActiveJokerPlayer(state jokerdraw.State, current int) int {
	for step := 1; step <= len(state.Players); step++ {
		next := (current + step) % len(state.Players)
		if state.Players[next].Active {
			return next
		}
	}
	return -1
}

func firstOtherWerewolfPlayer(state werewolf.State, playerID string) string {
	for _, player := range state.Players {
		if player.PlayerID != playerID {
			return player.PlayerID
		}
	}
	return ""
}

func twoOtherWerewolfPlayers(t *testing.T, state werewolf.State, playerID string) (string, string) {
	t.Helper()
	others := []string{}
	for _, player := range state.Players {
		if player.PlayerID != playerID {
			others = append(others, player.PlayerID)
		}
	}
	if len(others) < 2 {
		t.Fatalf("expected two troublemaker targets, got %+v", state.Players)
	}
	return others[0], others[1]
}

func applyGameplayAction(t *testing.T, service *Service, testRoom room.Room, current Session, playerID string, actionType string, payload any) Session {
	t.Helper()
	updated, _, err := service.ApplyAction(context.Background(), testRoom, current.ID, gamecore.Action{
		Type:     actionType,
		PlayerID: gamecore.PlayerID(playerID),
		Payload:  payload,
	})
	if err != nil {
		t.Fatalf("%s by %s failed: %v", actionType, playerID, err)
	}
	return updated
}

func gameplayRegistry() *gamecore.Registry {
	return gamecore.NewRegistry(
		bang.NewModule(),
		dalmuti.NewModule(),
		davinci.NewModule(),
		gostop.NewModule(),
		halligalli.NewModule(),
		jokerdraw.NewModule(),
		onecard.NewModule(),
		rummikub.NewModule(),
		splendor.NewModule(),
		sutda.NewModule(),
		werewolf.NewModule(),
	)
}

func gameplayRoom(gameID string, playerCount int, options map[string]any) room.Room {
	participants := make([]room.Participant, 0, playerCount)
	for index := 0; index < playerCount; index++ {
		userID := fmt.Sprintf("%s_p%d", gameID, index+1)
		participants = append(participants, room.Participant{
			User:      guest.PublicUser{ID: userID, Nickname: userID},
			Ready:     true,
			Host:      index == 0,
			SeatIndex: index,
			JoinedAt:  gameplayClock(),
		})
	}
	return room.Room{
		ID:           "room_gameplay_" + gameID,
		Code:         "CODE" + gameID,
		GameID:       gameID,
		Visibility:   room.VisibilityPrivate,
		Status:       room.StatusLobby,
		MaxPlayers:   playerCount,
		HostUserID:   participants[0].User.ID,
		Participants: participants,
		Options:      room.Options{TurnSeconds: 60, MaxWaitSeconds: 60, AllowSpectators: true},
		GameRules:    options,
		CreatedAt:    gameplayClock(),
		UpdatedAt:    gameplayClock(),
	}
}

func gameplayClock() time.Time {
	return time.Date(2026, 7, 27, 1, 0, 0, 0, time.UTC)
}
