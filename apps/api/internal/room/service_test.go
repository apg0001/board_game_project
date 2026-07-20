package room

import (
	"testing"
	"time"

	"board-game-platform/apps/api/internal/guest"
)

func TestCreateRoom(t *testing.T) {
	service := NewService(NewMemoryStore(), fixedClock())

	room, err := service.Create(testUser("u1", "Guest_1001"), "davinci", 4)
	if err != nil {
		t.Fatal(err)
	}

	if room.Code == "" {
		t.Fatal("expected room code")
	}
	if len(room.Participants) != 1 {
		t.Fatalf("expected host participant, got %d", len(room.Participants))
	}
	if !room.Participants[0].Host {
		t.Fatal("expected first participant to be host")
	}
}

func TestJoinByCode(t *testing.T) {
	service := NewService(NewMemoryStore(), fixedClock())
	created, err := service.Create(testUser("u1", "Guest_1001"), "davinci", 4)
	if err != nil {
		t.Fatal(err)
	}

	joined, err := service.JoinByCode(created.Code, testUser("u2", "Guest_1002"))
	if err != nil {
		t.Fatal(err)
	}

	if len(joined.Participants) != 2 {
		t.Fatalf("expected two participants, got %d", len(joined.Participants))
	}
	if joined.Participants[1].SeatIndex != 1 {
		t.Fatalf("expected seat index 1, got %d", joined.Participants[1].SeatIndex)
	}
}

func TestJoinByCodeRejectsFullRoom(t *testing.T) {
	service := NewService(NewMemoryStore(), fixedClock())
	created, err := service.Create(testUser("u1", "Guest_1001"), "davinci", 1)
	if err != nil {
		t.Fatal(err)
	}

	_, err = service.JoinByCode(created.Code, testUser("u2", "Guest_1002"))
	if err != ErrRoomFull {
		t.Fatalf("expected ErrRoomFull, got %v", err)
	}
}

func TestToggleReady(t *testing.T) {
	service := NewService(NewMemoryStore(), fixedClock())
	created, err := service.Create(testUser("u1", "Guest_1001"), "davinci", 4)
	if err != nil {
		t.Fatal(err)
	}

	updated, err := service.ToggleReady(created.ID, "u1", true)
	if err != nil {
		t.Fatal(err)
	}

	if !updated.Participants[0].Ready {
		t.Fatal("expected participant to be ready")
	}
}

func testUser(id string, nickname string) guest.PublicUser {
	now := time.Date(2026, 7, 20, 1, 0, 0, 0, time.UTC)
	return guest.PublicUser{
		ID:         id,
		Nickname:   nickname,
		CreatedAt:  now,
		LastSeenAt: now,
	}
}

func fixedClock() Clock {
	return func() time.Time {
		return time.Date(2026, 7, 20, 1, 0, 0, 0, time.UTC)
	}
}
