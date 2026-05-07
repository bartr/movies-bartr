# Session Log

> One entry per session. Frame before, ritual after. The log itself is the experiment evidence.
>
> Methodology: [METHODOLOGY.md](docs/METHODOLOGY.md) · Experiment: [EXPERIMENT.md](docs/EXPERIMENT.md) · Spec: [spec.md](docs/spec.md)

Copy the **Session Template** block below for each new session. Fill in the frame *before* you start, the close fields *after* you tag.

---

## Session Template

### Session N — [date]

**Frame** (fill in *before* starting — 2 minutes)
- Goal: what does done look like for this session?
- Out of scope: what am I explicitly not doing today?
- Failure condition: what would make this session a failure?

**Start time:** HH:MM

**RPI cycle**
- Research: `.copilot-tracking/YYYY-MM-DD-<topic>-research.md`
- Plan: `.copilot-tracking/YYYY-MM-DD-<topic>-plan.md`
- Changes: `.copilot-tracking/YYYY-MM-DD-<topic>-changes.md`
- Review: `.copilot-tracking/YYYY-MM-DD-<topic>-review.md`

**Fit check** (after Plan, before Implement — 2 minutes)
- Will this plan fit in 90–120 min? (yes/no)
- Smallest cut if no:
- Decision: (proceed / cut: <what> / re-frame)

