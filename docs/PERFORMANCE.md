# Performance — measured against spec §10.4

> Spec: [spec.md §10.4](spec.md). Methodology: [METHODOLOGY.md](METHODOLOGY.md).
> Experiment framing: [EXPERIMENT.md](EXPERIMENT.md).

## TL;DR

**Go is fast.** Sub-millisecond p95 on a 50 ms target — between **50× and
500× under spec** with no caching layer, no tuning, no tricks. The
service is a chi handler, an in-memory `map` lookup, and `encoding/json`.

| Route                  | Spec target | Measured p95 (0.9.0) | Headroom |
|------------------------|------------:|---------------------:|---------:|
| `GET /api/movies`      |       50 ms |              ~0.23 ms |     217× |
| `GET /api/actors`      |       50 ms |              ~0.21 ms |     238× |
| `GET /api/genres`      |       50 ms |              ~0.10 ms |     500× |
| `GET /api/movies/{id}` |       10 ms |              ~0.10 ms |     100× |
| `GET /api/actors/{id}` |       10 ms |              ~0.10 ms |     100× |

Throughput on a single 500m-CPU pod: **584 RPS sustained, 0% 5xx**
(spec target: 500 RPS, < 1% errors). The pod can drive ~800 RPS
flat-out before CPU throttling; we deliberately pace the load
generator just above the SLO to validate steady-state behaviour
rather than peak.

## Why this matters

[EXPERIMENT.md §"What this experiment is not"](EXPERIMENT.md) is explicit
that this is **not** a language benchmark — Go, Rust, Python,
TypeScript, .NET should all clear the bar. They should. But the
margin matters: when p95 is **two orders of magnitude under spec**,
the operator can spend the budget elsewhere — bigger payloads,
richer middleware, a slower disk, a noisier neighbor — and the SLO
still holds. That's the real win, and it's what made Go the
session-1 stack pick (alongside the deployable-binary story).

## How we measured

- **Load generator:** `webv` (in-cluster Deployment, same image as
  the API), running `--threads=2 --sleep=3ms --loop` against the
  `benchmark.yaml` suite (200 entries spanning every route + path-id
  + query-string variants). The kernel timer on this host quantizes
  Go's `time.Sleep` to roughly 1 ms slices, which means single-thread
  RPS jumps in coarse steps (800 → 428 → 293) with no value near
  500. Two threads at `sleep=3ms` give ~584 RPS sustained — just
  above the spec target without saturating the pod.
- **Histogram buckets:** custom sub-ms ladder starting at 100 µs
  (see [src/internal/httpapi/metrics.go](../src/internal/httpapi/metrics.go)).
  The Prometheus default starts at 5 ms, which is fine for typical
  web services but collapses every request on this API into a single
  bucket — `histogram_quantile` then interpolates inside that bucket
  and returns ~4.75 ms for every route. Before fixing the buckets,
  the dashboard told a uniform-but-wrong story; after, the routes
  separate cleanly.
- **Route labels:** five distinct templates (`/api/movies`,
  `/api/movies/{id}`, `/api/actors`, `/api/actors/{id}`, `/api/genres`)
  so list and detail latency can be observed independently.
  Cardinality stays bounded — IDs are templated, not raw.
- **Pod limits:** 500m CPU, 512Mi memory. Distroless static base.
- **Window:** 1-minute `rate()` over a sustained run; numbers above
  are representative steady-state.

## Why it's this fast

1. **In-process map lookups.** The dataset is loaded once at startup
   into Go maps + slice indexes (see
   [src/internal/store/store.go](../src/internal/store/store.go)).
   Detail routes are an O(1) map hit. List routes iterate a few
   thousand entries — still well under a millisecond.
2. **No serialization overhead worth talking about.** Responses are
   small (200–1000 bytes), `encoding/json` is fast enough that the
   JSON encode is dominated by the network write.
3. **No caching.** Verified — handlers do not set `Cache-Control`,
   the load generator sends `Cache-Control: no-cache` defensively,
   and the per-route mean latencies differ by an order of magnitude
   (genres is smallest, movies is largest). If anything were cached
   the means would converge.
4. **Per-router Prometheus registry.** Histogram observation cost is
   the dominant non-handler work on the request path; the
   `prometheus/client_golang` lock-free path keeps it in the
   sub-microsecond range.

## What this is *not*

- **Not a published benchmark of Go vs. anything.** This is one
  experiment-run's evidence against one spec.
- **Not a production load profile.** Real traffic isn't a single
  benchmark suite on a clean network; expect cold caches, GC
  pauses on bigger heaps, and tail latency from coordinated
  omission once the workload mix changes.
- **Not a license to skip future profiling.** The headroom we have
  today is the budget we'll spend on features tomorrow.

## Reproducing

```sh
make image import deploy            # ship the API
kubectl -n movies apply -k deploy/webv/overlays/dev   # start load
sleep 60                            # let histograms accumulate
curl -sS -G http://127.0.0.1:9090/api/v1/query \
  --data-urlencode 'query=histogram_quantile(0.95, sum by (le, route) (rate(http_request_duration_seconds_bucket{route=~"/api/.*"}[1m])))' \
  | jq -r '.data.result[] | [.metric.route, ((.value[1]|tonumber*1000)|tostring + " ms")] | @tsv'
```
