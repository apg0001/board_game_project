package session

import (
	"time"

	"board-game-platform/apps/api/internal/gamecore"
)

type Status string

const (
	StatusActive   Status = "ACTIVE"
	StatusFinished Status = "FINISHED"
	StatusAborted  Status = "ABORTED"
)

type Session struct {
	ID        string            `json:"id"`
	RoomID    string            `json:"roomId"`
	GameID    string            `json:"gameId"`
	Status    Status            `json:"status"`
	State     any               `json:"state"`
	Results   []gamecore.Result `json:"results"`
	CreatedAt time.Time         `json:"createdAt"`
	UpdatedAt time.Time         `json:"updatedAt"`
}
