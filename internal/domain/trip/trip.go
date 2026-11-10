package trip

import (
	"time"

	"github.com/google/uuid"

	"fmt"
)

// Status represents the current state of a trip.
type Status string

const (
	// StatusActive indicates that a trip is currently active.
	StatusActive Status = "active"
	// StatusCompleted indicates that a trip has been completed.
	StatusCompleted Status = "completed"
)

// Destination represents the destination of a trip.
type Destination struct {
	Latitude  float64
	Longitude float64
}

// Trip represents a trip and its current state.
type Trip struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	DriverID   uuid.UUID
	Start      Destination
	End        Destination
	Price      int64
	Status     Status
	StartedAt  time.Time
	FinishedAt *time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// Finish completes the trip at the specified time.
func (t *Trip) Finish(at time.Time) error {
	if t.Status == StatusCompleted {
		return ErrTripCompleted
	}

	if t.Status != StatusActive {
		return fmt.Errorf("unexpected trip status: %s", t.Status)
	}

	t.Status = StatusCompleted
	t.FinishedAt = &at
	t.UpdatedAt = at

	return nil
}
