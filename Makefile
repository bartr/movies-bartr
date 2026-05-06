# movies-api Makefile — minimal wrapper around the inner-loop steps
# documented in IMPL-README.md. Each target is independently runnable.

VERSION ?= 0.1.0
IMAGE   ?= movies-api:$(VERSION)
TARBALL ?= /tmp/movies-api-$(VERSION).tar
KCTL    ?= sudo k3s kubectl

.PHONY: help build test image import deploy verify undeploy clean

help:
	@echo "Targets:"
	@echo "  build    - go build ./..."
	@echo "  test     - go test -race ./..."
	@echo "  image    - docker build -t $(IMAGE) ."
	@echo "  import   - docker save | k3s ctr images import"
	@echo "  deploy   - kustomize build | kubectl apply"
	@echo "  verify   - port-forward + curl all 3 endpoints"
	@echo "  undeploy - delete dev overlay"
	@echo "  clean    - go clean + rm tarball"

build:
	go build ./...

test:
	go test -race ./...

image:
	docker build --build-arg VERSION=$(VERSION) -t $(IMAGE) .

import: image
	docker save $(IMAGE) -o $(TARBALL)
	sudo k3s ctr images import $(TARBALL)
	@echo "Imported $(IMAGE) into k3s containerd"

deploy:
	kustomize build deploy/k8s/overlays/dev | $(KCTL) apply -f -
	$(KCTL) -n movies rollout status deploy/movies-api --timeout=60s

verify:
	@bash -c '\
		set -e; \
		$(KCTL) -n movies port-forward svc/movies-api 18080:8080 >/tmp/movies-pf.log 2>&1 & \
		PF_PID=$$!; \
		trap "kill $$PF_PID 2>/dev/null || true" EXIT; \
		for i in $$(seq 1 30); do \
		  if curl -sSf -o /dev/null http://127.0.0.1:18080/version 2>/dev/null; then break; fi; \
		  sleep 0.5; \
		done; \
		echo "--- /version ---"; \
		V=$$(curl -sS -i http://127.0.0.1:18080/version); echo "$$V"; \
		echo "$$V" | grep -q "Content-Type: text/plain; charset=utf-8" || (echo "FAIL: /version content-type"; exit 1); \
		echo "$$V" | tail -n1 | grep -qx "$(VERSION)" || (echo "FAIL: /version body != $(VERSION)"; exit 1); \
		echo "--- /healthz ---"; \
		curl -sS http://127.0.0.1:18080/healthz | tee /dev/stderr | grep -qx "pass" || (echo "FAIL: /healthz"; exit 1); \
		echo "--- /readyz ---"; \
		curl -sS http://127.0.0.1:18080/readyz | tee /dev/stderr | grep -qx "pass" || (echo "FAIL: /readyz"; exit 1); \
		echo "OK: all three endpoints verified in-cluster"; \
	'

undeploy:
	kustomize build deploy/k8s/overlays/dev | $(KCTL) delete -f - --ignore-not-found

clean:
	go clean
	rm -f $(TARBALL)
