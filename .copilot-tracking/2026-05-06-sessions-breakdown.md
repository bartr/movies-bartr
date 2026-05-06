# Sessions Breakdown — Movies (Go)

> First-pass frames for sessions 1–10. Each session re-frames + fit-checks at
> its start; cuts and re-frames are logged honestly in
> [session-log.md](../session-log.md). This file is the *initial* plan, not the
> contract.
>
> Spec: [spec.md](../spec.md) · Methodology: [METHODOLOGY.md](../METHODOLOGY.md) · Experiment: [EXPERIMENT.md](../EXPERIMENT.md)

## Stack (decided session 1)

Go 1.26 · chi v5 · `log/slog` · `flag`+env (`MOVIES_*`) · distroless static ·
Kustomize on native k3s · `docker save` → `k3s ctr images import` (no registry).

## Sessions

| # | Tag | Frame goal | Out of scope |
|---|-----|------------|--------------|
| 1 | `0.1.0` | Walking skeleton: stack chosen, `/version` + `/healthz` + `/readyz` live on local k3s via Kustomize, distroless image, non-root, RO root FS | API, data load, metrics, dashboards |
| 2 | `0.2.0` | Data layer: infer schemas from `data/*.json`, in-memory store + indexes (id, genre, year, rating bucket, actorId→movies, movieId→roles); unit tests ≥80% on store + config | HTTP API, validation, metrics |
| 3 | `0.3.0` | Read API happy paths: `/api/movies`, `/api/movies/{id}`, `/api/actors`, `/api/actors/{id}`, `/api/genres`; integration tests | Validation 400s, OpenAPI, metrics |
| 4 | `0.4.0` | Validation: page/q/year/rating/id rules + `application/problem+json` (RFC 7807); one negative test per rule (mirroring `test.json`) | Metrics, dashboards |
| 5 | `0.5.0` | OpenAPI 3 doc, `/swagger`, `/swagger/v1/swagger.json`, `/` → `/swagger`; structured JSON log review; `/robots.txt` | Metrics, dashboards |
| 6 | `0.6.0` | Prometheus metrics + Prometheus Operator + `ServiceMonitor` + NetworkPolicy + tighter securityContext review | Grafana |
| 7 | `0.7.0` | Grafana deploy, provisioned Prometheus datasource, provisioned dashboard JSON showing live data; anonymous viewer for dev | Soak/bench |
| 8 | `0.8.0` | Web Validate-compatible suite from `test.json`; documented §12 inner-loop in implementation README | Bench tuning |
| 9 | `0.9.0` | Benchmarks: p95 `/api/movies` < 50 ms, p95 `/api/movies/{id}` < 10 ms, sustained 500 RPS on 500m pod with <1% error rate; close §14 gaps | New features |
| 10 | `1.0.0` | Acceptance run on freshly-wiped cluster against §14; `RETRO.md`; tag `1.0.0` | — |

## Notes

- Sessions 4–10 may shift up/down a slot if a fit check forces a cut. The
  *count* matters less than honest evidence; this is the experiment's point.
- Session 6 introduces the Prometheus Operator — confirm it's installable on
  the host's k3s during that session's Research phase before committing.
- Session 8 turns `test.json` (already in repo) into the authoritative
  Web Validate suite.
