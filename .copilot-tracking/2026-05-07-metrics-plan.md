# Metrics + ServiceMonitor + NetworkPolicy — Plan (2026-05-07)

## Files

### New

- `src/internal/httpapi/metrics.go` — registry, three metric vectors,
  `metricsMiddleware()`, `metricsHandler()`.
- `src/internal/httpapi/metrics_test.go` — unit tests.
- `deploy/k8s/base/servicemonitor.yaml` — ServiceMonitor with
  `monitoring.coreos.com/instance: prometheus`.
- `deploy/k8s/base/networkpolicy.yaml` — `default-deny` + `movies-api`
  ingress/egress allow.

### Edited

- `src/go.mod` / `src/go.sum` — add `github.com/prometheus/client_golang`.
- `src/internal/httpapi/router.go` — mount `/metrics`, install middleware
  *before* the request logger so it covers everything.
- `src/internal/httpapi/router.go` — exclude `/metrics` from the request
  log to avoid spamming logs with Prometheus scrapes.
- `deploy/k8s/base/kustomization.yaml` — add servicemonitor + networkpolicy
  resources.
- `deploy/k8s/base/deployment.yaml` — bump image tag to `0.6.0`; add
  container-level `seccompProfile: RuntimeDefault` and `runAsGroup: 1000`.
- `deploy/k8s/base/service.yaml` — no change needed; `port: http` already
  fits ServiceMonitor.
- `Makefile` — bump `VERSION` to `0.6.0`. Add `verify` step that also
  checks `/metrics` returns 200 and contains `http_requests_total`.
- `src/Dockerfile` — bump default `ARG VERSION` to `0.6.0`.
- `IMPL-README.md` — flip "0.5.0 = OpenAPI/Swagger, 0.6.0 = metrics"
  table; record what's done.
- `session-log.md` — Session 6 frame + close.

## Approach

### Metrics middleware

```go
var (
    httpReqs = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "http_requests_total",
            Help: "Total HTTP requests by method, route, and status code.",
        },
        []string{"method", "route", "code"},
    )
    httpDur = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "http_request_duration_seconds",
            Help:    "HTTP request latency in seconds by method, route, and status code.",
            Buckets: prometheus.DefBuckets, // default; spec §10.4 only constrains p95.
        },
        []string{"method", "route", "code"},
    )
    httpInFlight = prometheus.NewGauge(
        prometheus.GaugeOpts{
            Name: "http_requests_in_flight",
            Help: "Number of HTTP requests currently being served.",
        },
    )
)
```

Use `prometheus.NewRegistry()` + register Go + process collectors + the three
above. Do **not** use the default registry — keeps tests deterministic and
allows multiple parallel test routers without "AlreadyRegistered" panics.
Each `NewRouter` call creates its own `*prometheus.Registry`.

### Route label cardinality

After `next.ServeHTTP`, read `chi.RouteContext(r.Context()).RoutePattern()`.
For routes that 404 in chi (no match), the pattern is empty — record `"unmatched"`
so we don't blow up cardinality with raw URLs.

### `/metrics` exclusion from request log

Easiest: special-case in `requestLogger()` — if `r.URL.Path == "/metrics"`,
skip logging.

### NetworkPolicy

- `default-deny` selects all pods, allows nothing.
- `movies-api`:
  - `podSelector.matchLabels: app.kubernetes.io/name: movies-api`
  - Ingress[0]: TCP 8080 from
    `namespaceSelector.matchLabels.kubernetes.io/metadata.name: kube-system`
    (Traefik). The well-known label `kubernetes.io/metadata.name` is set
    automatically by kube-apiserver on every Namespace.
  - Ingress[1]: TCP 8080 from `default` (Prometheus pod).
  - Egress[0]: UDP+TCP 53 to `kube-system` (DNS).

Single combined policy keeps it readable; default-deny stays separate.

## Tests

- `metrics_test.go`:
  - `GET /metrics` → 200, content-type starts with `text/plain`, body contains
    `http_requests_total`, `go_goroutines`, `process_cpu_seconds_total`.
  - After a `GET /api/genres` (200) and `GET /api/movies/tt12345` (404), the
    counter has the right labels.
- Existing tests must keep passing.

## Fit check

- Metrics + middleware + tests: ~30 min.
- ServiceMonitor + NetworkPolicy: ~15 min.
- securityContext tightening: ~5 min.
- Build/import/deploy/verify + version bumps + docs + close: ~30 min.

Total target: 80–100 min. Fits. Smallest cut if blocked: drop NetworkPolicy
to a follow-up session — metrics + SM are the headline.

## Out of scope (frame)

- Grafana dashboards.
- Repairing the existing `default/prometheus` Prometheus pod (pre-existing).
- Web Validate runner. Benchmarks.
