---
title: "Integration with controller-runtime"
linkTitle: "Integration with controller-runtime"
weight: 30
type: "docs"
description: >
  Wiring the Object Reconciler into a controller-runtime operator
---

While the [Object Reconciler](../reconciler/) is the low-level engine, the
`pkg/component` package integrates it into the [controller-runtime](https://github.com/kubernetes-sigs/controller-runtime)
world. A generic `Reconciler[T]` drives a custom resource type — the *component* —
renders its dependent manifests through a *generator*, and delegates the actual cluster
synchronization to the Object Reconciler.

## Topics

- **[Components](components/)** — the `Component` interface and the optional
  configuration interfaces a component (or its spec) may implement.
- **[Generators](generators/)** — the `Generator` interface, the built-in generators,
  and the generate context.
- **[Reconciler](reconciler/)** — the `Reconciler[T]` implementation, its hooks, and the
  additional status fields it maintains.
