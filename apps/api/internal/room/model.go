package room

import (
	"time"

	"board-game-platform/apps/api/internal/guest"
)

type Status string
type Visibility string

const (
	StatusLobby    Status = "LOBBY"
	StatusPlaying  Status = "PLAYING"
	StatusFinished Status = "FINISHED"
	StatusClosed   Status = "CLOSED"
)

const (
	VisibilityPrivate Visibility = "PRIVATE"
	VisibilityPublic  Visibility = "PUBLIC"
)

type Participant struct {
	User      guest.PublicUser `json:"user"`
	Ready     bool             `json:"ready"`
	Host      bool             `json:"host"`
	SeatIndex int              `json:"seatIndex"`
	JoinedAt  time.Time        `json:"joinedAt"`
}

type Spectator struct {
	User     guest.PublicUser `json:"user"`
	JoinedAt time.Time        `json:"joinedAt"`
}

type RuleVote struct {
	UserID    string            `json:"userId"`
	Choices   map[string]string `json:"choices"`
	CreatedAt time.Time         `json:"createdAt"`
}

type Options struct {
	TurnSeconds     int  `json:"turnSeconds"`
	MaxWaitSeconds  int  `json:"maxWaitSeconds"`
	AutoStart       bool `json:"autoStart"`
	AllowSpectators bool `json:"allowSpectators"`
}

type Room struct {
	ID               string         `json:"id"`
	Code             string         `json:"code"`
	GameID           string         `json:"gameId"`
	Visibility       Visibility     `json:"visibility"`
	Status           Status         `json:"status"`
	ActiveSessionID  string         `json:"activeSessionId,omitempty"`
	PlayingPlayerIDs []string       `json:"playingPlayerIds,omitempty"`
	MaxPlayers       int            `json:"maxPlayers"`
	HostUserID       string         `json:"hostUserId"`
	Participants     []Participant  `json:"participants"`
	Spectators       []Spectator    `json:"spectators"`
	Options          Options        `json:"options"`
	RuleVotes        []RuleVote     `json:"ruleVotes,omitempty"`
	GameRules        map[string]any `json:"gameRules,omitempty"`
	RuleMessages     []string       `json:"ruleMessages,omitempty"`
	CreatedAt        time.Time      `json:"createdAt"`
	UpdatedAt        time.Time      `json:"updatedAt"`
}

func (r Room) IsFull() bool {
	return len(r.Participants) >= r.MaxPlayers
}

func (r Room) HasSpectator(userID string) bool {
	for _, spectator := range r.Spectators {
		if spectator.User.ID == userID {
			return true
		}
	}
	return false
}

func (r Room) HasParticipant(userID string) bool {
	for _, participant := range r.Participants {
		if participant.User.ID == userID {
			return true
		}
	}
	return false
}
