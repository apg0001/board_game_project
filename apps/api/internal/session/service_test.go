package session

import (
	"errors"
	"testing"
	"time"

	"board-game-platform/apps/api/internal/gamecore"
	"board-game-platform/apps/api/internal/games/gostop"
	"board-game-platform/apps/api/internal/guest"
	"board-game-platform/apps/api/internal/room"
)

func TestStartRejectsPlayersOverGameCapacity(t *testing.T) {
	service := NewService(NewMemoryStore(), gamecore.NewRegistry(gostop.NewModule()), func() time.Time {
		return time.Date(2026, 7, 21, 1, 0, 0, 0, time.UTC)
	})
	testRoom := room.Room{
		ID:         "room_1",
		GameID:     "gostop",
		MaxPlayers: 4,
		Participants: []room.Participant{
			{User: testUser("u1"), Ready: true},
			{User: testUser("u2"), Ready: true},
			{User: testUser("u3"), Ready: true},
			{User: testUser("u4"), Ready: true},
		},
	}

	_, err := service.Start(testRoom)
	if !errors.Is(err, ErrRoomOverCapacity) {
		t.Fatalf("expected ErrRoomOverCapacity, got %v", err)
	}
}

func testUser(id string) guest.PublicUser {
	return guest.PublicUser{ID: id, Nickname: id}
}
