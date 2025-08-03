# Set shell to bash
SHELL = /usr/bin/env bash
.SHELLFLAGS = -o pipefail -ec

##@ Development

.PHONY: generate
generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(LOCALBIN)/controller-gen object:headerFile="hack/boilerplate.go.txt" paths="./pkg/..." paths="./internal/..."

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: test
test:
	go test ./internal/... ./pkg/...

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