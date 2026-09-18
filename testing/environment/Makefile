SHELL = /usr/bin/env bash
.SHELLFLAGS = -o pipefail -ec
BASEDIR = ../..

.PHONY: all
all: generate manifests

.PHONY: generate
generate:
	$(BASEDIR)/bin/controller-gen object:headerFile="$(BASEDIR)/hack/boilerplate.go.txt" paths="./apis/..."

.PHONY: manifests
manifests:
	$(BASEDIR)/bin/controller-gen crd paths="./apis/..." output:crd:artifacts:config=crds