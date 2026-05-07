# movies-api Makefile — minimal wrapper around the inner-loop steps
# documented in IMPL-README.md. Each target is independently runnable.

VERSION     ?= 0.9.0
IMAGE       ?= movies-api:$(VERSION)
TARBALL     ?= /tmp/movies-api-$(VERSION).tar
KCTL        ?= sudo k3s kubectl
SRC         ?= src
MOVIES_DIR  ?= deploy/movies/overlays/dev
PROM_OP_DIR ?= deploy/prometheus-operator/overlays/dev
PROM_DIR    ?= deploy/prometheus/overlays/dev
GRAFANA_DIR ?= deploy/grafana/overlays/dev
TRAEFIK_DIR ?= deploy/traefik/overlays/dev
WEBV_DIR    ?= deploy/webv/overlays/dev

.PHONY: help build test image import deploy verify verify-ingress undeploy clean \
	prom-operator-deploy prom-deploy prom-verify prom-undeploy prom-operator-undeploy \
	prom-tombstone-stale-routes \
	grafana-deploy grafana-verify grafana-undeploy \
	traefik-apply \
	webv-install webv-smoke webv-deploy webv-verify webv-undeploy

help:
	@echo "Targets:"
	@echo "  build                 - go build ./..."
	@echo "  test                  - go test -race ./..."
	@echo "  image                 - docker build -t $(IMAGE) ./src"
	@echo "  import                - docker save | k3s ctr images import"
	@echo "  deploy                - kustomize build movies | kubectl apply"
	@echo "  verify                - curl /version /healthz /readyz /metrics"
	@echo "  verify-ingress        - alias of verify"
	@echo "  undeploy              - delete movies overlay"
	@echo "  prom-operator-deploy  - apply prometheus-operator (CRDs + operator)"
	@echo "  prom-deploy           - apply Prometheus instance (monitoring ns)"
	@echo "  prom-verify           - confirm Prometheus is scraping movies-api"
	@echo "  prom-undeploy         - delete Prometheus instance"
	@echo "  prom-tombstone-stale-routes - delete http_* series with route!~/api/.*"
	@echo "  prom-operator-undeploy- delete prometheus-operator"
	@echo "  grafana-deploy        - apply Grafana (monitoring ns)"
	@echo "  grafana-verify        - check /api/health, datasource, dashboard, star"
	@echo "  grafana-undeploy      - delete Grafana"
	@echo "  traefik-apply         - apply Traefik HelmChartConfig (entrypoints)"
	@echo "  webv-install          - go install webv to ~/go/bin (with version ldflags)"
	@echo "  webv-smoke            - run webv against http://127.0.0.1 with src/webv/test.yaml"
	@echo "  webv-deploy           - apply webv Job (movies ns)"
	@echo "  webv-verify           - tail webv pod logs and confirm passes"
	@echo "  webv-undeploy         - delete webv overlay"
	@echo "  clean                 - go clean + rm tarball"

build:
	cd $(SRC) && go build ./...

test:
	cd $(SRC) && go test -race ./...

image:
	docker build --build-arg VERSION=$(VERSION) -t $(IMAGE) $(SRC)

import: image
	docker save $(IMAGE) -o $(TARBALL)
	sudo k3s ctr images import $(TARBALL)
	@echo "Imported $(IMAGE) into k3s containerd"

deploy:
	kustomize build $(MOVIES_DIR) | $(KCTL) apply -f -
	$(KCTL) -n movies rollout status deploy/movies-api --timeout=60s

undeploy:
	kustomize build $(MOVIES_DIR) | $(KCTL) delete -f - --ignore-not-found

# verify hits Traefik on host port 80 (k3s default) using the Ingress host
# `localhost`. No port-forward needed.
verify verify-ingress:
	@bash -c '\
		set -e; \
		echo "--- /version ---"; \
		V=$$(curl -sS -i -H "Host: localhost" http://127.0.0.1/version); echo "$$V"; \
		echo "$$V" | grep -q "Content-Type: text/plain; charset=utf-8" || (echo "FAIL: /version content-type"; exit 1); \
		echo "$$V" | tail -n1 | grep -qx "$(VERSION)" || (echo "FAIL: /version body != $(VERSION)"; exit 1); \
		echo "--- /healthz ---"; \
		curl -sS -H "Host: localhost" http://127.0.0.1/healthz | tee /dev/stderr | grep -qx "pass" || (echo "FAIL: /healthz"; exit 1); \
		echo "--- /readyz ---"; \
		curl -sS -H "Host: localhost" http://127.0.0.1/readyz | tee /dev/stderr | grep -qx "pass" || (echo "FAIL: /readyz"; exit 1); \
		echo "--- /metrics ---"; \
		M=$$(curl -sS -H "Host: localhost" http://127.0.0.1/metrics); \
		echo "$$M" | head -3; \
		echo "$$M" | grep -q "^# HELP http_requests_total" || (echo "FAIL: /metrics missing http_requests_total"; exit 1); \
		echo "$$M" | grep -q "^go_goroutines " || (echo "FAIL: /metrics missing go_goroutines"; exit 1); \
		echo "OK: all endpoints verified via Traefik on http://localhost"; \
	'

