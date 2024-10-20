# Image URL to use all building/pushing image targets
IMG ?= {{ .image }}
# K8s version used by envtest
ENVTEST_K8S_VERSION = {{ .envtestKubernetesVersion }}

# Set shell to bash
SHELL = /usr/bin/env bash
.SHELLFLAGS = -o pipefail -ec

.PHONY: all
all: build

##@ General

.PHONY: help
help: ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: manifests
manifests: controller-gen ## Generate CustomResourceDefinition objects
	$(CONTROLLER_GEN) crd paths="./api/..." output:crd:artifacts:config=crds ;\
	test ! -d chart || test -e chart/crds || ln -s ../crds chart/crds

.PHONY: generate
generate: generate-deepcopy ## Generate required code pieces

.PHONY: generate-deepcopy
generate-deepcopy: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./api/..."

.PHONY: generate-client
generate-client: ## Generate typed client
	./hack/genclient.sh

.PHONY: fmt
fmt: ## Run go fmt against code
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code
	go vet ./...

##@ Testing

.PHONY: test
test: manifests generate-deepcopy fmt vet envtest ## Run tests
	KUBEBUILDER_ASSETS="$(LOCALBIN)/k8s/current" go test ./... -coverprofile cover.out

##@ Build

.PHONY: build
build: generate-deepcopy fmt vet ## Build manager binary
	go build -o bin/manager main.go

.PHONY: run
run: manifests generate-deepcopy fmt vet ## Run a controller from your host
	go run ./main.go

# Build docker image in current architecture and tag it as ${IMG}
.PHONY: docker-build
docker-build: ## Build docker image with the manager
	docker build -t ${IMG} .

# Push docker image to the target specified in ${IMG}
.PHONY: docker-push
docker-push: ## Push docker image with the manager
	docker push ${IMG}

# Build and push docker image for all given platforms
PLATFORMS ?= linux/arm64,linux/amd64,linux/s390x,linux/ppc64le
.PHONY: docker-buildx
docker-buildx: ## Build and push docker image for the manager for cross-platform support
	- docker buildx create --name project-v3-builder
	docker buildx use project-v3-builder
	- docker buildx build --push --platform=$(PLATFORMS) --tag ${IMG} .
	- docker buildx rm project-v3-builder

##@ Build Dependencies

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
SETUP_ENVTEST ?= $(LOCALBIN)/setup-envtest

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Install controller-gen
$(CONTROLLER_GEN): $(LOCALBIN)
	go mod download sigs.k8s.io/controller-tools
	GOBIN=$(LOCALBIN) go install $$(go list -m -f '{{`{{`}}.Dir{{`}}`}}' sigs.k8s.io/controller-tools)/cmd/controller-gen

.PHONY: setup-envtest
setup-envtest: $(SETUP_ENVTEST) ## Install setup-envtest
$(SETUP_ENVTEST): $(LOCALBIN)
	go mod download sigs.k8s.io/controller-runtime
	GOBIN=$(LOCALBIN) go install $$(go list -m -f '{{`{{`}}.Dir{{`}}`}}' sigs.k8s.io/controller-runtime/tools/setup-envtest)

.PHONY: envtest
envtest: setup-envtest ## Install envtest binaries
	ENVTESTDIR=$$($(SETUP_ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path) ;\
	chmod -R u+w $$ENVTESTDIR ;\
	rm -f $(LOCALBIN)/k8s/current ;\
	ln -s $$ENVTESTDIR $(LOCALBIN)/k8s/current
