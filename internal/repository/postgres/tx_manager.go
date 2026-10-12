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

func NewTxManager(
	pool *pgxpool.Pool,
	queryTimeout time.Duration,
) *PoolTxManager {
	return &PoolTxManager{
		pool:         pool,
		queryTimeout: queryTimeout,
	}
}

func (m *PoolTxManager) Do(
	ctx context.Context,
	fn func(ctx context.Context) error,
) (err error) {
	if _, ok := extractTx(ctx); ok {
		return fn(ctx)
	}

	beginCtx, beginCancel := context.WithTimeout(
		ctx,
		m.queryTimeout,
	)
	defer beginCancel()

	tx, err := m.pool.BeginTx(
		beginCtx,
		pgx.TxOptions{
			IsoLevel: pgx.ReadCommitted,
		},
	)
	if err != nil {
		return fmt.Errorf(
			"begin tx: %w",
			err,
		)
	}

	defer func() {
		if panicValue := recover(); panicValue != nil {
			rollbackCtx, cancel := context.WithTimeout(
				context.WithoutCancel(ctx),
				m.queryTimeout,
			)
			defer cancel()

			_ = tx.Rollback(rollbackCtx)

			err = fmt.Errorf(
				"transaction panic: %v",
				panicValue,
			)
		}
	}()

	ctxWithTx := context.WithValue(
		ctx,
		txKey,
		tx,
	)

	if err := fn(ctxWithTx); err != nil {
		rollbackCtx, cancel := context.WithTimeout(
			context.WithoutCancel(ctx),
			m.queryTimeout,
		)
		defer cancel()

		_ = tx.Rollback(rollbackCtx)

		return err
	}

	commitCtx, commitCancel := context.WithTimeout(
		ctx,
		m.queryTimeout,
	)
	defer commitCancel()

	if err := tx.Commit(commitCtx); err != nil {
		return fmt.Errorf(
			"commit tx: %w",
			err,
		)
	}

	return nil
}

func extractTx(ctx context.Context) (pgx.Tx, bool) {
	tx, ok := ctx.Value(txKey).(pgx.Tx)
	return tx, ok
}
