---
title: "Using kubebuilder"
linkTitle: "Using kubebuilder"
weight: 20
type: "docs"
description: >
  Integrate the framework into a kubebuilder project
---

This guide builds the same operator as the [Scaffolder tutorial](../scaffolder/) — an
operator that manages resources of type `Gizmo` in the API group `tools.acme.io`
(version `v1alpha1`) — but instead of using the provided scaffolding tool, it starts
from a plain [kubebuilder](https://book.kubebuilder.io/) project and wires
component-operator-runtime into it by hand.

## Prerequisites

- A Kubernetes cluster to deploy to later. If you do not have one, you can quickly
  spin up a local [kind](https://kind.sigs.k8s.io/) cluster:

  ```bash
  kind create cluster
  ```

- A recent [kubebuilder](https://book.kubebuilder.io/) binary. Follow the
  [installation instructions](https://book.kubebuilder.io/quick-start.html#installation)
  from the kubebuilder book and make sure `kubebuilder` is on your `PATH`.

## Initializing the project

First, create an empty working directory for the project, change into it, and initialize
the Go module:

```bash
mkdir gizmo-operator
cd gizmo-operator
go mod init acme.io/gizmo-operator
```

Then initialize the kubebuilder project and create an API and kind for `Gizmo`:

```bash
kubebuilder init --domain acme.io --project-name gizmo-operator
kubebuilder create api --group tools --version v1alpha1 --kind Gizmo
make manifests
```

When prompted by `kubebuilder create api`, answer `y` to both *Create Resource* and
*Create Controller*.

At this point you have a standard kubebuilder layout, with the API types under
`api/v1alpha1/`, the controller under `internal/controller/`, and the manager entry
point in `cmd/main.go`.

## Adding the dependency

Add component-operator-runtime to the module:

```bash
go get github.com/sap/component-operator-runtime
```

## Making `Gizmo` a component

For the framework to be able to reconcile `Gizmo`, the type has to implement the
[`Component`](../../concepts/controller-runtime/components/) interface. This means three
things:

- the spec has to implement the `componenttypes.Unstructurable` interface,
- the status has to embed `component.Status`,
- the `Gizmo` type has to expose `GetSpec()` and `GetStatus()` accessors.

Edit `api/v1alpha1/gizmo_types.go` accordingly. Start by adjusting the imports and the
spec and status structs:

```go
import (
	"github.com/sap/component-operator-runtime/pkg/component"
	componenttypes "github.com/sap/component-operator-runtime/pkg/types"
)

// GizmoSpec defines the desired state of Gizmo.
type GizmoSpec struct {
	// Add your component's parameters here, e.g.:
	// Replicas int `json:"replicas,omitempty"`
}

// GizmoStatus defines the observed state of Gizmo.
type GizmoStatus struct {
	component.Status `json:",inline"`
}
```

Then let the spec implement `componenttypes.Unstructurable`, and add the `GetSpec()` and
`GetStatus()` accessors to `Gizmo`:

```go
// ToUnstructured implements the componenttypes.Unstructurable interface.
func (s *GizmoSpec) ToUnstructured() map[string]any {
	result, err := runtime.DefaultUnstructuredConverter.ToUnstructured(s)
	if err != nil {
		panic(err)
	}
	return result
}

// GetSpec implements the component.Component interface.
func (c *Gizmo) GetSpec() componenttypes.Unstructurable {
	return &c.Spec
}

// GetStatus implements the component.Component interface.
func (c *Gizmo) GetStatus() *component.Status {
	return &c.Status.Status
}
```

Kubebuilder generates a `Ready` marker on the default status; the embedded
`component.Status` provides its own `Conditions` and `State` fields, so you can remove
any conflicting fields that `kubebuilder create api` may have added to `GizmoStatus`.

Whenever the API types change, regenerate the deepcopy code and the CRD manifests:

```bash
make generate manifests
```

## Wiring up the reconciler

Kubebuilder generated a controller in `internal/controller/gizmo_controller.go` and
registered it in `cmd/main.go`. Because component-operator-runtime brings its own
generic reconciler, the generated `GizmoReconciler` is not needed — delete
`internal/controller/gizmo_controller.go` (and its test file), and register the
framework's reconciler directly in `cmd/main.go` instead.

In `cmd/main.go`, add the imports:

```go
import (
	"github.com/sap/component-operator-runtime/pkg/component"
	"github.com/sap/component-operator-runtime/pkg/manifests"
)
```

Then, in the block where the generated code called
`(&controller.GizmoReconciler{...}).SetupWithManager(mgr)`, create and register the
component reconciler:

```go
resourceGenerator, err := manifests.NewDummyGenerator()
if err != nil {
	setupLog.Error(err, "unable to initialize resource generator")
	os.Exit(1)
}

if err := component.NewReconciler[*toolsv1alpha1.Gizmo](
	"gizmo-operator.acme.io",
	resourceGenerator,
	component.ReconcilerOptions{},
).SetupWithManager(mgr); err != nil {
	setupLog.Error(err, "unable to create controller", "controller", "Gizmo")
	os.Exit(1)
}
```

The first argument is a unique, fully qualified name identifying this operator in the
cluster; it is used for the field owner, the finalizer, and as the prefix of the
annotations and labels the reconciler writes on dependent objects. See the
[Reconciler reference](../../concepts/controller-runtime/reconciler/) for the full list
of options.

Here we use the built-in `DummyGenerator`, which renders no dependent objects, so the
project compiles and runs immediately. You will replace it with a real generator below.

The package `internal/controller` is not needed any longer, and can be deleted.

## Running the operator

Make sure `KUBECONFIG` points at your playground cluster, then install the CRD and run
the operator locally:

```bash
make install
make run
```

Now you can apply a `Gizmo` instance, for example the sample generated by kubebuilder:

```bash
kubectl apply -f config/samples/tools_v1alpha1_gizmo.yaml
```

The reconciler picks it up and — because the `DummyGenerator` produces no objects —
moves it straight to the `Ready` state.

## Developing the operator

The remaining work is identical to the scaffolder-based project: replace the
`DummyGenerator` with a real [generator](../../concepts/controller-runtime/generators/)
that renders the dependent objects for your component.