package storage

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Postgres struct {
	pool *pgxpool.Pool
}

func OpenPostgres(ctx context.Context, databaseURL string) (*Postgres, error) {
	if databaseURL == "" {
		return nil, errors.New("database url is required")
	}
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, err
	}
	db := &Postgres{pool: pool}
	if err := db.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return db, nil
}

func (db *Postgres) Pool() *pgxpool.Pool {
	return db.pool
}

func (db *Postgres) Ping(ctx context.Context) error {
	pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return db.pool.Ping(pingCtx)
}

func (db *Postgres) Close() {
	db.pool.Close()
}

func (db *Postgres) Migrate(ctx context.Context) error {
	statements := []string{
		`create table if not exists schema_migrations (
			version integer primary key,
			applied_at timestamptz not null default now()
		)`,
		`create table if not exists guest_users (
			id text primary key,
			nickname text not null unique,
			session_token text not null unique,
			created_at timestamptz not null,
			last_seen_at timestamptz not null
		)`,
		`create table if not exists app_users (
			id text primary key,
			username text not null unique,
			nickname text not null,
			role text not null default 'USER',
			created_at timestamptz not null,
			salt text not null,
			password_hash text not null
		)`,
		`alter table app_users add column if not exists role text not null default 'USER'`,
		`create table if not exists auth_sessions (
			token text primary key,
			user_id text not null references app_users(id) on delete cascade,
			created_at timestamptz not null default now()
		)`,
		`create table if not exists user_game_stats (
			user_id text not null,
			game_id text not null,
			wins integer not null default 0,
			losses integer not null default 0,
			draws integer not null default 0,
			play_count integer not null default 0,
			mmr integer not null default 1000,
			updated_at timestamptz not null,
			primary key (user_id, game_id)
		)`,
		`insert into schema_migrations (version) values (1) on conflict (version) do nothing`,
	}
	for _, statement := range statements {
		if _, err := db.pool.Exec(ctx, statement); err != nil {
			return err
		}
	}
	return nil
}
