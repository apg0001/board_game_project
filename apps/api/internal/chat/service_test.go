package chat

import (
	"testing"
	"time"

	"board-game-platform/apps/api/internal/guest"
)

func TestAddAndRecentKeepsLimit(t *testing.T) {
	service := NewService(func() time.Time { return time.Date(2026, 7, 20, 1, 0, 0, 0, time.UTC) }, 2)
	user := guest.PublicUser{ID: "u1", Nickname: "Guest_1001"}

	service.Add("r1", user, "hello", "chat")
	service.Add("r1", user, "🙂", "emoji")
	service.Add("r1", user, "last", "chat")

	messages := service.Recent("r1")
	if len(messages) != 2 {
		t.Fatalf("expected two messages, got %d", len(messages))
	}
	if messages[0].Text != "🙂" || messages[1].Text != "last" {
		t.Fatalf("unexpected messages: %+v", messages)
	}
}
