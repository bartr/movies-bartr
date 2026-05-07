# Metrics + Operator + ServiceMonitor + NetworkPolicy — Research (2026-05-07)

## Goal (from frame)

- Prometheus metrics on `/metrics`, idiomatic Go (`prometheus/client_golang`).
- Scraped by the existing in-cluster Prometheus Operator via a `ServiceMonitor`.
- NetworkPolicy locking down the `movies` namespace.
- Tighter container/pod `securityContext` review.
- **Out of scope:** Grafana dashboards (deferred to a later session).

## State of the world (verified)

- Tag `0.5.0` shipped OpenAPI 3 + Swagger UI + JSON request log + robots.txt.
  Session-log only captures sessions 1–3; that's a known reporting gap, not
  fixed here.
- Cluster: native `k3s` on host. Prometheus Operator CRDs already installed:
  `monitoring.coreos.com/v1` ServiceMonitor, PodMonitor, Prometheus, etc.
  CRDs created 2026-04-24.
- A `Prometheus` instance `default/prometheus` exists with:
  - `serviceMonitorNamespaceSelector: {}` (all namespaces)
  - `serviceMonitorSelector.matchLabels: monitoring.coreos.com/instance: prometheus`
  - `scrapeInterval: 30s`
  - status: 0/1 ready (image-pull or storage). Not in scope to fix; we only
    need to ship a correctly-labeled `ServiceMonitor` so the operator picks it
    up the moment Prometheus is healthy.
- `movies` namespace: Deployment + Service + Ingress, no ServiceMonitor, no
  NetworkPolicy. Pod runs uid 1000, RO root FS, drop ALL caps,
  allowPrivilegeEscalation:false, pod-level seccompProfile RuntimeDefault.
- spec §6 lists `/metrics`. spec §7.1 says "Prometheus text exposition format,
  idiomatic client library". spec §8.1 says scraping is via ServiceMonitor —
  no `prometheus.io/scrape` annotations.

## Library choice

- `github.com/prometheus/client_golang` (v1.x) — the idiomatic Go library.
  Use `promhttp.Handler()` for `/metrics`, default registry already includes
  Go runtime + process collectors.
- For HTTP instrumentation, wrap chi handlers with
  `promhttp.InstrumentHandlerCounter` + `InstrumentHandlerDuration` keyed by
  method/code, OR write a small middleware that records:
  - `http_requests_total{method,route,code}` (counter)
  - `http_request_duration_seconds{method,route,code}` (histogram, default buckets)
  - `http_requests_in_flight` (gauge)
  Preferred: small middleware that uses `chi.RouteContext(r.Context()).RoutePattern()`
  AFTER ServeHTTP so the label is the *templated* path (`/api/movies/{id}`),
  not the raw URL — keeps cardinality bounded.

## ServiceMonitor

- Namespace: `movies`.
- Labels: `monitoring.coreos.com/instance: prometheus` (matches the
  `Prometheus.serviceMonitorSelector` above), plus standard
  `app.kubernetes.io/name: movies-api`.
- `spec.selector.matchLabels` matches the `movies-api` Service labels.
- `spec.endpoints[0]`: `port: http`, `path: /metrics`, `interval: 30s`,
  `scheme: http`.
- We are not adding a separate metrics port; spec §8.1 explicitly allows
  `/metrics` on the same 8080 port.

## NetworkPolicy

Two policies in the `movies` namespace:

1. `default-deny` — all pods, denies all ingress + egress.
2. `movies-api` — allows:
   - **Ingress** on TCP 8080 from:
     - `kube-system` namespace (Traefik) — for Ingress traffic.
     - `default` namespace (Prometheus pod) — for `/metrics` scrape.
   - **Egress** to:
     - `kube-system` DNS (UDP+TCP 53) — for general resolution. The pod
       currently makes no other network calls, so nothing else is opened.

## Tighter securityContext

Current pod has runAsNonRoot/runAsUser/runAsGroup/fsGroup + seccompProfile
RuntimeDefault. Container has runAsNonRoot/runAsUser/RO root FS/drop ALL/
allowPrivilegeEscalation:false.

Tightenings to add:

- Container-level `seccompProfile: RuntimeDefault` (defense in depth — pods
  inherit, but the Pod Security Admission "restricted" profile checks the
  *container* level for seccomp).
- Container-level `runAsGroup: 1000` (currently only at pod).
- Container-level `capabilities.drop: [ALL]` is already set; nothing to add.

## Tests

- `httpapi`: assert `GET /metrics` returns 200 with
  `Content-Type: text/plain; version=0.0.4; charset=utf-8` (default from
  `promhttp`) and the body contains a known counter (`http_requests_total`)
  after at least one instrumented request.
- Middleware test: hit a 200 + a 404 + a 4xx route and assert each shows up
  exactly once in the registry with the expected `route` label (templated).

## Verification (cluster)

After deploy:

- `curl http://localhost/metrics | head` — HELP/TYPE lines, `go_*` runtime,
  `http_requests_total`.
- `kubectl -n movies get servicemonitor` — exists with correct labels.
- `kubectl -n movies get networkpolicy` — two policies present.
- Hitting `/healthz` from a pod *outside* the allowed sources should be
  blocked (skipped if Prometheus pod not running — record as known gap, not
  a session blocker since policy correctness is verifiable by inspection).

## Risks / unknowns

- Prometheus pod itself is currently 0/1 ready (storage / image issue
  pre-dating this session). Out of scope to repair; we only ensure our
  manifests are correct so when Prometheus comes back the scrape works.
- The default Go-collector cardinality is fine; histogram buckets are the
  client_golang defaults. Keep it boring.
