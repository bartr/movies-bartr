# Read API + Validation — Research (2026-05-07)

Source-of-truth signals: spec.md §6, `test.json`, and the fixtures under
`src/data/`.

## Routes

| Method | Path                  | Behavior                                                  |
|--------|-----------------------|-----------------------------------------------------------|
| GET    | `/api/movies`         | Filter by `q`, `genre`, `year`, `rating`, `actorId`; page |
| GET    | `/api/movies/{id}`    | 400 on bad id, 404 on miss, else movie object             |
| GET    | `/api/actors`         | Filter by `q`; page                                       |
| GET    | `/api/actors/{id}`    | 400 on bad id, 404 on miss, else actor object             |
| GET    | `/api/genres`         | Sorted distinct genre strings                             |

Response `Content-Type: application/json; charset=utf-8`. List endpoints
return a bare JSON array (the page slice). No pagination envelope this
session — the contract in `test.json` does not require one and the spec
does not specify response shape. We ship the simpler shape and revisit
when OpenAPI lands.

## Validation rules (derived from test.json)

| Param        | Rule                                                                                       | Default |
|--------------|--------------------------------------------------------------------------------------------|---------|
| `pageNumber` | integer, `[1, 10000]`                                                                      | `1`     |
| `pageSize`   | integer, `[1, 1000]`                                                                       | `25`    |
| `q`          | length `[2, 20]` when present                                                              | unset   |
| `genre`      | length `[3, 20]` when present                                                              | unset   |
| `year`       | integer, `[1874, 2025]`                                                                    | unset   |
| `rating`     | float, `[0.0, 10.0]`                                                                       | unset   |
| `actorId`    | `^nm\d{5,9}$` AND digits not all zero                                                      | unset   |
| path movieId | `^tt\d{5,9}$` AND digits not all zero                                                      | n/a     |
| path actorId | `^nm\d{5,9}$` AND digits not all zero                                                      | n/a     |

The "digits not all zero" rule is forced by `test.json`: `tt12345` and
`nm12345` are both 5-digit ids that pass validation (they 404), while
`tt00000` and `nm00000` (also 5 digits) return 400. That rules out
"min 7 digits" and rules in the all-zero exclusion. The spec's draft
range `\d{5,9}` is preserved; we update §6 to record the not-all-zero
rule explicitly.

`year` bounds: data range is 1999–2009, but `test.json` accepts `1999`
and rejects `1873` and `2026`. The current real-world year is 2026 and
the cinema era starts 1874. We freeze the constants `[1874, 2025]` —
not "currentYear-1" — because year validation must be deterministic
across builds; the upper bound bumps as part of an annual housekeeping
PR if needed.

`rating` bounds: rating values in data are 8.0–8.9. `test.json` rejects
`-1` and `10.1`. IMDb scale is `[0.0, 10.0]`, so we use that.

## RFC 7807 shape

```json
{
  "type":     "about:blank",
  "title":    "Bad Request",
  "status":   400,
  "detail":   "<rule-specific message>",
  "instance": "/api/movies?q=a"
}
```

`Content-Type: application/problem+json` (per `test.json`). 404 responses
also use `application/problem+json` for consistency.

## Filter composition (`/api/movies`)

All present filters AND together. Order chosen for cheapest-pruning first:

1. `actorId` → `MoviesByActor` (set membership)
2. `genre`   → `MoviesByGenre`
3. `year`    → equality on Movie.Year
4. `rating`  → equality on `floor(Movie.Rating)` matches `floor(param)`
   - We expose this as floor-bucket filter to match the existing store
     index. Test only asserts `rating=8.5` returns 200 with a non-empty
     body — happy path is preserved.
5. `q`       → substring over the per-movie haystack
6. Sort by id, slice by page.

For `/api/actors` only `q` filters; sort by id; page.

## Coverage target

Existing `internal/httpapi` coverage was 100 % of three trivial
handlers. Session goal: keep ≥ 80 % on the package after adding the
data API + validators.
