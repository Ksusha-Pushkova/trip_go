package create

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

	newTrip := tripdomain.Trip{
		ID:     uuid.New(),
		Status: tripdomain.StatusActive,
	}

	tx.EXPECT().
		Do(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		})

	repo.EXPECT().
		Create(ctx, newTrip).
		Return(nil)

	repo.EXPECT().
		AddStatusHistory(
			ctx,
			newTrip.ID,
			tripdomain.Status(""),
			tripdomain.StatusActive,
			"created",
		).
		Return(nil)

	err := uc.Execute(ctx, newTrip)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestUseCase_Execute_CreateError(t *testing.T) {
	ctrl := gomock.NewController(t)

	repo := NewMocktripRepository(ctrl)
	tx := NewMocktxManager(ctrl)

	uc := New(repo, tx)

	ctx := context.Background()
	newTrip := tripdomain.Trip{
		ID:     uuid.New(),
		Status: tripdomain.StatusActive,
	}

	createErr := errors.New("database error")

	tx.EXPECT().
		Do(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		})

	repo.EXPECT().
		Create(ctx, newTrip).
		Return(createErr)

	err := uc.Execute(ctx, newTrip)

	if !errors.Is(err, createErr) {
		t.Errorf("expected create error, got %v", err)
	}
}

func TestUseCase_Execute_AddStatusHistoryError(t *testing.T) {
	ctrl := gomock.NewController(t)

	repo := NewMocktripRepository(ctrl)
	tx := NewMocktxManager(ctrl)

	uc := New(repo, tx)

	ctx := context.Background()
	newTrip := tripdomain.Trip{
		ID:     uuid.New(),
		Status: tripdomain.StatusActive,
	}

	historyErr := errors.New("history database error")

	tx.EXPECT().
		Do(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		})

	repo.EXPECT().
		Create(ctx, newTrip).
		Return(nil)

	repo.EXPECT().
		AddStatusHistory(
			ctx,
			newTrip.ID,
			tripdomain.Status(""),
			tripdomain.StatusActive,
			"created",
		).
		Return(historyErr)

	err := uc.Execute(ctx, newTrip)

	if !errors.Is(err, historyErr) {
		t.Errorf("expected history error, got %v", err)
	}
}
