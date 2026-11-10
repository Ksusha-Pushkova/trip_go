//go:generate go run go.uber.org/mock/mockgen -source=contract.go -destination=mock_test.go -package=finish
package finish

import (
	"context"
	"time"

	tripdomain "github.com/Ksusha-Pushkova/trip_go/internal/domain/trip"
	"github.com/google/uuid"
)

type tripRepository interface {
	Finish(ctx context.Context, id uuid.UUID, at time.Time) (bool, error)
	GetByID(ctx context.Context, id uuid.UUID) (*tripdomain.Trip, error)
	AddStatusHistory(
		ctx context.Context,
		tripID uuid.UUID,
		from tripdomain.Status,
		to tripdomain.Status,
		reason string,
	) error
}

type txManager interface {
	Do(ctx context.Context, fn func(context.Context) error) error
}
