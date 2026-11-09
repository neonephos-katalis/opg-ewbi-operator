# Image URL to use all building/pushing image targets
IMG ?= ghcr.io/neonephos-katalis/opg-ewbi-operator:neonephos
IMG_DEBUG ?= ghcr.io/neonephos-katalis/opg-ewbi-operator:dev-callback-1.1.3
IMG_API ?= ghcr.io/neonephos-katalis/opg-ewbi-api:neonephos
PLATFORM ?= linux/amd64
# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = 1.31.0

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# CONTAINER_TOOL defines the container tool to be used for building images.
# Be aware that the target commands are only tested with Docker which is
# scaffolded by default. However, you might want to replace it to use other
# tools. (i.e. podman)
CONTAINER_TOOL ?= docker

# Setting SHELL to bash allows bash commands to be executed by recipes.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL ?= /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

.PHONY: all
all: build

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk command is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: manifests
manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases

.PHONY: generate
generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: test
test: manifests generate fmt vet envtest ## Run tests.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" go test $$(go list ./... | grep -v /e2e) -coverprofile cover.out

# To use a different vendor for e2e tests, modify the setup under 'tests/e2e'.
# The default setup assumes Kind is pre-installed and builds/loads the Manager Docker image locally.
# Prometheus and CertManager are installed by default; skip with:
# - PROMETHEUS_INSTALL_SKIP=true
# - CERT_MANAGER_INSTALL_SKIP=true
.PHONY: test-e2e
test-e2e: manifests generate fmt vet ## Run the e2e tests. Expected an isolated environment using Kind.
	@command -v kind >/dev/null 2>&1 || { \
		echo "Kind is not installed. Please install Kind manually."; \
		exit 1; \
	}
	@kind get clusters | grep -q 'kind' || { \
		echo "No Kind cluster is running. Please start a Kind cluster before running the e2e tests."; \
		exit 1; \
	}
	go test ./test/e2e/ -v -ginkgo.v

.PHONY: lint
lint: golangci-lint ## Run golangci-lint linter
	$(GOLANGCI_LINT) run

.PHONY: lint-fix
lint-fix: golangci-lint ## Run golangci-lint linter and perform fixes
	$(GOLANGCI_LINT) run --fix

##@ E2E workflows (local)

# Run the GitHub e2e workflows locally with act (https://nektosact.com).
# Requirements on the host: docker and act. Each workflow creates the kind clusters
# partnerop/originatingop and a local registry on the host docker daemon, so the
# workflows are run one at a time. GITHUB_TOKEN is only needed to fetch private Go
# modules (e.g. GITHUB_TOKEN=$(gh auth token) make e2e-rest-local).
ACT ?= act
ACT_IMAGE ?= catthehacker/ubuntu:act-latest
ACT_EVENT ?= workflow_dispatch
ACT_FLAGS ?= -P ubuntu-latest=$(ACT_IMAGE) --container-options "--privileged" --network host --pull=false

.PHONY: act-check
act-check: ## Check that act and docker are available and pull the act runner image if missing
	@command -v $(ACT) >/dev/null 2>&1 || { echo "act is not installed: see https://nektosact.com/installation/"; exit 1; }
	@docker info >/dev/null 2>&1 || { echo "docker is not reachable"; exit 1; }
	@docker image inspect $(ACT_IMAGE) >/dev/null 2>&1 || { \
		echo "Pulling the act runner image $(ACT_IMAGE) (first run only, ~1.5 GB)..."; \
		docker pull $(ACT_IMAGE); \
	}

.PHONY: e2e-local-clean
e2e-local-clean: ## Remove leftovers of previous local e2e runs (kind clusters, local registry, act containers)
	@echo "==> Removing kind clusters partnerop and originatingop"
	@for c in partnerop originatingop; do \
		if command -v kind >/dev/null 2>&1; then kind delete cluster --name $$c >/dev/null 2>&1 || true; fi; \
		docker rm -f $$c-control-plane >/dev/null 2>&1 || true; \
	done
	@echo "==> Removing the local registry kind-registry"
	@docker rm -f kind-registry >/dev/null 2>&1 || true
	@echo "==> Removing leftover act runner containers"
	@ids=$$(docker ps -aq --filter "name=^act-"); \
		if [ -n "$$ids" ]; then docker rm -f $$ids >/dev/null; echo "    removed: $$(echo $$ids | wc -w) container(s)"; else echo "    none"; fi
	@echo "==> Remaining kind/act containers (should be empty):"
	@docker ps -a --format '    {{.Names}}' | grep -E 'control-plane|^    act-|kind-registry' || echo "    none"
	@if [ -r /proc/sys/fs/inotify/max_user_instances ] && [ "$$(cat /proc/sys/fs/inotify/max_user_instances)" -lt 8192 ]; then \
		echo ""; \
		echo "WARNING: inotify limits are low on this host, kind clusters may fail to start. Run:"; \
		echo "  sudo sysctl -w fs.inotify.max_user_instances=8192 fs.inotify.max_user_watches=1048576"; \
	fi

