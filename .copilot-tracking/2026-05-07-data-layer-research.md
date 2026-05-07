# Session 2 — Data layer research (2026-05-07)

## Inputs

- Spec: [spec.md §5, §6](../docs/spec.md)
- Data files (live in [src/data/](../src/data)):
  - `movies.json`  — 100 records
  - `actors.json`  — 553 records
  - `ratings.json` — 100 records (1:1 with movies)

## Inferred schema

### `movies.json` — `[]Movie`
| Field    | Type        | Notes |
|----------|-------------|-------|
| id, key, _id, movieId | string `tt\d{7,9}` | All four are duplicates of the IMDb tt-id. Treat `id` (or `movieId`) as canonical. |
| type     | string `"movie"` | Constant. |
| title    | string      | |
| year     | int         | Observed range 1999–2009. |
| runtime  | int         | Minutes. |
| genres   | []string    | 20 distinct values: Action, Adventure, Animation, Biography, Comedy, Crime, Documentary, Drama, Family, Fantasy, History, Horror, Music, Musical, Mystery, Romance, Sci-Fi, Sport, Thriller, War. |
| roles    | []Role      | See below. |

### `Role` (inside Movie.roles)
| Field      | Type     | Notes |
|------------|----------|-------|
| order      | int      | Display order within the movie's cast/crew. |
| actorId    | string `nm\d{7,9}` | |
| name       | string   | Person's name (denormalized — also in actors.json). |
| category   | string   | Observed values: actor, actress, director, producer, self. |
| characters | []string | Optional — only when category is actor/actress/self (character names). |
| job        | string   | Optional — accompanies category=producer (e.g. "producer"). |

### `actors.json` — `[]Actor`
| Field      | Type     | Notes |
|------------|----------|-------|
| id, key, _id, actorId | string `nm\d{7,9}` | Canonical `id`. |
| type       | string `"actor"` | Constant. |
| name       | string   | |
| birthYear  | int      | |
| deathYear  | int      | `0` is sentinel for "living / unknown". |
| profession | []string | 30+ distinct values. Includes empty string `""` in the data. |
| movies     | []{ movieId, title } | Denormalized cross-references. |

### `ratings.json` — `[]Rating`
| Field      | Type     | Notes |
|------------|----------|-------|
| id, key, _id, movieId | string `tt…` | Canonical `movieId`. |
| type       | string `"Rating"` | Constant. |
| rating     | float    | Observed range 8.0–8.9. |
| votes      | int      | |

Set difference movies↔ratings: empty (every movie has a rating).

## Index / search design

Spec §6 query params on `/api/movies`: `q, genre, year, rating, actorId`. On `/api/actors`: `q`. The store therefore needs:

- `byID` for both entities (O(1) lookup).
- `byGenre`, `byYear`, `byRatingBucket`, `byActor` for movies (slice-of-movieID per key).
- `byMovie` for roles (movieId → []Role; lives on Movie itself but exposed as a method for symmetry with `actorId→movies`).

Rating bucket: integer floor of the rating (`int(rating)`). With the current data the only bucket is `8`, but the index is general.

Text search `q`:
- **Movies** indexed haystack (lowercased): title, each genre, year-as-string, each role's actor name, each role's characters.
- **Actors** indexed haystack (lowercased): name, each profession, each linked movie title.
- Match: substring contains. (Tokens-only match would miss multi-word titles; substring is simpler and adequate at this dataset size.)

## Decisions

- Canonical id: the `id` field. `key` / `_id` / `movieId` / `actorId` are ignored at load time after a sanity check that they all match.
- `Movie.Rating` and `Movie.Votes` are merged in from `ratings.json` at load. A movie with no rating record is a load error (spec §5.3 — dataset must be coherent for `/readyz`).
- Roles are kept as-is on `Movie`; no separate `roles` map duplicating data.
- Iteration order is deterministic: every result slice is sorted by id.
- Search is case-insensitive substring on a precomputed lowercased haystack string per record.
- HTTP-layer query validation (page sizes, q length 2–20, etc.) is **session 3**. Store accepts any `q`.
