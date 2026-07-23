---
title: "Client and Scheme"
linkTitle: "Client and Scheme"
weight: 100
type: "docs"
description: >
  Requirements on the client and scheme passed to the reconciler
---

The Object Reconciler talks to the target cluster through a `cluster.Client`. This page
describes what that client must provide, and what its scheme must recognize.

## The `cluster.Client` interface

```go
package cluster

// The Client interface extends the controller-runtime client by discovery and
// event recording capabilities.
type Client interface {
	client.Client
	// Return a discovery client.
	DiscoveryClient() discovery.DiscoveryInterface
	// Return an event recorder.
	EventRecorder() record.EventRecorder
	// Return a rest config for this client.
	Config() *rest.Config
	// Return a http client for this client.
	HttpClient() *http.Client
}
```

In other words, a `cluster.Client` is a controller-runtime `client.Client` augmented
with a few extras the reconciler relies on:

- the embedded **`client.Client`** for the usual CRUD operations, and — through it — a
  **`RESTMapper()`** and a **`Scheme()`**;
- a **discovery client**, used to resolve served resources;
- an **event recorder**, used to emit events on dependent objects (created / updated /
  deleted), unless events are disabled via `ReconcilerOptions.EnableEvents` or per object annotation `mycomponent-operator.mydomain.io/disable-events`;
- a **REST config** and **HTTP client** for lower-level access.

## Scheme requirements

The scheme of the passed client must recognize at least:

- the **core group** (`v1`),
- **`apiextensions.k8s.io/v1`** (for `CustomResourceDefinition`), and
- **`apiregistration.k8s.io/v1`** (for `APIService`).

These are required because the reconciler inspects and specially handles namespaces, secrets, CRDs and
APIServices. Beyond that, the scheme should recognize the concrete types of the objects
you apply if you pass typed (non-`Unstructured`) manifests; unstructured objects with
valid type information are also accepted and are converted to their concrete type when
known to the scheme.

## See also

- [Overview](../overview/) — where the client is passed to `NewReconciler`.
