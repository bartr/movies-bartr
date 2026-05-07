# Grafana session — Plan (2026-05-07)

## Files

### New

- `deploy/grafana/base/namespace.yaml` — empty file? No — `monitoring`
  namespace already exists from `deploy/prometheus`. Skip; just set
  `namespace: monitoring` in kustomization.
- `deploy/grafana/base/serviceaccount.yaml` — `grafana` SA.
- `deploy/grafana/base/configmap-grafana-ini.yaml` — `grafana.ini` with
  anonymous viewer + admin password env reference.
- `deploy/grafana/base/configmap-datasource.yaml` — Prometheus
  datasource provisioning YAML.
- `deploy/grafana/base/configmap-dashboard.yaml` — wraps the dashboard
  JSON for the bootstrap Job to read.
- `deploy/grafana/base/dashboards/movies-api.json` — dashboard JSON.
- `deploy/grafana/base/deployment.yaml` — Grafana Deployment.
- `deploy/grafana/base/service.yaml` — ClusterIP `grafana:3000`.
- `deploy/grafana/base/ingress.yaml` — Ingress pinned to Traefik
  `grafana` entrypoint.
- `deploy/grafana/base/job-bootstrap.yaml` — Job that POSTs the
  dashboard via API and stars it.
- `deploy/grafana/base/kustomization.yaml` — wires the resources +
  pulls the dashboard JSON into the dashboard ConfigMap via
  `configMapGenerator`.
- `deploy/grafana/overlays/dev/kustomization.yaml` — references base +
  `secretGenerator` for `grafana-admin` (admin password = `Passw0rd`).

### Edited

- `Makefile` — bump `VERSION` to `0.7.0`. Add targets:
  `grafana-deploy`, `grafana-verify`, `grafana-undeploy`. Wire into
  the help block.
- `AGENTS.md` — append a Session 7 entry under "Where the next session
  starts" → flip to point at session 8 (web-validate).
- `IMPL-README.md` — add Session 7 line in "What's done"; remove
  `0.7.0` row from the deferred table.
- `session-log.md` — Session 7 frame + close (close written after
  user review).

## Approach

### Bootstrap Job

```sh
#!/bin/sh
set -eu
H="http://grafana.monitoring.svc:3000"
PW="$GF_SECURITY_ADMIN_PASSWORD"
echo "waiting for grafana..."
for i in $(seq 1 60); do
  curl -fsS "$H/api/health" >/dev/null 2>&1 && break
  sleep 2
done
echo "creating dashboard..."
# wrap the raw dashboard JSON in {dashboard:..., overwrite:true, message:...}
jq -n --slurpfile d /dashboards/movies-api.json \
  '{dashboard: $d[0], overwrite: true, message: "bootstrap 0.7.0"}' \
  | curl -fsS -u "admin:$PW" \
      -H 'Content-Type: application/json' \
      -X POST "$H/api/dashboards/db" -d @-
echo "starring dashboard..."
curl -fsS -u "admin:$PW" -X POST "$H/api/user/stars/dashboard/uid/movies-api" || true
echo "done."
```

Image: `alpine/curl:8.10.1` (has both curl and jq via apk? no — use
`ghcr.io/jqlang/jq:latest` won't have curl). Cleaner: `alpine:3.20` +
`apk add --no-cache curl jq` in command. Even cleaner: do without `jq`
and inline-construct JSON with a heredoc. **Decision:** skip `jq`. Use
`alpine/curl` and embed JSON construction in shell with a `cat <<EOF`
that wraps the dashboard. We avoid the apk install hop.

But then we'd have to embed the dashboard JSON in shell, fragile with
quoting. Better path: `curlimages/curl` + write the wrapped payload
with a tiny POSIX script that reads the dashboard file and prepends/
appends the wrapper bytes. Easier still: ship two files — the inner
dashboard JSON, and a precomputed wrapper template — and concatenate
them at runtime. Simplest correct approach: use the
`grafana/grafana:11.3.0` image itself for the bootstrap Job (it ships
both `curl` *and* the dashboard via the same provisioning ConfigMap
mount). Wait — that's overkill image-wise but pragmatic. Actually
`grafana/grafana` does not ship `curl` reliably.

**Final decision:** use `alpine:3.20` and run
`apk add --no-cache curl jq` once. Cost is ~3 sec on first pull, but
the Job is run-to-completion so it's a one-off. Use `jq` to wrap the
dashboard JSON robustly.

### configMapGenerator for dashboard JSON

```yaml
configMapGenerator:
  - name: grafana-dashboard-movies-api
    files:
      - dashboards/movies-api.json
```

The Job mounts that ConfigMap at `/dashboards/`.

### Datasource provisioning

```yaml
# /etc/grafana/provisioning/datasources/prometheus.yaml
apiVersion: 1
datasources:
  - name: prometheus
    type: prometheus
    access: proxy
    url: http://prometheus.monitoring.svc:9090
    isDefault: true
    editable: false
```

Marking it `editable: false` keeps a viewer from accidentally repointing
it (anonymous role is Viewer anyway, but safer).

### Secret

Dev overlay:

```yaml
secretGenerator:
  - name: grafana-admin
    literals:
      - GF_SECURITY_ADMIN_PASSWORD=Passw0rd
```

`disableNameSuffixHash` is **not** set → kustomize hashes the name, the
deployment's `envFrom.secretRef.name` resolves through the kustomize
name reference machinery automatically.

### Ingress

```yaml
metadata:
  annotations:
    traefik.ingress.kubernetes.io/router.entrypoints: grafana
spec:
  ingressClassName: traefik
  rules:
    - http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: grafana
                port:
                  number: 3000
```

Reachable at `http://127.0.0.1:3000/`.

## Verify

`make grafana-verify`:

1. `grafana` Deployment Available.
2. `grafana-bootstrap` Job Complete.
3. `curl http://127.0.0.1:3000/api/health` → 200 + `database: ok`.
4. `curl http://127.0.0.1:3000/api/datasources/name/prometheus` (anon)
   → 200, type prometheus, url matches.
5. `curl http://127.0.0.1:3000/api/dashboards/uid/movies-api` (anon)
   → 200, title "Movies API".
6. `curl -u admin:Passw0rd .../api/user/stars` → JSON array containing
   the dashboard's numeric id.
7. PromQL probe: `curl http://127.0.0.1:3000/api/datasources/proxy/uid/<ds-uid>/api/v1/query?query=up`
   returns at least one series (proves data flow Grafana → Prometheus).

## Out of scope

- NetworkPolicy for monitoring ns.
- PVC for Grafana.
- Soak/bench.

## Fit check

90–120 min: yes. Smallest cut if we run long: drop the favorite-star
step (item 6) — it's the smallest deliverable in the frame and the
dashboard would still satisfy spec §7.3.
