# Metrics + ServiceMonitor + NetworkPolicy — Review (2026-05-07)

## Did the frame hold?

Yes. Goal was Prometheus metrics + Operator wiring + NetworkPolicy +
securityContext review. All four landed; Grafana stayed out of scope as
declared.

## Surprises

- During the first `make deploy`, `make verify` got a 502 from Traefik on
  the very first `/version`. The pod was Running and serving correctly —
  it was the rolling-update window between Traefik's old endpoint cache
  and the new pod. A second `make verify` was clean. **Followup:** the
  Makefile's `verify` could add a small retry, but that's polish, not
  this session's frame.
- `go mod tidy` initially stripped `prometheus/client_golang` because no
  package imported it yet. Order had to be: write `metrics.go` first, then
  `go mod tidy`. Recorded in repo memory.
- Nothing in the existing tests failed when the metrics middleware was
  inserted before `requestLogger`. Coverage rose 91.2 → 92.7 %.

## What I'd change next time

- Bundle this with Grafana provisioning if the dashboards are mostly
  copy-paste; could fit if metrics work was already reviewed offline.
- Consider exporting the metrics endpoint on a *separate* port (spec §8.1
  permits 9090) so NetworkPolicy can lock the public path (8080 from
  Traefik) without also opening it for Prometheus. Not worth doing now —
  same-port keeps the manifests simpler.

## Health signal (self-graded)

- Framing quality: 5 — frame named all four deliverables, all four shipped.
- Drift: no.
- Fit check honest: yes — recorded "proceed", with `NetworkPolicy` cut as
  the listed fallback, and didn't need it.

## Next session starter

Session 7 — Grafana + provisioned datasource + provisioned dashboard for the
movies-api `http_requests_total` / `http_request_duration_seconds` metrics.
The ServiceMonitor is in place; the cluster Prometheus comes-and-goes
(currently 0/1 in `default/`) — restoring it is a tactical pre-step. Next
slice after that: Web Validate runner.
