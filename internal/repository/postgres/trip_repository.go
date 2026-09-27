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
	Finish(ctx context.Context, id uuid.UUID) (*domain.Trip, error)
	AddStatusHistory(ctx context.Context, tripID uuid.UUID, from, to domain.TripStatus, reason string) error
}

type executor interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}
type tripRepository struct {
	pool *pgxpool.Pool
}

func NewTripRepository(pool *pgxpool.Pool) *tripRepository {
	return &tripRepository{pool: pool}
}
func (r *tripRepository) getExecutor(ctx context.Context) executor {
	if tx, ok := extractTx(ctx); ok {
		return tx
	}
	return r.pool
}

func (r *tripRepository) Create(ctx context.Context, trip domain.Trip) error {
	query, args, err := squirrel.
		Insert("trips").
		Columns(
			"id", "user_id", "driver_id",
			"start_latitude", "start_longitude",
			"end_latitude", "end_longitude",
			"price", "status",
			"started_at", "finished_at",
			"created_at", "updated_at",
		).
		Values(
			trip.ID, trip.UserID, trip.DriverID,
			trip.StartLatitude, trip.StartLongitude,
			trip.EndLatitude, trip.EndLongitude,
			trip.Price, trip.Status,
			trip.StartedAt, trip.FinishedAt,
			trip.CreatedAt, trip.UpdatedAt,
		).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert query: %w", err)
	}

	_, err = r.getExecutor(ctx).Exec(ctx, query, args...)
	if err != nil {
		if isUniqueViolation(err) {
			return domain.ErrDriverBusy
		}
		return fmt.Errorf("exec insert: %w", err)
	}

	return nil
}

func (r *tripRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Trip, error) {
	query, args, err := squirrel.
		Select(
			"id", "user_id", "driver_id",
			"start_latitude", "start_longitude",
			"end_latitude", "end_longitude",
			"price", "status",
			"started_at", "finished_at",
			"created_at", "updated_at",
		).
		From("trips").
		Where(squirrel.Eq{"id": id}).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build select query: %w", err)
	}

	row := r.getExecutor(ctx).QueryRow(ctx, query, args...)

	var trip domain.Trip
	err = row.Scan(
		&trip.ID, &trip.UserID, &trip.DriverID,
		&trip.StartLatitude, &trip.StartLongitude,
		&trip.EndLatitude, &trip.EndLongitude,
		&trip.Price, &trip.Status,
		&trip.StartedAt, &trip.FinishedAt,
		&trip.CreatedAt, &trip.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrTripNotFound
		}
		return nil, fmt.Errorf("scan trip: %w", err)
	}

	return &trip, nil
}

func (r *tripRepository) Finish(ctx context.Context, id uuid.UUID) (*domain.Trip, error) {
	now := time.Now().UTC()

	query, args, err := squirrel.
		Update("trips").
		Set("status", domain.TripStatusCompleted).
		Set("finished_at", now).
		Set("updated_at", now).
		Where(squirrel.Eq{"id": id}).
		Where(squirrel.Eq{"status": domain.TripStatusActive}).   
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build update query: %w", err)
	}

	tag, err := r.getExecutor(ctx).Exec(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("exec update: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, domain.ErrTripCompleted   
	}

	return r.GetByID(ctx, id)
}

func (r *tripRepository) AddStatusHistory(
	ctx context.Context,
	tripID uuid.UUID,
	from, to domain.TripStatus,
	reason string,
) error {
	var fromVal any
	if from != "" {
		fromVal = string(from)
	}
	query, args, err := squirrel.
		Insert("trip_status_history").
		Columns("trip_id", "from_status", "to_status", "reason").
		Values(tripID, fromVal, string(to), reason).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert: %w", err)
	}

	if _, err := r.getExecutor(ctx).Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("exec insert: %w", err)
	}

	return nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}
