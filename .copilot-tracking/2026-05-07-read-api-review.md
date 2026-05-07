# Read API + Validation — Review (2026-05-07)

## What landed vs. frame

| Frame item                                                  | Status |
|-------------------------------------------------------------|--------|
| `/api/movies` happy path + filters                           | ✅     |
| `/api/movies/{id}` happy + 400 + 404                         | ✅     |
| `/api/actors` happy path + `q`                               | ✅     |
| `/api/actors/{id}` happy + 400 + 404                         | ✅     |
| `/api/genres`                                                | ✅     |
| RFC 7807 `application/problem+json`                          | ✅     |
| One negative test per validation rule (mirrors test.json)    | ✅     |
| `internal/httpapi` coverage ≥ 80 %                            | ✅ 91.2 % |
| Spec §6 updated to match enforced rules                      | ✅     |
| In-cluster verify on 0.3.0                                   | ✅     |

## Out-of-scope items kept out

OpenAPI/Swagger UI, Prometheus metrics, ServiceMonitor, NetworkPolicy,
Grafana dashboards, Web Validate runner — none touched.

## Notable decisions

1. **`StoreFunc` accessor instead of swapping handlers.** Router built once
   in main; an `atomic.Pointer[store.Store]` is read on each request.
   `/api/*` returns 503 problem+json while the dataset is loading. Smaller
   diff than wiring a router-swap.

2. **Frozen year window `[1874, 2025]`.** Not derived from build time so
   tests stay deterministic across machines and over time. Bumping the
   upper bound is an explicit annual change, not a side-effect of the
   build.

3. **Rating filter compares floor buckets.** Reuses the existing
   `MoviesByRatingBucket` index. `test.json` only asserts `rating=8.5`
   returns 200; behavior is consistent with that contract.

4. **Path id rule is `regex && not-all-zero`.** This is what `test.json`
   actually enforces (`tt12345` 404s, `tt00000` 400s). Spec §6 was
   updated to record the second clause.

5. **List endpoints return bare arrays.** Spec is silent on shape; the
   `test.json` cases only check status + content type. Pagination
   metadata can be added when OpenAPI lands without breaking the
   negative-test surface.

6. **Data baked into the image now.** `COPY data /data` was the deferred
   step from session 2's "don't bake data into an image stage other than
   the one defined when session 2 lands" — it lands in session 3
   because /readyz needs the dataset.

## Risks / followups

- `firstQueryParam` silently picks the first value when a key is repeated.
  Acceptable for this surface; revisit if OpenAPI requires a stricter
  contract.
- The 503 problem path is covered by tests but not by the in-cluster
  verify. Acceptable because /readyz already covers the timing window.

## Health signal candidates (filled into session-log on close)

- Framing quality: bundle of sessions 3+4 fit comfortably; no scope
  reduction needed at fit check.
- Drift: none. Image-tag bump + data bake were necessary side-effects of
  shipping the slice end-to-end.