clean:
	cd $(SRC) && go clean
	rm -f $(TARBALL)

# ---------------------------------------------------------------- prometheus
# `prom-operator-deploy` installs (or upgrades) the upstream operator.
# `prom-deploy` creates the Prometheus instance in the `monitoring` ns and
# waits for it to become Ready. `prom-verify` checks that movies-api shows
# up as a healthy scrape target.
prom-operator-deploy:
	kustomize build $(PROM_OP_DIR) | $(KCTL) apply --server-side --force-conflicts -f -
	$(KCTL) -n default rollout status deploy/prometheus-operator --timeout=120s

prom-deploy:
	kustomize build $(PROM_DIR) | $(KCTL) apply -f -
	$(KCTL) -n monitoring wait --for=condition=Available prometheus/prometheus --timeout=120s

prom-verify:
	@bash -c '\
		set -e; \
		echo "--- prometheus pod ---"; \
		$(KCTL) -n monitoring get pod -l app.kubernetes.io/name=prometheus -o wide; \
		echo "--- targets (movies-api) ---"; \
		POD=$$($(KCTL) -n monitoring get pod -l app.kubernetes.io/name=prometheus -o jsonpath="{.items[0].metadata.name}"); \
		$(KCTL) -n monitoring exec $$POD -c prometheus -- wget -qO- "http://127.0.0.1:9090/api/v1/targets?state=active" | grep -o "\"health\":\"[a-z]*\"" | sort -u; \
		echo "--- query http_requests_total ---"; \
		$(KCTL) -n monitoring exec $$POD -c prometheus -- wget -qO- "http://127.0.0.1:9090/api/v1/query?query=http_requests_total" | head -c 400; echo; \
		echo "OK: Prometheus is up and scraping"; \
	'

prom-undeploy:
	kustomize build $(PROM_DIR) | $(KCTL) delete --ignore-not-found -f -

# Tombstone every http_* series whose `route` label is NOT `/api/...`.
# Used after the metrics middleware is tightened (e.g. dropping
# /healthz, /version, or `unmatched`) — Prometheus retains old labels
# for the full retention window otherwise. Requires `enableAdminAPI:
# true` on the Prometheus CR (already set in deploy/prometheus/base/
# prometheus.yaml).
prom-tombstone-stale-routes:
	@bash -c '\
		set -e; \
		H=http://127.0.0.1:9090; \
		echo "--- before ---"; \
		curl -sS -G $$H/api/v1/label/route/values | head -c 500; echo; \
		for m in http_requests_total http_request_duration_seconds_bucket http_request_duration_seconds_sum http_request_duration_seconds_count; do \
			echo "--- delete_series $$m route!~/api/.* ---"; \
			curl -sS -X POST -G "$$H/api/v1/admin/tsdb/delete_series" \
				--data-urlencode "match[]=$$m{route!~\"/api/.*\"}"; echo; \
		done; \
		echo "--- clean_tombstones ---"; \
		curl -sS -X POST $$H/api/v1/admin/tsdb/clean_tombstones; echo; \
		echo "--- after ---"; \
		curl -sS -G $$H/api/v1/label/route/values | head -c 500; echo; \
	'

prom-operator-undeploy:
	kustomize build $(PROM_OP_DIR) | $(KCTL) delete --ignore-not-found -f -

# ------------------------------------------------------------------- grafana
# `grafana-deploy` applies Grafana into the `monitoring` namespace and
# waits for the Deployment + bootstrap Job to complete. The Job is what
# POSTs the dashboard via the Grafana HTTP API (so the UI stays
# editable, unlike file-provisioned dashboards) and then stars it for
# admin. `grafana-verify` exercises the Ingress on host port 3000 and
# the dashboard end-to-end.
grafana-deploy:
	# Delete the bootstrap Job before re-applying: Job pod templates are
	# immutable and the configMap volume reference changes every time the
	# dashboard JSON does (kustomize hashes the dashboard ConfigMap).
	$(KCTL) -n monitoring delete job grafana-bootstrap --ignore-not-found
	kustomize build $(GRAFANA_DIR) | $(KCTL) apply -f -
	$(KCTL) -n monitoring rollout status deploy/grafana --timeout=120s
	$(KCTL) -n monitoring wait --for=condition=complete job/grafana-bootstrap --timeout=120s

