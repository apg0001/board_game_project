package guest

import (
	"strings"
	"testing"
	"time"
)

func TestCreateGuest(t *testing.T) {
	service := NewService(NewMemoryStore(), fixedClock())

	user, err := service.Create()
	if err != nil {
		t.Fatal(err)
	}

	if !strings.HasPrefix(user.ID, "guest_") {
		t.Fatalf("expected guest id prefix, got %s", user.ID)
	}
	if !strings.HasPrefix(user.Nickname, "Guest_") {
		t.Fatalf("expected guest nickname prefix, got %s", user.Nickname)
	}
	if user.SessionToken == "" {
		t.Fatal("expected session token")
	}
}

func TestMeRefreshesLastSeenAt(t *testing.T) {
	first := time.Date(2026, 7, 20, 1, 0, 0, 0, time.UTC)
	second := first.Add(10 * time.Second)
	now := first
	service := NewService(NewMemoryStore(), func() time.Time { return now })

	user, err := service.Create()
	if err != nil {
		t.Fatal(err)
	}

	now = second
	found, err := service.Me(user.SessionToken)
	if err != nil {
		t.Fatal(err)
	}

	if !found.LastSeenAt.Equal(second) {
		t.Fatalf("expected refreshed last seen, got %s", found.LastSeenAt)
	}
}

func TestMeRejectsUnknownToken(t *testing.T) {
	service := NewService(NewMemoryStore(), fixedClock())

	_, err := service.Me("missing")
	if err == nil {
		t.Fatal("expected error")
	}
	if err != ErrSessionNotFound {
		t.Fatalf("expected ErrSessionNotFound, got %v", err)
	}
}

func fixedClock() Clock {
	return func() time.Time {
		return time.Date(2026, 7, 20, 1, 0, 0, 0, time.UTC)
	}
}
