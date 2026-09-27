package httptransport

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/Ksusha-Pushkova/trip_go/api"
)

func NewRouter(handlers api.ServerInterface) http.Handler {
	r := chi.NewRouter()

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