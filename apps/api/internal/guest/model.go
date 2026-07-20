package guest

import "time"

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
		Nickname:   u.Nickname,
		CreatedAt:  u.CreatedAt,
		LastSeenAt: u.LastSeenAt,
	}
}
