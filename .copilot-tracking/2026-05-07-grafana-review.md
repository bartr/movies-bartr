# Grafana session — Review (2026-05-07)

## Frame check

Goal was: Grafana in `monitoring` namespace, anonymous viewer in dev,
admin password `Passw0rd` via Secret, datasource provisioned, dashboard
created via the **Grafana HTTP API** (so it stays editable), dashboard
added to admin's favorites, Ingress on the Traefik `grafana` entrypoint
(host port 3000). Tag `0.7.0`.

| Frame item                                            | Status | Evidence |
|-------------------------------------------------------|--------|----------|
| Grafana running in `monitoring` namespace             | ✅     | `kubectl -n monitoring get deploy/grafana` Available |
| Ingress on host port 3000                             | ✅     | `curl http://127.0.0.1:3000/api/health` 200 |
| Admin password `Passw0rd` via Kubernetes Secret       | ✅     | `secretGenerator` in dev overlay → `grafana-admin-<hash>`, mounted via `envFrom: secretRef` (no plaintext outside the dev overlay tree) |
| Anonymous viewer in dev                               | ✅     | `/api/datasources/name/prometheus` returns 200 with no auth |
| Prometheus datasource provisioned automatically      | ✅     | uid `prometheus`, url `http://prometheus.monitoring.svc:9090`, `editable: false` |
| Dashboard created via Grafana API (not file-provisioned, so editable) | ✅ | Bootstrap Job POSTs `/api/dashboards/db`; `meta.canEdit` is gated only by org role (admin sees true; anon sees false because Viewer); no "provisioned" warning in the UI |
| Dashboard added to admin's favorites via API          | ✅     | `curl -u admin:Passw0rd .../api/user/stars` → `["movies-api"]` |
| Live data on the dashboard                            | ✅     | Datasource proxy `up` query returns `success` with `up{job="movies-api"}=1` |
| Tag `0.7.0`                                           | pending | Tag deferred to user review per repo memory |

## Out-of-scope items honoured

- No NetworkPolicy in `monitoring` (parking-lot for a future session).
- No PVC for Grafana (emptyDir is fine for dev; documented in research).
- No soak/bench, no Web Validate.

## Drift / fixups during the session

1. **Bootstrap Job v1 (alpine + apk) crash-looped.** Non-root pod
   couldn't `apk add`. Replaced with `curlimages/curl:8.10.1` and a
   `printf` + `cat` JSON wrapper (no jq). Saved as a repo-memory note
   for future "shell + curl in a Job" patterns.
2. **`make grafana-verify` grep patterns.** First two passes used
   `'\"type\":\"prometheus\"'` style quoting that didn't survive the
   bash `-c` heredoc; matched the existing `prom-verify` style with
   `\"...\"` escapes and it cleared.
3. **Star check.** Grafana 11 returns `["movies-api"]` (uids), not
   numeric ids; relaxed the verify grep to look for `movies-api`.

None of these were scope creep — all were straight bugfixes inside the
named deliverable.

## Tests

- `make test` → all green with `-race`.
- `make verify` → `/version` 0.7.0 + `/healthz` + `/readyz` + `/metrics`
  green (after the documented rolling-update retry).
- `make grafana-verify` → all six checks green.

## Health signal

- Framing quality (1–5): 4. The frame correctly anticipated the
  API-vs-file-provisioning trade-off and the Traefik entrypoint pin.
  One missed detail: the bootstrap Job container choice — should have
  picked `curlimages/curl` from the start instead of detouring through
  `alpine + apk`.
- Drift: no.
- Fit check honest: yes — bundled the dashboard, the API call, *and*
  the favorite-star into one frame, with star-only as the named cut;
  didn't need the cut.
- Close: pending user review per repo memory rule (do **not** auto-tag).

## Next session starter

Session 8 — Web Validate runner. Cluster has Prometheus + Grafana
healthy; movies-api 0.7.0 is up with the dashboard on
`http://127.0.0.1:3000/d/movies-api/movies-api`. Web Validate suite
should hit the `/api/*` routes through the Traefik `web` entrypoint and
the dashboard panels should light up the requests/p95 timeseries.
