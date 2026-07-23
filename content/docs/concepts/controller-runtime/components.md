---
title: "Components"
linkTitle: "Components"
weight: 10
type: "docs"
description: >
  The Component interface and related configuration interfaces
---

The `Component` interface is the central abstraction of the higher-level programming
model. It abstracts any custom resource type that describes and models a *component* in
the sense of the [terminology](../../../#terminology): a coherent set of dependent
objects that are to be applied to a cluster. A component operator built on top of
component-operator-runtime provides one such custom resource type per component, and
that type has to implement `Component`.

## The `Component` interface

```go
package component

// Component is the central interface that component operators have to implement.
// Besides being a controller-runtime client.Object, the implementing type has to expose
// accessor methods for the component's spec and status, GetSpec() and GetStatus().
type Component interface {
	client.Object
	// Return a read-only accessor to the component's spec.
	// The returned value has to implement the types.Unstructurable interface.
	GetSpec() types.Unstructurable
	// Return a read-write (usually a pointer) accessor to the component's status,
	// resp. to the corresponding substruct if the status extends component.Status.
	GetStatus() *Status
}
```

Because the implementing type embeds controller-runtime's `client.Object`, it is a
regular Kubernetes API type (with `TypeMeta` and `ObjectMeta`). On top of that, only two
accessor methods have to be provided:

- **`GetSpec()`** exposes the parameterization of the component. Its return value only
  has to implement the `types.Unstructurable` interface (see below); the framework
  passes it verbatim to the generator's `Generate()` method as `parameters`. In most
  cases this is the spec itself, or a deep copy of it. The framework treats the returned
  value as read-only and never modifies it, so there is no requirement that changes to
  it are reflected back into the component.
- **`GetStatus()`** returns a pointer to the component's status, or to the substructure
  of the status that embeds `component.Status`. In contrast to `GetSpec()`, the
  framework *does* write to the returned structure, so the pointer should reference the
  actual status of the API object (not a copy).

The component's custom resource type is expected to be namespaced. By default, dependent
objects are placed in the component's own namespace, and the `namespace` and `name`
arguments of `Generate()` are set to the component's `metadata.namespace` and
`metadata.name`. These defaults can be overridden through the optional
[`PlacementConfiguration`](#placement) interface.

## The `Unstructurable` interface

The value returned by `GetSpec()` must implement `types.Unstructurable`, a minimal
interface for anything that can be represented as a string-keyed map:

```go
package types

// Unstructurable represents objects which can be converted into a string-keyed map.
// All Kubernetes API types, as well as all JSON objects could be modelled as
// Unstructurable objects.
type Unstructurable interface {
	ToUnstructured() map[string]any
}
```

The package also provides a ready-made implementation for the common case where the
parameters are already available as a plain map:

```go
package types

// UnstructurableMap is a string-keyed map, implementing the Unstructurable interface
// in the natural way.
type UnstructurableMap map[string]any

func (m UnstructurableMap) ToUnstructured() map[string]any {
	return m
}
```

A typed spec struct can implement `Unstructurable` by marshalling itself to a map (for
example via JSON round-tripping), while dynamically shaped parameters can simply be
wrapped in an `UnstructurableMap`.

## The `Status` type

Components must include `component.Status` into their status, in some way; typically, using a flat embedding. The framework maintains this
structure across reconciliations:

```go
package component

// Component Status. Components must include this into their status.
type Status struct {
	ObservedGeneration   int64                       `json:"observedGeneration"`
	AppliedGeneration    int64                       `json:"appliedGeneration,omitempty"`
	LastObservedAt       *metav1.Time                `json:"lastObservedAt,omitempty"`
	LastAppliedAt        *metav1.Time                `json:"lastAppliedAt,omitempty"`
	ProcessingDigest     string                      `json:"processingDigest,omitempty"`
	ProcessingSince      *metav1.Time                `json:"processingSince,omitempty"`
	LastProcessingDigest string                      `json:"lastProcessingDigest,omitempty"`
	Revision             int64                       `json:"revision,omitempty"`
	Conditions           []Condition                 `json:"conditions,omitempty"`
	State                State                       `json:"state,omitempty"`
	Inventory            []*reconciler.InventoryItem `json:"inventory,omitempty"`
}
```

The most relevant fields are:

- **`ObservedGeneration`** / **`AppliedGeneration`** — the `metadata.generation` last
  observed by the reconciler, respectively the generation whose dependent objects were
  last successfully applied.
- **`LastObservedAt`** / **`LastAppliedAt`** - timestamps related to `ObservedGeneration` / `AppliedGeneration`.
- **`Conditions`** — currently the single `Ready` condition; its `status` is `True`,
  `False`, or `Unknown` and carries `reason` and `message`.
- **`State`** — a coarse summary of the component's lifecycle phase, one of `Ready`,
  `Pending`, `Processing`, `DeletionPending`, `Deleting`, or `Error`.
- **`Inventory`** — the list of dependent objects the reconciler currently tracks for
  this component (see [Inventory](../../reconciler/inventory/)).

`Status` also exposes helper methods such as `IsReady()`, `GetState()`, and
`SetState()`, so component controllers rarely have to manipulate conditions directly.

## Optional configuration interfaces

Beyond the mandatory `Component` interface, a component may opt into additional behavior
by implementing any of the interfaces below. Each of them may be implemented **either by
the component type itself or by its spec type**. If both implement the same interface,
the implementation on the component takes precedence.

### Placement

```go
// The PlacementConfiguration interface is meant to be implemented by components (or
// their spec) which allow to explicitly specify target namespace and name of the
// deployment (otherwise this will be defaulted as the namespace and name of the
// component object itself).
type PlacementConfiguration interface {
	GetDeploymentNamespace() string
	GetDeploymentName() string
}
```

Overrides the target namespace and name passed to the generator. A non-empty
`GetDeploymentNamespace()` additionally becomes the namespace for rendered namespaced
resources that do not specify one themselves.

### Remote deployment

```go
// The ClientConfiguration interface is meant to be implemented by components (or their
// spec) which offer remote deployments.
type ClientConfiguration interface {
	GetKubeConfig() []byte
}
```

Deploys the dependent objects to a remote cluster using the returned kubeconfig. Return `nil`
to use the default local client.

### Impersonation

```go
// The ImpersonationConfiguration interface is meant to be implemented by components (or
// their spec) which offer impersonated deployments.
type ImpersonationConfiguration interface {
	GetImpersonationUser() string
	GetImpersonationGroups() []string
}
```

Deploys the dependent objects while impersonating the given user (and optionally groups).
Returning an empty user / `nil` groups disables impersonation.

### Suspension

```go
// The SuspensionConfiguration interface is meant to be implemented by components (or
// their spec) which offer the ability to suspend reconciliation.
type SuspensionConfiguration interface {
	IsSuspended() bool
}
```

If `IsSuspended()` returns `true`, the component moves into the `Pending` state with
reason `Suspended` and apply reconciliation is paused. Deletion is not affected by
suspension.

### Requeue, retry, and timeout intervals

```go
// The RequeueConfiguration interface tweaks the interval after which a component is
// re-reconciled following a successful reconciliation (default 10 minutes).
type RequeueConfiguration interface {
	GetRequeueInterval() time.Duration
}

// The RetryConfiguration interface tweaks the interval after which a component is
// re-reconciled following a failed reconciliation (default: the requeue interval).
type RetryConfiguration interface {
	GetRetryInterval() time.Duration
}

// The TimeoutConfiguration interface tweaks the processing timeout, after which the
// component goes into Error state if not all dependents became ready (default: the
// requeue interval).
type TimeoutConfiguration interface {
	GetTimeout() time.Duration
}
```

For all three, a return value of zero means "use the framework default".
If non-zero values are returned, they should be greater than one minute.

### Policies

```go
// The PolicyConfiguration interface is meant to be implemented by components (or their
// spec) which offer tweaking policies affecting the dependents handling.
type PolicyConfiguration interface {
	GetAdoptionPolicy() reconciler.AdoptionPolicy
	GetUpdatePolicy() reconciler.UpdatePolicy
	GetDeletePolicy() reconciler.DeletePolicy
	GetMissingNamespacesPolicy() reconciler.MissingNamespacesPolicy
}
```

Lets the component set default [policies](../../reconciler/policies/) for its dependent
objects. Each method may return the empty string to fall back to the
reconciler/framework default. Individual dependents can still override these via
annotations.

### Additional managed types

```go
// The TypeConfiguration interface is meant to be implemented by components (or their
// spec) which allow to specify additional managed types.
type TypeConfiguration interface {
	GetAdditionalManagedTypes() []reconciler.TypeInfo
}
```

Declares extra managed types (beyond those explicitly part of the manifests). The
returned `TypeInfo` structs may use concrete groups/kinds or wildcards. See
[Managed Types](../../reconciler/managed-types/).

### Force-reapply interval

```go
// The ReapplyConfiguration interface is meant to be implemented by components (or their
// spec) which allow to tune the force-reapply interval.
type ReapplyConfiguration interface {
	GetReapplyInterval() time.Duration
}
```

Tunes how often dependent objects are force-reapplied even when they appear to be in
sync. See [Drift Detection](../../reconciler/drift-detection/).

## Convenience spec structs

For every optional configuration interface, the framework ships a small struct that
already implements it. Implementing types can simply embed the relevant struct into
their spec to gain the corresponding behavior, without writing accessor methods by hand.
All of them are annotated for deepcopy generation and carry the appropriate JSON tags and
validation markers.

| Embed this struct | To implement | Key field(s) |
|---|---|---|
| `PlacementSpec` | `PlacementConfiguration` | `namespace`, `name` |
| `ClientSpec` | `ClientConfiguration` | `kubeConfig` (secret reference) |
| `ImpersonationSpec` | `ImpersonationConfiguration` | `serviceAccountName` |
| `SuspensionSpec` | `SuspensionConfiguration` | `suspend` |
| `RequeueSpec` | `RequeueConfiguration` | `requeueInterval` |
| `RetrySpec` | `RetryConfiguration` | `retryInterval` |
| `TimeoutSpec` | `TimeoutConfiguration` | `timeout` |
| `PolicySpec` | `PolicyConfiguration` | `adoptionPolicy`, `updatePolicy`, `deletePolicy`, `missingNamespacesPolicy` |
| `TypeSpec` | `TypeConfiguration` | `additionalManagedTypes` |
| `ReapplySpec` | `ReapplyConfiguration` | `reapplyInterval` |

For example, a component spec that wants to expose a configurable target placement and a
suspend switch could look like:

```go
type MyComponentSpec struct {
	component.PlacementSpec  `json:",inline"`
	component.SuspensionSpec `json:",inline"`

	// component-specific parameters follow
	// ...
}
```