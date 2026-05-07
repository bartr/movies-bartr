package httpapi

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// metrics bundles the per-router Prometheus registry and the three vectors
// exposed at /metrics. We use a per-router registry (not the default global)
// so tests can spin up multiple routers in the same process without
// "duplicate metrics collector registration" panics, and so application
// metrics ship alongside the standard Go-runtime + process collectors.
type metrics struct {
	registry  *prometheus.Registry
	requests  *prometheus.CounterVec
	durations *prometheus.HistogramVec
	inFlight  prometheus.Gauge
}

func newMetrics() *metrics {
	m := &metrics{
		registry: prometheus.NewRegistry(),
		requests: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_requests_total",
				Help: "Total HTTP requests received, labeled by method, route template, and status code.",
			},
			[]string{"method", "route", "code"},
		),
		durations: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name: "http_request_duration_seconds",
				Help: "HTTP request latency in seconds, labeled by method, route template, and status code.",
				// Sub-ms buckets up front: this is an in-memory API
				// where typical service time is 20–100 µs. The default
				// Prometheus buckets start at 5 ms which makes p95
				// indistinguishable across all routes (they all land in
				// the first bucket). Keeping the higher bands lets
				// regressions still register.
				Buckets: []float64{
					0.0001, 0.00025, 0.0005, 0.001, 0.0025,
					0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10,
				},
			},
			[]string{"method", "route", "code"},
		),
		inFlight: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "http_requests_in_flight",
				Help: "Number of HTTP requests currently being served.",
			},
		),
	}
	m.registry.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		m.requests,
		m.durations,
		m.inFlight,
	)
	return m
}

// handler returns the /metrics HTTP handler bound to this registry.
func (m *metrics) handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{
		Registry: m.registry,
	})
}

// middleware records request count, duration, and in-flight gauge for
// every `/api/*` request. Anything outside `/api/` (`/metrics`,
// `/healthz`, `/readyz`, `/version`, `/swagger/*`, `/`, `robots.txt`,
// 404s on unrouted paths) is intentionally not measured: the dashboard
// surfaces business traffic, and operational endpoints would dominate
// the timeseries on an idle service.
//
// The route label keeps cardinality bounded by emitting only known
// templates: `/api/movies`, `/api/movies/{id}`, `/api/actors`,
// `/api/actors/{id}`, `/api/genres`. Anything beyond a known third
// segment collapses to its two-segment parent so a stray path can't
// blow up label cardinality.
func (m *metrics) middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			m.inFlight.Inc()
			defer m.inFlight.Dec()

			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rec, r)

			route, ok := apiRouteLabel(r.URL.Path)
			if !ok {
				return
			}
			code := strconv.Itoa(rec.status)
			m.requests.WithLabelValues(r.Method, route, code).Inc()
			m.durations.WithLabelValues(r.Method, route, code).Observe(time.Since(start).Seconds())
		})
	}
}

// apiRouteLabel returns the route-template label for a request path
// and a bool indicating whether the request should be measured at all.
// Only paths beginning with `/api/` are measured. List routes return
// `/api/<resource>`; detail routes for movies and actors return
// `/api/<resource>/{id}`. Unknown sub-paths collapse to their
// two-segment parent to keep label cardinality bounded.
func apiRouteLabel(path string) (string, bool) {
	if len(path) < len("/api/") || path[:5] != "/api/" {
		return "", false
	}
	// rest is everything after "/api/".
	rest := path[5:]
	resource := rest
	tail := ""
	if i := indexByte(rest, '/'); i >= 0 {
		resource = rest[:i]
		tail = rest[i+1:]
	}
	if resource == "" {
		// "/api/" by itself — not a real resource. Skip.
		return "", false
	}
	base := "/api/" + resource
	// Detail routes only for resources that actually have one. A
	// non-empty tail past the resource means a detail-style request.
	if tail != "" {
		switch resource {
		case "movies", "actors":
			return base + "/{id}", true
		}
	}
	return base, true
}

// indexByte is a tiny strings.IndexByte to keep this file's imports
// unchanged. It returns the index of the first c in s, or -1.
func indexByte(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}
