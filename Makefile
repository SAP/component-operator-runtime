# Set shell to bash
SHELL = /usr/bin/env bash
.SHELLFLAGS = -o pipefail -ec

# K8s version used by envtest
ENVTEST_K8S_VERSION = 1.35.0

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
test: generate fmt vet envtest ## Run tests
	KUBEBUILDER_ASSETS="$(LOCALBIN)/k8s/current" go test ./internal/... ./pkg/... -coverprofile cover.out

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

.PHONY: setup-envtest
setup-envtest: $(LOCALBIN) ## Install setup-envtest
	@go mod download sigs.k8s.io/controller-runtime/tools/setup-envtest && \
	VERSION=$$(go list -m -f '{{.Version}}' sigs.k8s.io/controller-runtime/tools/setup-envtest) && \
	if [ ! -L $(LOCALBIN)/setup-envtest ] || [ "$$(readlink $(LOCALBIN)/setup-envtest)" != "setup-envtest-$$VERSION" ]; then \
	echo "Installing setup-envtest $$VERSION" && \
	rm -f $(LOCALBIN)/setup-envtest && \
	GOBIN=$(LOCALBIN) go install $$(go list -m -f '{{.Dir}}' sigs.k8s.io/controller-runtime/tools/setup-envtest) && \
	mv $(LOCALBIN)/setup-envtest $(LOCALBIN)/setup-envtest-$$VERSION && \
	ln -s setup-envtest-$$VERSION $(LOCALBIN)/setup-envtest; \
	fi

.PHONY: envtest
envtest: setup-envtest ## Install envtest binaries
	@ENVTESTDIR=$$($(LOCALBIN)/setup-envtest use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path) && \
	chmod -R u+w $$ENVTESTDIR && \
	rm -f $(LOCALBIN)/k8s/current && \
	ln -s $$ENVTESTDIR $(LOCALBIN)/k8s/current

# Set the year for SPDX header updates (default: current year)
YEAR ?= $(shell date +%Y)

.PHONY: update-header-year
update-header-year:
    # Go + TXT
	@find . -type f \( -name "*.go" -o -name "*.txt" \) -exec sed -i \
	's/^SPDX-FileCopyrightText: [0-9]\{4\}\( SAP SE or an SAP affiliate company and [^"]\+ contributors\)/SPDX-FileCopyrightText: $(YEAR)\1/' {} +

    # TOML
	@find . -type f -name "*.toml" -exec sed -i \
	's/^SPDX-FileCopyrightText = "[0-9]\{4\}\( SAP SE or an SAP affiliate company and [^"]\+ contributors\)"/SPDX-FileCopyrightText = "$(YEAR)\1"/' {} +