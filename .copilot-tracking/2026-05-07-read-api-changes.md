# Read API + Validation — Changes (2026-05-07)

## New files

- `src/internal/httpapi/problem.go` — RFC 7807 helper (`writeProblem`).
- `src/internal/httpapi/validate.go` — strict parsers + bounds constants + id regex with not-all-zero rule.
- `src/internal/httpapi/api.go` — five `/api/*` handlers + filtering helpers.
- `src/internal/httpapi/api_test.go` — happy paths + 1 negative per validation rule + 503/404 paths.

## Edits

- `src/internal/httpapi/router.go` — `NewRouter(version, ready, store StoreFunc)`; mounts `/api/*` under chi.Route.
- `src/cmd/movies-api/main.go` — `atomic.Pointer[store.Store]`; pass accessor into `NewRouter`.
- `src/internal/httpapi/handlers_test.go` — pass nil StoreFunc to existing tests.
- `src/Dockerfile` — `COPY data /data/`; bump default `VERSION` arg to `0.3.0`.
- `src/internal/version/version.go` — default Version `"0.3.0"`.
- `deploy/k8s/base/deployment.yaml` — image `movies-api:0.3.0`.
- `Makefile` — default `VERSION ?= 0.3.0`.
- `docs/spec.md` §6 — pin year/rating/genre/q/page rules + not-all-zero id rule + RFC 7807 reference.
- `IMPL-README.md` — record what 0.3.0 ships; refresh deferred-tag table.
- `AGENTS.md` — bump "where the next session starts".
- `session-log.md` — Session 3 frame + close ritual.

## Verification

- `cd src && go test -race -cover ./...` — all green; `internal/httpapi` 91.2 %.
- `make image import deploy verify VERSION=0.3.0` — `/version` returns `0.3.0`, `/healthz` and `/readyz` `pass`.
- Live spot checks: `/api/genres` returns the 20-genre array, `/api/movies?year=1999` returns 1, `/api/movies?q=a` 400 with `application/problem+json`, `/api/movies/tt0133093` returns The Matrix.
