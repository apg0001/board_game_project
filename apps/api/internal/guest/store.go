package guest

import (
	"errors"
	"sync"
)

var ErrSessionNotFound = errors.New("guest session not found")

type Store interface {
	Save(user User) error
	FindByToken(token string) (User, error)
	NicknameExists(nickname string) bool
}

type MemoryStore struct {
	mu      sync.RWMutex
	byToken map[string]User
	byName  map[string]struct{}
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		byToken: make(map[string]User),
		byName:  make(map[string]struct{}),
	}
}

func (s *MemoryStore) Save(user User) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	user.Nickname = DisplayNickname(user.Nickname, user.ID)
	s.byToken[user.SessionToken] = user
	s.byName[user.Nickname] = struct{}{}
	return nil
}

func (s *MemoryStore) FindByToken(token string) (User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, ok := s.byToken[token]
	if !ok {
		return User{}, ErrSessionNotFound
	}
	user.Nickname = DisplayNickname(user.Nickname, user.ID)
	return user, nil
}

func (s *MemoryStore) NicknameExists(nickname string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, ok := s.byName[nickname]
	return ok
}
