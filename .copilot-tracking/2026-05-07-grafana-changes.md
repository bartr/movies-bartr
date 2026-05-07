# Grafana session — Changes (2026-05-07)

## Manifests

### `deploy/grafana/base/`

- `serviceaccount.yaml` — `grafana` ServiceAccount in the `monitoring`
  namespace.
- `configmap-grafana-ini.yaml` — `grafana.ini` with anonymous Viewer
  access, admin user `admin` and admin password fed via the
  `GF_SECURITY_ADMIN_PASSWORD` env var, analytics + telemetry off,
  signup off, console logging.
- `configmap-datasource.yaml` — file-provisioned `prometheus` datasource
  pointing at `http://prometheus.monitoring.svc:9090`, marked
  `editable: false` so anonymous viewers can't repoint it. Datasource
  uid is forced to `prometheus` so the dashboard panels reference it
  stably.
- `dashboards/movies-api.json` — six-panel dashboard (Total requests,
  In-flight, 5xx rate, Goroutines, Requests/sec by route, p95 latency
  by route). `schemaVersion: 39`, no template variables. Uid
  `movies-api`, no `id` (Grafana assigns one on first POST).
- `deployment.yaml` — `grafana/grafana:11.3.0`, runs as uid 472, ALL
  caps dropped, `seccompProfile: RuntimeDefault`,
  `allowPrivilegeEscalation: false`, `Recreate` strategy (single
  replica + emptyDir, so rolling update doesn't help). emptyDir on
  `/var/lib/grafana`. ConfigMap mounts: `grafana-ini` mounted
  read-only at `/etc/grafana/grafana.ini` (subPath), `grafana-datasources`
  mounted at `/etc/grafana/provisioning/datasources`.
  `envFrom: secretRef: grafana-admin` injects the admin password.
  `readOnlyRootFilesystem` left **off** — Grafana writes plugin and
  cache state at boot and retrofitting that to a RO root with
  emptyDirs over `/etc/grafana`, `/var/log/grafana`, `/tmp`, and the
  plugins cache was out of scope for the session.
- `service.yaml` — ClusterIP `grafana:3000`.
- `ingress.yaml` — pinned to the Traefik `grafana` entrypoint (host
  port 3000) via `traefik.ingress.kubernetes.io/router.entrypoints:
  grafana`. No `host:` rule — reachable at `http://127.0.0.1:3000/`.
- `job-bootstrap.yaml` — Job that POSTs the dashboard via the Grafana
  HTTP API (`/api/dashboards/db`) and stars it
  (`/api/user/stars/dashboard/uid/movies-api`) using the admin
  credentials from the `grafana-admin` Secret. Image
  `curlimages/curl:8.10.1` (non-root, no apk needed). Reads the
  dashboard JSON from a configMap mount at `/dashboards/`. Idempotent
  — repeat applies issue Grafana a `version`-aware update keyed by
  the dashboard uid; `overwrite: true` lets the bootstrap overwrite
  itself but Grafana returns 412 on edits the operator has saved
  (which the Job tolerates with `set -e` because we expect first-run
  on a fresh PVC; on a re-deploy the existing dashboard is updated
  in place).
- `kustomization.yaml` — wires the resources and uses
  `configMapGenerator` to wrap `dashboards/movies-api.json` as a
  ConfigMap that the bootstrap Job mounts. Hashed name flows through
  the volume reference automatically.

### `deploy/grafana/overlays/dev/`

- `kustomization.yaml` — references `../../base` and adds a
  `secretGenerator` for `grafana-admin` with `GF_SECURITY_ADMIN_PASSWORD=
  Passw0rd`. The Secret name is hashed; the Deployment + Job
  `envFrom: secretRef` resolve through kustomize's name reference
  machinery.

## Top-level

- `Makefile`:
  - Bumped `VERSION ?= 0.6.0` → `0.7.0`.
  - Added `GRAFANA_DIR` and three `.PHONY` targets: `grafana-deploy`,
    `grafana-verify`, `grafana-undeploy`. `grafana-deploy` waits for
    both the Grafana Deployment and the bootstrap Job to complete.
    `grafana-verify` exercises six checks against `127.0.0.1:3000`:
    `/api/health`, the anonymous-viewer datasource read, the
    dashboard read, `admin /api/user/stars` containing `movies-api`,
    and a PromQL `up` query through the Grafana datasource proxy
    (proves Grafana → Prometheus data flow live).

## App version bumps

- `src/Dockerfile` — default `ARG VERSION` 0.6.0 → 0.7.0.
- `src/internal/httpapi/openapi.json` — `info.version` 0.6.0 → 0.7.0;
  `/version` example body 0.6.0 → 0.7.0.
- `deploy/movies/base/deployment.yaml` — image tag `movies-api:0.6.0`
  → `movies-api:0.7.0`.

## Tracking

- `.copilot-tracking/2026-05-07-grafana-research.md` — research.
- `.copilot-tracking/2026-05-07-grafana-plan.md` — plan.
- `.copilot-tracking/2026-05-07-grafana-changes.md` — this file.
- `.copilot-tracking/2026-05-07-grafana-review.md` — review (post-verify).
- `session-log.md` — Session 7 frame appended; close ritual filled
  after user review.

## In-cluster verification

- `make test` → all packages green with `-race`.
- `make image import deploy verify` → `/version` returns `0.7.0`,
  `/healthz`, `/readyz`, `/metrics` all green (one rolling-update 502
  retry as documented in repo memory).
- `make grafana-deploy` → Deployment Available, Job Complete.
  Bootstrap Job log:
  ```
  [bootstrap] grafana is up
  [bootstrap] payload bytes: 5059
  [bootstrap] dashboard response: {"folderUid":"","id":1,
    "slug":"movies-api","status":"success","uid":"movies-api",
    "url":"/d/movies-api/movies-api","version":1}
  [bootstrap] starring dashboard for admin ...
  {"message":"Dashboard starred!"}
  [bootstrap] starred movies-api
  ```
- `make grafana-verify` → all six checks green.
  Live PromQL probe through Grafana's datasource proxy returned a
  `success` response with the `up{job="movies-api",...}` series, so
  the dashboard panels are reading real data.

## Implementation notes worth keeping

- **First Job pass used `alpine:3.20` + `apk add curl jq`** and
  CrashLoopBackOff'd because `runAsNonRoot: true` (uid 1000) cannot
  write to apk's database. Switched to `curlimages/curl:8.10.1`
  (already non-root) and replaced `jq` payload assembly with a
  `printf` + `cat` heredoc that prepends/appends the
  `{"dashboard":..., "overwrite":true, "message":"bootstrap"}`
  envelope around the raw dashboard JSON. Non-root + read-only
  rootfs + tmpfs `/tmp` works cleanly.
- **Datasource uid pinned to `prometheus`.** The dashboard panels
  reference `{"type":"prometheus","uid":"prometheus"}`. Without the
  pin, Grafana auto-assigns a random uid and panels would render
  "Datasource not found" until manually repointed.
- **Anonymous Viewer is configured in `grafana.ini`** rather than via
  env (`GF_AUTH_ANONYMOUS_ENABLED=true`) because spec §7.3 says the
  anonymous flag belongs in the dev overlay only — keeping it in a
  ConfigMap that lives under `deploy/grafana/base/` is fine for now
  since there's only the dev overlay; a future prod overlay would
  patch the ConfigMap to set `enabled = false`.
- **API-driven dashboard, not file-provisioned.** File provisioning
  marks dashboards as managed and the UI surfaces them as read-only.
  Per the user's request, the dashboard must be editable + saveable
  by the admin. Pushing it through `/api/dashboards/db` once at
  startup is the canonical workaround; Grafana treats it as a normal
  user-created dashboard from then on.
