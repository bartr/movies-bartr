# Repo memory for AI assistants

> This file is the durable scratchpad for any AI working in this repo. Read it
> first; update it at the close of every session. Keep it short.

## Project

- This repo is the **Movies** experiment harness ([README.md](README.md)).
- Spec: [spec.md](docs/spec.md). Methodology: [METHODOLOGY.md](docs/METHODOLOGY.md). Experiment: [EXPERIMENT.md](docs/EXPERIMENT.md). Log: [session-log.md](session-log.md).
- We follow **sessions + RPI**. Every session = one RPI cycle = artifacts in `.copilot-tracking/` + a tagged dot-release.

## Stack (decided session 1)

- Go 1.26 · chi v5 · `log/slog` (JSON) · `flag`+env (`MOVIES_*`) for config.
- Module: `github.com/bartr/bartr-movies`.
- Layout: Go module + Dockerfile + `data/` live under `src/`. Manifests under `deploy/<component>/{base,overlays/dev}` — `deploy/movies/` for the API, `deploy/prometheus/` and `deploy/prometheus-operator/` for monitoring, and `deploy/traefik/` for the k3s Traefik HelmChartConfig (entrypoints). Makefile at repo root drives both.
- Image base: `gcr.io/distroless/static-debian12:nonroot`. Pod runs uid 1000, RO root FS, drop ALL caps.
- Manifests: Kustomize only (`deploy/<component>/base` + `deploy/<component>/overlays/dev`; components are `movies`, `prometheus`, `prometheus-operator`, `traefik`). **No Helm charts authored here**; the `traefik` component only patches the k3s-bundled chart via a `HelmChartConfig`.
- Local cluster: native `k3s` on the host (no k3d/kind).
- Image distribution: `docker save` → `sudo k3s ctr images import`. **No local registry.**

## Conventions

- Tests use `httptest` + table-driven cases; run with `go test -race ./...`.
- Endpoints return `Content-Type: text/plain; charset=utf-8` + body ending in `\n` for plaintext (spec §6.1 explicit for `/version`).
- Errors for `/api/*` will return `application/problem+json` (RFC 7807) — landing in session 4.
- Effective config logged once at info on startup (spec §11).

## Cluster facts (this host)

- Traefik entrypoints (declared by the `kube-system/traefik` HelmChartConfig — already on the cluster, not managed by this repo). Each entrypoint is published on the listed host port, so an Ingress pinned to it needs **no host header**:

  | Entrypoint   | Host port | Used by                       |
  |--------------|-----------|-------------------------------|
  | `web`        | 80        | `movies-api` (`Host: localhost`) |
  | `websecure`  | 443       | TLS (unused here)             |
  | `prometheus` | 9090      | `deploy/prometheus` Ingress   |
  | `grafana`    | 3000      | `deploy/grafana` Ingress      |
  | `vllm`       | 8000      | other workload on this host   |
  | `cllm`       | 8088      | other workload on this host   |
  | `ask`        | 8008      | other workload on this host   |

  Pin to a specific entrypoint with the annotation `traefik.ingress.kubernetes.io/router.entrypoints: <name>` on the Ingress.
- On this box, `localhost` resolves to `::1` and the k3s CNI hostport DNAT is IPv4-only. Use `127.0.0.1` (or the host's LAN IP) in verify scripts.

## Where the next session starts

After tag `0.9.0`: every box in spec §14 acceptance is `[x]` with measured evidence — p95 list routes 0.10–0.23 ms · p95 detail routes ≈ 0.10 ms · 585 RPS sustained · 0% 5xx · 0% 4xx (single 500m pod). Bench instrumentation: `prometheus.DefBuckets` replaced with a sub-ms ladder (100 µs floor) in `src/internal/httpapi/metrics.go`; `apiRouteLabel` emits five route templates including `/api/movies/{id}` and `/api/actors/{id}`; webv `--sleep` takes Go durations and renders timing as `%.3fms`. In-cluster load gen pinned to `--threads=2 --sleep=3ms`. Grafana dashboard v3: 5-tile row 1 (Total · In-flight · 4xx/s · 5xx/s · Workers) + stacked req/s by route + sub-ms p95 timeseries. New `docs/PERFORMANCE.md` documents the 50–500× headroom against spec. **Session 10** is the 1.0.0 cut: pick the hardening direction (TLS / authn / rate limit / RETRO write-up), close any remaining experiment-evidence gaps, tag `1.0.0`.

## Inner loop quickref

```
make test image import deploy verify        # one cycle
make image import deploy verify VERSION=X.Y.Z   # bump
```

## Don't

- Don't add Helm.
- Don't introduce a third-party logger; `log/slog` is enough.
- Don't add a config file format; defaults < env < flags only.
- Don't bake data into an image stage other than the one defined when session 2 lands.
- Don't start a session without filling in the Frame in `session-log.md`.
