#!/usr/bin/env bash

set -eo pipefail

BASEDIR=$(realpath "$(dirname "$0")"/..)

export GOBIN=$BASEDIR/bin
mkdir -p "$GOBIN"

go mod download k8s.io/code-generator
CODEGEN_DIR=$(go list -m -f '{{`{{`}}.Dir{{`}}`}}' k8s.io/code-generator)
go install "$CODEGEN_DIR"/cmd/*

TEMPDIR=$BASEDIR/tmp/gen
trap 'rm -rf "$TEMPDIR"' EXIT
mkdir -p "$TEMPDIR"

mkdir -p "$TEMPDIR"/apis/{{ .groupName }}
ln -s "$BASEDIR"/api/{{ .groupVersion }} "$TEMPDIR"/apis/{{ .groupName }}/{{ .groupVersion }}

"$GOBIN"/client-gen \
  --clientset-name versioned \
  --input-base "$TEMPDIR"/apis \
  --input {{ .groupName }}/{{ .groupVersion }} \
  --go-header-file "$BASEDIR"/hack/boilerplate.go.txt \
  --output-pkg {{ .goModule }}/pkg/client/clientset \
  --output-dir "$TEMPDIR"/pkg/client/clientset \
  --plural-exceptions {{ .kind }}:{{ .resource }}

"$GOBIN"/lister-gen \
  --go-header-file "$BASEDIR"/hack/boilerplate.go.txt \
  --output-pkg {{ .goModule }}/pkg/client/listers \
  --output-dir "$TEMPDIR"/pkg/client/listers \
  --plural-exceptions {{ .kind }}:{{ .resource }} \
  {{ .goModule }}/tmp/gen/apis/{{ .groupName }}/{{ .groupVersion }}

"$GOBIN"/informer-gen \
  --versioned-clientset-package {{ .goModule }}/pkg/client/clientset/versioned \
  --listers-package {{ .goModule }}/pkg/client/listers \
  --go-header-file "$BASEDIR"/hack/boilerplate.go.txt \
  --output-pkg {{ .goModule }}/pkg/client/informers \
  --output-dir "$TEMPDIR"/pkg/client/informers \
  --plural-exceptions {{ .kind }}:{{ .resource }} \
  {{ .goModule }}/tmp/gen/apis/{{ .groupName }}/{{ .groupVersion }}

find "$TEMPDIR"/pkg/client -name "*.go" -exec \
  perl -pi -e "s#{{ .goModule | regexQuoteMeta }}/tmp/gen/apis/{{ .groupName | regexQuoteMeta }}/{{ .groupVersion | regexQuoteMeta }}#{{ .goModule }}/api/{{ .groupVersion }}#g" \
  {} +

rm -rf "$BASEDIR"/pkg/client
mv "$TEMPDIR"/pkg/client "$BASEDIR"/pkg

cd "$BASEDIR"
go mod tidy
go fmt ./pkg/client/...
go vet ./pkg/client/...
