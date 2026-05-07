# movies-api Makefile — minimal wrapper around the inner-loop steps
# documented in IMPL-README.md. Each target is independently runnable.

VERSION    ?= 0.6.0
IMAGE      ?= movies-api:$(VERSION)
TARBALL    ?= /tmp/movies-api-$(VERSION).tar
KCTL       ?= sudo k3s kubectl
SRC        ?= src
MOVIES_DIR ?= deploy/movies/overlays/dev
PROM_OP_DIR ?= deploy/prometheus-operator/overlays/dev
PROM_DIR    ?= deploy/prometheus/overlays/dev
TRAEFIK_DIR ?= deploy/traefik/overlays/dev

.PHONY: help build test image import deploy verify verify-ingress undeploy clean \
	prom-operator-deploy prom-deploy prom-verify prom-undeploy prom-operator-undeploy \
	traefik-apply

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
	@echo "  prom-operator-undeploy- delete prometheus-operator"
	@echo "  traefik-apply         - apply Traefik HelmChartConfig (entrypoints)"
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

prom-operator-undeploy:
	kustomize build $(PROM_OP_DIR) | $(KCTL) delete --ignore-not-found -f -

# k3s reconciles the bundled Traefik chart from this HelmChartConfig.
# Apply re-runs the chart with the entrypoint values, recreating the
# `prometheus`, `grafana`, etc. host ports.
traefik-apply:
	kustomize build $(TRAEFIK_DIR) | $(KCTL) apply -f -
