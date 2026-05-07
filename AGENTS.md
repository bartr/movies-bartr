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

After tag `0.7.0`: Grafana 11.3.0 runs in the `monitoring` namespace alongside Prometheus, Ingress on the Traefik `grafana` entrypoint at host port 3000, anonymous Viewer enabled for dev, admin password `Passw0rd` injected via the `grafana-admin` Secret (dev overlay only). The `prometheus` datasource (uid `prometheus`, url `http://prometheus.monitoring.svc:9090`) is provisioned via file. The movies-api dashboard (uid `movies-api`) is created at boot through the Grafana **HTTP API** by a one-shot `Job` running `curlimages/curl` — so it stays editable + saveable in the UI — and starred for admin via the same Job. **Session 8** picks the Web Validate runner: hit `/api/*` through the Traefik `web` entrypoint, dashboard panels light up the requests/p95 timeseries.

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
