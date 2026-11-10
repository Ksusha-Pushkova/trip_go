package get

import (
	"context"

	tripdomain "github.com/Ksusha-Pushkova/trip_go/internal/domain/trip"
	"github.com/google/uuid"
)

// UseCase handles retrieving trips.
type UseCase struct {
	repo tripRepository
}

// New creates a new trip retrieval use case.
func New(repo tripRepository) *UseCase {
	return &UseCase{repo: repo}
}

// Execute retrieves a trip by its ID.
func (u *UseCase) Execute(ctx context.Context, id uuid.UUID) (*tripdomain.Trip, error) {
	return u.repo.GetByID(ctx, id)
}
