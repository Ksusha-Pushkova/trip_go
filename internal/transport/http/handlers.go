package httptransport

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Ksusha-Pushkova/trip_go/api"
	"github.com/Ksusha-Pushkova/trip_go/internal/domain"
	"github.com/Ksusha-Pushkova/trip_go/internal/usecase"
)

type Handlers struct {
	logger               *slog.Logger
	tripUsecase          usecase.TripUsecase
	pool                 *pgxpool.Pool
	databaseQueryTimeout time.Duration
}

func NewHandlers(
	logger *slog.Logger,
	tripUsecase usecase.TripUsecase,
	pool *pgxpool.Pool,
	databaseQueryTimeout time.Duration,
) *Handlers {
	return &Handlers{
		logger:               logger,
		tripUsecase:          tripUsecase,
		pool:                 pool,
		databaseQueryTimeout: databaseQueryTimeout,
	}
}

var _ api.ServerInterface = (*Handlers)(nil)

func (h *Handlers) Health(
	w http.ResponseWriter,
	r *http.Request,
) {
	writeJSON(
		w,
		http.StatusOK,
		api.HealthResponse{Status: api.Ok},
	)
}

func (h *Handlers) Ready(
	w http.ResponseWriter,
	r *http.Request,
) {
	ctx, cancel := context.WithTimeout(
		r.Context(),
		h.databaseQueryTimeout,
	)
	defer cancel()

	if err := h.pool.Ping(ctx); err != nil {
		h.logger.Warn(
			"readiness check failed",
			"error",
			err,
		)

		writeJSON(
			w,
			http.StatusServiceUnavailable,
			api.HealthResponse{Status: api.Unavailable},
		)

		return
	}

	writeJSON(
		w,
		http.StatusOK,
		api.HealthResponse{Status: api.Ok},
	)
}

func (h *Handlers) CreateTrip(
	w http.ResponseWriter,
	r *http.Request,
	_ api.CreateTripParams,
) {
	var body api.TripData

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		h.logger.Warn(
			"invalid createTrip body",
			"error",
			err,
		)

		writeProblem(
			w,
			r,
			http.StatusBadRequest,
			"invalid_request",
			"Invalid request",
			"Request body is not valid JSON",
		)

		return
	}

	if err := validateTripData(body); err != nil {
		h.logger.Warn(
			"createTrip validation failed",
			"error",
			err,
		)

		writeProblem(
			w,
			r,
			http.StatusBadRequest,
			"invalid_request",
			"Invalid request",
			err.Error(),
		)

		return
	}

	now := time.Now().UTC()

	trip := domain.Trip{
		ID:       uuid.New(),
		UserID:   uuid.UUID(body.UserId),
		DriverID: uuid.UUID(body.DriverId),

		Start: domain.Destination{
			Latitude:  body.StartPoint.Latitude,
			Longitude: body.StartPoint.Longitude,
		},

		End: domain.Destination{
			Latitude:  body.EndPoint.Latitude,
			Longitude: body.EndPoint.Longitude,
		},

		Price:     body.Price,
		Status:    domain.TripStatusActive,
		StartedAt: now,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := h.tripUsecase.Create(r.Context(), trip); err != nil {
		h.writeError(w, r, err)
		return
	}

	w.Header().Set(
		"Location",
		"/api/v1/trips/"+trip.ID.String(),
	)

	writeJSON(
		w,
		http.StatusCreated,
		toAPITrip(&trip),
	)
}

func (h *Handlers) GetTrip(
	w http.ResponseWriter,
	r *http.Request,
	tripID api.TripId,
) {
	trip, err := h.tripUsecase.GetByID(
		r.Context(),
		uuid.UUID(tripID),
	)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	writeJSON(
		w,
		http.StatusOK,
		toAPITrip(trip),
	)
}

func (h *Handlers) FinishTrip(
	w http.ResponseWriter,
	r *http.Request,
	tripID api.TripId,
) {
	trip, err := h.tripUsecase.Finish(
		r.Context(),
		uuid.UUID(tripID),
	)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	writeJSON(
		w,
		http.StatusOK,
		toAPITrip(trip),
	)
}

func (h *Handlers) writeError(
	w http.ResponseWriter,
	r *http.Request,
	err error,
) {
	switch {
	case errors.Is(err, domain.ErrTripNotFound):
		writeProblem(
			w,
			r,
			http.StatusNotFound,
			"trip_not_found",
			"Trip not found",
			"Trip was not found",
		)

	case errors.Is(err, domain.ErrDriverBusy):
		writeProblem(
			w,
			r,
			http.StatusConflict,
			"driver_busy",
			"Driver busy",
			"Driver already has an active trip",
		)

	case errors.Is(err, domain.ErrTripCompleted):
		writeProblem(
			w,
			r,
			http.StatusConflict,
			"trip_completed",
			"Trip completed",
			"Operation is not allowed for a completed trip",
		)

	default:
		h.logger.Error(
			"internal error",
			"error",
			err,
		)

		writeProblem(
			w,
			r,
			http.StatusInternalServerError,
			"internal_error",
			"Internal Server Error",
			"Internal server error",
		)
	}
}

func validateTripData(d api.TripData) error {
	if d.UserId == (api.TripId{}) {
		return errors.New("user_id is required")
	}

	if d.DriverId == (api.TripId{}) {
		return errors.New("driver_id is required")
	}

	if err := validateCoordinates(d.StartPoint); err != nil {
		return errors.New("start_point: " + err.Error())
	}

	if err := validateCoordinates(d.EndPoint); err != nil {
		return errors.New("end_point: " + err.Error())
	}

	if d.Price < 0 {
		return errors.New("price must be >= 0")
	}

	return nil
}

func validateCoordinates(c api.Coordinates) error {
	if c.Latitude < -90 || c.Latitude > 90 {
		return errors.New(
			"latitude must be between -90 and 90",
		)
	}

	if c.Longitude < -180 || c.Longitude > 180 {
		return errors.New(
			"longitude must be between -180 and 180",
		)
	}

	return nil
}

func toAPITrip(t *domain.Trip) api.Trip {
	return api.Trip{
		Id:       t.ID,
		UserId:   t.UserID,
		DriverId: t.DriverID,

		StartPoint: api.Coordinates{
			Latitude:  t.Start.Latitude,
			Longitude: t.Start.Longitude,
		},

		EndPoint: api.Coordinates{
			Latitude:  t.End.Latitude,
			Longitude: t.End.Longitude,
		},

		Price:          t.Price,
		Status:         api.TripStatus(t.Status),
		StartedAt:      t.StartedAt,
		FinishedAt:     t.FinishedAt,
		LastPositionAt: nil,
	}
}

func writeJSON(
	w http.ResponseWriter,
	status int,
	v any,
) {
	w.Header().Set(
		"Content-Type",
		"application/json",
	)

	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(v)
}

func writeProblem(
	w http.ResponseWriter,
	r *http.Request,
	status int,
	code,
	title,
	detail string,
) {
	instance := r.URL.Path

	problem := api.Problem{
		Type: "https://tripgo.example/problems/" +
			strings.ReplaceAll(code, "_", "-"),
		Title:    title,
		Status:   int32(status),
		Code:     code,
		Detail:   &detail,
		Instance: &instance,
	}

	w.Header().Set(
		"Content-Type",
		"application/problem+json",
	)

	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(problem)
}
