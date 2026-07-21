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
	if room.Spectators == nil {
		t.Fatal("expected empty spectator list")
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

func TestReturnToLobbyClearsReadyAndSession(t *testing.T) {
	service := NewService(NewMemoryStore(), fixedClock())
	created, err := service.Create(testUser("u1", "Guest_1001"), "davinci", 4)
	if err != nil {
		t.Fatal(err)
	}
	playing, err := service.SetPlaying(created.ID, "session_1", []string{"u1"})
	if err != nil {
		t.Fatal(err)
	}
	ready, err := service.ToggleReady(playing.ID, "u1", true)
	if err != nil {
		t.Fatal(err)
	}

	returned, err := service.ReturnToLobby(ready.ID)
	if err != nil {
		t.Fatal(err)
	}

	if returned.Status != StatusLobby || returned.ActiveSessionID != "" {
		t.Fatalf("expected lobby without active session, got %+v", returned)
	}
	if returned.Participants[0].Ready {
		t.Fatal("expected ready to be reset")
	}
}

func TestLeaveReassignsHost(t *testing.T) {
	service := NewService(NewMemoryStore(), fixedClock())
	created, err := service.Create(testUser("u1", "Guest_1001"), "davinci", 4)
	if err != nil {
		t.Fatal(err)
	}
	joined, err := service.JoinByCode(created.Code, testUser("u2", "Guest_1002"))
	if err != nil {
		t.Fatal(err)
	}

	updated, err := service.Leave(joined.ID, "u1")
	if err != nil {
		t.Fatal(err)
	}

	if updated.HostUserID != "u2" {
		t.Fatalf("expected u2 host, got %s", updated.HostUserID)
	}
	if !updated.Participants[0].Host {
		t.Fatal("expected remaining participant to be host")
	}
}

func TestTransferHost(t *testing.T) {
	service := NewService(NewMemoryStore(), fixedClock())
	created, err := service.Create(testUser("u1", "Guest_1001"), "davinci", 4)
	if err != nil {
		t.Fatal(err)
	}
	joined, err := service.JoinByCode(created.Code, testUser("u2", "Guest_1002"))
	if err != nil {
		t.Fatal(err)
	}
	updated, err := service.TransferHost(joined.ID, "u2")
	if err != nil {
		t.Fatal(err)
	}
	if updated.HostUserID != "u2" || !updated.Participants[1].Host {
		t.Fatalf("expected transferred host, got %+v", updated)
	}
}

func TestLeaveClosesEmptyRoom(t *testing.T) {
	service := NewService(NewMemoryStore(), fixedClock())
	created, err := service.Create(testUser("u1", "Guest_1001"), "davinci", 4)
	if err != nil {
		t.Fatal(err)
	}

	updated, err := service.Leave(created.ID, "u1")
	if err != nil {
		t.Fatal(err)
	}

	if updated.Status != StatusClosed {
		t.Fatalf("expected closed room, got %s", updated.Status)
	}
}

func TestJoinSpectatorDoesNotOccupySeat(t *testing.T) {
	service := NewService(NewMemoryStore(), fixedClock())
	created, err := service.Create(testUser("u1", "Guest_1001"), "davinci", 4)
	if err != nil {
		t.Fatal(err)
	}

	updated, err := service.JoinSpectator(created.ID, testUser("u2", "Guest_1002"))
	if err != nil {
		t.Fatal(err)
	}

	if len(updated.Participants) != 1 || len(updated.Spectators) != 1 {
		t.Fatalf("unexpected room users: %+v", updated)
	}
}

func TestUpdateOptionsClampsTurnSeconds(t *testing.T) {
	service := NewService(NewMemoryStore(), fixedClock())
	created, err := service.Create(testUser("u1", "Guest_1001"), "davinci", 4)
	if err != nil {
		t.Fatal(err)
	}

	updated, err := service.UpdateOptions(created.ID, 3)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Options.TurnSeconds != 10 {
		t.Fatalf("expected clamped turn seconds, got %d", updated.Options.TurnSeconds)
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
