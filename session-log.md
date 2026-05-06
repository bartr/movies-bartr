# Session Log

> One entry per session. Frame before, ritual after. The log itself is the experiment evidence.
>
> Methodology: [METHODOLOGY.md](METHODOLOGY.md) · Experiment: [EXPERIMENT.md](EXPERIMENT.md) · Spec: [spec.md](spec.md)

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

---

<!-- Copy the Session Template block above for each new session. -->
