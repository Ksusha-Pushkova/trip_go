package finish

import (
	"context"
	"fmt"
	"time"

	tripdomain "github.com/Ksusha-Pushkova/trip_go/internal/domain/trip"
	"github.com/google/uuid"
)

type tripMetrics interface {
	TripCompleted(context.Context)
}

// UseCase handles the completion of trips.
type UseCase struct {
	repo      tripRepository
	txManager txManager
	metrics   tripMetrics
}

// New creates a new trip completion use case.
func New(repo tripRepository, txManager txManager, metrics ...tripMetrics) *UseCase {
	var m tripMetrics
	if len(metrics) > 0 {
		m = metrics[0]
	}

	return &UseCase{
		repo:      repo,
		txManager: txManager,
		metrics:   m,
	}
}

// Execute completes a trip and records its status history.
func (u *UseCase) Execute(ctx context.Context, id uuid.UUID) (*tripdomain.Trip, error) {
	var result *tripdomain.Trip

	err := u.txManager.Do(ctx, func(ctx context.Context) error {
		trip, err := u.repo.GetByID(ctx, id)
		if err != nil {
			return err
		}

		now := time.Now().UTC()

		if err := trip.Finish(now); err != nil {
			return err
		}

		updated, err := u.repo.Finish(ctx, id, now)
		if err != nil {
			return err
		}

		if !updated {
			currentTrip, err := u.repo.GetByID(ctx, id)
			if err != nil {
				return err
			}

			if currentTrip.Status == tripdomain.StatusCompleted {
				return tripdomain.ErrTripCompleted
			}

			return fmt.Errorf("unexpected trip status: %s", currentTrip.Status)
		}

		if err := u.repo.AddStatusHistory(
			ctx,
			id,
			tripdomain.StatusActive,
			tripdomain.StatusCompleted,
			"completed by driver",
		); err != nil {
			return fmt.Errorf("add status history: %w", err)
		}

		result = trip
		return nil
	})

	if err != nil {
		return nil, err
	}
	if u.metrics != nil {
		u.metrics.TripCompleted(ctx)
	}

	return result, nil
}
