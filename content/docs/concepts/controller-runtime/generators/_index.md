---
title: "Generators"
linkTitle: "Generators"
weight: 20
type: "docs"
description: >
  The Generator interface, built-in generators, and how to transform them
---

A *generator* is the recipe that turns a component's parameters into the concrete,
applyable resource manifests of its dependent objects. Where the [`Component`](../components/)
interface models the desired and observed state of a component, the `Generator`
interface encapsulates *how* the dependent objects are rendered from the component's
parameterization (its spec):

```go
package manifests

// Resource generator interface.
// When called from the reconciler, the arguments namespace and name will match the
// component's namespace and name or, if the component or its spec implement the
// PlacementConfiguration interface, the return values of the GetDeploymentNamespace(),
// GetDeploymentName() methods (if non-empty). The parameters argument will be assigned
// the return value of the component's GetSpec() method.
type Generator interface {
	Generate(ctx context.Context, namespace string, name string, parameters types.Unstructurable) ([]client.Object, error)
}
```

The returned `[]client.Object` is exactly the manifest list that the framework hands to
the low-level [object reconciler](../../reconciler/overview/), which then creates,
updates, and deletes the dependent objects in the cluster.

The generation logic can be implemented natively in Go: a component controller is free
to provide its own `Generator`, assembling and returning `client.Object` values by
whatever means it likes. In practice, however, most components are described through some
form of templating. For these common cases, the framework ships two ready-made
generators:

- the [Helm generator](helm/), which renders a Helm chart, and
- the [Kustomize generator](kustomize/), which renders a (possibly templatized)
  kustomization.

Both are passed to `component.NewReconciler[T]()` as the `resourceGenerator` argument.

Additional contextual information (such as a client for the deployment target) can be
retrieved inside `Generate()` from the passed `context.Context` via the accessor
functions in package `pkg/component`, for example `component.ClientFromContext()` and
`component.LocalClientFromContext()`.

Generators may optionally implement the `SchemeBuilder` interface

```go
package types

type SchemeBuilder interface {
	AddToScheme(scheme *runtime.Scheme) error
}
```

in order to enhance the scheme used by the dependent objects deployer.

## Topics

- **[Transforming existing generators](transformers/)** — wrap an existing generator to
  adjust its input parameters or output objects.
- **[Helm generator](helm/)** — render a Helm chart into dependent objects.
- **[Kustomize generator](kustomize/)** — render a (templatized) kustomization into
  dependent objects.
