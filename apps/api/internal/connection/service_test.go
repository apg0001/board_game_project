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
