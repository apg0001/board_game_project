package connection

import (
	"sync"
	"time"
)

type Status string

const (
	StatusOnline       Status = "ONLINE"
	StatusDisconnected Status = "DISCONNECTED"
)

type Presence struct {
	UserID         string    `json:"userId"`
	RoomID         string    `json:"roomId"`
	SessionID      string    `json:"sessionId,omitempty"`
	Status         Status    `json:"status"`
	DisconnectedAt time.Time `json:"disconnectedAt,omitempty"`
	ExpiresAt      time.Time `json:"expiresAt,omitempty"`
}

type Clock func() time.Time

type Service struct {
	mu     sync.RWMutex
	clock  Clock
	grace  time.Duration
	byUser map[string]Presence
}

func NewService(clock Clock, grace time.Duration) *Service {
	if clock == nil {
		clock = time.Now
	}
	if grace <= 0 {
		grace = 60 * time.Second
	}
	return &Service{
		clock:  clock,
		grace:  grace,
		byUser: make(map[string]Presence),
	}
}

func (s *Service) MarkOnline(userID string, roomID string, sessionID string) Presence {
	s.mu.Lock()
	defer s.mu.Unlock()

	presence := Presence{
		UserID:    userID,
		RoomID:    roomID,
		SessionID: sessionID,
		Status:    StatusOnline,
	}
	s.byUser[userID] = presence
	return presence
}

func (s *Service) MarkDisconnected(userID string) (Presence, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, ok := s.byUser[userID]
	if !ok {
		return Presence{}, false
	}

	now := s.clock().UTC()
	current.Status = StatusDisconnected
	current.DisconnectedAt = now
	current.ExpiresAt = now.Add(s.grace)
	s.byUser[userID] = current
	return current, true
}

func (s *Service) Resume(userID string) (Presence, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, ok := s.byUser[userID]
	if !ok {
		return Presence{}, false
	}
	if current.Status == StatusDisconnected && s.clock().UTC().After(current.ExpiresAt) {
		delete(s.byUser, userID)
		return Presence{}, false
	}

	current.Status = StatusOnline
	current.DisconnectedAt = time.Time{}
	current.ExpiresAt = time.Time{}
	s.byUser[userID] = current
	return current, true
}

func (s *Service) ExpireDisconnected() []Presence {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.clock().UTC()
	expired := make([]Presence, 0)
	for userID, current := range s.byUser {
		if current.Status != StatusDisconnected || current.ExpiresAt.IsZero() || now.Before(current.ExpiresAt) {
			continue
		}
		expired = append(expired, current)
		delete(s.byUser, userID)
	}
	return expired
}

func (s *Service) GracePeriod() time.Duration {
	return s.grace
}
