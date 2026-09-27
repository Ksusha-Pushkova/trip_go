package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/Ksusha-Pushkova/trip_go/internal/domain"
	"github.com/Ksusha-Pushkova/trip_go/internal/repository/postgres"
)

type TripUsecase interface {
	Create(ctx context.Context, input CreateTripInput) (*domain.Trip, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Trip, error)
	Finish(ctx context.Context, id uuid.UUID) (*domain.Trip, error)
}

type CreateTripInput struct {
	UserID         uuid.UUID
	DriverID       uuid.UUID
	StartLatitude  float64
	StartLongitude float64
	EndLatitude    float64
	EndLongitude   float64
	Price          int64
}

type tripUsecase struct {
	tripRepo  postgres.TripRepository
	txManager postgres.TxManager
}

func NewTripUsecase(repo postgres.TripRepository, txManager postgres.TxManager) TripUsecase {
	return &tripUsecase{tripRepo: repo, txManager: txManager}
}

func (u *tripUsecase) Create(ctx context.Context, input CreateTripInput) (*domain.Trip, error) {
	now := time.Now().UTC()
	trip := domain.Trip{
		ID:             uuid.New(),
		UserID:         input.UserID,
		DriverID:       input.DriverID,
		StartLatitude:  input.StartLatitude,
		StartLongitude: input.StartLongitude,
		EndLatitude:    input.EndLatitude,
		EndLongitude:   input.EndLongitude,
		Price:          input.Price,
		Status:         domain.TripStatusActive,
		StartedAt:      now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	err := u.txManager.Do(ctx, func(ctx context.Context) error {
		if err := u.tripRepo.Create(ctx, trip); err != nil {
			return fmt.Errorf("create trip: %w", err)
		}
		if err := u.tripRepo.AddStatusHistory(ctx, trip.ID, "", domain.TripStatusActive, "created"); err != nil {
			return fmt.Errorf("add status history: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &trip, nil
}

func (u *tripUsecase) GetByID(ctx context.Context, id uuid.UUID) (*domain.Trip, error) {
	return u.tripRepo.GetByID(ctx, id)
}

func (u *tripUsecase) Finish(ctx context.Context, id uuid.UUID) (*domain.Trip, error) {
	var result *domain.Trip

	err := u.txManager.Do(ctx, func(ctx context.Context) error {
		trip, err := u.tripRepo.Finish(ctx, id)
		if err != nil {
			return err
		}
		if err := u.tripRepo.AddStatusHistory(ctx, id, domain.TripStatusActive, domain.TripStatusCompleted, "completed by driver"); err != nil {
			return fmt.Errorf("add status history: %w", err)
		}
		result = trip
		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}