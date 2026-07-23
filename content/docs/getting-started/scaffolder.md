---
title: "Using the Scaffolder"
linkTitle: "Using the Scaffolder"
weight: 10
type: "docs"
description: >
  Bootstrap a component operator with the scaffolding tool
---

This guide walks through bootstrapping a new component operator with the
[Scaffolder](../../tools/scaffolder/). As a running example we will build an operator
that manages resources of type `Gizmo` in the API group `tools.acme.io` (version
`v1alpha1`).

## Prerequisites

- A Kubernetes cluster to deploy to later. If you do not have one, you can quickly
  spin up a local [kind](https://kind.sigs.k8s.io/) cluster:

  ```bash
  kind create cluster
  ```

- The `scaffold` binary. Download the latest release from the
  [GitHub Releases](https://github.com/SAP/component-operator-runtime/releases) page
  and make sure it is on your `PATH`. See the
  [Scaffolder tool reference](../../tools/scaffolder/) for the full list of options.

## Generating the skeleton

First, create an empty working directory for the project:

```bash
mkdir gizmo-operator
```

Change into it:

```bash
cd gizmo-operator
```

Then run the scaffolder, pointing it at the current directory (`.`) as the output
target:

```bash
scaffold \
  --group-name tools.acme.io \
  --group-version v1alpha1 \
  --kind Gizmo \
  --operator-name gizmo-operator.acme.io \
  --go-module acme.io/gizmo-operator \
  --image ghcr.io/acme/gizmo-operator:latest \
  .
```

This generates a complete, ready-to-build operator project in the current directory,
pre-wired to component-operator-runtime for managing `Gizmo` resources.

## Understand the Makefile

A Makefile is generated. Run `make help` to see what it can do:

```bash
Usage:
  make <target>

General
  help                  Display this help

Development
  manifests             Generate CustomResourceDefinition objects
  generate              Generate required code pieces
  generate-deepcopy     Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations
  generate-client       Generate typed client
  fmt                   Run go fmt against code
  vet                   Run go vet against code

Testing
  test                  Run tests

Build
  build                 Build manager binary
  run                   Run a controller from your host
  docker-build          Build docker image with the manager
  docker-push           Push docker image with the manager
  docker-buildx         Build and push docker image for the manager for cross-platform support

Build Dependencies
  controller-gen        Install controller-gen
  setup-envtest         Install setup-envtest
  envtest               Install envtest binaries
```

## Run the Operator

Running `make build` compiles the operator into `bin/manager`.

Ensure that KUBECONFIG is set to the kubeconfig for your playground cluster.
Then, install the CRD into your cluster:

```bash
kubectl apply -f crds
```

And start the operator

```bash
./bin/manager
```

If you are using Visual Studio Code, you can use the generated `./vscode/launch.json` to start the operator from vscode. Note that this requires to copy the kubeconfig into the `./tmp` directory:

```bash
cp $KUBECONFIG tmp/kubeconfig
```

## Developing the Operator

Now it's time to breathe some life into the operator.

The first step is to replace the `DummyGenerator` in `./pkg/operator/operator.go`

```go
func (o *Operator) Setup(mgr ctrl.Manager) error {
	// Replace this by a real resource generator (e.g. HelmGenerator or KustomizeGenerator, or your own one).
	resourceGenerator, err := manifests.NewDummyGenerator()
	if err != nil {
		return fmt.Errorf("error initializing resource generator: %w", err)
	}

	if err := component.NewReconciler[*operatorv1alpha1.Gizmo](
		o.options.Name,
		resourceGenerator,
		component.ReconcilerOptions{},
	).SetupWithManager(mgr); err != nil {
		return fmt.Errorf("unable to create controller: %w", err)
	}

	return nil
}
```

with your own implementation. There are various alternatives how to do this. For example ...

### Implement the Generator from Scratch

Let's define a package `./internal/generator`, and a file `./internal/generator/generator.go`:

```go
package generator

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/sap/component-operator-runtime/pkg/manifests"
	"github.com/sap/component-operator-runtime/pkg/types"

	operatorv1alpha1 "acme.io/gizmo-operator/api/v1alpha1"
)

type Generator struct{}

var _ manifests.Generator = &Generator{}

func New() (*Generator, error) {
	return &Generator{}, nil
}

func (g *Generator) Generate(ctx context.Context, namespace string, name string, parameters types.Unstructurable) ([]client.Object, error) {
	// the following cast is safe because of the way how Gizmo.GetSpec() is implemented
	spec := parameters.(*operatorv1alpha1.GizmoSpec)
	_ = spec

	return []client.Object{}, nil
}
```

And replace the ` manifests.NewDummyGenerator()` invocation above by `generator.New()`.

Now you have full control and can construct your dependent objects in whatever way you want.

### Embed an Existing Helm Chart

Another common approach is to vendor an existing Helm Chart.

To do so, copy the extracted Helm Chart to `./pkg/operator/data/chart`, and reference it from `./pkg/operator/operator.go` as 

```go
//go:embed all:data
var data embed.FS
```

Then you can consume it in `./pkg/operator/operator.go` using

```go
resourceGenerator, err := helm.NewHelmGenerator(data, "data/chart", nil)
```

In this form, the spec of the reconciled `Gizmo` object would, after being converted to unstructured, be passed unchanged as values to the Helm chart. Often this does not fit to the needs. In such cases you could attach a `ParameterTransformer`:

```go
parameterTransformer := // implement your own custom ParameterTransformer
resourceGenerator, err := helm.NewHelmGeneratorWithParameterTransformer(data, "data/chart", nil, parameterTransformer)
```

Similarly, if you want to mutate the output of the rendered Helm chart, an `ObjectTransformer` can be used:

```go
	helmGenerator, err := helm.NewHelmGenerator(data, "data/chart", nil)
	if err != nil {
		return fmt.Errorf("error initializing resource generator: %w", err)
	}

	// the file ./pkg/operator/data/parameter-transformer.yaml must be created, containing
	// a go template taking the unstructured Gizmo spec as input, and returning the final helm values as yaml
	parameterTransformer, err := manifests.NewTemplateParameterTransformer(data, "data/parameter-transformer.yaml")
	if err != nil {
		return fmt.Errorf("error initializing parameter transformer: %w", err)
	}

	// the directory ./pkg/operator/data/patches must be created, containing
	// patches that shall be applied to the Helm output
	patches, err := readPatches(data, "data/patches") // to be implemented ...
	if err != nil {
		return fmt.Errorf("error reading patches: %w", err)
	}
	objectTransformer, err := manifests.NewKustomizeObjectTransformer(patches, nil)
	if err != nil {
		return fmt.Errorf("error initializing object transformer: %w", err)
	}

	resourceGenerator := manifests.NewGenerator(helmGenerator).
		WithParameterTransformer(parameterTransformer).
		WithObjectTransformer(objectTransformer)
```

Check [the generators documentation](../../concepts/controller-runtime/generators/transformers/) for more details.