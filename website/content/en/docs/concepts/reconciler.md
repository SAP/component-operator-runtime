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
the set of dependent object potentially changes, and the cluster state has to be synchronized with that new declared state.
This synchronization is the job of the reconciler provided by this framework.

## Creating the reconciler instance

Typically, a component operator runs one reconciler which is instantiated by calling the following constructor:

```go
package component

func NewReconciler[T Component](
  name              string,
  resourceGenerator manifests.Generator
  options           ReconcilerOptions
) *Reconciler[T]
```

The passed type parameter `T Component` is the concrete runtime type of the component's custom resource type (respectively, a pointer to that). Furthermore,
- `name` is supposed to be a unique name (typically a DNS name) identifying this component operator in the cluster; ìt will be used in annotations, labels, for leader election, ...
- `resourceGenerator` is an implementation of the `Generator` interface, describing how the dependent objects are rendered from the component's spec.
- `options` can be used to tune the behavior of the reconciler:

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
    // Of course, components can still customize impersonation by implementing the ImpersonationConfiguration interface.
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
    // The returned client is used by the reconciler to manage the component instances, and passed to hooks.
    // Its scheme therefore must recognize the component type.
    NewClient NewClientFunc
  }
  ```

The object returned by `NewReconciler` implements controller-runtime's `Reconciler` interface, and can therefore be used as a drop-in
in kubebuilder managed projects. After creation, the reconciler can be registered with the responsible controller-runtime manager instance by calling

```go
package component

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

## Reconciler hooks

Component operators may register hooks to enhance the reconciler logic at certain points, by passing functions of type

```go
package component

// HookFunc is the function signature that can be used to
// establish callbacks at certain points in the reconciliation logic.
// Hooks will be passed the current (potentially unsaved) state of the component.
// Post-hooks will only be called if the according operation (read, reconcile, delete)
// has been successful.
type HookFunc[T Component] func(ctx context.Context, client client.Client, component T) error
```

to the desired registration functions:

```go
package component

// Register post-read hook with reconciler.
// This hook will be called after the reconciled component object has been retrieved from the Kubernetes API.
func (r *Reconciler[T]) WithPostReadHook(hook HookFunc[T]) *Reconciler[T]

// Register pre-reconcile hook with reconciler.
// This hook will be called if the reconciled component is not in deletion (has no deletionTimestamp set),
// right before the reconcilation of the dependent objects starts.
func (r *Reconciler[T]) WithPreReconcileHook(hook HookFunc[T]) *Reconciler[T]

// Register post-reconcile hook with reconciler.
// This hook will be called if the reconciled component is not in deletion (has no deletionTimestamp set),
// right after the reconcilation of the dependent objects happened, and was successful.
func (r *Reconciler[T]) WithPostReconcileHook(hook HookFunc[T]) *Reconciler[T]

// Register pre-delete hook with reconciler.
// This hook will be called if the reconciled component is in deletion (has a deletionTimestamp set),
// right before the deletion of the dependent objects starts.
func (r *Reconciler[T]) WithPreDeleteHook(hook HookFunc[T]) *Reconciler[T]

// Register post-delete hook with reconciler.
// This hook will be called if the reconciled component is in deletion (has a deletionTimestamp set),
// right after the deletion of the dependent objects happened, and was successful.
func (r *Reconciler[T]) WithPostDeleteHook(hook HookFunc[T]) *Reconciler[T]
```

Note that the client passed to the hook functions is the client of the manager that was used when calling `SetupWithManager()` (that is, the return value of that manager's `GetClient()` method). In addition, reconcile and delete hooks (that is, all except the post-read hook) can retrieve a client for the deployment target by calling `ClientFromContext()`.

## Tuning the retry behavior

By default, errors returned by the component's generator or by a registered hook will make the reconciler go
into a backoff managed by controller-runtime (which usually is an exponential backoff, capped at 10 minutes).
However, if the error is or unwraps to a `types.RetriableError`, then the retry delay specified at the error
will be used instead of the backoff. Implementations should use

```go
pacakge types

func NewRetriableError(err error, retryAfter *time.Duration) RetriableError {
  return RetriableError{err: err, retryAfter: retryAfter}
}
```

to wrap an error into a `RetriableError`. It is allowed to pass `retryAfter` as nil; in that case the retry delay
will be determined by calling the component's `GetRetryInterval()` method (if the component or its spec implements
the

```go
package component

// The RetryConfiguration interface is meant to be implemented by components (or their spec) which offer
// tweaking the retry interval (by default, it would be the value of the requeue interval).
type RetryConfiguration interface {
  // Get retry interval. Should be greater than 1 minute.
  // A return value of zero means to use the framework default.
  GetRetryInterval() time.Duration
}
```

interface), or otherwise will be set to the effective requeue interval (see below).

## Tuning the requeue behavior

If a component was successfully reconciled, another reconciliation will be scheduled after 10 minutes, by default.
This default requeue interval may be overridden by the component by implementing the

```go
package component

// The RequeueConfiguration interface is meant to be implemented by components (or their spec) which offer
// tweaking the requeue interval (by default, it would be 10 minutes).
type RequeueConfiguration interface {
  // Get requeue interval. Should be greater than 1 minute.
  // A return value of zero means to use the framework default.
  GetRequeueInterval() time.Duration
}
```