.PHONY: e2e-k8s-local
e2e-k8s-local: act-check e2e-local-clean ## Run the K8s e2e workflow (.github/workflows/e2e-federation-k8s.yml) locally with act
	$(ACT) $(ACT_EVENT) -W .github/workflows/e2e-federation-k8s.yml $(ACT_FLAGS) -s GITHUB_TOKEN="$${GITHUB_TOKEN:-}"

.PHONY: e2e-rest-local
e2e-rest-local: act-check e2e-local-clean ## Run the REST e2e workflow (.github/workflows/e2e-federation-rest.yml) locally with act
	$(ACT) $(ACT_EVENT) -W .github/workflows/e2e-federation-rest.yml $(ACT_FLAGS) -s GITHUB_TOKEN="$${GITHUB_TOKEN:-}"

.PHONY: e2e-local
e2e-local: act-check ## Run the K8s and the REST e2e workflows locally, one after the other, and print a summary
	@k8s=PASSED; rest=PASSED; \
	$(MAKE) --no-print-directory e2e-k8s-local  || k8s=FAILED; \
	$(MAKE) --no-print-directory e2e-rest-local || rest=FAILED; \
	echo ""; \
	echo "==================== E2E SUMMARY ===================="; \
	echo "  K8s  (e2e-federation-k8s.yml)  : $$k8s"; \
	echo "  REST (e2e-federation-rest.yml) : $$rest"; \
	echo "====================================================="; \
	[ "$$k8s" = PASSED ] && [ "$$rest" = PASSED ]

##@ Build

.PHONY: build
build: manifests generate fmt vet ## Build manager binary.
	go build -o bin/manager cmd/main.go

.PHONY: build-api
build-api: fmt vet ## Build API server binary.
	go build -o bin/api ./cmd/api

.PHONY: run
run: manifests generate fmt vet ## Run a controller from your host.
	go run ./cmd/main.go

.PHONY: run-api
run-api: fmt vet ## Run the API server locally (requires CONTROLLER_NAMESPACE, CAMARA_* env vars).
	go run ./cmd/api

# If you wish to build the manager image targeting other platforms you can use the --platform flag.
# (i.e. docker build --platform linux/arm64). However, you must enable docker buildKit for it.
# More info: https://docs.docker.com/develop/develop-images/build_enhancements/
.PHONY: docker-build
docker-build: docker-build-controller ## Build docker image for the manager

.PHONY: docker-build-controller
docker-build-controller:
	DOCKER_BUILDKIT=1 $(CONTAINER_TOOL) build --platform=${PLATFORM} -t ${IMG} --secret id=netrc,src=$(HOME)/.netrc .

.PHONY: docker-build-debug
docker-build-debug: docker-build-controller-debug ## Build docker image for the manager

.PHONY: docker-build-controller-debug
docker-build-controller-debug:
	DOCKER_BUILDKIT=1 $(CONTAINER_TOOL) build --target debug --platform=${PLATFORM} -t ${IMG_DEBUG} --secret id=netrc,src=$(HOME)/.netrc .

.PHONY: docker-push
docker-push: docker-push-controller

.PHONY: docker-push-controller
docker-push-controller: ## Push docker image with the manager.
	$(CONTAINER_TOOL) push ${IMG}

.PHONY: docker-push-debug
docker-push-debug:
	$(CONTAINER_TOOL) push ${IMG_DEBUG}

.PHONY: docker-build-api
docker-build-api: ## Build docker image for the API server.
	DOCKER_BUILDKIT=1 $(CONTAINER_TOOL) build --platform=${PLATFORM} -t ${IMG_API} -f Dockerfile.api --secret id=netrc,src=$(HOME)/.netrc .

.PHONY: docker-push-api
docker-push-api: ## Push docker image for the API server.
	$(CONTAINER_TOOL) push ${IMG_API}

.PHONY: docker-itest
docker-itest: ## Run itests
	 DOCKER_BUILDKIT=1 $(CONTAINER_TOOL) build --target test -t ${IMG}-test --secret id=netrc,src=$(HOME)/.netrc .

.PHONY: docker-build-apigen
docker-build-apigen: ## Build the API code generator image.
	DOCKER_BUILDKIT=1 $(CONTAINER_TOOL) build --platform=${PLATFORM} -t opg-ewbi-apigen -f Dockerfile.apigen .

