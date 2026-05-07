# Read API + Validation — Plan (2026-05-07)

## Files

- `src/internal/httpapi/problem.go` — RFC 7807 helper: `writeProblem(w, r, status, detail)`.
- `src/internal/httpapi/validate.go` — pure validators returning `(value, *problem)`. Constants for bounds/regex.
- `src/internal/httpapi/api.go` — five handlers; receive `*store.Store`.
- `src/internal/httpapi/router.go` — accept `*store.Store`, mount `/api/*`.
- `src/internal/httpapi/api_test.go` — happy paths + one negative per rule (mirrors test.json).
- `src/cmd/movies-api/main.go` — pass loaded store into `NewRouter`. Router built only after dataset load (use a swap or a closure).
- `docs/spec.md` — pin year/rating/genre/q/page rules + not-all-zero id rule.

## Approach

- Introduce `chi.URLParam` for path ids.
- Validators return a `*problem` (nil = ok). One negative test per rule.
- Router signature: `NewRouter(version string, ready ReadyFunc, s *store.Store) http.Handler`. `s` may be nil; `/api/*` returns 503 problem when nil (covers the pre-load window). Simpler: build router twice — first without store (for /version, /healthz, /readyz), and after load swap an `atomic.Pointer[http.Handler]`. **Picked simpler:** main builds router once, passing a `func() *store.Store` accessor; handlers call it and 503-problem when nil.

## Validation logic

- Single helper `requireID(prefix, id) *problem` — checks regex + not-all-zero.
- `parseIntInRange(name, raw string, lo, hi int) (int, *problem)` — strict integer parse (no decimals, no `foo`).
- `parseFloatInRange(name, raw string, lo, hi float64) (float64, *problem)` — strict float parse.
- `parseLenString(name, raw string, lo, hi int) (string, *problem)`.

## Tests

- Build a small in-memory `*store.Store` from the `internal/store/testdata/good` fixtures (already loadable).
- Table-driven negative tests, one row per rule from test.json.
- Happy-path tests: list, by-id, genres, q, filters.

## Out of scope (frame)

- OpenAPI/Swagger, metrics, ServiceMonitor, NetworkPolicy, dashboards, Web Validate runner.
