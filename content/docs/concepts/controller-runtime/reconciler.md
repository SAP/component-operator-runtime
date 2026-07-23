---
title: "Reconciler"
linkTitle: "Reconciler"
weight: 30
type: "docs"
description: >
  The Reconciler[T] implementation, its options, hooks, and processing cycles
---

`Reconciler[T]` is the heart of the higher-level programming model. It implements
controller-runtime's `reconcile.Reconciler` interface, so it can be plugged into any
controller-runtime manager, and it drives a single [component](../components/) type
`T` end to end: it reads the component, resolves its references, renders the dependent
objects through a [generator](../generators/), and delegates the actual cluster
synchronization to the low-level [object reconciler](../../reconciler/overview/).

## A parameterized Go interface

The reconciler is a *generic* type, parameterized by the component type it manages:

```go
package component

// Reconciler provides the implementation of controller-runtime's Reconciler interface,
// for a given Component type T.
type Reconciler[T Component] struct {
	// ...
}
```

The type parameter `T` is constrained by the [`Component`](../components/) interface,
which is itself a plain Go interface (a `client.Object` plus `GetSpec()` and
`GetStatus()` accessors). This is the central design decision of the framework: instead
of forcing a fixed, concrete resource type onto operator authors, component-operator-runtime
lets each operator define its own custom resource type and simply requires it to satisfy
`Component`. The reconciler is then instantiated with that concrete type as `T`.

Because `T` is a real Go type — not an `unstructured.Unstructured` — the reconciler can
construct fresh instances, deep-copy them, and hand strongly typed values to the
generator and to hooks, all without reflection or type assertions on the operator side.
Everything below is expressed in terms of `T`.

## Creating the reconciler

A component operator typically runs exactly one reconciler, created through the
constructor:

```go
package component

// Create a new Reconciler.
// Here, name should be a meaningful and unique name identifying this reconciler within
// the Kubernetes cluster; it will be used in annotations, finalizers, and so on;
// resourceGenerator must be an implementation of the manifests.Generator interface.
func NewReconciler[T Component](
	name string,
	resourceGenerator manifests.Generator,
	options ReconcilerOptions,
) *Reconciler[T]
```

- **`name`** must be a unique, fully qualified identifier for this operator in the
  cluster (typically a DNS-style name such as `mycomponent-operator.mydomain.io`). It is
  used as the default field owner and finalizer, for the controller name, and — after
  being passed down to the object reconciler — as the prefix of all annotations and
  labels written on dependent objects.
- **`resourceGenerator`** is the [`Generator`](../generators/) that renders the dependent
  objects from the component's spec.
- **`options`** tunes the reconciler's behavior; see below.

`NewReconciler` returns a a prepared `*Reconciler[T]`. Before using the `Reconcile()` method it must be
regsitered with a manager; see below.

## Reconciler options

```go
package component

// ReconcilerOptions are creation options for a Reconciler.
type ReconcilerOptions struct {
	// Which field manager to use in API calls.
	// If unspecified, the reconciler name is used.
	FieldOwner *string
	// Which finalizer to use.
	// If unspecified, the reconciler name is used.
	Finalizer *string
	// Default service account used for impersonation of clients.
	// Of course, components can still customize impersonation by implementing the
	// ImpersonationConfiguration interface.
	DefaultServiceAccount *string
	// How to react if a dependent object exists but has no or a different owner.
	// If unspecified, AdoptionPolicyIfUnowned is assumed.
	// Can be overridden by annotation on object level.
	AdoptionPolicy *reconciler.AdoptionPolicy
	// How to perform updates to dependent objects.
	// If unspecified, UpdatePolicyReplace is assumed.
	// Can be overridden by annotation on object level.
	UpdatePolicy *reconciler.UpdatePolicy
	// How to perform deletion of dependent objects.
	// If unspecified, DeletePolicyDelete is assumed.
	// Can be overridden by annotation on object level.
	DeletePolicy *reconciler.DeletePolicy
	// Whether namespaces are auto-created if missing.
	// If unspecified, MissingNamespacesPolicyCreate is assumed.
	MissingNamespacesPolicy *reconciler.MissingNamespacesPolicy
	// Interval after which an object will be force-reapplied, even if it seems to be synced.
	ReapplyInterval *time.Duration
	// SchemeBuilder allows to define additional schemes to be made available in the
	// target client.
	SchemeBuilder types.SchemeBuilder
	// NewClient allows to modify or replace the default client used by the reconciler.
	// The returned client is used by the reconciler to manage the component instances,
	// and passed to hooks. Its scheme therefore must recognize the component type.
	NewClient NewClientFunc
}
```

All pointer fields are optional; the constructor defaults them to the same values used
by the object reconciler:

| Option | Default |
|--------|---------|
| `FieldOwner` | the reconciler `name` |
| `Finalizer` | the reconciler `name` |
| `AdoptionPolicy` | `AdoptionPolicyIfUnowned` |
| `UpdatePolicy` | `UpdatePolicyReplace` |
| `DeletePolicy` | `DeletePolicyDelete` |
| `MissingNamespacesPolicy` | `MissingNamespacesPolicyCreate` |
| `ReapplyInterval` | 60 minutes |

