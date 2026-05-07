package httpapi

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
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
				Name:    "http_request_duration_seconds",
				Help:    "HTTP request latency in seconds, labeled by method, route template, and status code.",
				Buckets: prometheus.DefBuckets,
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

// middleware records request count, duration, and in-flight gauge for every
// request that flows through the router. The route label is the chi
// templated pattern (e.g. "/api/movies/{id}"), not the raw URL — that keeps
// label cardinality bounded. Requests that don't match any route are
// labeled "unmatched".
func (m *metrics) middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			m.inFlight.Inc()
			defer m.inFlight.Dec()

			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rec, r)

			route := "unmatched"
			if rc := chi.RouteContext(r.Context()); rc != nil {
				if p := rc.RoutePattern(); p != "" {
					route = p
				}
			}
			code := strconv.Itoa(rec.status)
			m.requests.WithLabelValues(r.Method, route, code).Inc()
			m.durations.WithLabelValues(r.Method, route, code).Observe(time.Since(start).Seconds())
		})
	}
}
