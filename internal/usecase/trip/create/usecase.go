package create

import (
	"context"
	"fmt"

	tripdomain "github.com/Ksusha-Pushkova/trip_go/internal/domain/trip"
)

type tripMetrics interface {
	TripCreated(context.Context)
}

// UseCase handles the creation of trips.
type UseCase struct {
	repo      tripRepository
	txManager txManager
	metrics   tripMetrics
}

// New creates a new trip creation use case.
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

// Execute creates a trip and records its initial status history.
func (u *UseCase) Execute(ctx context.Context, trip tripdomain.Trip) error {
	err := u.txManager.Do(ctx, func(ctx context.Context) error {
		if err := u.repo.Create(ctx, trip); err != nil {
			return fmt.Errorf("create trip: %w", err)
		}

		if err := u.repo.AddStatusHistory(
			ctx,
			trip.ID,
			"",
			tripdomain.StatusActive,
			"created",
		); err != nil {
			return fmt.Errorf("add status history: %w", err)
		}

		return nil
	})
	if err != nil {
		return err
	}

	if u.metrics != nil {
		u.metrics.TripCreated(ctx)
	}

	return nil
}
