package finish

import (
	"context"
	"errors"
	"testing"

	tripdomain "github.com/Ksusha-Pushkova/trip_go/internal/domain/trip"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

func TestUseCase_Execute_Success(t *testing.T) {
	ctrl := gomock.NewController(t)

	repo := NewMocktripRepository(ctrl)
	tx := NewMocktxManager(ctrl)

	uc := New(repo, tx)

	ctx := context.Background()
	tripID := uuid.New()

	activeTrip := &tripdomain.Trip{
		ID:     tripID,
		Status: tripdomain.StatusActive,
	}

	tx.EXPECT().
		Do(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		})

	repo.EXPECT().
		GetByID(ctx, tripID).
		Return(activeTrip, nil)

	repo.EXPECT().
		Finish(ctx, tripID, gomock.Any()).
		Return(true, nil)

	repo.EXPECT().
		AddStatusHistory(
			ctx,
			tripID,
			tripdomain.StatusActive,
			tripdomain.StatusCompleted,
			"completed by driver",
		).
		Return(nil)

	result, err := uc.Execute(ctx, tripID)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result == nil {
		t.Fatal("expected trip, got nil")
	}

	if result.Status != tripdomain.StatusCompleted {
		t.Errorf("expected status %q, got %q",
			tripdomain.StatusCompleted, result.Status)
	}

	if result.FinishedAt == nil {
		t.Error("expected FinishedAt to be set")
	}
}

func TestUseCase_Execute_AlreadyCompleted(t *testing.T) {
	ctrl := gomock.NewController(t)

	repo := NewMocktripRepository(ctrl)
	tx := NewMocktxManager(ctrl)

	uc := New(repo, tx)

	ctx := context.Background()
	tripID := uuid.New()

	completedTrip := &tripdomain.Trip{
		ID:     tripID,
		Status: tripdomain.StatusCompleted,
	}

	tx.EXPECT().
		Do(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		})

	repo.EXPECT().
		GetByID(ctx, tripID).
		Return(completedTrip, nil)

	result, err := uc.Execute(ctx, tripID)

	if result != nil {
		t.Error("expected nil trip")
	}

	if !errors.Is(err, tripdomain.ErrTripCompleted) {
		t.Errorf("expected ErrTripCompleted, got %v", err)
	}
}

func TestUseCase_Execute_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)

	repo := NewMocktripRepository(ctrl)
	tx := NewMocktxManager(ctrl)

	uc := New(repo, tx)

	ctx := context.Background()
	tripID := uuid.New()

	tx.EXPECT().
		Do(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		})

	repo.EXPECT().
		GetByID(ctx, tripID).
		Return(nil, tripdomain.ErrTripNotFound)

	result, err := uc.Execute(ctx, tripID)

	if result != nil {
		t.Error("expected nil trip")
	}

	if !errors.Is(err, tripdomain.ErrTripNotFound) {
		t.Errorf("expected ErrTripNotFound, got %v", err)
	}
}

func TestUseCase_Execute_FinishError(t *testing.T) {
	ctrl := gomock.NewController(t)

	repo := NewMocktripRepository(ctrl)
	tx := NewMocktxManager(ctrl)

	uc := New(repo, tx)

	ctx := context.Background()
	tripID := uuid.New()
	dbErr := errors.New("database error")

	activeTrip := &tripdomain.Trip{
		ID:     tripID,
		Status: tripdomain.StatusActive,
	}

	tx.EXPECT().
		Do(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		})

	repo.EXPECT().
		GetByID(ctx, tripID).
		Return(activeTrip, nil)

	repo.EXPECT().
		Finish(ctx, tripID, gomock.Any()).
		Return(false, dbErr)

	result, err := uc.Execute(ctx, tripID)

	if result != nil {
		t.Error("expected nil trip")
	}

	if !errors.Is(err, dbErr) {
		t.Errorf("expected database error, got %v", err)
	}
}

func TestUseCase_Execute_RaceCondition(t *testing.T) {
	ctrl := gomock.NewController(t)

	repo := NewMocktripRepository(ctrl)
	tx := NewMocktxManager(ctrl)

	uc := New(repo, tx)

	ctx := context.Background()
	tripID := uuid.New()

	activeTrip := &tripdomain.Trip{
		ID:     tripID,
		Status: tripdomain.StatusActive,
	}

	completedTrip := &tripdomain.Trip{
		ID:     tripID,
		Status: tripdomain.StatusCompleted,
	}

	tx.EXPECT().
		Do(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		})

	repo.EXPECT().
		GetByID(ctx, tripID).
		Return(activeTrip, nil)

	repo.EXPECT().
		Finish(ctx, tripID, gomock.Any()).
		Return(false, nil)

	repo.EXPECT().
		GetByID(ctx, tripID).
		Return(completedTrip, nil)

	result, err := uc.Execute(ctx, tripID)

	if result != nil {
		t.Error("expected nil trip")
	}

	if !errors.Is(err, tripdomain.ErrTripCompleted) {
		t.Errorf("expected ErrTripCompleted, got %v", err)
	}
}
