# Set shell to bash
SHELL = /usr/bin/env bash
.SHELLFLAGS = -o pipefail -ec

.PHONY: all
all: scaffold clm

##@ General

.PHONY: help
help: ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: generate
generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations
	$(LOCALBIN)/controller-gen object:headerFile="hack/boilerplate.go.txt" paths="./pkg/..." paths="./internal/..."

.PHONY: fmt
fmt: ## Run go fmt against code
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code
	go vet ./...

.PHONY: test
test: generate fmt vet ## Run tests
	go test ./internal/... ./pkg/...

##@ Build

.PHONY: scaffold
scaffold: generate fmt vet ## Build scaffold
	GIT_BRANCH=$$(git rev-parse --abbrev-ref @) && \
	GIT_COMMIT=$$(git rev-parse @) && \
	if out=$$(git status --porcelain) && [ -z "$$out" ]; then GIT_TREE_STATE=clean; else GIT_TREE_STATE=dirty; fi && \
	LDFLAGS=" -X \"github.com/sap/component-operator-runtime/internal/version.version=$$GIT_BRANCH\" \
	  -X \"github.com/sap/component-operator-runtime/internal/version.gitCommit=$$GIT_COMMIT\" \
	  -X \"github.com/sap/component-operator-runtime/internal/version.gitTreeState=$$GIT_TREE_STATE\"" && \
	CGO_ENABLED=0 go build -ldflags "$$LDFLAGS" -o ./bin/scaffold ./scaffold

.PHONY: clm
clm: generate fmt vet ## Build clm
	GIT_BRANCH=$$(git rev-parse --abbrev-ref @) && \
	GIT_COMMIT=$$(git rev-parse @) && \
	if out=$$(git status --porcelain) && [ -z "$$out" ]; then GIT_TREE_STATE=clean; else GIT_TREE_STATE=dirty; fi && \
	LDFLAGS=" -X \"github.com/sap/component-operator-runtime/internal/version.version=$$GIT_BRANCH\" \
	  -X \"github.com/sap/component-operator-runtime/internal/version.gitCommit=$$GIT_COMMIT\" \
	  -X \"github.com/sap/component-operator-runtime/internal/version.gitTreeState=$$GIT_TREE_STATE\"" && \
	CGO_ENABLED=0 go build -ldflags "$$LDFLAGS" -o ./bin/clm ./clm

##@ Build Dependencies

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

.PHONY: controller-gen
controller-gen: $(LOCALBIN) ## Install controller-gen
	@go mod download sigs.k8s.io/controller-tools && \
	VERSION=$$(go list -m -f '{{.Version}}' sigs.k8s.io/controller-tools) && \
	if [ ! -L $(LOCALBIN)/controller-gen ] || [ "$$(readlink $(LOCALBIN)/controller-gen)" != "controller-gen-$$VERSION" ]; then \
	echo "Installing controller-gen $$VERSION" && \
	rm -f $(LOCALBIN)/controller-gen && \
	GOBIN=$(LOCALBIN) go install $$(go list -m -f '{{.Dir}}' sigs.k8s.io/controller-tools)/cmd/controller-gen && \
	mv $(LOCALBIN)/controller-gen $(LOCALBIN)/controller-gen-$$VERSION && \
	ln -s controller-gen-$$VERSION $(LOCALBIN)/controller-gen; \
	fi