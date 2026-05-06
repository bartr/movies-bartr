// Package httpapi builds the HTTP router and exposes session-1 health
// endpoints. The data API (§6) lands in later sessions.
package httpapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// ReadyFunc reports whether the service is ready to serve data-bearing
// endpoints. The walking skeleton has no dataset, so the main package flips a
// flag synchronously after a no-op load.
type ReadyFunc func() bool

// NewRouter constructs the chi router with the session-1 endpoints wired up.
func NewRouter(version string, ready ReadyFunc) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Get("/version", versionHandler(version))
	r.Get("/healthz", healthzHandler())
	r.Get("/readyz", readyzHandler(ready))
	return r
}
