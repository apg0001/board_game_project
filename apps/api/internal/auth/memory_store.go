package auth

import "sync"

type MemoryStore struct {
	mu      sync.RWMutex
	byName  map[string]Account
	byToken map[string]User
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		byName:  make(map[string]Account),
		byToken: make(map[string]User),
	}
}

func (s *MemoryStore) FindAccountByUsername(username string) (Account, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	account, ok := s.byName[username]
	return account, ok
}

func (s *MemoryStore) SaveAccount(account Account) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.byName[account.Username] = account
	return nil
}

func (s *MemoryStore) SaveSession(token string, user User) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.byToken[token] = user
	return nil
}

func (s *MemoryStore) FindSession(token string) (User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	user, ok := s.byToken[token]
	return user, ok
}
