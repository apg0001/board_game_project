package guest

import (
	"strings"
	"time"
)

type User struct {
	ID           string    `json:"id"`
	Nickname     string    `json:"nickname"`
	SessionToken string    `json:"sessionToken,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
	LastSeenAt   time.Time `json:"lastSeenAt"`
}

type PublicUser struct {
	ID         string    `json:"id"`
	Nickname   string    `json:"nickname"`
	CreatedAt  time.Time `json:"createdAt"`
	LastSeenAt time.Time `json:"lastSeenAt"`
}

func (u User) Public() PublicUser {
	return PublicUser{
		ID:         u.ID,
		Nickname:   DisplayNickname(u.Nickname, u.ID),
		CreatedAt:  u.CreatedAt,
		LastSeenAt: u.LastSeenAt,
	}
}

func DisplayNickname(nickname string, fallback string) string {
	nickname = strings.TrimSpace(nickname)
	if nickname != "" {
		return nickname
	}
	fallback = strings.TrimSpace(fallback)
	if fallback != "" {
		return fallback
	}
	return "Player"
}
