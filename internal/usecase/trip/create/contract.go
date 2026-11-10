//go:generate go run go.uber.org/mock/mockgen -source=contract.go -destination=mock_test.go -package=create
package create

import (
	"context"

	tripdomain "github.com/Ksusha-Pushkova/trip_go/internal/domain/trip"
	"github.com/google/uuid"
)

type tripRepository interface {
	Create(ctx context.Context, trip tripdomain.Trip) error
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
