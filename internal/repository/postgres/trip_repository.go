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

	"github.com/Ksusha-Pushkova/trip_go/internal/domain"
)

type TripRepository interface {
	Create(ctx context.Context, trip domain.Trip) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Trip, error)
	Finish(ctx context.Context, id uuid.UUID) (bool, error)
	AddStatusHistory(
		ctx context.Context,
		tripID uuid.UUID,
		from,
		to domain.TripStatus,
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

func NewTripRepository(
	pool *pgxpool.Pool,
	queryTimeout time.Duration,
) *tripRepository {
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
	trip domain.Trip,
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
		return fmt.Errorf(
			"build insert query: %w",
			err,
		)
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
			return domain.ErrDriverBusy
		}

		return fmt.Errorf(
			"exec insert: %w",
			err,
		)
	}

	return nil
}

func (r *tripRepository) GetByID(
	ctx context.Context,
	id uuid.UUID,
) (*domain.Trip, error) {
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
		return nil, fmt.Errorf(
			"build select query: %w",
			err,
		)
	}

	queryCtx, cancel := r.queryContext(ctx)
	defer cancel()

	row := r.getExecutor(ctx).QueryRow(
		queryCtx,
		query,
		args...,
	)

	var trip domain.Trip

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
			return nil, domain.ErrTripNotFound
		}

		return nil, fmt.Errorf(
			"scan trip: %w",
			err,
		)
	}

	return &trip, nil
}

func (r *tripRepository) Finish(
	ctx context.Context,
	id uuid.UUID,
) (bool, error) {
	now := time.Now().UTC()

	query, args, err := squirrel.
		Update("trips").
		Set("status", domain.TripStatusCompleted).
		Set("finished_at", now).
		Set("updated_at", now).
		Where(squirrel.Eq{"id": id}).
		Where(squirrel.Eq{
			"status": domain.TripStatusActive,
		}).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()

	if err != nil {
		return false, fmt.Errorf(
			"build update query: %w",
			err,
		)
	}

	queryCtx, cancel := r.queryContext(ctx)
	defer cancel()

	tag, err := r.getExecutor(ctx).Exec(
		queryCtx,
		query,
		args...,
	)
	if err != nil {
		return false, fmt.Errorf(
			"exec update: %w",
			err,
		)
	}

	return tag.RowsAffected() == 1, nil
}

func (r *tripRepository) AddStatusHistory(
	ctx context.Context,
	tripID uuid.UUID,
	from,
	to domain.TripStatus,
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
		return fmt.Errorf(
			"build insert: %w",
			err,
		)
	}

	queryCtx, cancel := r.queryContext(ctx)
	defer cancel()

	if _, err := r.getExecutor(ctx).Exec(
		queryCtx,
		query,
		args...,
	); err != nil {
		return fmt.Errorf(
			"exec insert: %w",
			err,
		)
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