interface.

## Tuning the timeout behavior

If the dependent objects of a component do not reach a ready state after a certain period, the component enters a timeout state. That means:
- if the component was in `Processing` state, then the state switches to `Error`, and the reason of the `Ready` condition is set to `Timeout`
- if the reconciler encounters a retriable error, then the state is `Pending`, and the reason of the `Ready` condition is set to `Timeout`
- if the reconciler encounters a non-retriable error, then the state is `Error`, and the reason of the `Ready` condition is set to `Timeout`.

This timeout restarts counting down whenever something changed in the component or its references, and by default has the value
of the effective requeue interval, which in turn defaults to 10 minutes.
The timeout may be overridden by the component by implementing the 

```go
package component

// The TimeoutConfiguration interface is meant to be implemented by components (or their spec) which offer
// tweaking the processing timeout (by default, it would be the value of the requeue interval).
type TimeoutConfiguration interface {
  // Get timeout. Should be greater than 1 minute.
  // A return value of zero means to use the framework default.
  GetTimeout() time.Duration
}
```

interface.

## Tuning the handling of dependent objects

The reconciler allows to tweak how dependent objects are applied to or deleted from the cluster.
To change the shipped framework defaults, a component type can implement the 

```go
package component

// The PolicyConfiguration interface is meant to be implemented by compoments (or their spec) which offer
// tweaking policies affecting the dependents handling.
type PolicyConfiguration interface {
  // Get adoption policy.
  // Must return a valid AdoptionPolicy, or the empty string (then the reconciler/framework default applies).
  GetAdoptionPolicy() reconciler.AdoptionPolicy
  // Get update policy.
  // Must return a valid UpdatePolicy, or the empty string (then the reconciler/framework default applies).
  GetUpdatePolicy() reconciler.UpdatePolicy
  // Get delete policy.
  // Must return a valid DeletePolicy, or the empty string (then the reconciler/framework default applies).
  GetDeletePolicy() reconciler.DeletePolicy
  // Get namspace auto-creation policy.
  // Must return a valid MissingNamespacesPolicy, or the empty string (then the reconciler/framework default applies).
  GetMissingNamespacesPolicy() reconciler.MissingNamespacesPolicy
}
```

interface. Note that most of the above policies can be overridden on a per-object level by setting certain annotations, as described [here](../dependents).

## Declaring additional (implicit) managed types

One of component-operator-runtime's core features is the special handling of instances of managed types.
Managed types are API extension types (such as custom resource definitions or types added by federation of an aggregated API server). Instances of these types which are part of the component (so-called managed instances) are treated differently. For example, the framework tries to process these instances as late as possible when applying the component, and as early as possible when the component is deleted. Other instances of these types which are not part of the component (so-called foreign instances) block the deletion of the whole component.

Sometimes, components are implicitly adding extension types. That means that the type definition is not part of the component manifest, but the types are just created at runtime by controllers or operators contained in the component. A typical example are crossplane providers. These types are of course not recognized by the framework as managed types. However it is probably desired that (both managed and foreign) instances of these types experience the same special handling like instances of real managed types.

To make this possible, components can implement the

```go
package component

// The TypeConfiguration interface is meant to be implemented by compoments (or their spec) which allow
// to specify additional managed types.
type TypeConfiguration interface {
  // Get additional managed types; instances of these types are handled differently during
  // apply and delete; foreign instances of these types will block deletion of the component.
  // The fields of the returned TypeInfo structs can be concrete api groups, kinds,
  // or wildcards ("*"); in addition, groups can be specified as a pattern of the form "*.<suffix>"",
  // where the wildcard matches one or multiple dns labels.
  GetAdditionalManagedTypes() []reconciler.TypeInfo
}
```

interface. The types returned by `GetAdditionalManagedTypes()` contain a group and a kind, such as

```go
// TypeInfo represents a Kubernetes type.
type TypeInfo struct {
  // API group.
  Group string `json:"group"`
  // API kind.
  Kind string `json:"kind"`
}
```

To match multiple types, the following pattern syntax is supported:
- the `Group` can be just `*` or have the form `*.domain.suffix`; note that the second case, the asterisk matches one or multiple DNS labels
- the `Kind` can be just `*`, which matches any kind.

## Tuning the reapply behavior

If a requeue of a successfully reconciled dependent object happens, the reconciler usually only compares the object's current digest with the target digest, and decides based on that comparison whether a reapply is needed or not. In addition, the framework performs forced reapplies from time to time,
in order to correct out-of-band discrepancies (e.g. resulting from manual changes). These forced repplies happen be default every 60 minutes, but this
interval can be tuned by the component by implementing the 

```go
package component

// The ReapplyConfiguration interface is meant to be implemented by components (or their spec) which allow
// to tune the force-reapply interval.
type ReapplyConfiguration interface {
  // Get force-reapply interval. Should be greater than the effective requeue interval. If a value smaller than the
  // effective requeue interval is specified, the force-reapply might be delayed until the requeue happens.
  // A return value of zero means to use the framework default.
  GetReapplyInterval() time.Duration
}
```

interface. Note that the reapply interval can also be overridden on a per-object level, as described [here](../dependents).