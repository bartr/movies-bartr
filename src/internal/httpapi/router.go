// Package httpapi builds the HTTP router. Session 1 shipped /version,
// /healthz, /readyz; session 3 added the /api/* read endpoints with full
// query/path validation and RFC 7807 error bodies; session 5 added the
// OpenAPI document, Swagger UI, root redirect, robots.txt, and a JSON
// request-log middleware; session 6 adds Prometheus metrics on /metrics
// with a per-router registry and a chi-aware instrumentation middleware.
package httpapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// ReadyFunc reports whether the service is ready to serve data-bearing
// endpoints. Wired by main from the dataset-load goroutine.
type ReadyFunc func() bool

// NewRouter constructs the chi router. store may return nil; /api/* will
// respond 503 problem+json until it is non-nil.
func NewRouter(version string, ready ReadyFunc, store StoreFunc) http.Handler {
	r := chi.NewRouter()
	m := newMetrics()
	r.Use(middleware.Recoverer)
	r.Use(m.middleware())
	r.Use(requestLogger())

	r.Get("/", rootRedirectHandler())
	r.Get("/robots.txt", robotsHandler())
	r.Get("/swagger", swaggerUIHandler())
	r.Get("/swagger/", swaggerUIHandler())
	r.Get("/swagger/v1/swagger.json", swaggerJSONHandler())
	r.Method(http.MethodGet, "/metrics", m.handler())

	r.Get("/version", versionHandler(version))
	r.Get("/healthz", healthzHandler())
	r.Get("/readyz", readyzHandler(ready))

	r.Route("/api", func(api chi.Router) {
		api.Get("/movies", listMoviesHandler(store))
		api.Get("/movies/{id}", getMovieHandler(store))
		api.Get("/actors", listActorsHandler(store))
		api.Get("/actors/{id}", getActorHandler(store))
		api.Get("/genres", genresHandler(store))
	})

	return r
}
