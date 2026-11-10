package httptransport

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/Ksusha-Pushkova/trip_go/api"
	"github.com/Ksusha-Pushkova/trip_go/internal/telemetry"
)

// NewRouter creates an HTTP router with middleware and API handlers.
func NewRouter(
	handlers api.ServerInterface,
	logger *slog.Logger,
	metrics ...*telemetry.Metrics,
) http.Handler {
	var m *telemetry.Metrics
	if len(metrics) > 0 {
		m = metrics[0]
	}

	r := chi.NewRouter()

	r.Use(RecoverMiddleware(logger))
	r.Use(TracingMiddleware())
	r.Use(LoggingMiddleware(logger))
	r.Use(MetricsMiddleware(m))

	options := api.ChiServerOptions{
		BaseRouter:       r,
		ErrorHandlerFunc: newErrorHandler(),
	}

	return api.HandlerWithOptions(handlers, options)
}

func newErrorHandler() func(w http.ResponseWriter, r *http.Request, err error) {
	return func(w http.ResponseWriter, r *http.Request, err error) {
		var invalidParamErr *api.InvalidParamFormatError
		if errors.As(err, &invalidParamErr) {
			writeProblem(w, r, http.StatusBadRequest, "invalid_request",
				"Invalid request", "Invalid format for parameter "+invalidParamErr.ParamName)
			return
		}

		var requiredParamErr *api.RequiredParamError
		if errors.As(err, &requiredParamErr) {
			writeProblem(w, r, http.StatusBadRequest, "invalid_request",
				"Invalid request", "Missing required parameter "+requiredParamErr.ParamName)
			return
		}
		writeProblem(w, r, http.StatusBadRequest, "invalid_request",
			"Invalid request", err.Error())
	}
}
