# Session 2 — Data layer changes (2026-05-07)

Branch: `session/0.2.0-data-layer`. Target tag: `0.2.0`.

## New code

- [src/internal/store/store.go](../src/internal/store/store.go) — `Movie`, `Actor`, `Role`, `MovieRef`, `Counts`, `Store` with all six required indexes + `q` substring search over precomputed lowercased haystacks for both movies and actors.
- [src/internal/store/load.go](../src/internal/store/load.go) — `Load(dir)` reads `movies.json` / `actors.json` / `ratings.json`, validates id consistency (`id == key == _id == movieId|actorId`), enforces no duplicate ids, and requires every movie to have a matching rating record. Rating + votes are merged onto each `Movie` at load.

## New tests

- [src/internal/store/store_test.go](../src/internal/store/store_test.go) — table-driven coverage for every accessor + `SearchMovies` / `SearchActors` (title, genre, year, actor name, character name, profession, linked-movie title, empty/whitespace, no-match), all five load-error paths (missing dir, malformed JSON, inconsistent canonical id, duplicate movie id, movie without rating), plus a smoke test that loads the real `src/data/` and asserts 100 movies / 553 actors.
- [src/internal/config/config_test.go](../src/internal/config/config_test.go) — defaults, env-only, env-then-flag override, `--help` returns `flag.ErrHelp` and prints expected sections, invalid env-port, validation failures (log-level, port-low, port-high, empty data-dir, positional arg, unknown flag), and `Redacted()`.
- Fixtures under [src/internal/store/testdata/](../src/internal/store/testdata): `good/`, `no_rating/`, `dup_movie/`, `bad_id/`, `bad_json/`.

## Wiring

- [src/cmd/movies-api/main.go](../src/cmd/movies-api/main.go) — replaced session-1 synthetic readiness with a real background `store.Load(cfg.DataDir)`. `/readyz` stays 503 until load succeeds; counts are logged at `info`.

## Verification

- `make test` ⇒ green, `-race` clean.
- `go test -coverprofile=/tmp/cover.out ./...`:
  - `internal/store`  **94.0 %** (≥80 % required)
  - `internal/config` **100.0 %** (≥80 % required)
  - `internal/httpapi` **100.0 %** (unchanged from session 1)
- `go vet ./...` clean. `go build ./...` clean.

## Out of scope (deferred, all per the frame)

- `/api/movies`, `/api/movies/{id}`, `/api/actors`, `/api/actors/{id}`, `/api/genres` — session 3.
- HTTP-layer query validation (`q` length 2–20, page sizes, id regex, year/rating bounds) — session 3.
- Pagination — session 3.
- Prometheus metrics, OpenAPI/Swagger, NetworkPolicy, ServiceMonitor.
