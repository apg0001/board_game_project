package session

import (
	"errors"
	"sync"
)

var ErrSessionNotFound = errors.New("session not found")

type Store interface {
	Save(session Session) error
	FindByID(id string) (Session, error)
}

type MemoryStore struct {
	mu   sync.RWMutex
	byID map[string]Session
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{byID: make(map[string]Session)}
}

func (s *MemoryStore) Save(session Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.byID[session.ID] = session
	return nil
}

func (s *MemoryStore) FindByID(id string) (Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, ok := s.byID[id]
	if !ok {
		return Session{}, ErrSessionNotFound
	}
	return session, nil
}