.PHONY: apigen
apigen: docker-build-apigen ## Generate API models, server, and client code from OpenAPI spec.
	$(CONTAINER_TOOL) run --rm -v $(shell pwd)/api/ewbi:/api/ewbi opg-ewbi-apigen

.PHONY: swagger
swagger: ## Start Swagger UI for the EWBI API spec on http://localhost:8082
	$(CONTAINER_TOOL) run --rm -p 8082:8080 \
		-v $(shell pwd)/api/ewbi/swagger.yaml:/usr/share/nginx/html/doc/swagger.yaml \
		-e API_URL=doc/swagger.yaml \
		swaggerapi/swagger-ui

# PLATFORMS defines the target platforms for the manager image be built to provide support to multiple
# architectures. (i.e. make docker-buildx IMG=myregistry/mypoperator:0.0.1). To use this option you need to:
# - be able to use docker buildx. More info: https://docs.docker.com/build/buildx/
# - have enabled BuildKit. More info: https://docs.docker.com/develop/develop-images/build_enhancements/
# - be able to push the image to your registry (i.e. if you do not set a valid value via IMG=<myregistry/image:<tag>> then the export will fail)
# To adequately provide solutions that are compatible with multiple platforms, you should consider using this option.
PLATFORMS ?= linux/arm64,linux/amd64,linux/s390x,linux/ppc64le
.PHONY: docker-buildx
docker-buildx: ## Build and push docker image for the manager for cross-platform support
	# copy existing Dockerfile and insert --platform=${BUILDPLATFORM} into Dockerfile.cross, and preserve the original Dockerfile
	sed -e '1 s/\(^FROM\)/FROM --platform=\$$\{BUILDPLATFORM\}/; t' -e ' 1,// s//FROM --platform=\$$\{BUILDPLATFORM\}/' Dockerfile > Dockerfile.cross
	- $(CONTAINER_TOOL) buildx create --name opg-ewbi-operator-builder
	$(CONTAINER_TOOL) buildx use opg-ewbi-operator-builder
	- $(CONTAINER_TOOL) buildx build --secret id=netrc,src=$(HOME)/.netrc --push --platform=$(PLATFORMS) --tag ${IMG} -f Dockerfile.cross .
	- $(CONTAINER_TOOL) buildx rm opg-ewbi-operator-builder
	rm Dockerfile.cross

.PHONY: build-installer
build-installer: manifests generate kustomize ## Generate a consolidated YAML with CRDs and deployment.
	mkdir -p dist
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default > dist/install.yaml

##@ Deployment

ifndef ignore-not-found
  ignore-not-found = false
endif

.PHONY: install
install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | $(KUBECTL) apply -f -

.PHONY: uninstall
uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/crd | $(KUBECTL) delete --ignore-not-found=$(ignore-not-found) -f -

.PHONY: deploy
deploy: manifests kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default | $(KUBECTL) apply -f -

.PHONY: undeploy
undeploy: kustomize ## Undeploy controller from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/default | $(KUBECTL) delete --ignore-not-found=$(ignore-not-found) -f -

##@ Dependencies

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
KUBECTL ?= kubectl
KUSTOMIZE ?= $(LOCALBIN)/kustomize
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
ENVTEST ?= $(LOCALBIN)/setup-envtest
GOLANGCI_LINT = $(LOCALBIN)/golangci-lint

## Tool Versions
KUSTOMIZE_VERSION ?= v5.5.0
CONTROLLER_TOOLS_VERSION ?= v0.16.5
ENVTEST_VERSION ?= release-0.19
GOLANGCI_LINT_VERSION ?= v1.61.0

.PHONY: kustomize
kustomize: $(KUSTOMIZE) ## Download kustomize locally if necessary.
$(KUSTOMIZE): $(LOCALBIN)
	$(call go-install-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v5,$(KUSTOMIZE_VERSION))

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary.
$(CONTROLLER_GEN): $(LOCALBIN)
	$(call go-install-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen,$(CONTROLLER_TOOLS_VERSION))

.PHONY: envtest
envtest: $(ENVTEST) ## Download setup-envtest locally if necessary.
$(ENVTEST): $(LOCALBIN)
	$(call go-install-tool,$(ENVTEST),sigs.k8s.io/controller-runtime/tools/setup-envtest,$(ENVTEST_VERSION))

.PHONY: golangci-lint
golangci-lint: $(GOLANGCI_LINT) ## Download golangci-lint locally if necessary.
$(GOLANGCI_LINT): $(LOCALBIN)
	$(call go-install-tool,$(GOLANGCI_LINT),github.com/golangci/golangci-lint/cmd/golangci-lint,$(GOLANGCI_LINT_VERSION))

