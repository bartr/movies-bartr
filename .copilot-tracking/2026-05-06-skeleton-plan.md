# Plan — Walking Skeleton (Session 1, tag 0.1.0)

**Date:** 2026-05-06
**Phase constraint:** Plan only. Numbered, checkbox-able tasks with file targets and exit criteria. No code.

Inputs: [research](2026-05-06-stack-research.md), [spec.md](../docs/spec.md), [README.md](../README.md).

## Tasks

- [ ] **T1. Branch.** Create `git checkout -b session/0.1.0-skeleton`. Exit: branch checked out.
- [ ] **T2. Go module.** `go mod init github.com/bartr/bartr-movies`; pin Go directive to `1.24` for distroless toolchain compat. Exit: `go.mod` exists with single `chi` direct dep added in T3.
- [ ] **T3. Dependencies.** `go get github.com/go-chi/chi/v5@latest`. Exit: `go.sum` populated; `go mod tidy` clean.
- [ ] **T4. Source layout.** Create:
  - `cmd/movies-api/main.go` — flag/env config, slog JSON, build the router, ready flag flipped after (synthetic) load, `http.Server` with sane timeouts, graceful shutdown on SIGTERM.
  - `internal/config/config.go` — `Config` struct + `Load()` from defaults < env (`MOVIES_*`) < flags. Booleans parsed per spec §11. `--help`/`-h` prints flags + env + defaults + effective; effective config logged once at info on startup.
  - `internal/version/version.go` — `var Version = "0.1.0"`; overridable via `-ldflags "-X .../internal/version.Version=..."`.
  - `internal/httpapi/router.go` — builds `chi.Mux` with handlers for `/version`, `/healthz`, `/readyz`; takes a `func() bool` for ready.
  - `internal/httpapi/handlers.go` — the three handlers.
  Exit: files exist with implementations; `go build ./...` succeeds.
- [ ] **T5. Handlers behaviour.**
  - `/version`: `200`, `Content-Type: text/plain; charset=utf-8`, body = `Version + "\n"`. Independent of ready state.
  - `/healthz`: `200` plaintext `pass\n` (always pass in session 1; richer states are a later concern).
  - `/readyz`: `200` plaintext `pass\n` once ready, else `503` plaintext `not ready\n`. Ready flag flipped synchronously in `main` after a no-op "load" (placeholder for session 2).
  Exit: behaviour matches spec §6 / §6.1.
- [ ] **T6. Unit tests.** `internal/httpapi/handlers_test.go` table-driven, using `httptest.NewRecorder` against the router. Cover: version body + content-type; healthz body; readyz both states. Exit: `go test ./... -race` green.
- [ ] **T7. Dockerfile.** Multi-stage:
  - Stage 1: `golang:1.26-bookworm` → build static binary `CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X .../version.Version=$VERSION" -o /out/movies-api ./cmd/movies-api`.
  - Stage 2: `gcr.io/distroless/static-debian12:nonroot` → `COPY` binary, `USER 1000:1000`, `EXPOSE 8080`, `ENTRYPOINT ["/movies-api"]`.
  Exit: `docker build -t movies-api:0.1.0 .` succeeds; `docker run --rm -p 8080:8080 movies-api:0.1.0` serves all three endpoints locally.
- [ ] **T8. Kustomize base.** `deploy/k8s/base/` with: `namespace.yaml` (ns `movies`), `deployment.yaml` (1 replica, probes, resources, securityContext per spec §8.1, port 8080, image `movies-api:0.1.0`, `imagePullPolicy: IfNotPresent`), `service.yaml` (ClusterIP :8080), `kustomization.yaml`. Exit: `kustomize build deploy/k8s/base | kubectl apply --dry-run=client -f -` clean.
- [ ] **T9. Kustomize dev overlay.** `deploy/k8s/overlays/dev/kustomization.yaml` referencing base. Exit: dry-run apply clean.
- [ ] **T10. k3s image import.** Document and run: `docker save movies-api:0.1.0 -o /tmp/movies-api-0.1.0.tar && sudo k3s ctr images import /tmp/movies-api-0.1.0.tar`. Exit: image visible via `sudo k3s ctr images ls | grep movies-api`.
- [ ] **T11. Deploy.** `kustomize build deploy/k8s/overlays/dev | sudo k3s kubectl apply -f -`. Exit: pod `Ready 1/1`.
- [ ] **T12. In-cluster verification.** `sudo k3s kubectl -n movies port-forward svc/movies-api 8080:8080 &` then `curl` each endpoint. Exit: `/version` returns `0.1.0\n` with the right content-type, `/healthz` returns `pass`, `/readyz` returns `pass`.
- [ ] **T13. README + Makefile.** Top-level `README-IMPL.md` (or extend project root README later) with the dev-loop steps actually used. Add a small `Makefile` with `build`, `test`, `image`, `import`, `deploy`, `verify`. Exit: a fresh terminal can run `make image import deploy verify` and reach a green verify.
- [ ] **T14. Repo memory.** Create `AGENTS.md` (or update `.github/copilot-instructions.md`) with: stack chosen, Go version, image/runtime conventions, k3s import pattern, where the next session begins.
- [ ] **T15. Close.** Commit, PR, FF-merge, tag `0.1.0`, fill close fields in `session-log.md`.

## File targets summary

```
cmd/movies-api/main.go
internal/config/config.go
internal/version/version.go
internal/httpapi/router.go
internal/httpapi/handlers.go
internal/httpapi/handlers_test.go
Dockerfile
.dockerignore
Makefile
go.mod
go.sum
deploy/k8s/base/{namespace,deployment,service,kustomization}.yaml
deploy/k8s/overlays/dev/kustomization.yaml
AGENTS.md
.copilot-tracking/2026-05-06-skeleton-changes.md
.copilot-tracking/2026-05-06-skeleton-review.md
```

## Parking lot (defer)

- Ingress, TLS, host-based routing.
- richer `/healthz` states (`warn` semantics).
- `/swagger` and `/` redirect.
- ServiceMonitor, NetworkPolicy.
- Image scanning, SBOM.
- Liveness probe tuning beyond defaults.
- Multi-arch image build.

## Exit criteria for the session

1. `go test ./... -race` green.
2. `docker run` of `movies-api:0.1.0` serves all three endpoints with the contracted content-types and bodies.
3. Pod is `1/1 Ready` in namespace `movies` on the host's k3s.
4. `curl http://127.0.0.1:8080/version` (via port-forward) returns exactly `0.1.0\n`.
5. Container runs as uid 1000 with read-only root FS (verified by inspecting the live pod's `securityContext`).
6. `0.1.0` git tag pushed; PR FF-merged; session log closed.
