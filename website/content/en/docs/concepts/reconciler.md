---
title: "Component Reconciler"
linkTitle: "Component Reconciler"
weight: 20
type: "docs"
description: >
  Reconciliation logic for dependent objects
---

Dependent objects are - by definition - the resources returned by the `Generate()` method of the used resource generator.
Whenever a component resource (that is, an instance of the component's custom resource type) is created, udpated, or deleted,
the set of dependent object probably changes, and the cluster state has to be synchronized with that new declared state.

This is the job of the reconciler which is instantiated by calling the following constructor:

```go
package component

func NewReconciler[T Component](
  name              string,
  client            client.Client,
  discoveryClient   discovery.DiscoveryInterface,
  eventRecorder     record.EventRecorder,
  scheme            *runtime.Scheme,
  resourceGenerator manifests.Generator
) *Reconciler[T]
```

The passed type parameter `T Component` is the concrete runtime type of the component's custom resource type. Furthermore,
- `name` is supposed to be a unique name (typically a DNS name) identifying this component operator in the cluster; ìt will be used in annotations, labels, for leader election, ...
- `client`, `discoveryClient`, `eventRecorder` and `scheme` are deprecated and will be removed in future releases; they can be passed as `nil`
- `resourceGenerator` is an implementation of the `Generator` interface, describing how the dependent objects are rendered from the component's spec.

The object returned by `NewReconciler` implements the controller-runtime `Reconciler` interface, and can therefore be used as a drop-in
in kubebuilder managed projects. After creation, the reconciler  has to be registered with the responsible controller-runtime manager instance by calling

```go
func (r *Reconciler[T]) SetupWithManager(mgr ctrl.Manager) error
```

The used manager `mgr` has to fulfill a few requirements:
- its client must bypass informer caches for the following types:
  - the type `T` itself
  - the type `CustomResourceDefinition` from the `apiextensions.k8s.io/v1` group
  - the type `APIService` from the `apiregistration.k8s.io/v1` group
- its scheme must recognize at least the following types:
  - the types in the API group defined in this repository
  - the core group  (`v1`)
  - group `apiextensions.k8s.io/v1`
  - group `apiregistration.k8s.io/v1`.