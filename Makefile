# movies-api Makefile — minimal wrapper around the inner-loop steps
# documented in IMPL-README.md. Each target is independently runnable.

VERSION ?= 0.3.0
IMAGE   ?= movies-api:$(VERSION)
TARBALL ?= /tmp/movies-api-$(VERSION).tar
KCTL    ?= sudo k3s kubectl
SRC     ?= src

.PHONY: help build test image import deploy verify verify-ingress undeploy clean

help:
	@echo "Targets:"
	@echo "  build          - go build ./..."
	@echo "  test           - go test -race ./..."
	@echo "  image          - docker build -t $(IMAGE) ./src"
	@echo "  import         - docker save | k3s ctr images import"
	@echo "  deploy         - kustomize build | kubectl apply"
	@echo "  verify         - port-forward + curl all 3 endpoints (Service)"
	@echo "  verify-ingress - curl all 3 endpoints via Traefik on http://localhost"
	@echo "  undeploy       - delete dev overlay"
	@echo "  clean          - go clean + rm tarball"

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
	kustomize build deploy/k8s/overlays/dev | $(KCTL) apply -f -
	$(KCTL) -n movies rollout status deploy/movies-api --timeout=60s

undeploy:
	kustomize build deploy/k8s/overlays/dev | $(KCTL) delete -f - --ignore-not-found

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
		echo "OK: all three endpoints verified via Traefik on http://localhost"; \
	'

clean:
	cd $(SRC) && go clean
	rm -f $(TARBALL)
