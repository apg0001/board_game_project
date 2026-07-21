package chat

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
	"sync"
	"time"

	"board-game-platform/apps/api/internal/guest"
)

type Message struct {
	ID        string           `json:"id"`
	RoomID    string           `json:"roomId"`
	User      guest.PublicUser `json:"user"`
	Text      string           `json:"text"`
	Kind      string           `json:"kind"`
	CreatedAt time.Time        `json:"createdAt"`
}

type Clock func() time.Time

type Service struct {
	mu     sync.RWMutex
	clock  Clock
	limit  int
	byRoom map[string][]Message
}

func NewService(clock Clock, limit int) *Service {
	if clock == nil {
		clock = time.Now
	}
	if limit <= 0 {
		limit = 50
	}
	return &Service{clock: clock, limit: limit, byRoom: make(map[string][]Message)}
}

func (s *Service) Add(roomID string, user guest.PublicUser, text string, kind string) (Message, bool) {
	user.Nickname = guest.DisplayNickname(user.Nickname, user.ID)
	text = strings.TrimSpace(text)
	if text == "" {
		return Message{}, false
	}
	if len([]rune(text)) > 160 {
		text = string([]rune(text)[:160])
	}
	if kind == "" {
		kind = "chat"
	}

	message := Message{
		ID:        "msg_" + randomHex(8),
		RoomID:    roomID,
		User:      user,
		Text:      text,
		Kind:      kind,
		CreatedAt: s.clock().UTC(),
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.byRoom[roomID] = append(s.byRoom[roomID], message)
	if len(s.byRoom[roomID]) > s.limit {
		s.byRoom[roomID] = s.byRoom[roomID][len(s.byRoom[roomID])-s.limit:]
	}
	return message, true
}

func (s *Service) Recent(roomID string) []Message {
	s.mu.RLock()
	defer s.mu.RUnlock()
	messages := s.byRoom[roomID]
	if len(messages) == 0 {
		return []Message{}
	}
	return append([]Message{}, messages...)
}

func randomHex(byteLength int) string {
	buffer := make([]byte, byteLength)
	_, _ = rand.Read(buffer)
	return hex.EncodeToString(buffer)
}
