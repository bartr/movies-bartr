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
  | `grafana`    | 3000      | reserved for session 7        |
  | `vllm`       | 8000      | other workload on this host   |
  | `cllm`       | 8088      | other workload on this host   |
  | `ask`        | 8008      | other workload on this host   |

  Pin to a specific entrypoint with the annotation `traefik.ingress.kubernetes.io/router.entrypoints: <name>` on the Ingress.
- On this box, `localhost` resolves to `::1` and the k3s CNI hostport DNAT is IPv4-only. Use `127.0.0.1` (or the host's LAN IP) in verify scripts.

## Where the next session starts

After tag `0.6.0`: `/metrics` on the same 8080 port, `prometheus/client_golang` v1.23 with a per-router registry; Go + process collectors plus `http_requests_total`, `http_request_duration_seconds`, `http_requests_in_flight` keyed by templated chi route. `ServiceMonitor` labeled `monitoring.coreos.com/instance: prometheus` for the cluster Prometheus operator. `default-deny` + `movies-api` `NetworkPolicy` pair locks the namespace down to Traefik + scrape ingress + DNS egress. Container `securityContext` now sets `runAsGroup` and `seccompProfile: RuntimeDefault` explicitly. `internal/httpapi` coverage 92.7 %. **Session 7** picks Grafana + provisioned dashboard (the cluster `default/prometheus` instance has been 0/1 for a while — restoring it is a tactical pre-step, not a frame-buster).

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
