---
title: "Overview"
linkTitle: "Overview"
weight: 10
type: "docs"
description: >
  The Reconciler type, its constructor, options, and core methods
---

This page introduces the main type of package `pkg/reconciler` and the methods used
to drive it.

## The `Reconciler` type

`Reconciler` manages a set of specified objects in a given target cluster. It is
created with the `NewReconciler` constructor:

```go
package reconciler

// Create new reconciler.
// The passed name should be fully qualified; by default it will be used as
// field owner and finalizer.
// The passed client's scheme must recognize at least the core group (v1) and
// apiextensions.k8s.io/v1 and apiregistration.k8s.io/v1.
func NewReconciler(name string, clnt cluster.Client, options ReconcilerOptions) *Reconciler
```

- **`name`** should be a fully qualified, unique identifier for the operator, typically
  a DNS-style name such as `mycomponent-operator.mydomain.io`. It is used, by default,
  as the field owner and finalizer, and — importantly — as the **prefix of all
  annotations and labels** the reconciler reads and writes. Throughout this
  documentation, annotations are written as `mycomponent-operator.mydomain.io/apply-order`
  and similar; the prefix is exactly this `name`, so it varies from operator to operator.

- **`clnt`** is a `cluster.Client` (from `pkg/cluster`) pointing at the target cluster.
  See [Client and Scheme](../client/) for the requirements on this client and its scheme.

- **`options`** is a `ReconcilerOptions` value.

### `ReconcilerOptions`

```go
package reconciler

// ReconcilerOptions are creation options for a Reconciler.
type ReconcilerOptions struct {
	// Which field manager to use in API calls.
	// If unspecified, the reconciler name is used.
	FieldOwner *string
	// Which finalizer to use.
	// If unspecified, the reconciler name is used.
	Finalizer *string
	// How to react if a dependent object exists but has no or a different owner.
	// If unspecified, AdoptionPolicyIfUnowned is assumed.
	// Can be overridden by annotation on object level.
	AdoptionPolicy *AdoptionPolicy
	// How to perform updates to dependent objects.
	// If unspecified, UpdatePolicyReplace is assumed.
	// Can be overridden by annotation on object level.
	UpdatePolicy *UpdatePolicy
	// How to perform deletion of dependent objects.
	// If unspecified, DeletePolicyDelete is assumed.
	// Can be overridden by annotation on object level.
	DeletePolicy *DeletePolicy
	// Whether namespaces are auto-created if missing.
	// If unspecified, MissingNamespacesPolicyCreate is assumed.
	MissingNamespacesPolicy *MissingNamespacesPolicy
	// Additional managed types. Instances of these types are handled differently
	// during apply and delete; foreign instances of these types will block deletion
	// of the component. A typical example are CRDs which are implicitly created by
	// the workloads of the component, but are not part of the manifests.
	AdditionalManagedTypes []TypeInfo
	// Interval after which an object will be force-reapplied, even if it seems synced.
	ReapplyInterval *time.Duration
	// How to analyze the state of the dependent objects.
	// If unspecified, an optimized kstatus based implementation is used.
	StatusAnalyzer status.StatusAnalyzer
	// Prometheus metrics to be populated by the reconciler.
	Metrics ReconcilerMetrics
	// Whether to disable sending events on the dependent objects.
	// If unspecified, true is assumed.
	EnableEvents *bool
}
```

All pointer fields are optional; when left unspecified, the framework defaults apply:

| Option | Default |
|--------|---------|
| `FieldOwner` | the reconciler `name` |
| `Finalizer` | the reconciler `name` |
| `AdoptionPolicy` | `AdoptionPolicyIfUnowned` |
| `UpdatePolicy` | `UpdatePolicyReplace` |
| `DeletePolicy` | `DeletePolicyDelete` |
| `MissingNamespacesPolicy` | `MissingNamespacesPolicyCreate` |
| `ReapplyInterval` | 60 minutes |
| `StatusAnalyzer` | optimized kstatus implementation |
| `EnableEvents` | `true` |

See [Policies](../policies/) for the meaning of each policy, and
[Enhanced Status Detection](../status-detection/) for the default status analyzer.

## Core methods

The reconciler exposes three methods that make up its public surface.

### `Apply`

```go
func (r *Reconciler) Apply(
	ctx context.Context,
	inventory *[]*InventoryItem,
	objects []client.Object,
	namespace string,
	ownerId string,
	componentDigest string,
) (bool, error)
```

`Apply` synchronizes the target cluster with the desired set of `objects`:

- non-existent objects are **created**;
- existing objects are **updated** if they are out of sync;
- objects that are in the inventory but no longer in `objects` are **removed**.

Key points:

- The `inventory` is passed by pointer and **mutated in place**. Callers persist it
  between invocations (typically in the status of a custom resource) and pass the same
  inventory back on the next call.
- `namespace` is the fallback namespace for namespaced objects that do not specify one.
- `ownerId` identifies the owner; it is used for the ownership / adoption checks (see
  [Policies](../policies/)). It must not change across invocations for the same inventory.
- `componentDigest` is folded into the per-object digest only when the effective
  reconcile policy is `OnObjectOrComponentChange`, so that a change of the owning
  component triggers reconciliation of its dependents.
- Objects are applied in **waves** according to their apply order, and objects with a
  purge order are completed at the end of their wave (see
  [Waves](../waves/) and [Completion](../completion/)).

`Apply` returns `(true, nil)` once **everything** is reconciled and ready. If it
returns `(false, nil)`, the work is not finished — the caller should persist the
inventory and call `Apply` again later. A non-nil error indicates a genuine failure.

> It is guaranteed that `Apply` returns `(true, nil)` immediately, on the first call,
> if the cluster is already in the desired state.

### `Delete`

```go
func (r *Reconciler) Delete(
	ctx context.Context,
	inventory *[]*InventoryItem,
	ownerId string,
) (bool, error)
```

`Delete` removes all objects tracked in the `inventory` from the target cluster,
proceeding in **delete waves** (see [Waves](../waves/)). Objects whose effective delete
policy is `Orphan` or `OrphanOnDelete` are left in the cluster but dropped from the
inventory.

Like `Apply`, `Delete` mutates the inventory in place and returns `(true, nil)` only
once all objects are gone. Otherwise it returns `(false, nil)` and should be re-called
until it returns `true`.

### `IsDeletionAllowed`

```go
func (r *Reconciler) IsDeletionAllowed(
	ctx context.Context,
	inventory *[]*InventoryItem,
	ownerId string,
) (bool, string, error)
```

`IsDeletionAllowed` checks whether the object set may be safely deleted. It returns
`false` (together with a human-readable reason) if the inventory contains an extension
type — a CRD or a type served by an APIService — for which **foreign instances** (that
is, instances not managed by this owner) still exist in the cluster. As an exception,
deletion is allowed immediately if *all* inventory items have delete policy `Orphan` or
`OrphanOnDelete`.

This is the mechanism that guards against the *stuck finalizer* problem; see
[Managed Types](../managed-types/) for the full story.

## The inventory

The current cluster state is tracked in the **inventory**, a `[]*InventoryItem`. Each
`InventoryItem` records the identity, applied policies, orders, digest, phase, and
observed status of one managed object. The inventory is what makes the reconciler
level-based: it is the memory that lets successive `Apply` / `Delete` calls know what
has already been done. See [The Inventory](../inventory/) for a field-by-field walkthrough.
