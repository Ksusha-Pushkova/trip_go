package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	tripdomain "github.com/Ksusha-Pushkova/trip_go/internal/domain/trip"
)

// TripRepository defines operations for managing trips in PostgreSQL.
type TripRepository interface {
	Create(ctx context.Context, trip tripdomain.Trip) error
	GetByID(ctx context.Context, id uuid.UUID) (*tripdomain.Trip, error)
	Finish(ctx context.Context, id uuid.UUID, at time.Time) (bool, error)
	CountActive(ctx context.Context) (int64, error)
	AddStatusHistory(
		ctx context.Context,
		tripID uuid.UUID,
		from,
		to tripdomain.Status,
		reason string,
	) error
}

type executor interface {
	Exec(
		ctx context.Context,
		sql string,
		args ...any,
	) (pgconn.CommandTag, error)

	Query(
		ctx context.Context,
		sql string,
		args ...any,
	) (pgx.Rows, error)

	QueryRow(
		ctx context.Context,
		sql string,
		args ...any,
	) pgx.Row
}

type tripRepository struct {
	pool         *pgxpool.Pool
	queryTimeout time.Duration
}

// NewTripRepository creates a new PostgreSQL trip repository.
func NewTripRepository(
	pool *pgxpool.Pool,
	queryTimeout time.Duration,
) TripRepository {
	return &tripRepository{
		pool:         pool,
		queryTimeout: queryTimeout,
	}
}

func (r *tripRepository) getExecutor(
	ctx context.Context,
) executor {
	if tx, ok := extractTx(ctx); ok {
		return tx
	}

	return r.pool
}

func (r *tripRepository) queryContext(
	ctx context.Context,
) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, r.queryTimeout)
}

func (r *tripRepository) Create(
	ctx context.Context,
	trip tripdomain.Trip,
) error {
	query, args, err := squirrel.
		Insert("trips").
		Columns(
			"id",
			"user_id",
			"driver_id",
			"start_latitude",
			"start_longitude",
			"end_latitude",
			"end_longitude",
			"price",
			"status",
			"started_at",
			"finished_at",
			"created_at",
			"updated_at",
		).
		Values(
			trip.ID,
			trip.UserID,
			trip.DriverID,
			trip.Start.Latitude,
			trip.Start.Longitude,
			trip.End.Latitude,
			trip.End.Longitude,
			trip.Price,
			trip.Status,
			trip.StartedAt,
			trip.FinishedAt,
			trip.CreatedAt,
			trip.UpdatedAt,
		).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()

	if err != nil {
		return fmt.Errorf("build insert query: %w", err)
	}

	queryCtx, cancel := r.queryContext(ctx)
	defer cancel()

	_, err = r.getExecutor(ctx).Exec(
		queryCtx,
		query,
		args...,
	)
	if err != nil {
		if isDriverBusyViolation(err) {
			return tripdomain.ErrDriverBusy
		}

		return fmt.Errorf("exec insert: %w", err)
	}

	return nil
}

func (r *tripRepository) GetByID(
	ctx context.Context,
	id uuid.UUID,
) (*tripdomain.Trip, error) {
	query, args, err := squirrel.
		Select(
			"id",
			"user_id",
			"driver_id",
			"start_latitude",
			"start_longitude",
			"end_latitude",
			"end_longitude",
			"price",
			"status",
			"started_at",
			"finished_at",
			"created_at",
			"updated_at",
		).
		From("trips").
		Where(squirrel.Eq{"id": id}).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()

	if err != nil {
		return nil, fmt.Errorf("build select query: %w", err)
	}

	queryCtx, cancel := r.queryContext(ctx)
	defer cancel()

	row := r.getExecutor(ctx).QueryRow(
		queryCtx,
		query,
		args...,
	)

	var trip tripdomain.Trip

	err = row.Scan(
		&trip.ID,
		&trip.UserID,
		&trip.DriverID,
		&trip.Start.Latitude,
		&trip.Start.Longitude,
		&trip.End.Latitude,
		&trip.End.Longitude,
		&trip.Price,
		&trip.Status,
		&trip.StartedAt,
		&trip.FinishedAt,
		&trip.CreatedAt,
		&trip.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, tripdomain.ErrTripNotFound
		}

		return nil, fmt.Errorf("scan trip: %w", err)
	}

	return &trip, nil
}

func (r *tripRepository) CountActive(ctx context.Context) (int64, error) {
	queryCtx, cancel := r.queryContext(ctx)
	defer cancel()

	var count int64

	err := r.getExecutor(ctx).QueryRow(
		queryCtx,
		`SELECT COUNT(*) FROM trips WHERE status = $1`,
		tripdomain.StatusActive,
	).Scan(&count)

	if err != nil {
		return 0, fmt.Errorf("count active trips: %w", err)
	}

	return count, nil
}

func (r *tripRepository) Finish(
	ctx context.Context,
	id uuid.UUID,
	at time.Time,
) (bool, error) {
	query, args, err := squirrel.
		Update("trips").
		Set("status", tripdomain.StatusCompleted).
		Set("finished_at", at).
		Set("updated_at", at).
		Where(squirrel.Eq{"id": id}).
		Where(squirrel.Eq{
			"status": tripdomain.StatusActive,
		}).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()

	if err != nil {
		return false, fmt.Errorf("build update query: %w", err)
	}

	queryCtx, cancel := r.queryContext(ctx)
	defer cancel()

	tag, err := r.getExecutor(ctx).Exec(
		queryCtx,
		query,
		args...,
	)
	if err != nil {
		return false, fmt.Errorf("exec update: %w", err)
	}

	return tag.RowsAffected() == 1, nil
}

func (r *tripRepository) AddStatusHistory(
	ctx context.Context,
	tripID uuid.UUID,
	from,
	to tripdomain.Status,
	reason string,
) error {
	var fromValue any

	if from != "" {
		fromValue = string(from)
	}

	query, args, err := squirrel.
		Insert("trip_status_history").
		Columns(
			"trip_id",
			"from_status",
			"to_status",
			"reason",
		).
		Values(
			tripID,
			fromValue,
			string(to),
			reason,
		).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()

	if err != nil {
		return fmt.Errorf("build insert: %w", err)
	}

	queryCtx, cancel := r.queryContext(ctx)
	defer cancel()

	if _, err := r.getExecutor(ctx).Exec(
		queryCtx,
		query,
		args...,
	); err != nil {
		return fmt.Errorf("exec insert: %w", err)
	}

	return nil
}

func isDriverBusyViolation(err error) bool {
	var pgErr *pgconn.PgError

	if !errors.As(err, &pgErr) {
		return false
	}

	return pgErr.Code == "23505" &&
		pgErr.ConstraintName == "trips_driver_active_uniq"
}
