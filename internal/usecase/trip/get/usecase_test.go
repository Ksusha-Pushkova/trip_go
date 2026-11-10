package get

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

	id := uuid.New()
	expectedTrip := &tripdomain.Trip{
		ID:     id,
		Status: tripdomain.StatusActive,
	}

	repo.EXPECT().
		GetByID(gomock.Any(), id).
		Return(expectedTrip, nil)

	uc := New(repo)

	result, err := uc.Execute(context.Background(), id)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != expectedTrip {
		t.Fatalf("expected trip %v, got %v", expectedTrip, result)
	}
}

func TestUseCase_Execute_RepositoryError(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := NewMocktripRepository(ctrl)

	id := uuid.New()
	expectedErr := errors.New("database error")

	repo.EXPECT().
		GetByID(gomock.Any(), id).
		Return(nil, expectedErr)

	uc := New(repo)

	result, err := uc.Execute(context.Background(), id)

	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}

	if result != nil {
		t.Fatalf("expected nil trip, got %v", result)
	}
}
