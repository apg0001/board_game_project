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

type Service struct {
	mu    sync.RWMutex
	clock Clock
	stats map[string]UserGameStats
}

func NewService(clock Clock) *Service {
	if clock == nil {
		clock = time.Now
	}
	return &Service{clock: clock, stats: make(map[string]UserGameStats)}
}

func (s *Service) RecordSession(finished session.Session) {
	if finished.Status != session.StatusFinished || len(finished.Results) == 0 {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	for _, result := range finished.Results {
		key := string(result.PlayerID) + ":" + finished.GameID
		stats := s.stats[key]
		if stats.UserID == "" {
			stats.UserID = string(result.PlayerID)
			stats.GameID = finished.GameID
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
		stats.UpdatedAt = s.clock().UTC()
		s.stats[key] = stats
	}
}

func (s *Service) Leaderboard(gameID string, limit int) []UserGameStats {
	if limit <= 0 {
		limit = 20
	}
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
