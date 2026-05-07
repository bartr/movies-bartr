# Session 2 — Data layer plan (2026-05-07)

## Goal (Frame)

- Build `internal/store` with all indexes called out in research, loaded from `src/data/{movies,actors,ratings}.json`.
- Unit tests ≥80% line coverage on `internal/store` and `internal/config`.
- `/api/*` HTTP work stays out of scope (session 3).

## Cuts available if fit-check fails

1. Drop the wired-in load-on-startup in `main.go`; ship store + tests only.
2. Drop actor-side search index; keep only movies search (still meets "q on /api/movies").

## Steps

1. Create `internal/store/store.go` — types + indexes + accessors + search.
2. Create `internal/store/load.go` — read three JSON files, build the store, return errors for malformed/missing/unreferenced input.
3. Add fixture under `internal/store/testdata/` (3 movies / 3 actors / 3 ratings) for deterministic tests, plus a malformed fixture dir.
4. `internal/store/store_test.go` — table-driven tests for every accessor + search + load error paths. Smoke test against the real `src/data/` to keep us honest.
5. `internal/config/config_test.go` — table-driven tests for defaults / env / flag precedence + `--help` + invalid values.
6. Wire loader into `cmd/movies-api/main.go`: load store before flipping `ready`. Log effective record counts.
7. `make test` (race + cover) and confirm ≥80% on both packages.

## Acceptance

- `go test -race -coverprofile=cover.out ./internal/store ./internal/config` reports ≥80% coverage on both.
- `go vet ./...` clean.
- `make test` green.
- Store-level smoke test loads `src/data/` and confirms 100 movies, 553 actors, 100 ratings, every movie has a rating.
- No `/api/*` route changes.
