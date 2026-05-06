# Review — Walking Skeleton (Session 1, tag 0.1.0)

**Date:** 2026-05-06
**Phase constraint:** Review only — validate against research + plan. No code changes.

## Plan task status

| # | Task                                 | Status |
|---|--------------------------------------|--------|
| T1 | Branch `session/0.1.0-skeleton`     | ✅ |
| T2 | `go mod init`                       | ✅ |
| T3 | `chi` dep added; `go mod tidy`      | ✅ |
| T4 | Source layout (5 files)             | ✅ |
| T5 | Handlers behaviour                  | ✅ — `/version` body `0.1.0\n`, ct `text/plain; charset=utf-8`; `/healthz` `pass\n`; `/readyz` 503 / 200 |
| T6 | Unit tests + `-race`                | ✅ — `httpapi` 100.0% coverage |
| T7 | Multi-stage distroless Dockerfile   | ✅ — image 3.72 MB content, 16.2 MB disk; smoke tested |
| T8 | Kustomize base                      | ✅ — `kubectl apply --dry-run=client` clean |
| T9 | Kustomize dev overlay               | ✅ |
| T10 | k3s image import                   | ✅ — `docker.io/library/movies-api:0.1.0` listed |
| T11 | Deploy                             | ✅ — rollout succeeded; pod 1/1 Running |
| T12 | In-cluster verification            | ✅ — port-forward + curl, all three endpoints correct |
| T13 | README + Makefile                  | ✅ — `IMPL-README.md`, `Makefile` |
| T14 | Repo memory                        | ✅ — `AGENTS.md` |
| T15 | Close                              | (next) |

## Spec checks pulled forward

- §6.1 `/version` — body is exactly `0.1.0\n`, single trailing newline; `Content-Type: text/plain; charset=utf-8`. ✅
- §6 `/healthz` plaintext. ✅ (only `pass` for now; `warn`/`fail` is later)
- §6 `/readyz` gates on dataset; ✅ (synthetic flip; real loader in session 2)
- §8.1 probes wired to right paths. ✅
- §8.1 resources 100m/128Mi req, 500m/512Mi lim. ✅
- §8.1 securityContext non-root, RO root FS, drop ALL caps, no priv esc. ✅ (verified on the live pod)
- §8.1 port 8080. ✅
- §11 defaults < env < flags; validates log level + port; effective config logged once at info on startup. ✅ (`{"level":"INFO","msg":"starting movies-api","version":"0.1.0","config":"data_dir=/data log_level=info port=8080"}`)
- §13 non-root container, RO root FS, no caps, no priv esc. ✅
- §7.2 logs are JSON-per-line via `slog.NewJSONHandler` to stdout; level configurable via `MOVIES_LOG_LEVEL`. ✅

## Verified out-of-scope

- No `/api/*` shipped (correctly deferred to session 3).
- No metrics, OpenAPI, dashboards (deferred to sessions 5–7).
- No NetworkPolicy / ServiceMonitor (session 6).

## Quality signals

- `go vet ./...` clean.
- `gofmt -l .` clean (after one normalization pass during review).
- `httpapi` test coverage 100% (handlers + router code paths exercised).
- `config` and `version` have 0% direct test coverage today — acceptable for session 1 (`config` is exercised end-to-end via the `docker run` smoke and the cluster deploy; targeted unit tests will land in session 2 alongside the data layer where validation rules grow).

## Follow-ups (parking lot for next sessions)

- Add unit tests for `internal/config` (env precedence, flag override, invalid log level, invalid port). Owner: session 2.
- Decide whether `commonLabels`-replacement `labels` entry should also include selectors once we have multiple workloads sharing the namespace (Prometheus, Grafana). Owner: session 6.
- Image tag strategy when `VERSION` bumps — currently `make import` always re-saves; fine for now; revisit if cycle time grows.

## Verdict

Walking skeleton is honest. All §1 frame goals met; nothing in the out-of-scope list leaked. Ready to tag `0.1.0`.
