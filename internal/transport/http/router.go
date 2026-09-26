package httptransport

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/Ksusha-Pushkova/trip_go/api"
)
func NewRouter(handlers api.ServerInterface) http.Handler {
	r := chi.NewRouter()
	return api.HandlerFromMux(handlers, r)
}