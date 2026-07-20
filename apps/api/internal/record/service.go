package record

import (
	"sort"
	"sync"
	"time"

	"board-game-platform/apps/api/internal/gamecore"
	"board-game-platform/apps/api/internal/session"
)

type UserGameStats struct {
	UserID    string    `json:"userId"`
	GameID    string    `json:"gameId"`
	Wins      int       `json:"wins"`
	Losses    int       `json:"losses"`
	Draws     int       `json:"draws"`
	PlayCount int       `json:"playCount"`
	MMR       int       `json:"mmr"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type Clock func() time.Time

type Store interface {
	ApplyResult(gameID string, result gamecore.Result, updatedAt time.Time) error
	Leaderboard(gameID string, limit int) []UserGameStats
}

type Service struct {
	clock Clock
	store Store
}

func NewService(clock Clock) *Service {
	return NewServiceWithStore(NewMemoryStore(), clock)
}

func NewServiceWithStore(store Store, clock Clock) *Service {
	if clock == nil {
		clock = time.Now
	}
	return &Service{clock: clock, store: store}
}

func (s *Service) RecordSession(finished session.Session) {
	if finished.Status != session.StatusFinished || len(finished.Results) == 0 {
		return
	}

	updatedAt := s.clock().UTC()
	for _, result := range finished.Results {
		_ = s.store.ApplyResult(finished.GameID, result, updatedAt)
	}
}

func (s *Service) Leaderboard(gameID string, limit int) []UserGameStats {
	if limit <= 0 {
		limit = 20
	}
	return s.store.Leaderboard(gameID, limit)
}

type MemoryStore struct {
	mu    sync.RWMutex
	stats map[string]UserGameStats
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{stats: make(map[string]UserGameStats)}
}

func (s *MemoryStore) ApplyResult(gameID string, result gamecore.Result, updatedAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := string(result.PlayerID) + ":" + gameID
	stats := s.stats[key]
	if stats.UserID == "" {
		stats.UserID = string(result.PlayerID)
		stats.GameID = gameID
		stats.MMR = 1000
	}
	stats.PlayCount++
	switch result.Outcome {
	case gamecore.OutcomeWin:
		stats.Wins++
		stats.MMR += 24
	case gamecore.OutcomeDraw:
		stats.Draws++
		stats.MMR += 4
	default:
		stats.Losses++
		stats.MMR -= 16
	}
	if stats.MMR < 0 {
		stats.MMR = 0
	}
	stats.UpdatedAt = updatedAt
	s.stats[key] = stats
	return nil
}

func (s *MemoryStore) Leaderboard(gameID string, limit int) []UserGameStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows := make([]UserGameStats, 0, len(s.stats))
	for _, stats := range s.stats {
		if gameID == "" || stats.GameID == gameID {
			rows = append(rows, stats)
		}
	}
	sort.SliceStable(rows, func(i, j int) bool {
		return rows[i].MMR > rows[j].MMR
	})
	if len(rows) > limit {
		rows = rows[:limit]
	}
	return rows
}
