package match

import "sync"

type Service struct {
	mu            sync.Mutex
	waitingByGame map[string][]string
}

func NewService() *Service {
	return &Service{waitingByGame: make(map[string][]string)}
}

func (s *Service) AddWaiting(gameID string, roomID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.waitingByGame[gameID] = append(s.waitingByGame[gameID], roomID)
}

func (s *Service) NextWaiting(gameID string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rooms := s.waitingByGame[gameID]
	if len(rooms) == 0 {
		return "", false
	}
	roomID := rooms[0]
	s.waitingByGame[gameID] = rooms[1:]
	return roomID, true
}

func (s *Service) Cancel(gameID string, roomID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	rooms := s.waitingByGame[gameID]
	next := make([]string, 0, len(rooms))
	removed := false
	for _, id := range rooms {
		if id == roomID {
			removed = true
			continue
		}
		next = append(next, id)
	}
	s.waitingByGame[gameID] = next
	return removed
}
