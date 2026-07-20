package record

import (
	"testing"
	"time"

	"board-game-platform/apps/api/internal/gamecore"
	"board-game-platform/apps/api/internal/session"
)

func TestRecordSessionUpdatesLeaderboard(t *testing.T) {
	service := NewService(func() time.Time { return time.Date(2026, 7, 20, 1, 0, 0, 0, time.UTC) })
	service.RecordSession(session.Session{
		ID:     "s1",
		GameID: "davinci",
		Status: session.StatusFinished,
		Results: []gamecore.Result{
			{PlayerID: "u1", Outcome: gamecore.OutcomeWin},
			{PlayerID: "u2", Outcome: gamecore.OutcomeLose},
		},
	})

	rows := service.Leaderboard("davinci", 10)
	if len(rows) != 2 {
		t.Fatalf("expected two rows, got %d", len(rows))
	}
	if rows[0].UserID != "u1" || rows[0].MMR <= rows[1].MMR {
		t.Fatalf("unexpected leaderboard: %+v", rows)
	}
}
