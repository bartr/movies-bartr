# Research — Stack & Walking-Skeleton Approach

**Date:** 2026-05-06
**Phase constraint:** Research only. No planning, no code. Cite findings; recommend one path.

## 1. The decision space

The spec ([spec.md §1](../docs/spec.md)) is deliberately stack-agnostic. Candidates considered:

| Stack            | Image size (distroless/alpine, hello-world API) | Cold start | Prom client | OpenAPI story  | Notes |
|------------------|------------------------------------------------|------------|-------------|----------------|-------|
| Go + chi + slog  | ~15–25 MB on `gcr.io/distroless/static`        | <50 ms     | mature (`prometheus/client_golang`) | `kin-openapi`, hand-written JSON, `swaggo/swag` | Static binary; trivial non-root + RO root FS |
| Rust + axum      | ~10–15 MB                                      | <50 ms     | mature      | `utoipa`        | Longer compile loop; harder for an inner loop in 90 min |
| Python + FastAPI | ~80–150 MB                                     | 200–800 ms | mature      | built-in        | Heavier image, slower start |
| TS + Fastify     | ~80–120 MB (alpine node)                       | 100–300 ms | ok          | `@fastify/swagger` | Fine, but image not "tiny" |
| .NET 8 minimal   | ~80–110 MB (chiseled)                          | 100–300 ms | mature      | first-class     | Larger images; great DX |

**User constraint:** "Go, because it's fast and the docker images are tiny." This is consistent with the table above.

## 2. Recommended stack — Go

- **Language:** Go 1.26 (host has `go version go1.26.2`, confirmed via `go version`).
- **Router:** `github.com/go-chi/chi/v5` — minimal, idiomatic, plays well with `net/http`, supports sub-routers + middleware composition needed for `/api/*` later.
- **Logging:** standard library `log/slog` with `slog.NewJSONHandler` — meets §7.2 (one JSON object per line, levels, configurable level via env). No third-party dep.
- **Metrics (future session):** `github.com/prometheus/client_golang/prometheus/promhttp` — defer the dep until session 6.
- **Config:** standard library `flag` + env merge in `main` — meets §11 (defaults < env < flags). No `cobra`/`viper` needed.
- **Container base:** `gcr.io/distroless/static-debian12:nonroot` (uid 65532 — but spec §8.1 wants uid 1000; need to set USER 1000 explicitly in the builder + use `distroless/static-debian12` and run as 1000 via securityContext, since distroless `static` does not have shells/users defined for arbitrary UIDs but the kernel only needs the numeric UID; verified pattern: COPY binary, set in Pod's `runAsUser: 1000`. Alternative: `static-debian12:nonroot` + accept uid 65532, but spec is explicit on 1000).

**What we are NOT taking:**
- No web framework beyond chi (no Echo, Gin, Fiber).
- No code-generated OpenAPI in session 1; OpenAPI is session 5.
- No `viper`/`cobra` config — `flag` + env is enough.
- No structured-logging library beyond `log/slog`.
- No metrics library yet.
- No Helm — Kustomize only ([spec §8](../docs/spec.md#8-kubernetes-manifests)).

## 3. Local k8s — what's actually on this host

Toolchain probe (run today):

```
/usr/local/go/bin/go      go1.26.2
/usr/bin/docker
/usr/local/bin/kubectl
/usr/local/bin/kustomize
/usr/local/bin/k3s        (native; no k3d/kind/minikube installed)
/usr/bin/gh
/usr/bin/make
```

`k3d` and `kind` are absent. Native `k3s` is installed. The README ([README.md §1](../README.md)) lists k3d/kind/minikube as acceptable, but the spec ([spec.md §2](../docs/spec.md#2-goals)) says "single-node k3s cluster (and any conformant Kubernetes: k3d, kind, minikube, AKS, EKS, GKE)". Native k3s is in-spec. **Use native k3s.** This avoids installing yet another tool inside session 1.

For loading a locally-built image into k3s without a registry: `k3s ctr images import` from a `docker save` tarball. Documented pattern. Avoids running a local registry in session 1.

## 4. Endpoint contracts for session 1

From spec §6:

- `GET /version` → `200 OK`, `Content-Type: text/plain; charset=utf-8`, body = semver + single `\n`. Must work even before `/readyz`. Source: [spec.md §6.1](../docs/spec.md#61-version-response).
- `GET /healthz` → plaintext `pass` / `warn` / `fail`. Source: [spec.md §6](../docs/spec.md#6-api-surface).
- `GET /readyz` → 200 only after dataset loaded. Session 1 has no dataset, so `/readyz` will return 200 immediately after a synthetic "ready" flag is flipped at startup (no data loading yet — that's session 2).

`/` → `/swagger` redirect is in §6 but bound to swagger which is a session-5 concern. Defer.

## 5. Container & deployment requirements pulled forward to session 1

Spec §8.1 + §13 require, *for any deployment*:
- `livenessProbe: GET /healthz`, `readinessProbe: GET /readyz`
- `resources.requests` 100m/128Mi, `limits` 500m/512Mi
- `securityContext`: `runAsNonRoot: true`, `runAsUser: 1000`, `readOnlyRootFilesystem: true`, `allowPrivilegeEscalation: false`, `capabilities.drop: [ALL]`
- Port 8080 (we'll use a single port for now).
- Non-root in the image itself.

Deferring to later sessions:
- ServiceMonitor (no Prometheus Operator yet — session 6).
- NetworkPolicy (session 6).
- Ingress (not strictly required for session 1; `kubectl port-forward` is sufficient to verify endpoints).

## 6. Risks / unknowns

1. **distroless + uid 1000.** distroless images don't define an `/etc/passwd` entry for 1000; pods can still run as 1000 because the container runtime accepts a numeric UID. Read-only root FS works because the binary is static and we don't write anywhere. Mitigation: set `runAsUser: 1000` in the Pod, `USER 1000:1000` in the Dockerfile; add an `emptyDir` for `/tmp` if any lib touches it (none expected for session 1).
2. **k3s image import.** Image must be importable via `sudo k3s ctr images import`. Confirmed pattern on k3s.io docs.
3. **Time budget.** Cluster-side verification of three trivial endpoints is the long pole. Mitigation: portable shell script for build → import → kustomize apply → port-forward → curl.

## 7. Recommendation

**Go 1.26 + chi v5 + log/slog + flag/env + distroless static + Kustomize on native k3s.** Single binary, single Dockerfile, one base + one overlay, no external deps beyond `chi`. Walking skeleton ships `/version` + `/healthz` + `/readyz` and proves the deploy path. Everything else is a later session.
