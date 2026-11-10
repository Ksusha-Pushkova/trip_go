// Package trip contains the domain model and errors for trips.
package trip

import "errors"

var (
	// ErrTripNotFound is returned when a trip cannot be found.
	ErrTripNotFound = errors.New("trip not found")
	// ErrDriverBusy indicates that the driver already has an active trip.
	ErrDriverBusy = errors.New("driver already has an active trip")
	// ErrTripCompleted indicates that the trip has already been completed.
	ErrTripCompleted = errors.New("trip already completed")
)
