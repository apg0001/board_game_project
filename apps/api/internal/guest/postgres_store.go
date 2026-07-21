package guest

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct {
	pool *pgxpool.Pool
}

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool}
}

func (s *PostgresStore) Save(user User) error {
	user.Nickname = DisplayNickname(user.Nickname, user.ID)
	_, err := s.pool.Exec(context.Background(), `
		insert into guest_users (id, nickname, session_token, created_at, last_seen_at)
		values ($1, $2, $3, $4, $5)
		on conflict (id) do update set
			nickname = excluded.nickname,
			session_token = excluded.session_token,
			last_seen_at = excluded.last_seen_at
	`, user.ID, user.Nickname, user.SessionToken, user.CreatedAt, user.LastSeenAt)
	return err
}

func (s *PostgresStore) FindByToken(token string) (User, error) {
	row := s.pool.QueryRow(context.Background(), `
		select id, nickname, session_token, created_at, last_seen_at
		from guest_users
		where session_token = $1
	`, token)

	var user User
	if err := row.Scan(&user.ID, &user.Nickname, &user.SessionToken, &user.CreatedAt, &user.LastSeenAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrSessionNotFound
		}
		return User{}, err
	}
	user.Nickname = DisplayNickname(user.Nickname, user.ID)
	return user, nil
}

func (s *PostgresStore) NicknameExists(nickname string) bool {
	var exists bool
	err := s.pool.QueryRow(context.Background(), `select exists(select 1 from guest_users where nickname = $1)`, nickname).Scan(&exists)
	return err == nil && exists
}