# go-install-tool will 'go install' any package with custom target and name of binary, if it doesn't exist
# $1 - target path with name of binary
# $2 - package url which can be installed
# $3 - specific version of package
define go-install-tool
@[ -f "$(1)-$(3)" ] || { \
set -e; \
package=$(2)@$(3) ;\
echo "Downloading $${package}" ;\
rm -f $(1) || true ;\
GOBIN=$(LOCALBIN) go install $${package} ;\
mv $(1) $(1)-$(3) ;\
} ;\
ln -sf $(1)-$(3) $(1)
endef


## HELM

.PHONY: helm
helm: base-chart crd-chart ## build the operator's helm chart

define CLUSTER_WIDE_RBAC_SCRIPT
set -euo pipefail
role=dist/chart/templates/rbac/role.yaml
binding=dist/chart/templates/rbac/role_binding.yaml
if [ "$(grep -c '^kind: Role$' "$role")" != 1 ] || [ "$(grep -c '^rules:$' "$role")" != 2 ]; then
  echo "error: expected one ClusterRole and one Role in $role" >&2
  exit 1
fi
if ! grep -q '^kind: RoleBinding$' "$binding" || ! grep -q '^  kind: Role$' "$binding" || ! grep -q 'name: opg-ewbi-manager-rolebinding$' "$binding"; then
  echo "error: expected the manager RoleBinding in $binding" >&2
  exit 1
fi

awk '
  { lines[NR] = $0 }
  /^kind: Role$/ { role = NR }
  END {
    start = role
    while (start > 1 && lines[start] != "---") start--
    end = role
    while (end <= NR && lines[end] != "rules:") end++
    for (i = 1; i <= NR; i++) {
      if (i == start) print "{{- if not .Values.clusterWide }}"
      print lines[i]
      if (i == end) print "{{- end }}"
      if (lines[i] == "rules:" && i < start) {
        print "{{- if .Values.clusterWide }}"
        print "- apiGroups:"
        print "  - \"\""
        print "  resources:"
        print "  - namespaces"
        print "  verbs:"
        print "  - create"
        print "{{- end }}"
      }
    }
  }
' "$role" > "$role.tmp"
cat "$role.tmp" > "$role"
rm "$role.tmp"
sed -i \
  -e 's/^kind: RoleBinding$/kind: {{ if .Values.clusterWide }}ClusterRoleBinding{{ else }}RoleBinding{{ end }}/' \
  -e 's/^  kind: Role$/  kind: {{ if .Values.clusterWide }}ClusterRole{{ else }}Role{{ end }}/' \
  -e 's/name: opg-ewbi-manager-rolebinding$/name: opg-ewbi-manager-rolebinding{{ if .Values.clusterWide }}-{{ .Release.Namespace }}{{ end }}/' \
  "$binding"
endef
export CLUSTER_WIDE_RBAC_SCRIPT := $(value CLUSTER_WIDE_RBAC_SCRIPT)

.PHONY: base-chart
base-chart: manifests ## build the operator's helm chart
	kubebuilder edit  --plugins=helm/v1-alpha
# 	fix the namespace in the role.yaml file otherwise it hardcodes the namespace to "foo"
	sed -i 's/namespace: foo/namespace: {{ .Release.Namespace }}/g' dist/chart/templates/rbac/role.yaml
	sed -i 's/name: opg-ewbi-manager-role$$/name: opg-ewbi-manager-role-{{ .Release.Namespace }}/' dist/chart/templates/rbac/role.yaml
	sed -i 's/name: opg-ewbi-manager-role$$/name: opg-ewbi-manager-role-{{ .Release.Namespace }}/g' dist/chart/templates/rbac/role_binding.yaml
	bash -c "$$CLUSTER_WIDE_RBAC_SCRIPT"
	helm lint dist/chart


chart_yaml := dist/chart/Chart.yaml
crd_chart_yaml := dist/chart-crd/Chart.yaml

.PHONY: crd-chart
crd-chart: base-chart ## build the crd's helm chart
	mkdir -p dist/chart-crd
	cp -r config/crd/bases/ dist/chart-crd/templates
	if [ ! -e $(crd_chart_yaml) ]; then \
		cp -n $(chart_yaml) $(crd_chart_yaml);  \
		sed -i'' 's/^\(name:\).*/\1 opg-ewbi-crd/' $(crd_chart_yaml);  \
		sed -i'' 's/^\(description:\).*/\1 A Helm chart to distribute the opg-ewbi-crd/' $(crd_chart_yaml); \
	fi
	helm lint dist/chart-crd
