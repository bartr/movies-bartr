# Movies API — Implementation Notes (Go)

> Living document for participants of the sessions+RPI experiment. Updated each session.
>
> Spec: [spec.md](docs/spec.md) · Methodology: [METHODOLOGY.md](docs/METHODOLOGY.md) · Sessions: [session-log.md](session-log.md)

## Stack (Session 1)

- **Language:** Go 1.26
- **HTTP:** `net/http` + `github.com/go-chi/chi/v5`
- **Logging:** `log/slog` (JSON handler, stdout, level via `MOVIES_LOG_LEVEL`)
- **Config:** `flag` + env (`MOVIES_*`); precedence defaults < env < flags (spec §11)
- **Container:** distroless `gcr.io/distroless/static-debian12:nonroot`
- **Manifests:** Kustomize `deploy/k8s/{base,overlays/dev}` (no Helm, per spec §8)
- **Local k8s:** native `k3s` on the host

## Layout

```
src/                       Go module + Dockerfile + data
  cmd/movies-api/          entrypoint
  internal/config/         flag/env config
  internal/httpapi/        chi router + handlers
  internal/version/        embedded semver
  data/                    source-of-truth JSON (baked into image, spec §5.2)
  Dockerfile               multi-stage; build context is ./src
  go.mod, go.sum
deploy/k8s/base/           ns + deployment + service + kustomization
deploy/k8s/overlays/dev    seam for dev-only resources (Prometheus etc.)
.copilot-tracking/         RPI artifacts (research/plan/changes/review)
Makefile                   inner-loop wrapper (run from repo root)
```

## Inner loop (§12)

```bash
make test          # 1. unit tests
make image         # 2. docker build (sets VERSION via build arg + ldflags)
make import        # 3. docker save | k3s ctr images import   (no registry needed)
make deploy        # 4. kustomize build | kubectl apply  (waits for rollout)
make verify        # 5. port-forward + curl /version /healthz /readyz
```

Once deployed, Traefik (bundled with k3s, listening on host port 80) routes
`localhost` → the `movies-api` Service:

```bash
curl http://localhost/version    # 0.1.0
curl http://localhost/healthz    # pass
curl http://localhost/readyz     # pass
```

To bump the version, override `VERSION`:

```bash
make image import deploy verify VERSION=0.1.1
```

## What's done (tag 0.6.0)

- **Session 1 (0.1.0):** `/version`, `/healthz`, `/readyz` walking skeleton on distroless; non-root, RO root FS, all caps dropped; Kustomize-only manifests; Traefik Ingress on `localhost`.
- **Session 2 (0.2.0):** `internal/store` with id/genre/year/rating-bucket/actor indexes + `q=` substring search; loader cross-references all four duplicate id fields and gates `/readyz` until the dataset is in memory.
- **Session 3 (0.3.0):** `/api/movies`, `/api/movies/{id}`, `/api/actors`, `/api/actors/{id}`, `/api/genres` with full validation (`pageNumber`, `pageSize`, `q`, `genre`, `year`, `rating`, `actorId`, path ids — see spec §6). Errors are RFC 7807 `application/problem+json`. `internal/httpapi` coverage 91.2 % with one negative test per rule mirroring `test.json`.
- **Session 5 (0.5.0):** OpenAPI 3 doc embedded at compile time + Swagger UI at `/swagger`, root redirect, `robots.txt`, JSON request-log middleware.
- **Session 6 (0.6.0):** Prometheus metrics on `/metrics` (`prometheus/client_golang` v1.23, per-router registry, Go + process collectors, `http_requests_total` / `http_request_duration_seconds` / `http_requests_in_flight` with templated chi route labels). `ServiceMonitor` labeled for the cluster Prometheus operator; `default-deny` + `movies-api` NetworkPolicy pair (Traefik + scrape ingress, DNS egress); container `securityContext` tightened with explicit `runAsGroup` and `seccompProfile: RuntimeDefault`. `internal/httpapi` coverage 92.7 %.

## What's deferred

See [session-log.md](session-log.md) for the per-session frame. Headline:

| Tag    | Adds                                                       |
|--------|------------------------------------------------------------|
| 0.7.0  | Grafana + provisioned dashboard                            |
| 0.8.0  | Web Validate runner + documented inner loop                |
| 0.9.0  | Benchmarks (p95 + 500 RPS)                                 |
| 1.0.0  | §14 acceptance run + RETRO.md                              |
