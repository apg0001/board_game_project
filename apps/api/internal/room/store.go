package room

import (
	"errors"
	"sync"
)

var (
	ErrRoomNotFound = errors.New("room not found")
	ErrCodeExists   = errors.New("room code already exists")
)

type Store interface {
	Save(room Room) error
	FindByID(id string) (Room, error)
	FindByCode(code string) (Room, error)
	List() []Room
	CodeExists(code string) bool
}

type MemoryStore struct {
	mu     sync.RWMutex
	byID   map[string]Room
	byCode map[string]string
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		byID:   make(map[string]Room),
		byCode: make(map[string]string),
	}
}

func (s *MemoryStore) Save(room Room) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if existingID, ok := s.byCode[room.Code]; ok && existingID != room.ID {
		return ErrCodeExists
	}

	s.byID[room.ID] = room
	s.byCode[room.Code] = room.ID
	return nil
}

func (s *MemoryStore) FindByID(id string) (Room, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	room, ok := s.byID[id]
	if !ok {
		return Room{}, ErrRoomNotFound
	}
	return room, nil
}

func (s *MemoryStore) FindByCode(code string) (Room, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	id, ok := s.byCode[code]
	if !ok {
		return Room{}, ErrRoomNotFound
	}
	return s.byID[id], nil
}

func (s *MemoryStore) List() []Room {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rooms := make([]Room, 0, len(s.byID))
	for _, room := range s.byID {
		rooms = append(rooms, room)
	}
	return rooms
}

func (s *MemoryStore) CodeExists(code string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, ok := s.byCode[code]
	return ok
}
