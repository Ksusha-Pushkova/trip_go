package httptransport

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/Ksusha-Pushkova/trip_go/api"
)

type Handlers struct {
	logger *slog.Logger
}

func NewHandlers(logger *slog.Logger) *Handlers {
	return &Handlers{logger: logger}
}

var _ api.ServerInterface = (*Handlers)(nil)

func (h *Handlers) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, api.HealthResponse{Status: api.Ok})
}

func (h *Handlers) Ready(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, api.HealthResponse{Status: api.Ok})
}

func (h *Handlers) CreateTrip(w http.ResponseWriter, r *http.Request, params api.CreateTripParams) {
	h.logger.Warn("createTrip is not implemented yet")
	writeProblem(w, r, http.StatusNotImplemented, "not_implemented",
		"Not implemented", "CreateTrip is not implemented yet")
}

func (h *Handlers) GetTrip(w http.ResponseWriter, r *http.Request, tripId api.TripId) {
	h.logger.Warn("getTrip is not implemented yet", "trip_id", tripId)
	writeProblem(w, r, http.StatusNotImplemented, "not_implemented",
		"Not implemented", "GetTrip is not implemented yet")
}

func (h *Handlers) FinishTrip(w http.ResponseWriter, r *http.Request, tripId api.TripId) {
	h.logger.Warn("finishTrip is not implemented yet", "trip_id", tripId)
	writeProblem(w, r, http.StatusNotImplemented, "not_implemented",
		"Not implemented", "FinishTrip is not implemented yet")
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		_ = err
	}
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
