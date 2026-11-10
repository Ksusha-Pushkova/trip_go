// Package telemetry provides application metrics and telemetry instrumentation.
package telemetry

import (
	"context"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// Metrics stores application metrics for HTTP requests and trips.
type Metrics struct {
	httpDuration   metric.Int64Histogram
	activeRequests metric.Int64UpDownCounter
	tripsCreated   metric.Int64Counter
	tripsCompleted metric.Int64Counter

	activeTrips atomic.Int64
}

// NewMetrics creates and registers application metrics.
func NewMetrics() (*Metrics, error) {
	meter := otel.Meter("trip-service")

	httpDuration, err := meter.Int64Histogram(
		"http.server.request.duration",
		metric.WithUnit("ms"),
		metric.WithDescription("Duration of HTTP server requests"),
	)
	if err != nil {
		return nil, err
	}

	activeRequests, err := meter.Int64UpDownCounter(
		"http.server.active_requests",
		metric.WithDescription("Number of active HTTP requests"),
	)
	if err != nil {
		return nil, err
	}

	tripsCreated, err := meter.Int64Counter(
		"trips.created",
		metric.WithDescription("Number of created trips"),
	)
	if err != nil {
		return nil, err
	}

	tripsCompleted, err := meter.Int64Counter(
		"trips.completed",
		metric.WithDescription("Number of completed trips"),
	)
	if err != nil {
		return nil, err
	}

	metrics := &Metrics{
		httpDuration:   httpDuration,
		activeRequests: activeRequests,
		tripsCreated:   tripsCreated,
		tripsCompleted: tripsCompleted,
	}

	activeTripsGauge, err := meter.Int64ObservableGauge(
		"trips.active",
		metric.WithDescription("Number of active trips"),
	)
	if err != nil {
		return nil, err
	}

	_, err = meter.RegisterCallback(
		func(_ context.Context, observer metric.Observer) error {
			observer.ObserveInt64(activeTripsGauge, metrics.activeTrips.Load())
			return nil
		},
		activeTripsGauge,
	)
	if err != nil {
		return nil, err
	}

	return metrics, nil
}

// RequestStarted records the start of an HTTP request.
func (m *Metrics) RequestStarted(ctx context.Context) {
	if m == nil {
		return
	}

	m.activeRequests.Add(ctx, 1)
}

// RequestFinished records the completion and duration of an HTTP request.
func (m *Metrics) RequestFinished(
	ctx context.Context,
	method string,
	route string,
	status int,
	duration time.Duration,
) {
	if m == nil {
		return
	}

	attributes := metric.WithAttributes(
		attribute.String("http.request.method", method),
		attribute.String("http.route", route),
		attribute.Int("http.response.status_code", status),
	)

	m.activeRequests.Add(ctx, -1)
	m.httpDuration.Record(ctx, duration.Milliseconds(), attributes)
}

// TripCreated records the creation of a trip.
func (m *Metrics) TripCreated(ctx context.Context) {
	if m == nil {
		return
	}

	m.tripsCreated.Add(ctx, 1)
	m.activeTrips.Add(1)
}

// TripCompleted records the completion of a trip.
func (m *Metrics) TripCompleted(ctx context.Context) {
	if m == nil {
		return
	}

	m.tripsCompleted.Add(ctx, 1)
	m.activeTrips.Add(-1)
}

// SetActiveTrips updates the current number of active trips.
func (m *Metrics) SetActiveTrips(count int64) {
	if m == nil {
		return
	}

	m.activeTrips.Store(count)
}
