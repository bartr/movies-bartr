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
- Layout: Go module + Dockerfile + `data/` live under `src/`. Manifests under `deploy/k8s/`. Makefile at repo root drives both.
- Image base: `gcr.io/distroless/static-debian12:nonroot`. Pod runs uid 1000, RO root FS, drop ALL caps.
- Manifests: Kustomize only (`deploy/k8s/base` + `deploy/k8s/overlays/dev`). **No Helm.**
- Local cluster: native `k3s` on the host (no k3d/kind).
- Image distribution: `docker save` → `sudo k3s ctr images import`. **No local registry.**

## Conventions

- Tests use `httptest` + table-driven cases; run with `go test -race ./...`.
- Endpoints return `Content-Type: text/plain; charset=utf-8` + body ending in `\n` for plaintext (spec §6.1 explicit for `/version`).
- Errors for `/api/*` will return `application/problem+json` (RFC 7807) — landing in session 4.
- Effective config logged once at info on startup (spec §11).

## Where the next session starts

After tag `0.3.0`: `/api/movies`, `/api/movies/{id}`, `/api/actors`, `/api/actors/{id}`, `/api/genres` are live with full query/path validation and RFC 7807 error bodies; `internal/httpapi` coverage is 91.2 %. **Session 4** picks up the next slice — top candidates are OpenAPI + Swagger UI (spec §6 routes `/`, `/swagger`, `/swagger/v1/swagger.json`) **or** Prometheus metrics + `ServiceMonitor` + NetworkPolicy. Frame should fill 90–120 min — last session over-shot the "fits" claim again, so default to bundling adjacent slices and only cut at the fit check.

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
