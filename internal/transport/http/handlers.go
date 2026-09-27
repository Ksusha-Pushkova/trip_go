package httptransport

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/Ksusha-Pushkova/trip_go/api"
	"github.com/Ksusha-Pushkova/trip_go/internal/domain"
	"github.com/Ksusha-Pushkova/trip_go/internal/usecase"
)

type Handlers struct {
	logger      *slog.Logger
	tripUsecase usecase.TripUsecase
}

func NewHandlers(logger *slog.Logger, tripUsecase usecase.TripUsecase) *Handlers {
	return &Handlers{
		logger:      logger,
		tripUsecase: tripUsecase,
	}
}

var _ api.ServerInterface = (*Handlers)(nil)

func (h *Handlers) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, api.HealthResponse{Status: api.Ok})
}

func (h *Handlers) Ready(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, api.HealthResponse{Status: api.Ok})
}

func (h *Handlers) CreateTrip(w http.ResponseWriter, r *http.Request, _ api.CreateTripParams) {
	var body api.TripData
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		h.logger.Warn("invalid createTrip body", "error", err)
		writeProblem(w, r, http.StatusBadRequest, "invalid_request",
			"Invalid request", "Request body is not valid JSON")
		return
	}

	if err := validateTripData(body); err != nil {
		h.logger.Warn("createTrip validation failed", "error", err)
		writeProblem(w, r, http.StatusBadRequest, "invalid_request",
			"Invalid request", err.Error())
		return
	}

	trip, err := h.tripUsecase.Create(r.Context(), usecase.CreateTripInput{
		UserID:         uuid.UUID(body.UserId),
		DriverID:       uuid.UUID(body.DriverId),
		StartLatitude:  body.StartPoint.Latitude,
		StartLongitude: body.StartPoint.Longitude,
		EndLatitude:    body.EndPoint.Latitude,
		EndLongitude:   body.EndPoint.Longitude,
		Price:          body.Price,
	})
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	w.Header().Set("Location", "/api/v1/trips/"+trip.ID.String())
	writeJSON(w, http.StatusCreated, toAPITrip(trip))
}

func (h *Handlers) GetTrip(w http.ResponseWriter, r *http.Request, tripId api.TripId) {
	trip, err := h.tripUsecase.GetByID(r.Context(), uuid.UUID(tripId))
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, toAPITrip(trip))
}

func (h *Handlers) FinishTrip(w http.ResponseWriter, r *http.Request, tripId api.TripId) {
	trip, err := h.tripUsecase.Finish(r.Context(), uuid.UUID(tripId))
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, toAPITrip(trip))
}
func (h *Handlers) writeError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, domain.ErrTripNotFound):
		writeProblem(w, r, http.StatusNotFound, "trip_not_found",
			"Trip not found", "Trip was not found")
	case errors.Is(err, domain.ErrDriverBusy):
		writeProblem(w, r, http.StatusConflict, "driver_busy",
			"Driver busy", "Driver already has an active trip")
	case errors.Is(err, domain.ErrTripCompleted):
		writeProblem(w, r, http.StatusConflict, "trip_completed",
			"Trip completed", "Operation is not allowed for a completed trip")
	default:
		h.logger.Error("internal error", "error", err)
		writeProblem(w, r, http.StatusInternalServerError, "internal_error",
			"Internal Server Error", "Internal server error")
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
		return errors.New("latitude must be between -90 and 90")
	}
	if c.Longitude < -180 || c.Longitude > 180 {
		return errors.New("longitude must be between -180 and 180")
	}
	return nil
}

func toAPITrip(t *domain.Trip) api.Trip {
	return api.Trip{
		Id:             t.ID,
		UserId:         t.UserID,
		DriverId:       t.DriverID,
		StartPoint:     api.Coordinates{Latitude: t.StartLatitude, Longitude: t.StartLongitude},
		EndPoint:       api.Coordinates{Latitude: t.EndLatitude, Longitude: t.EndLongitude},
		Price:          t.Price,
		Status:         api.TripStatus(t.Status),
		StartedAt:      t.StartedAt,
		FinishedAt:     t.FinishedAt,
		LastPositionAt: nil,
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeProblem(w http.ResponseWriter, r *http.Request, status int, code, title, detail string) {
	instance := r.URL.Path

	p := api.Problem{
		Type:     "https://tripgo.example/problems/" + strings.ReplaceAll(code, "_", "-"),
		Title:    title,
		Status:   int32(status),
		Code:     code,
		Detail:   &detail,
		Instance: &instance,
	}

	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(p)
}