grafana-verify:
	@bash -c '\
		set -e; \
		H=http://127.0.0.1:3000; \
		echo "--- /api/health ---"; \
		curl -fsS $$H/api/health | tee /dev/stderr | grep -q "\"database\": \"ok\"" || (echo "FAIL: grafana health"; exit 1); \
		echo; \
		echo "--- datasource (anon viewer) ---"; \
		DS=$$(curl -fsS $$H/api/datasources/name/prometheus); echo "$$DS"; \
		echo "$$DS" | grep -q "\"type\":\"prometheus\"" || (echo "FAIL: datasource type"; exit 1); \
		echo "$$DS" | grep -q "prometheus.monitoring.svc:9090" || (echo "FAIL: datasource url"; exit 1); \
		echo "--- dashboard movies-api (anon viewer) ---"; \
		DB=$$(curl -fsS $$H/api/dashboards/uid/movies-api); \
		echo "$$DB" | head -c 240; echo; \
		echo "$$DB" | grep -q "\"title\":\"Movies API\"" || (echo "FAIL: dashboard title"; exit 1); \
		echo "--- admin stars (auth) ---"; \
		ST=$$(curl -fsS -u admin:Passw0rd $$H/api/user/stars); echo "$$ST"; \
		echo "$$ST" | grep -q "movies-api" || (echo "FAIL: movies-api not in admin stars"; exit 1); \
		echo "--- live data via grafana datasource proxy ---"; \
		Q=$$(curl -fsS $$H/api/datasources/proxy/uid/prometheus/api/v1/query?query=up); \
		echo "$$Q" | head -c 240; echo; \
		echo "$$Q" | grep -q "\"status\":\"success\"" || (echo "FAIL: prometheus proxy query"; exit 1); \
		echo "OK: grafana up, datasource live, dashboard provisioned + starred"; \
	'

grafana-undeploy:
	kustomize build $(GRAFANA_DIR) | $(KCTL) delete --ignore-not-found -f -

# k3s reconciles the bundled Traefik chart from this HelmChartConfig.
# Apply re-runs the chart with the entrypoint values, recreating the
# `prometheus`, `grafana`, etc. host ports.
traefik-apply:
	kustomize build $(TRAEFIK_DIR) | $(KCTL) apply -f -

# ---------------------------------------------------------------------- webv
# `webv-install` builds the CLI and installs it into $GOBIN (~/go/bin),
# which is on PATH on this host. Same VERSION as movies-api.
# `webv-smoke` runs one pass against the local Traefik web entrypoint
# using the same suite as the in-cluster Job.
# `webv-deploy` applies the Job into the `movies` namespace; the Job
# uses the in-cluster Service URL and runs --loop, so the pod stays up
# until `make webv-undeploy`.
webv-install:
	cd $(SRC) && go install \
		-trimpath \
		-ldflags "-s -w -X github.com/bartr/bartr-movies/internal/version.Version=$(VERSION)" \
		./cmd/webv
	@command -v webv >/dev/null && webv --version || echo "webv installed; ensure ~/go/bin is on PATH"

webv-smoke:
	webv --url http://127.0.0.1 --files $(SRC)/webv/test.yaml --threads 2 --verbose | tail -20

webv-deploy:
	kustomize build $(WEBV_DIR) | $(KCTL) apply -f -
	$(KCTL) -n movies rollout status deploy/webv --timeout=60s

webv-verify:
	@bash -c '\
		set -e; \
		echo "--- webv pod ---"; \
		$(KCTL) -n movies get pod -l app.kubernetes.io/name=webv -o wide; \
		echo "--- recent log lines (last 20) ---"; \
		POD=$$($(KCTL) -n movies get pod -l app.kubernetes.io/name=webv -o jsonpath="{.items[0].metadata.name}"); \
		$(KCTL) -n movies logs $$POD --tail=20; \
		echo "--- summary so far ---"; \
		$(KCTL) -n movies logs $$POD | tail -5; \
		echo "OK: webv is running"; \
	'

webv-undeploy:
	kustomize build $(WEBV_DIR) | $(KCTL) delete --ignore-not-found -f -