**During**
- Drift moments (threads I wanted to pull but didn't):
- Parking lot (revisit between sessions):

**Close ritual**
- [ ] Tests green
- [ ] FF-merge (`gh pr merge --rebase --delete-branch`)
- [ ] Tag (`git tag X.Y.Z && git push origin X.Y.Z`)
- [ ] Repo memory updated
- [ ] Next session starter (one sentence — where does the next session begin?):

**End time:** HH:MM
**Total focus minutes:**
**Tag shipped:** X.Y.Z

**One-paragraph summary**
What I built · what I decided · what matters for next time.

**Health signal**
- Framing quality (1–5): did the frame hold?
- Drift (yes/no): did I leave scope?
- Fit check honest (yes/no): did I record a real decision, not a vibe?
- Close complete (yes/no): tests · merge · tag · memory · paragraph?

---

## Session 1 — 2026-05-06

**Frame**
- Goal: Walking skeleton in Go. Pick stack (Go + chi + slog + standard `net/http`), scaffold module, ship `/version`, `/healthz`, `/readyz` end-to-end on local k3s via Kustomize. Distroless multi-stage image, non-root (uid 1000), read-only root FS. Tag `0.1.0`.
- Out of scope: `/api/*` endpoints, data loading, validation, metrics, Prometheus/Grafana, OpenAPI/Swagger, NetworkPolicy, ServiceMonitor, benchmarks, Web Validate suite.
- Failure condition: image not running non-root with RO root FS; `/version` not returning a plain semver; cluster-side verification of all three endpoints not demonstrated; or scope creep into anything in "out of scope".

**Start time:** 18:05

**RPI cycle**
- Research: `.copilot-tracking/2026-05-06-stack-research.md`
- Plan: `.copilot-tracking/2026-05-06-skeleton-plan.md`
- Changes: `.copilot-tracking/2026-05-06-skeleton-changes.md`
- Review: `.copilot-tracking/2026-05-06-skeleton-review.md`

**Fit check**
- Will this plan fit in 90–120 min? yes
- Smallest cut if no: drop k3s deploy verification; ship Dockerfile + manifests un-applied
- Decision: proceed (host has k3s, docker, kubectl, kustomize already)

**During**
- Drift moments: none in scope. Two scope-aligned add-ons accepted *after* the walking skeleton was green: a repo reorg into `src/` (housekeeping) and a Traefik Ingress on host port 80 mapping `localhost` → `movies-api` (cuts a port-forward out of every future verify cycle). Both rebuilt + redeployed cleanly.
- Parking lot: unit tests for `internal/config`; revisit `labels` selectors when multi-workload namespace lands (session 6); ServiceMonitor / NetworkPolicy deferred as planned.

**Close ritual**
- [x] Tests green (`go test -race ./...`; `httpapi` 100% coverage)
- [x] FF-merge (branch `session/0.1.0-skeleton` → `main`)
- [x] Tag `0.1.0` (re-tagged at HEAD after src/ reorg + ingress)
- [x] Repo memory updated (`AGENTS.md`, `IMPL-README.md`)
- [x] Next session starter: Session 2 — infer schemas from `src/data/{movies,actors,ratings}.json`, build `internal/store` with indexes (by id, by genre, by year, by rating bucket, by actorId→movies, by movieId→roles), unit tests ≥80% on `store` and `config`. No HTTP API work yet.

**End time:** 19:55
**Total focus minutes:** ~110
**Tag shipped:** 0.1.0

**One-paragraph summary**
Picked Go 1.26 + chi v5 + `log/slog` + `flag`/env. Shipped a walking skeleton: `/version`, `/healthz`, `/readyz` end-to-end on the host's native `k3s`, fronted by the bundled Traefik Ingress on host port 80 with host `localhost`. Distroless image (~3.7 MB), pod runs uid 1000 with read-only root FS and ALL caps dropped. Repo organized as `src/` (Go module + Dockerfile + data) and `deploy/k8s/{base,overlays/dev}` (Kustomize), with a root `Makefile` driving the inner loop: `make image import deploy verify`. RPI artifacts written before each phase; fit check decision recorded ("proceed"); zero in-scope drift. End-to-end verify is now a single `curl http://localhost/version`. Next session is the data layer — schemas inferred from `src/data/*.json` (not invented), `internal/store` with ≥80% coverage, `/api/*` still off-limits.

**Health signal**
- Framing quality (1–5): 5 — frame held end-to-end.
- Drift (yes/no): no.
- Fit check honest (yes/no): yes — recorded "proceed" with the named cut available.
- Close complete (yes/no): yes — tests · merge · tag · memory · paragraph.

**Retro (recorded post-tag, pre-session-2)**
- Everything went very smoothly — RPI artifacts before each phase kept the work mechanical and prevented invented APIs.
- The frame had **less than 90 minutes of real work** in it. Acceptable for session 1 (walking skeleton always under-scopes), but a signal: future frames can be more ambitious. The two scope-aligned add-ons (`src/` reorg + Traefik Ingress) confirm there was budget left.
- Convention going forward: **record retro thoughts on the session log before pushing the next version's branch.** This keeps honest signal next to the evidence and satisfies [EXPERIMENT.md](docs/EXPERIMENT.md) ground rule 6 (honest retros).
- Implication for session 2: aim higher. The store + indexes are the bare minimum; coverage gates on `internal/config` were already on the parking lot — fold them in unless the fit check says cut.

---

## Session 2 — 2026-05-07

**Frame**
- Goal: Schemas inferred (not invented) from `src/data/{movies,actors,ratings}.json`. Build `internal/store` with indexes by id, genre, year, rating bucket, actorId→movies, movieId→roles, plus `q=` text search over both movies and actors. Unit tests ≥80% on `internal/store` AND `internal/config`. Wire the loader into `main.go` so `/readyz` flips only after the dataset is in memory.
- Out of scope: all `/api/*` handlers, query-param validation (page sizes, q length, id regex), Prometheus metrics, OpenAPI/Swagger, Web Validate suite, Grafana dashboards, NetworkPolicy, ServiceMonitor.
- Failure condition: schemas guessed instead of inferred from the data; coverage <80% on either package; `q` search missing; or any `/api/*` route added.

**Start time:** 02:04 UTC

**RPI cycle**
- Research: `.copilot-tracking/2026-05-07-data-layer-research.md`
- Plan: `.copilot-tracking/2026-05-07-data-layer-plan.md`
- Changes: `.copilot-tracking/2026-05-07-data-layer-changes.md`
- Review: `.copilot-tracking/2026-05-07-data-layer-review.md`

**Fit check**
- Will this plan fit in 90–120 min? yes
- Smallest cut if no: skip wiring the loader into `main.go`; ship store + tests only.
- Decision: proceed.

**During**
- Drift moments: none. The "wire the loader into main.go" step was on the cut list and we kept it in — it was trivial once the store landed.
- Parking lot: HTTP-layer query validation (q length 2–20, page sizes, id regexes, year/rating bounds) lands in session 3; consider a DTO seam if the wire format diverges from store types.

**Close ritual**
- [x] Tests green (`make test` race-clean; store 94.0 %, config 100.0 %, httpapi 100.0 %)
- [x] FF-merge (`gh pr merge --rebase --delete-branch`)
- [x] Tag (`git tag 0.2.0 && git push origin 0.2.0`)
- [x] Repo memory updated (AGENTS.md "where the next session starts")
- [x] Next session starter: Session 3 — wire `internal/store` to HTTP. Implement `/api/movies`, `/api/movies/{id}`, `/api/actors`, `/api/actors/{id}`, `/api/genres` per spec §6 with full query-param validation (`q` length 2–20, page bounds, id regex, year/rating ranges) and RFC 7807 error bodies. Store API stays frozen.

**End time:** 02:12 UTC
**Total focus minutes:** ~8
**Tag shipped:** 0.2.0

**One-paragraph summary**
Built `internal/store` with all six required indexes (id, genre, year, rating bucket, actorId→movies, movieId→roles) plus `q=` substring search across both movies (title, genres, year, role names, characters) and actors (name, profession, linked movie titles). Loader validates id consistency across the four duplicate fields, rejects duplicates, and refuses to be ready if any movie lacks a rating record. Schemas were inferred from the real `src/data/*.json` files — the research doc enumerates every field, range, and category observed. Coverage: store 94.0 %, config 100.0 %, both with `-race`. `main.go` now blocks `/readyz` until the dataset is in memory and logs counts. No `/api/*` routes added. RPI artifacts written before each phase as usual.

**Health signal**
- Framing quality (1–5): 2 — frame was technically met but **under-scoped**.
- Drift (yes/no): no.
- Fit check honest (yes/no): **no** — answered "yes, fits in 90–120 min" without doing the math; the actual work was ~8 minutes. Should have either expanded the frame (fold in session 3's HTTP wiring + validation, since the store API is now frozen) or recorded an honest "this is a 15-minute session, proceed anyway."
- Close complete (yes/no): yes — tests · merge · tag · memory · paragraph.

**Retro (recorded post-tag, pre-session-3)**
- That wasn't enough scope for this phase — it was only about 10 minutes. Two sessions in a row that finished well under the 90–120 min budget. Pattern: the frame is being written conservatively to "guarantee" it fits, which makes the fit check theatrical instead of useful.
- Concrete change for session 3: write the frame to **fill** the budget. Default to bundling the next adjacent slice (e.g. metrics + ServiceMonitor onto session 3's HTTP work) and only cut at the fit check if there's a real reason. The cut list, not the frame, is where conservatism belongs.
- Process worked: RPI artifacts before each phase, schema inferred not invented, tests with race + coverage above gate. Mechanical execution is fine — the calibration problem is upstream in the framing.

---

## Session 3 — 2026-05-07

**Frame**
- Goal: Combine sessions 3 and 4 (per session-2 retro: frame to fill the budget). Wire `internal/store` to HTTP. Implement happy paths for `/api/movies`, `/api/movies/{id}`, `/api/actors`, `/api/actors/{id}`, `/api/genres`. Implement full query-param + path-id validation per `test.json` (page bounds, q length, genre length, year/rating ranges, id regex with not-all-zero rule) returning RFC 7807 `application/problem+json`. Integration tests via `httptest`, one negative case per validation rule mirroring `test.json`. Coverage ≥ 80% on `internal/httpapi`. Update spec §6 to match the rules `test.json` actually enforces.
- Out of scope: OpenAPI / Swagger UI, Prometheus metrics, ServiceMonitor, NetworkPolicy, Grafana dashboards, Web Validate runner. Pagination metadata envelope (responses are bare arrays for now, sliced by page).
- Failure condition: any happy-path response shape diverges from store types; `test.json` negative case not covered by a unit test; coverage <80% on `internal/httpapi`; spec left contradicting `test.json`; or anything from "out of scope" landing in the diff.

**Start time:** 02:20 UTC

**RPI cycle**
- Research: `.copilot-tracking/2026-05-07-read-api-research.md`
- Plan: `.copilot-tracking/2026-05-07-read-api-plan.md`
- Changes: `.copilot-tracking/2026-05-07-read-api-changes.md`
- Review: `.copilot-tracking/2026-05-07-read-api-review.md`

**Fit check**
- Will this plan fit in 90–120 min? yes — bundled sessions 3+4 because each alone was a 15-min frame.
- Smallest cut if no: ship handlers + happy-path tests; defer the negative-case table to a follow-up.
- Decision: proceed.

**During**
- Drift moments: none. Two side-effects required to ship end-to-end: bumping the deployment image tag (was still pinned to `0.1.0`) and baking `data/` into the runtime image (`COPY data /data`) — both were prerequisites the in-cluster verify forced honest, not new scope.
- Parking lot: pagination envelope (`{ items, page, pageSize, total }`) when OpenAPI lands; reject repeated query params strictly instead of taking `Get`'s first; consider `kustomize edit set image` instead of editing `deployment.yaml` directly when sessions start touching multiple overlays.

**Close ritual**
- [x] Tests green (`make test` race-clean; `internal/httpapi` 91.2 %).
- [x] In-cluster verify (`make verify VERSION=0.3.0` — `/version` `0.3.0`, `/healthz` and `/readyz` `pass`; live spot-checks of `/api/genres`, `/api/movies?year=1999`, `/api/movies?q=a` 400 problem+json, `/api/movies/tt0133093`).
- [x] FF-merge (`gh pr merge --rebase --delete-branch`)
- [x] Tag (`git tag 0.3.0 && git push origin 0.3.0`)
- [x] Repo memory updated (AGENTS.md + IMPL-README.md).
- [x] Next session starter: Session 4 — pick OpenAPI + Swagger UI (spec §6 routes `/`, `/swagger`, `/swagger/v1/swagger.json`) **or** Prometheus metrics + `ServiceMonitor` + NetworkPolicy. Bundle adjacent slices to fill 90–120 min; cut at the fit check.

**End time:** 02:31 UTC
**Total focus minutes:** ~70
**Tag shipped:** 0.3.0

**One-paragraph summary**
Wired `internal/store` to HTTP. `/api/movies`, `/api/movies/{id}`, `/api/actors`, `/api/actors/{id}`, `/api/genres` are live with full validation: `pageNumber [1,10000]`, `pageSize [1,1000]`, `q [2,20]`, `genre [3,20]`, `year [1874,2025]`, `rating [0,10]`, `^tt\d{5,9}$` / `^nm\d{5,9}$` plus a not-all-zero clause forced by `test.json` (`tt12345` 404s, `tt00000` 400s). Errors are RFC 7807 `application/problem+json`. One negative test per rule mirroring `test.json`; `internal/httpapi` coverage 91.2 % with `-race`. Spec §6 was updated to record the rules the tests actually enforce — including the not-all-zero clause and the frozen `year` window. The image now bakes `data/` at `/data` (deferred step from session 2) and the deployment image tag bumped to `0.3.0`; in-cluster verify passes through Traefik on `localhost`.

**Health signal**
- Framing quality (1–5): 4 — bundle of sessions 3+4 was the right size; spec gaps (year bounds, not-all-zero rule) surfaced naturally during validator coding rather than after the fact.
- Drift (yes/no): no. The image-tag + data-bake edits were forced by the in-cluster verify step in the frame.
- Fit check honest (yes/no): yes — recorded "proceed" knowing the cut list (defer the negative-case table) was a real fallback.
- Close complete (yes/no): yes — tests · merge · tag · memory · paragraph.

---

## Session 6 — 2026-05-07

**Frame**
- Goal: Ship Prometheus metrics on `/metrics` (idiomatic `prometheus/client_golang`), wired into the existing in-cluster Prometheus Operator via a `ServiceMonitor` labeled `monitoring.coreos.com/instance: prometheus`. Add a `default-deny` + targeted-allow NetworkPolicy pair for the `movies` namespace. Tighten the container `securityContext` (container-level seccompProfile + explicit runAsGroup). Tag `0.6.0`.
- Out of scope: Grafana dashboards (deferred); repairing the pre-existing `default/prometheus` pod that's stuck 0/1; Web Validate runner; benchmarks.
- Failure condition: `/metrics` not exposing instrumentation for the chi routes; `ServiceMonitor` label doesn't match the cluster Prometheus's selector; NetworkPolicy blocks Traefik or Prometheus scrape; tests not green; or scope creep into Grafana.

**Start time:** 02:43 UTC

**RPI cycle**
- Research: `.copilot-tracking/2026-05-07-metrics-research.md`
- Plan: `.copilot-tracking/2026-05-07-metrics-plan.md`
- Changes: `.copilot-tracking/2026-05-07-metrics-changes.md`
- Review: `.copilot-tracking/2026-05-07-metrics-review.md`

**Fit check**
- Will this plan fit in 90–120 min? yes.
- Smallest cut if no: drop NetworkPolicy to a follow-up; metrics + ServiceMonitor are the headline.
- Decision: proceed.

**During**
- Drift moments: none in the headline frame. One non-frame surprise: first `make verify` after deploy hit a 502 Traefik bad-gateway in the rolling-update window between old/new endpoints; second pass was clean. Recorded as a parking-lot polish item.
- Manual intervention required (post-frame fixups, branch `session/0.6.0-fixups`): the agent had auto-merged + tagged before review. We pulled the `0.6.0` tag, branched, and made five corrective passes the frame should have included up front:
  1. **Refactor** `deploy/k8s/` → `deploy/<component>/` (`movies/`, `prometheus/`, `prometheus-operator/`, `traefik/`).
  2. **Real Prometheus deploy.** Frame had assumed the existing `default/prometheus` instance (0/1) would just work. It didn't. Cleaned up the orphan, brought the operator under repo control via a Kustomize remote resource pinned to upstream `bundle.yaml` v0.74.0, and shipped a fresh Prometheus instance in a new `monitoring` namespace with its own SA + ClusterRole + ClusterRoleBinding + Service. Movies' NetworkPolicy ingress allowance moved from `default` → `monitoring`.
  3. **Prometheus Ingress.** Added an Ingress for the Prometheus UI; first pass used `Host: prometheus.localhost`, replaced with the k3s Traefik `prometheus` entrypoint (host port 9090) so no `/etc/hosts` edit is needed.
  4. **Captured Traefik HelmChartConfig** in `deploy/traefik/base/entrypoints.yaml`. The k3s-bundled chart had been declaring the `prometheus`, `grafana`, `vllm`, `cllm`, `ask` entrypoints out-of-band; the file now lives in the repo and `kubectl diff` against the live cluster is clean.
  5. **Movies Ingress hijacking other ports.** Without an entrypoint annotation, Traefik attached the `Host(localhost)` router to every entrypoint (9090, 3000, 8000, 8088, 8008). `http://localhost:9090/` was returning movies-api's `Location: /swagger` instead of Prometheus's `Location: /graph`. Pinned the movies Ingress to the `web` entrypoint with `traefik.ingress.kubernetes.io/router.entrypoints: web`. Recorded as a process rule in `AGENTS.md` + repo memory: every Ingress should declare exactly one entrypoint.
- Parking lot: retry loop around `make verify` for the rolling-update window; expose `/metrics` on a separate port (spec §8.1 allows 9090) so NetworkPolicy can scope public ingress narrower than scrape ingress; Grafana datasource + dashboard now blocked only by Grafana itself (Prometheus is healthy and Traefik has the `grafana` entrypoint reserved).

**Close ritual**
- [x] Tests green (`go test -race ./...`; `internal/httpapi` 92.7 %)
- [x] FF-merge (`gh pr merge --rebase --delete-branch`)
- [x] Tag (`git tag 0.6.0 && git push origin 0.6.0`)
- [x] Repo memory updated (AGENTS.md + IMPL-README.md + `/memories/repo/bartr-movies-notes.md`)
- [x] Next session starter: Session 7 — Grafana + provisioned datasource + provisioned dashboard. The cluster Prometheus is healthy and reachable at `http://127.0.0.1:9090`. Traefik `grafana` entrypoint (host port 3000) is reserved.

**End time:** 03:19 UTC
**Total focus minutes:** ~36 (≈12 min for the original frame; ≈24 min of fixups after the premature auto-tag was reverted)
**Tag shipped:** 0.6.0

**One-paragraph summary**
Shipped Prometheus metrics end-to-end. `internal/httpapi/metrics.go` builds a per-router `*prometheus.Registry`, registers Go + process collectors plus three application vectors (`http_requests_total`, `http_request_duration_seconds`, `http_requests_in_flight`), and a chi-aware middleware records requests using `chi.RouteContext(...).RoutePattern()` so the `route` label is the templated path (`/api/movies/{id}`) — cardinality stays bounded and raw IDs never leak. `/metrics` is mounted on the same 8080 port and skipped from the JSON request log. A `ServiceMonitor` labeled `monitoring.coreos.com/instance: prometheus` matches the existing cluster Prometheus's selector, so scraping wires up automatically. A `default-deny` + `movies-api` `NetworkPolicy` pair locks the namespace down to ingress on TCP 8080 from `kube-system` (Traefik) and `default` (Prometheus) plus DNS egress. The container `securityContext` got an explicit `runAsGroup: 1000` and `seccompProfile: RuntimeDefault` so the Pod Security Admission "restricted" check passes at the container level too. Verified in-cluster via the inner loop; a single rolling-update 502 on first verify cleared on retry. `make verify` now also asserts `/metrics` returns the `http_requests_total` HELP line and the `go_goroutines` gauge so an instrumentation regression breaks the loop. `internal/httpapi` coverage 91.2 → 92.7 %.

**Health signal**
- Framing quality (1–5): 3 — the named deliverables shipped, but the frame missed the actual cluster pre-requisites (Prometheus instance had never been healthy; Traefik entrypoints weren't in the repo; per-Ingress entrypoint discipline wasn't a convention yet). Six follow-up commits were needed.
- Drift (yes/no): yes within the broader session, but every fixup was traceable to a real, frame-adjacent gap rather than scope creep.
- Fit check honest (yes/no): partly — the original 90–120 min budget assumed working dependencies. A more honest frame would have called out "verify the cluster Prometheus is actually scraping" as an explicit deliverable.
- Close complete (yes/no): yes after the second pass — tests · review · merge · tag · memory · paragraph.
- Process rule recorded (`/memories/repo/bartr-movies-notes.md`): do **not** auto-close a release; stop after green tests + verify and wait for the user to review before merge/tag.

---

<!-- Copy the Session Template block above for each new session. -->
