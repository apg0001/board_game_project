package connection

import (
	"testing"
	"time"
)

func TestResumeWithinGrace(t *testing.T) {
	now := time.Date(2026, 7, 20, 1, 0, 0, 0, time.UTC)
	service := NewService(func() time.Time { return now }, time.Minute)

	service.MarkOnline("u1", "r1", "s1")
	service.MarkDisconnected("u1")

	presence, ok := service.Resume("u1")
	if !ok {
		t.Fatal("expected resume")
	}
	if presence.Status != StatusOnline {
		t.Fatalf("expected online, got %s", presence.Status)
	}
}

func TestResumeExpiresAfterGrace(t *testing.T) {
	now := time.Date(2026, 7, 20, 1, 0, 0, 0, time.UTC)
	service := NewService(func() time.Time { return now }, time.Minute)

	service.MarkOnline("u1", "r1", "s1")
	service.MarkDisconnected("u1")
	now = now.Add(61 * time.Second)

	_, ok := service.Resume("u1")
	if ok {
		t.Fatal("expected expired resume")
	}
}

func TestExpireDisconnectedReturnsExpiredPresence(t *testing.T) {
	now := time.Date(2026, 7, 20, 1, 0, 0, 0, time.UTC)
	service := NewService(func() time.Time { return now }, time.Minute)

	service.MarkOnline("u1", "r1", "s1")
	service.MarkDisconnected("u1")
	service.MarkOnline("u2", "r1", "s1")
	now = now.Add(61 * time.Second)

	expired := service.ExpireDisconnected()
	if len(expired) != 1 {
		t.Fatalf("expected one expired presence, got %d", len(expired))
	}
	if expired[0].UserID != "u1" {
		t.Fatalf("expected u1 to expire, got %s", expired[0].UserID)
	}
	if _, ok := service.Resume("u1"); ok {
		t.Fatal("expired presence should be removed")
	}
	if _, ok := service.Resume("u2"); !ok {
		t.Fatal("online presence should remain resumable")
	}
}
