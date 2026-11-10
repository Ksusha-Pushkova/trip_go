//go:generate go run go.uber.org/mock/mockgen -source=contract.go -destination=mock_test.go -package=httptransport

// Package httptransport provides HTTP handlers, middleware, and server components.
package httptransport

import (
	"context"

	tripdomain "github.com/Ksusha-Pushkova/trip_go/internal/domain/trip"
	"github.com/google/uuid"
)

type createUseCase interface {
	Execute(ctx context.Context, trip tripdomain.Trip) error
}

type getUseCase interface {
	Execute(ctx context.Context, id uuid.UUID) (*tripdomain.Trip, error)
}

type finishUseCase interface {
	Execute(ctx context.Context, id uuid.UUID) (*tripdomain.Trip, error)
}

type readinessChecker interface {
	Ping(ctx context.Context) error
}
