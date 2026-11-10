//go:build integration

package postgres

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	tripdomain "github.com/Ksusha-Pushkova/trip_go/internal/domain/trip"
)

func newIntegrationPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Fatal("DATABASE_URL is required for integration tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("create database pool: %v", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Fatalf("ping database: %v", err)
	}

	t.Cleanup(pool.Close)

	return pool
}

func newTestTrip() tripdomain.Trip {
	now := time.Now().UTC().Truncate(time.Microsecond)

	return tripdomain.Trip{
		ID:       uuid.New(),
		UserID:   uuid.New(),
		DriverID: uuid.New(),
		Start: tripdomain.Destination{
			Latitude:  55.751244,
			Longitude: 37.618423,
		},
		End: tripdomain.Destination{
			Latitude:  55.761244,
			Longitude: 37.628423,
		},
		Price:     1500,
		Status:    tripdomain.StatusActive,
		StartedAt: now,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func cleanupTestTrip(t *testing.T, pool *pgxpool.Pool, id uuid.UUID) {
	t.Helper()

	t.Cleanup(func() {
		_, _ = pool.Exec(
			context.Background(),
			"DELETE FROM trips WHERE id = $1",
			id,
		)
	})
}

func TestTripRepository_CreateAndGetByID(t *testing.T) {
	pool := newIntegrationPool(t)
	repo := NewTripRepository(pool, 5*time.Second)

	expected := newTestTrip()
	cleanupTestTrip(t, pool, expected.ID)

	ctx := context.Background()

	if err := repo.Create(ctx, expected); err != nil {
		t.Fatalf("create trip: %v", err)
	}

	actual, err := repo.GetByID(ctx, expected.ID)
	if err != nil {
		t.Fatalf("get trip: %v", err)
	}

	if actual.ID != expected.ID {
		t.Errorf("ID = %v, want %v", actual.ID, expected.ID)
	}

	if actual.UserID != expected.UserID {
		t.Errorf("UserID = %v, want %v", actual.UserID, expected.UserID)
	}

	if actual.DriverID != expected.DriverID {
		t.Errorf("DriverID = %v, want %v", actual.DriverID, expected.DriverID)
	}

	if actual.Status != expected.Status {
		t.Errorf("Status = %v, want %v", actual.Status, expected.Status)
	}

	if actual.Price != expected.Price {
		t.Errorf("Price = %v, want %v", actual.Price, expected.Price)
	}
}

func TestTripRepository_Finish(t *testing.T) {
	pool := newIntegrationPool(t)
	repo := NewTripRepository(pool, 5*time.Second)

	trip := newTestTrip()
	cleanupTestTrip(t, pool, trip.ID)

	ctx := context.Background()

	if err := repo.Create(ctx, trip); err != nil {
		t.Fatalf("create trip: %v", err)
	}

	finishedAt := trip.StartedAt.Add(time.Hour)

	updated, err := repo.Finish(ctx, trip.ID, finishedAt)
	if err != nil {
		t.Fatalf("finish trip: %v", err)
	}

	if !updated {
		t.Fatal("expected trip to be updated")
	}

	actual, err := repo.GetByID(ctx, trip.ID)
	if err != nil {
		t.Fatalf("get trip: %v", err)
	}

	if actual.Status != tripdomain.StatusCompleted {
		t.Errorf("status = %v, want %v",
			actual.Status, tripdomain.StatusCompleted)
	}

	if actual.FinishedAt == nil {
		t.Fatal("expected FinishedAt to be set")
	}

	if !actual.FinishedAt.Equal(finishedAt) {
		t.Errorf("FinishedAt = %v, want %v",
			actual.FinishedAt, finishedAt)
	}
}

func TestTripRepository_FinishAlreadyCompleted(t *testing.T) {
	pool := newIntegrationPool(t)
	repo := NewTripRepository(pool, 5*time.Second)

	trip := newTestTrip()
	cleanupTestTrip(t, pool, trip.ID)

	ctx := context.Background()

	if err := repo.Create(ctx, trip); err != nil {
		t.Fatalf("create trip: %v", err)
	}

	finishedAt := trip.StartedAt.Add(time.Hour)

	updated, err := repo.Finish(ctx, trip.ID, finishedAt)
	if err != nil {
		t.Fatalf("first finish: %v", err)
	}
	if !updated {
		t.Fatal("expected first finish to update trip")
	}

	updated, err = repo.Finish(ctx, trip.ID, finishedAt.Add(time.Hour))
	if err != nil {
		t.Fatalf("second finish: %v", err)
	}
	if updated {
		t.Fatal("expected second finish not to update trip")
	}
}

func TestTripRepository_GetByIDNotFound(t *testing.T) {
	pool := newIntegrationPool(t)
	repo := NewTripRepository(pool, 5*time.Second)

	_, err := repo.GetByID(context.Background(), uuid.New())

	if !errors.Is(err, tripdomain.ErrTripNotFound) {
		t.Fatalf("error = %v, want ErrTripNotFound", err)
	}
}

func TestTripRepository_DriverBusy(t *testing.T) {
	pool := newIntegrationPool(t)
	repo := NewTripRepository(pool, 5*time.Second)

	firstTrip := newTestTrip()
	cleanupTestTrip(t, pool, firstTrip.ID)

	if err := repo.Create(context.Background(), firstTrip); err != nil {
		t.Fatalf("create first trip: %v", err)
	}

	secondTrip := newTestTrip()
	secondTrip.DriverID = firstTrip.DriverID

	err := repo.Create(context.Background(), secondTrip)

	if !errors.Is(err, tripdomain.ErrDriverBusy) {
		t.Fatalf("error = %v, want ErrDriverBusy", err)
	}
}

func TestTripRepository_TransactionRollback(t *testing.T) {
	pool := newIntegrationPool(t)
	repo := NewTripRepository(pool, 5*time.Second)
	txManager := NewTxManager(pool, 5*time.Second)

	trip := newTestTrip()
	cleanupTestTrip(t, pool, trip.ID)

	ctx := context.Background()

	err := txManager.Do(ctx, func(txCtx context.Context) error {
		if err := repo.Create(txCtx, trip); err != nil {
			return err
		}

		return repo.AddStatusHistory(
			txCtx,
			trip.ID,
			"",
			tripdomain.Status("invalid"),
			"test rollback",
		)
	})

	if err == nil {
		t.Fatal("expected transaction to fail")
	}

	_, err = repo.GetByID(ctx, trip.ID)

	if !errors.Is(err, tripdomain.ErrTripNotFound) {
		t.Fatalf("trip should be rolled back, got error: %v", err)
	}
}
