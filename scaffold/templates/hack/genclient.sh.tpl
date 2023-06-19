#!/usr/bin/env bash

set -eo pipefail

export GOROOT=$(go env GOROOT)

BASEDIR=$(realpath $(dirname "$0")/..)
TEMPDIR=$BASEDIR/tmp/gen
trap 'rm -rf "$TEMPDIR"' EXIT
mkdir -p "$TEMPDIR"

mkdir -p "$TEMPDIR"/apis/{{ .groupName }}
ln -s "$BASEDIR"/api/{{ .groupVersion }} "$TEMPDIR"/apis/{{ .groupName }}/{{ .groupVersion }}

"$BASEDIR"/bin/client-gen \
  --clientset-name versioned \
  --input-base "" \
  --input {{ .goModule }}/tmp/gen/apis/{{ .groupName }}/{{ .groupVersion }} \
  --go-header-file "$BASEDIR"/hack/boilerplate.go.txt \
  --output-package {{ .goModule }}/pkg/client/clientset \
  --output-base "$TEMPDIR"/pkg/client \
  --plural-exceptions {{ .kind }}:{{ .resource }}

"$BASEDIR"/bin/lister-gen \
  --input-dirs {{ .goModule }}/tmp/gen/apis/{{ .groupName }}/{{ .groupVersion }} \
  --go-header-file "$BASEDIR"/hack/boilerplate.go.txt \
  --output-package {{ .goModule }}/pkg/client/listers \
  --output-base "$TEMPDIR"/pkg/client \
  --plural-exceptions {{ .kind }}:{{ .resource }}

"$BASEDIR"/bin/informer-gen \
  --input-dirs {{ .goModule }}/tmp/gen/apis/{{ .groupName }}/{{ .groupVersion }} \
  --versioned-clientset-package {{ .goModule }}/pkg/client/clientset/versioned \
  --listers-package {{ .goModule }}/pkg/client/listers \
  --go-header-file "$BASEDIR"/hack/boilerplate.go.txt \
  --output-package {{ .goModule }}/pkg/client/informers \
  --output-base "$TEMPDIR"/pkg/client \
  --plural-exceptions {{ .kind }}:{{ .resource }}

find "$TEMPDIR"/pkg/client -name "*.go" -exec \
  sed -i -e "s#{{ .goModule | regexQuoteMeta }}/tmp/gen/apis/{{ .groupName | regexQuoteMeta }}/{{ .groupVersion | regexQuoteMeta }}#{{ .goModule }}/api/{{ .groupVersion }}#g" \
  {} +

rm -rf "$BASEDIR"/pkg/client
mv "$TEMPDIR"/pkg/client/{{ .goModule }}/pkg/client "$BASEDIR"/pkg

cd "$BASEDIR"
go fmt ./pkg/client/...
go vet ./pkg/client/...
