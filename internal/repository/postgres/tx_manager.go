package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type contextKey struct{}

var txKey = contextKey{}

type TxManager interface {
	Do(ctx context.Context, fn func(ctx context.Context) error) error
}

type PoolTxManager struct {
	pool         *pgxpool.Pool
	queryTimeout time.Duration
}

func NewTxManager(pool *pgxpool.Pool, queryTimeout time.Duration) *PoolTxManager {
	return &PoolTxManager{
		pool:         pool,
		queryTimeout: queryTimeout,
	}
}

func (m *PoolTxManager) Do(ctx context.Context, fn func(ctx context.Context) error) error {
	if _, ok := extractTx(ctx); ok {
		return fn(ctx)
	}

	beginCtx, beginCancel := context.WithTimeout(ctx, m.queryTimeout)
	tx, err := m.pool.BeginTx(
		beginCtx,
		pgx.TxOptions{IsoLevel: pgx.ReadCommitted},
	)
	beginCancel()

	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			rollbackCtx, cancel := context.WithTimeout(
				context.WithoutCancel(ctx),
				m.queryTimeout,
			)
			_ = tx.Rollback(rollbackCtx)
			cancel()

			panic(p)
		}
	}()

	ctxWithTx := context.WithValue(ctx, txKey, tx)

	if err := fn(ctxWithTx); err != nil {
		rollbackCtx, cancel := context.WithTimeout(
			context.WithoutCancel(ctx),
			m.queryTimeout,
		)
		_ = tx.Rollback(rollbackCtx)
		cancel()

		return err
	}

	commitCtx, commitCancel := context.WithTimeout(ctx, m.queryTimeout)
	err = tx.Commit(commitCtx)
	commitCancel()

	if err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	return nil
}

func extractTx(ctx context.Context) (pgx.Tx, bool) {
	tx, ok := ctx.Value(txKey).(pgx.Tx)
	return tx, ok
}
