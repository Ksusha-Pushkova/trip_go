// Package postgres provides PostgreSQL repository and connection pool implementations.
package postgres

import (
	"context"
	"fmt"

	"github.com/Ksusha-Pushkova/trip_go/internal/config"
	"github.com/exaring/otelpgx"
	"github.com/jackc/pgx/v5/pgxpool"
)

// NewPool creates and configures a PostgreSQL connection pool.
func NewPool(
	ctx context.Context,
	cfg config.Config,
) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.Database.URL)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}

	poolCfg.MaxConns = cfg.Database.MaxConns
	poolCfg.MinConns = cfg.Database.MinConns
	poolCfg.MaxConnLifetime = cfg.Database.MaxConnLifetime
	poolCfg.MaxConnIdleTime = cfg.Database.MaxConnIdleTime
	poolCfg.ConnConfig.ConnectTimeout = cfg.Database.ConnectTimeout
	poolCfg.ConnConfig.Tracer = otelpgx.NewTracer()

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}

	if err := otelpgx.RecordStats(pool); err != nil {
		pool.Close()
		return nil, fmt.Errorf("record postgres pool stats: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(
		ctx,
		cfg.Database.ConnectTimeout,
	)
	defer cancel()

	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return pool, nil
}
