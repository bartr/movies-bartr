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

After tag `0.8.0`: `webv` ships as a single-binary Web Validate-compatible CLI under `src/cmd/webv/` (flags `--url|-u --files|-f --loop|-l --threads|-t --random|-r --duration|-d --verbose|-v --version`, defaults `statusCode=200`/`contentType=application/json`, tab-delimited output with `# pass complete` heartbeats, exit 0 on signal). The same Dockerfile bakes both `/movies-api` and `/webv` plus the suites at `/webv-suites/`. `deploy/webv/` runs the load generator as a Deployment in the `movies` namespace pointing at `http://movies-api.movies.svc.cluster.local:8080` with `--loop --threads=2`; a sibling NetworkPolicy locks egress to DNS + movies-api on 8080 only. `make webv-install` installs the CLI into `~/go/bin`. New `src/webv/benchmark.json` has 200 expected-200 entries spanning `/api/{movies,actors,genres}` plus path-id and query-string variants. IMPL-README §12 documents the full five-step inner loop. **Session 9** picks Bench tuning — turn the runner into a p95 + 500 RPS gate.

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
