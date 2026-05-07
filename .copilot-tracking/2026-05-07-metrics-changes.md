# Metrics + ServiceMonitor + NetworkPolicy — Changes (2026-05-07)

## Source

- `src/internal/httpapi/metrics.go` (new): per-router `*prometheus.Registry` +
  Go runtime + process collectors + three vectors (`http_requests_total`,
  `http_request_duration_seconds`, `http_requests_in_flight`). `middleware()`
  records on every request using `chi.RouteContext(...).RoutePattern()` for
  the route label so cardinality stays bounded; unmatched requests get
  `route="unmatched"`. `handler()` returns `promhttp.HandlerFor(...)`.
- `src/internal/httpapi/router.go`: register `m := newMetrics()`, install
  `m.middleware()` after `Recoverer` and before `requestLogger()`, mount
  `m.handler()` at `GET /metrics`. Package comment updated.
- `src/internal/httpapi/middleware.go`: skip the structured request log line
  when `r.URL.Path == "/metrics"` so 30 s scrapes don't drown the log.
- `src/internal/httpapi/metrics_test.go` (new): asserts /metrics 200 +
  `text/plain` content-type, presence of all three application metrics +
  Go/process collectors, templated route labels, no raw-id leak,
  `route="unmatched"` for unrouted requests.
- `src/internal/httpapi/openapi.json`: bumped `info.version` to `0.6.0`,
  added `/metrics` path, refreshed `/version` example.
- `src/go.mod` / `src/go.sum`: added `github.com/prometheus/client_golang
  v1.23.2` (+ transitive `client_model`, `common`, `procfs`, `klauspost/
  compress`, `golang.org/x/sys`, `protobuf`, `kr/text`).

## Manifests

- `deploy/k8s/base/servicemonitor.yaml` (new): ServiceMonitor `movies-api`
  in `movies` ns with label `monitoring.coreos.com/instance: prometheus`
  matching the cluster Prometheus's `serviceMonitorSelector`. Endpoint
  `port=http path=/metrics interval=30s`.
- `deploy/k8s/base/networkpolicy.yaml` (new): two-policy pattern.
  `default-deny` selects every pod and drops ingress + egress; `movies-api`
  re-allows TCP 8080 ingress from `kube-system` (Traefik) and `default`
  (Prometheus operator scrape), plus DNS egress to `kube-system`.
- `deploy/k8s/base/kustomization.yaml`: added the two new resources.
- `deploy/k8s/base/deployment.yaml`: image bumped `0.3.0` → `0.6.0`;
  container `securityContext` tightened with `runAsGroup: 1000` and
  `seccompProfile: { type: RuntimeDefault }` (the Pod Security Admission
  "restricted" profile checks for these at the container level even when a
  pod-level profile is present).

## Build / docs / ops

- `Makefile`: `VERSION ?= 0.6.0`. `verify` now also asserts /metrics returns
  200 and contains both `http_requests_total` (HELP line) and
  `go_goroutines`, so a missing instrumentation regression breaks the
  inner loop.
- `src/Dockerfile`: default `ARG VERSION=0.6.0`.
- `IMPL-README.md`: "what's done" table + headline updated.
- `session-log.md`: Session 6 frame + close ritual.
- `.copilot-tracking/2026-05-07-metrics-{research,plan,changes,review}.md`:
  RPI artifacts.

## Verification (in-cluster)

- `make image import deploy verify VERSION=0.6.0` → green after the
  rolling-update window.
- Spot checks:
  - `kubectl -n movies get servicemonitor` → present.
  - `kubectl -n movies get networkpolicy` → `default-deny` + `movies-api`.
  - `curl /metrics | grep http_requests_total{...}` shows templated routes
    `/api/movies/{id}`, `/api/genres`, `/healthz`, `/readyz`, `/version`,
    `/metrics` with `code="200"` / `code="404"` labels — cardinality is
    bounded, raw IDs do not appear.
  - `/api/genres` returns the expected 20-genre array.

## Known not-fixed

- The `default/prometheus` Prometheus instance is still 0/1 (pre-existing,
  out of scope for this session). The ServiceMonitor is correctly labeled,
  so as soon as Prometheus comes back the scrape will pick up automatically.
- ServiceMonitor namespace selector for the cluster Prometheus is `{}`
  (all namespaces) and the label is `monitoring.coreos.com/instance:
  prometheus` — both verified against the live `Prometheus` resource.

## Coverage

- `internal/httpapi`: 92.7 % statements (was 91.2 %).
- All packages green under `go test -race ./...`.
