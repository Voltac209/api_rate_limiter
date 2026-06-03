package ratelimiter

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RateLimitConfig struct {
	Limit         uint32
	WindowSeconds uint32
}

type ConfigStore interface {
	Get(ctx context.Context, key string) (RateLimitConfig, bool, error)
	Close()
}

type PostgresConfigStore struct {
	pool *pgxpool.Pool
}

func NewPostgresConfigStore(
	ctx context.Context,
	databaseURL string,
) (*PostgresConfigStore, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, err
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}

	return &PostgresConfigStore{pool: pool}, nil
}

func (s *PostgresConfigStore) Get(
	ctx context.Context,
	key string,
) (RateLimitConfig, bool, error) {
	var limit int32
	var windowSeconds int32

	err := s.pool.QueryRow(
		ctx,
		`
  			SELECT limit_value, window_seconds
  			FROM rate_limit_configs
  			WHERE key = $1
  		`,
		key,
	).Scan(&limit, &windowSeconds)

	if errors.Is(err, pgx.ErrNoRows) {
		return RateLimitConfig{}, false, nil
	}
	if err != nil {
		return RateLimitConfig{}, false, err
	}

	return RateLimitConfig{
		Limit:         uint32(limit),
		WindowSeconds: uint32(windowSeconds),
	}, true, nil
}

func (s *PostgresConfigStore) Close() {
	s.pool.Close()
}
