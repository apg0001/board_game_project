package auth

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

func (s *PostgresStore) FindAccountByUsername(username string) (Account, bool) {
	row := s.pool.QueryRow(context.Background(), `
		select id, username, nickname, created_at, salt, password_hash
		from app_users
		where username = $1
	`, username)

	var account Account
	if err := row.Scan(&account.ID, &account.Username, &account.Nickname, &account.CreatedAt, &account.Salt, &account.PasswordHash); err != nil {
		return Account{}, false
	}
	return account, true
}

func (s *PostgresStore) SaveAccount(account Account) error {
	_, err := s.pool.Exec(context.Background(), `
		insert into app_users (id, username, nickname, created_at, salt, password_hash)
		values ($1, $2, $3, $4, $5, $6)
	`, account.ID, account.Username, account.Nickname, account.CreatedAt, account.Salt, account.PasswordHash)
	return err
}

func (s *PostgresStore) SaveSession(token string, user User) error {
	_, err := s.pool.Exec(context.Background(), `
		insert into auth_sessions (token, user_id, created_at)
		values ($1, $2, now())
		on conflict (token) do update set user_id = excluded.user_id
	`, token, user.ID)
	return err
}

func (s *PostgresStore) FindSession(token string) (User, bool) {
	row := s.pool.QueryRow(context.Background(), `
		select u.id, u.username, u.nickname, u.created_at
		from auth_sessions s
		join app_users u on u.id = s.user_id
		where s.token = $1
	`, token)

	var user User
	if err := row.Scan(&user.ID, &user.Username, &user.Nickname, &user.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, false
		}
		return User{}, false
	}
	return user, true
}
