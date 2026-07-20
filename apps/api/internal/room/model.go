package room

import (
	"time"

	"board-game-platform/apps/api/internal/guest"
)

type Status string

const (
	StatusLobby    Status = "LOBBY"
	StatusPlaying  Status = "PLAYING"
	StatusFinished Status = "FINISHED"
	StatusClosed   Status = "CLOSED"
)

type Participant struct {
	User      guest.PublicUser `json:"user"`
	Ready     bool             `json:"ready"`
	Host      bool             `json:"host"`
	SeatIndex int              `json:"seatIndex"`
	JoinedAt  time.Time        `json:"joinedAt"`
}

type Room struct {
	ID              string        `json:"id"`
	Code            string        `json:"code"`
	GameID          string        `json:"gameId"`
	Status          Status        `json:"status"`
	ActiveSessionID string        `json:"activeSessionId,omitempty"`
	MaxPlayers      int           `json:"maxPlayers"`
	HostUserID      string        `json:"hostUserId"`
	Participants    []Participant `json:"participants"`
	CreatedAt       time.Time     `json:"createdAt"`
	UpdatedAt       time.Time     `json:"updatedAt"`
}

func (r Room) IsFull() bool {
	return len(r.Participants) >= r.MaxPlayers
}

func (r Room) HasParticipant(userID string) bool {
	for _, participant := range r.Participants {
		if participant.User.ID == userID {
			return true
		}
	}
	return false
}
