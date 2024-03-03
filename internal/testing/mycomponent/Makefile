SHELL = /usr/bin/env bash
.SHELLFLAGS = -o pipefail -ec
BASEDIR = ../../..

.PHONY: all
all: generate manifests

.PHONY: generate
generate:
	$(BASEDIR)/bin/controller-gen object:headerFile="$(BASEDIR)/hack/boilerplate.go.txt" paths="./api/..."

.PHONY: manifests
manifests:
	$(BASEDIR)/bin/controller-gen crd paths="./api/..." output:crd:artifacts:config=crds