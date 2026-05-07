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
- **Manifests:** Kustomize `deploy/<component>/{base,overlays/dev}` — `movies/` today, `prometheus/` and `prometheus-operator/` are skeleton siblings (no Helm, per spec §8)
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
deploy/movies/base/        ns + deployment + service + ingress + servicemonitor + networkpolicy
deploy/movies/overlays/dev seam for dev-only resources
deploy/webv/base/          load-generator Deployment + NetworkPolicy (movies ns)
deploy/prometheus/         monitoring ns: SA + RBAC + Prometheus CR + Service + Ingress
deploy/prometheus-operator pinned upstream operator bundle (Kustomize remote resource)
deploy/traefik/            k3s Traefik HelmChartConfig (entrypoint ports)
.copilot-tracking/         RPI artifacts (research/plan/changes/review)
Makefile                   inner-loop wrapper (run from repo root)
```

## Inner loop (§12)

Five-step cycle wrapping the targets in [Makefile](Makefile). Each step is independently runnable.

```bash
make test          # 1. unit tests (-race) across the module
make image         # 2. docker build (sets VERSION via build arg + ldflags)
make import        # 3. docker save | k3s ctr images import   (no registry needed)
make deploy        # 4. kustomize build | kubectl apply  (waits for rollout)
make verify        # 5. curl /version /healthz /readyz /metrics through Traefik
```

Once deployed, Traefik (bundled with k3s, listening on host port 80) routes
`localhost` → the `movies-api` Service:

```bash
curl http://localhost/version    # 0.8.0
curl http://localhost/healthz    # pass
curl http://localhost/readyz     # pass
```

To bump the version, override `VERSION`:

```bash
make image import deploy verify VERSION=0.1.1
```

### Web Validate (`webv`) — session 8

`webv` is a small Web Validate-compatible runner ([cmd/webv](src/cmd/webv)).
It reads JSON suites in the shape used by [test.json](test.json),
issues HTTP requests against a base URL, and validates response
status code, content type, and (optionally) body length. Defaults are
`statusCode=200` and `contentType=application/json`; both can be
overridden per-request.

```bash
make webv-install  # go install -ldflags ... ./cmd/webv  → ~/go/bin/webv
make webv-smoke    # one pass against http://127.0.0.1 with src/webv/test.json
make webv-deploy   # apply the in-cluster Deployment (movies ns, --loop)
make webv-verify   # tail the pod logs and confirm pass=N fail=0 heartbeat
make webv-undeploy # delete the load-generator Deployment
```

CLI flags (each has a short alias):

| Flag                   | Purpose                                                  |
|------------------------|----------------------------------------------------------|
| `--url`     `-u`       | base URL (`http`/`https`); required                      |
| `--files`   `-f`       | suite file(s); repeatable or comma-separated; required   |
| `--loop`    `-l`       | run forever                                              |
| `--threads` `-t`       | concurrent worker goroutines (default 1)                 |
| `--random`  `-r`       | shuffle requests each pass                               |
| `--duration` `-d`      | total run time (`30s`, `5m`, `24h`); takes precedence over `--loop` |
| `--verbose` `-v`       | log successes too (failures always logged)               |
| `--version`            | print the shared semver and exit                         |

Output is one tab-delimited record per logged request plus a
`# pass complete` heartbeat line at the end of every pass and a
`# summary pass=N fail=N` line on shutdown.

The same Dockerfile bakes both `/movies-api` and `/webv` into the image
along with the suites at `/webv-suites/`, so the in-cluster Deployment
([deploy/webv](deploy/webv)) is just `image: movies-api:0.8.0` with
`command: ["/webv"]`. webv targets the in-cluster movies-api Service
(`http://movies-api.movies.svc.cluster.local:8080`) and runs in a loop;
it always exits 0 on signal so K8s does not flag it as failed.

## What's done (tag 0.7.0)

- **Session 1 (0.1.0):** `/version`, `/healthz`, `/readyz` walking skeleton on distroless; non-root, RO root FS, all caps dropped; Kustomize-only manifests; Traefik Ingress on `localhost`.
- **Session 2 (0.2.0):** `internal/store` with id/genre/year/rating-bucket/actor indexes + `q=` substring search; loader cross-references all four duplicate id fields and gates `/readyz` until the dataset is in memory.
- **Session 3 (0.3.0):** `/api/movies`, `/api/movies/{id}`, `/api/actors`, `/api/actors/{id}`, `/api/genres` with full validation (`pageNumber`, `pageSize`, `q`, `genre`, `year`, `rating`, `actorId`, path ids — see spec §6). Errors are RFC 7807 `application/problem+json`. `internal/httpapi` coverage 91.2 % with one negative test per rule mirroring `test.json`.
- **Session 5 (0.5.0):** OpenAPI 3 doc embedded at compile time + Swagger UI at `/swagger`, root redirect, `robots.txt`, JSON request-log middleware.
- **Session 6 (0.6.0):** Prometheus metrics on `/metrics` (`prometheus/client_golang` v1.23, per-router registry, Go + process collectors, `http_requests_total` / `http_request_duration_seconds` / `http_requests_in_flight` with templated chi route labels). `ServiceMonitor` labeled for the cluster Prometheus operator; `default-deny` + `movies-api` NetworkPolicy pair (Traefik + scrape ingress, DNS egress); container `securityContext` tightened with explicit `runAsGroup` and `seccompProfile: RuntimeDefault`. `internal/httpapi` coverage 92.7 %.
- **Session 7 (0.7.0):** Grafana 11.3.0 in the `monitoring` namespace, anonymous Viewer for dev, admin password `Passw0rd` injected via Kubernetes `Secret` (dev overlay `secretGenerator`), Ingress pinned to the Traefik `grafana` entrypoint on host port 3000. Prometheus datasource (uid `prometheus`) provisioned via file. The movies-api dashboard is created at boot through the Grafana **HTTP API** (`POST /api/dashboards/db`) by a one-shot bootstrap `Job` running `curlimages/curl` — so the dashboard stays editable + saveable in the UI rather than read-only like a file-provisioned dashboard. The same Job stars the dashboard for admin via `POST /api/user/stars/dashboard/uid/movies-api`.
- **Session 8 (0.8.0):** `webv` Web Validate-compatible runner ([cmd/webv](src/cmd/webv)) — single-binary CLI with `--url`/`--files`/`--loop`/`--threads`/`--random`/`--duration`/`--verbose`/`--version` flags (each has a short alias), tab-delimited output, per-pass heartbeat, defaults `statusCode=200`/`contentType=application/json` overridable per-request. Same Dockerfile now ships both `/movies-api` and `/webv` plus the suites at `/webv-suites/`. New `benchmark.json` (200 entries spanning `/api/{movies,actors,genres}` plus path-id and query-string variants — all expected 200). In-cluster Deployment under `deploy/webv/` (movies namespace, NetworkPolicy allowing egress only to movies-api on 8080) hits the in-cluster Service in `--loop` and pumps the Active workers / Requests-by-route / p95 panels on the Grafana dashboard. `make webv-install` puts the CLI in `~/go/bin`. §12 inner loop now documented end-to-end.

## What's deferred

See [session-log.md](session-log.md) for the per-session frame. Headline:

| Tag    | Adds                                                       |
|--------|------------------------------------------------------------|
| 0.9.0  | Benchmarks (p95 + 500 RPS)                                 |
| 1.0.0  | §14 acceptance run + RETRO.md                              |