### How options map to the object reconciler

The component reconciler does not synchronize the cluster itself; for each component it
builds a target client and constructs a fresh object reconciler, translating its own
options into the [object reconciler's `ReconcilerOptions`](../../reconciler/overview/#reconcileroptions).
The mapping is only *partial*:

- **Passed straight through** to the object reconciler: `FieldOwner`, `Finalizer`,
  `AdoptionPolicy`, `UpdatePolicy`, `DeletePolicy`, `MissingNamespacesPolicy`, and
  `ReapplyInterval`. On top of that, the component reconciler always injects its own
  status analyzer and Prometheus metrics.
- **Consumed at the component layer** and *not* forwarded: `DefaultServiceAccount`,
  `SchemeBuilder`, and `NewClient`. These influence how the target and component clients
  are constructed (impersonation, additional schemes, client wrapping), rather than how
  individual dependent objects are handled.
- **Contributed per component, not through `ReconcilerOptions`:** the object reconciler
  additionally receives the component's `AdditionalManagedTypes` (from the
  [`TypeConfiguration`](../components/#additional-managed-types) interface) and an
  `EnableEvents` flag derived from a per-component annotation.

Some of the passed-through options can be overridden per component (through the
[`PolicyConfiguration`](../components/#policies) and
[`ReapplyConfiguration`](../components/#force-reapply-interval) interfaces) and, further
down, per dependent object (through annotations). Note that the object reconciler's
`ReconcilePolicy` is *not* exposed here; the component reconciler always drives the
object reconciler with the default `OnObjectChange` policy.

## Registering with a manager

`NewReconciler` only creates the instance; to actually use it, it has to be registered
with a controller-runtime manager. There are two registration methods:

```go
package component

// Register the reconciler with a given controller-runtime Manager.
func (r *Reconciler[T]) SetupWithManager(mgr ctrl.Manager) error

// Register the reconciler with a given controller-runtime Manager and Builder.
// This will call For() and Complete() on the provided builder.
func (r *Reconciler[T]) SetupWithManagerAndBuilder(mgr ctrl.Manager, blder *ctrl.Builder) error
```

`SetupWithManager` is the convenience entry point; it constructs a default builder (with
`MaxConcurrentReconciles: 5`) and forwards to `SetupWithManagerAndBuilder`. Use
`SetupWithManagerAndBuilder` when you need to customize the underlying
`ctrl.Builder` (predicates, `Named()`, `Watches()`, concurrency, and so on). Either method may be
called only once, and hooks must be registered *before* setup.

### Assumptions on clients and schemes

During setup the reconciler wires up two distinct clients, and this is where the
requirements on the manager come from:

- A **dedicated component client** is built from `mgr.GetConfig()` and `mgr.GetScheme()`.
  It is used to read and update the reconciled component, to send events for it, and to
  resolve `configMap`/`secret` references in the component's spec. As a consequence,
  `mgr.GetScheme()` must recognize at least the core group (`v1`) and the component type
  `T`.
- The **manager's own client** (`mgr.GetClient()`) is passed to hooks, and is used to resolve generic references
  — spec fields implementing the `Reference[T]` interface.

## Local and Target Clients

During `Reconcile()`, two clients are constructed: the **local** and the **target** client. They are equal unless the component implements the `ClientConfiguration` interface.

The **local client** always uses `mgr.GetConfig()`. It is passed to hooks and generators.

The **target client** uses - if implemented - the kubeconfig from the `GetKubeConfig()` method of the `ClientConfiguration` interface, and `mgr.GetConfig()` otherwise. The target client is used by the [object reconciler](../reconciler/), and is passed to hooks and generators.

If the component implements the `ImpersonationConfiguration` interface, then the local client impersonates accordingly. The target client impersonates only if the `ClientConfiguration` interface is not implemented.

Both clients always recognize the builtin groups (`v1`, `apps/v1` and so on), and the `apiextensions.k8s.io/v1` and `apiregistration.k8s.io/v1` groups. More API groups can be added through the `SchemeBuilder` option, and by the generator itself, when implementing

```yaml
package types

// SchemeBuilder interface.
type SchemeBuilder interface {
	AddToScheme(scheme *runtime.Scheme) error
}
```

## Hooks

Operators can enhance the reconciliation at well-defined points by registering *hooks*.
A hook is a function of type:

```go
package component

// HookFunc is the function signature that can be used to establish callbacks at certain
// points in the reconciliation logic.
// Hooks will be passed a local client (to be precise, the one belonging to the owning
// manager), and the current (potentially unsaved) state of the component.
// Post-hooks will only be called if the according operation (read, reconcile, delete)
// has been successful.
type HookFunc[T Component] func(ctx context.Context, clnt client.Client, component T) error
```

Hooks are registered with the fluent `With*Hook` methods, each of which returns the
reconciler so calls can be chained. They must be called before `SetupWithManager`:

```go
package component

// Called after the component object has been retrieved from the API, on every reconcile.
func (r *Reconciler[T]) WithPostReadHook(hook HookFunc[T]) *Reconciler[T]

// Called (only if the component is not in deletion) right before the dependent objects
// are reconciled.
func (r *Reconciler[T]) WithPreReconcileHook(hook HookFunc[T]) *Reconciler[T]

// Called (only if the component is not in deletion) right after the dependent objects
// were successfully reconciled.
func (r *Reconciler[T]) WithPostReconcileHook(hook HookFunc[T]) *Reconciler[T]

// Called (only if the component is in deletion) right before the dependent objects are
// deleted.
func (r *Reconciler[T]) WithPreDeleteHook(hook HookFunc[T]) *Reconciler[T]

// Called (only if the component is in deletion) right after the dependent objects were
// successfully deleted.
func (r *Reconciler[T]) WithPostDeleteHook(hook HookFunc[T]) *Reconciler[T]
```

Ordering within a single reconcile is: **post-read** (always) → then either the
create/update branch (**pre-reconcile** → apply → **post-reconcile** once everything is
applied) or the deletion branch (**pre-delete** → delete → **post-delete** once
everything is gone). Multiple hooks of the same kind run in registration order.

A few important semantics:

- Hooks receive the *current, potentially unsaved* component. They may change the
  component's **status**, but must not alter its **metadata** or **spec** — such changes
  might be silently persisted (e.g. when the framework updates finalizers), which could
  invalidate the already computed component digest (see below), and would produce a confusing
  picture for the user.
- The `client.Client` passed to a hook is the manager's one, i.e. the return value of `mgr.GetClient()`. The reconcile and
  delete hooks (all except the post-read hook) run with a context that additionally
  carries the [target and local clients](#local-and-target-clients), retrievable via `ClientFromContext()` /
  `LocalClientFromContext()`.
- Post-hooks only fire on success. If a hook returns an error, reconciliation stops and
  is retried; if the hook returns a `types.RetriableError`, then this controls the retry delay, otherwise the standard
  backoff applies.

## Status fields maintained by the reconciler

Beyond the [`Status`](../components/#the-status-type) fields already described with the
`Component` interface, the reconciler maintains a small group of fields that together
model the notion of a **processing cycle** — a single "attempt" at driving a particular
desired state to readiness.

### The component digest

On every reconcile, after resolving all references (secrets, config maps, and generic
`Reference[T]` fields), the reconciler computes a **component digest** — a hash over the
effective input to the generation: the component's annotations and spec together with the resolved
reference values. Two reconciles that see the same spec and the same resolved references
produce the same digest; any meaningful change to the desired state changes it.

The digest is the trigger that decides whether the current work is a continuation of an
ongoing cycle or the start of a new one. It is also handed to the object reconciler so
it can be recorded on the dependent objects.

### `ProcessingDigest` and `ProcessingSince`

- **`ProcessingDigest`** is the component digest that the *current* processing cycle is
  working towards. When the freshly computed digest differs from `ProcessingDigest`, the
  reconciler starts a **new processing cycle**: it resets `ProcessingSince`, resets
  backoffs, sets the state to `Processing` (reason `Restarting`), and requeues. While the
  component is not in deletion, `ProcessingDigest` is never cleared — it always reflects
  the digest of the latest cycle. During deletion it is reset to the empty string.

- **`ProcessingSince`** is the timestamp at which the component actually started
  *waiting* for its dependents to become ready within the current cycle (i.e. when an
  apply completed but not all dependents were ready yet). It is the basis for the
  [processing timeout](../components/#requeue-retry-and-timeout-intervals): if the
  dependents do not all become ready within the effective timeout counted from
  `ProcessingSince`, the component transitions to `Error` (or `Pending`, for retriable
  errors) with reason `Timeout`. Once the component reaches the `Ready` state,
  `ProcessingSince` is cleared, ending the cycle.

Because a new cycle resets `ProcessingSince`, the timeout automatically restarts whenever
the desired state changes — a long-running rollout is not penalized by time spent on a
previous, now-superseded configuration.

### `LastProcessingDigest` and `Revision`

- **`LastProcessingDigest`** records the `ProcessingDigest` of the cycle for which
  dependent objects were last (re)generated and applied. In the create/update path, when
  `ProcessingDigest` differs from `LastProcessingDigest`, the reconciler recognizes that
  a new set of manifests is about to be applied.

- **`Revision`** is a monotonically increasing counter of processing cycles. Each time
  `ProcessingDigest` differs from `LastProcessingDigest`, `Revision` is incremented and
  `LastProcessingDigest` is set to the current `ProcessingDigest`. In other words, the
  revision counts how many *distinct* desired states have been applied over the lifetime
  of the component. It is made available to generators (for instance, the
  [Helm generator](../generators/helm/) uses it as the Helm release revision so that
  chart logic depending on `.Release.Revision` behaves as expected).

In summary, a change to the component's effective desired state changes the **component
digest**, which starts a new cycle by advancing **`ProcessingDigest`** (resetting
**`ProcessingSince`**) and, once the new manifests are applied, bumps the
**`Revision`** while updating **`LastProcessingDigest`**. `ProcessingSince` then bounds
how long that cycle may take before a timeout is declared.
