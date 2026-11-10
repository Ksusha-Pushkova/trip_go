//go:generate go run go.uber.org/mock/mockgen -source=contract.go -destination=mock_test.go -package=get
package get

import (
	"context"

	tripdomain "github.com/Ksusha-Pushkova/trip_go/internal/domain/trip"
	"github.com/google/uuid"
)

type tripRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*tripdomain.Trip, error)
}
