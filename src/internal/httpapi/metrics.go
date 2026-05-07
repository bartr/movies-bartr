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
// The route label is collapsed to the first two path segments — so
// `/api/movies/{id}` and `/api/movies` both record as `/api/movies`,
// and `/api/actors/{id}` collapses to `/api/actors`. That keeps label
// cardinality bounded (one series per top-level resource) and matches
// the granularity a business dashboard cares about.
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

// apiRouteLabel returns the two-level route label for a request path
// and a bool indicating whether the request should be measured at all.
// Only paths beginning with `/api/` are measured; the returned label is
// the first two segments (`/api/movies`, `/api/actors`, `/api/genres`).
func apiRouteLabel(path string) (string, bool) {
	if len(path) < len("/api/") || path[:5] != "/api/" {
		return "", false
	}
	// path[5:] is everything after "/api/". Split on the first '/'
	// (or end-of-string) to get the resource segment.
	rest := path[5:]
	for i := 0; i < len(rest); i++ {
		if rest[i] == '/' {
			rest = rest[:i]
			break
		}
	}
	if rest == "" {
		// "/api/" by itself — not a real resource. Skip.
		return "", false
	}
	return "/api/" + rest, true
}
