# Changes — Walking Skeleton (Session 1, tag 0.1.0)

**Date:** 2026-05-06

## Files added

- `cmd/movies-api/main.go` — entrypoint: config load, slog JSON, synthetic ready, chi router, http.Server with timeouts, graceful shutdown.
- `internal/config/config.go` — defaults < env (`MOVIES_*`) < flags. Validates log level, port, non-empty data dir. `--help` prints flags + env + effective.
- `internal/version/version.go` — exported `Version = "0.1.0"`, ldflags-overridable.
- `internal/httpapi/router.go` — chi router with `Recoverer` middleware; takes `version` and `ReadyFunc`.
- `internal/httpapi/handlers.go` — `/version`, `/healthz`, `/readyz`.
- `internal/httpapi/handlers_test.go` — table-driven tests for all four behaviours (incl. version-independent-of-ready, both readyz states).
- `Dockerfile` — multi-stage `golang:1.26-bookworm` → `gcr.io/distroless/static-debian12:nonroot`, `USER 1000:1000`, `EXPOSE 8080`. `VERSION` build-arg threaded into ldflags.
- `.dockerignore` — excludes manifests, tests, docs, tracking dir.
- `.gitignore` — Go + tarball + coverage artifacts.
- `deploy/k8s/base/{namespace,deployment,service,kustomization}.yaml`.
- `deploy/k8s/overlays/dev/kustomization.yaml`.
- `Makefile` — `build|test|image|import|deploy|verify|undeploy|clean`.
- `IMPL-README.md` — implementation notes; inner-loop quickref.
- `AGENTS.md` — repo memory for AI assistants.
- `go.mod`, `go.sum` — `github.com/go-chi/chi/v5 v5.2.5` only.

## Verifications run

1. `go test -race ./...` → ok.
2. `go vet ./...` → ok.
3. `kustomize build deploy/k8s/overlays/dev | kubectl apply --dry-run=client` → clean.
4. `make image` → distroless image, `~3.7 MB` content, `~16.2 MB` disk.
5. Smoke (`docker run`): `/version` returned `0.1.0\n` + `text/plain; charset=utf-8`; `/healthz` `pass\n`; `/readyz` `pass\n`.
6. `make import` → `docker.io/library/movies-api:0.1.0` listed in `k3s ctr images ls`.
7. `make deploy` → `deployment "movies-api" successfully rolled out`; pod `1/1 Running`.
8. Pod `securityContext` confirmed `{allowPrivilegeEscalation:false, capabilities.drop:[ALL], readOnlyRootFilesystem:true, runAsNonRoot:true, runAsUser:1000}`.
9. Cluster verify (port-forward + curl): all three endpoints return correct status, content-type, and body live in k3s.

## Deviations from plan

- Replaced deprecated `commonLabels` with `labels:` form in `kustomization.yaml` after `kustomize` warned. No semantic change.
- Added an `emptyDir` mount at `/tmp` proactively (plan listed it as conditional). Cost is zero and removes a class of "RO root FS" surprises later.
- Left `automountServiceAccountToken: false` in the pod spec; not in the plan but it is a free hardening win and aligns with §13.
