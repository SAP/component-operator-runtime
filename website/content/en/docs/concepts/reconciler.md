---
title: "Component Reconciler"
linkTitle: "Component Reconciler"
weight: 20
type: "docs"
description: >
  Reconciliation logic for dependent objects
---

Dependent objects are - by definition - the resources returned by the `Generate()` method of the provided resource generator.
Whenever a component resource (that is, an instance of the component's custom resource type) is created, udpated, or deleted,
the set of dependent object probably changes, and the cluster state has to be synchronized with that new declared state.

This happens in the reconciler which is instantiated by calling the following constructor:

```go
package component

func NewReconciler[T Component](
  name              string,
  client            client.Client,
  discoveryClient   discovery.DiscoveryInterface,
  recorder          record.EventRecorder,
  scheme            *runtime.Scheme,
  resourceGenerator manifests.Generator
) *Reconciler[T]
```

The passed type parameter `T Component` would be the concrete runtime type of the custom resource type. Furthermore,
- `name` is supposed to be a unique name (typically a DNS name) uniquely identifying this component operator in the cluster; ìt will be used in annotations, labels, for leader election, ...
- `client` is a controller-runtime `client.Client`, having all the needed privileges to manage the dependent objects (probably it will have cluster-admin rights); the `client` must bypass informer caches for the following types:
  - the type `T` itself
  - the type `CustomResourceDefinition` from the `apiextensions.k8s.io/v1` group
  - the type `APIService` from the `apiregistration.k8s.io/v1` group
- `discoveryClient` is a normal discovery client as provided by client-go
- `recorder` is a standard event recorder
- `scheme` is a `runtime.Scheme` which must recognize at least the following types:
  - the types in the API group defined in this repository
  - the core group  (`v1`)
  - group `apiextensions.k8s.io/v1`
  - group `apiregistration.k8s.io/v1`
  - all concrete (i.e. non-unstructured) types returned by the `Generate()` method of the passed resource generator
- `resourceGenerator` is an implementation of the `Generator` interface, describing how the dependent objects are rendered from the component's spec.

The object returned by `NewReconciler` implements the controller-runtime `Reconciler` interface, and can therefore be used as a drop-in
in kubebuilder managed projects.