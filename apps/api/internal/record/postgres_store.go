package record

import (
	"context"
	"time"

	"board-game-platform/apps/api/internal/gamecore"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct {
	pool *pgxpool.Pool
}

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool}
}

func (s *PostgresStore) ApplyResult(gameID string, result gamecore.Result, updatedAt time.Time) error {
	wins, losses, draws, mmrDelta := resultDelta(result.Outcome)
	_, err := s.pool.Exec(context.Background(), `
		insert into user_game_stats (user_id, game_id, wins, losses, draws, play_count, mmr, updated_at)
		values ($1, $2, $3, $4, $5, 1, greatest(0, 1000 + $6), $7)
		on conflict (user_id, game_id) do update set
			wins = user_game_stats.wins + excluded.wins,
			losses = user_game_stats.losses + excluded.losses,
			draws = user_game_stats.draws + excluded.draws,
			play_count = user_game_stats.play_count + 1,
			mmr = greatest(0, user_game_stats.mmr + $6),
			updated_at = excluded.updated_at
	`, string(result.PlayerID), gameID, wins, losses, draws, mmrDelta, updatedAt)
	return err
}

func (s *PostgresStore) Leaderboard(gameID string, limit int) []UserGameStats {
	if limit <= 0 {
		limit = 20
	}

	rows, err := s.pool.Query(context.Background(), `
		select user_id, game_id, wins, losses, draws, play_count, mmr, updated_at
		from user_game_stats
		where $1 = '' or game_id = $1
		order by mmr desc, wins desc, play_count asc
		limit $2
	`, gameID, limit)
	if err != nil {
		return []UserGameStats{}
	}
	defer rows.Close()

	stats := make([]UserGameStats, 0)
	for rows.Next() {
		var row UserGameStats
		if err := rows.Scan(&row.UserID, &row.GameID, &row.Wins, &row.Losses, &row.Draws, &row.PlayCount, &row.MMR, &row.UpdatedAt); err != nil {
			return []UserGameStats{}
		}
		stats = append(stats, row)
	}
	return stats
}

func resultDelta(outcome gamecore.Outcome) (wins int, losses int, draws int, mmrDelta int) {
	switch outcome {
	case gamecore.OutcomeWin:
		return 1, 0, 0, 24
	case gamecore.OutcomeDraw:
		return 0, 0, 1, 4
	default:
		return 0, 1, 0, -16
	}
}
