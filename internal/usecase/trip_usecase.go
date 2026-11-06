package usecase

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/Ksusha-Pushkova/trip_go/internal/domain"
	"github.com/Ksusha-Pushkova/trip_go/internal/repository/postgres"
)

type TripUsecase interface {
	Create(ctx context.Context, trip domain.Trip) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Trip, error)
	Finish(ctx context.Context, id uuid.UUID) (*domain.Trip, error)
}

type tripUsecase struct {
	tripRepo  postgres.TripRepository
	txManager postgres.TxManager
}

func NewTripUsecase(
	repo postgres.TripRepository,
	txManager postgres.TxManager,
) TripUsecase {
	return &tripUsecase{
		tripRepo:  repo,
		txManager: txManager,
	}
}

func (u *tripUsecase) Create(
	ctx context.Context,
	trip domain.Trip,
) error {
	return u.txManager.Do(ctx, func(ctx context.Context) error {
		if err := u.tripRepo.Create(ctx, trip); err != nil {
			return fmt.Errorf("create trip: %w", err)
		}

		if err := u.tripRepo.AddStatusHistory(
			ctx,
			trip.ID,
			"",
			domain.TripStatusActive,
			"created",
		); err != nil {
			return fmt.Errorf(
				"add status history: %w",
				err,
			)
		}

		return nil
	})
}

func (u *tripUsecase) GetByID(
	ctx context.Context,
	id uuid.UUID,
) (*domain.Trip, error) {
	return u.tripRepo.GetByID(ctx, id)
}

func (u *tripUsecase) Finish(
	ctx context.Context,
	id uuid.UUID,
) (*domain.Trip, error) {
	var result *domain.Trip

	err := u.txManager.Do(ctx, func(ctx context.Context) error {
		updated, err := u.tripRepo.Finish(ctx, id)
		if err != nil {
			return err
		}

		if !updated {
			trip, err := u.tripRepo.GetByID(ctx, id)
			if err != nil {
				return err
			}

			if trip.Status == domain.TripStatusCompleted {
				return domain.ErrTripCompleted
			}

			return fmt.Errorf(
				"unexpected trip status: %s",
				trip.Status,
			)
		}

		if err := u.tripRepo.AddStatusHistory(
			ctx,
			id,
			domain.TripStatusActive,
			domain.TripStatusCompleted,
			"completed by driver",
		); err != nil {
			return fmt.Errorf(
				"add status history: %w",
				err,
			)
		}

		result, err = u.tripRepo.GetByID(ctx, id)
		if err != nil {
			return fmt.Errorf(
				"get finished trip: %w",
				err,
			)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}
