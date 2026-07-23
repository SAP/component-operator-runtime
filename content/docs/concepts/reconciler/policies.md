---
title: "Policies"
linkTitle: "Policies"
weight: 50
type: "docs"
description: >
  Adoption (ownership), reconcile, update, delete, and missing-namespaces policies
---

The behavior of the Object Reconciler is governed by five policies. Each has a
reconciler-wide default (set through `ReconcilerOptions`), and most can be overridden
per object using an annotation whose prefix is the reconciler `name`.

## Adoption policy — and the ownership concept

Before explaining the policy, it helps to understand **ownership**. When the reconciler
manages an object, it stamps it with an *owner id* (as a label and annotation). This is
how the reconciler tells its own objects apart from *foreign* ones: on every apply and
delete it verifies the owner id before touching an object. Ownership is what makes it
safe for the reconciler to update and delete resources — it will not clobber objects it
does not own, unless told to.

The **adoption policy** decides what happens when an object already exists but carries
**no owner id**, or a **different** one:

```go
type AdoptionPolicy string

const (
	// Fail if the object exists but has no or a different owner.
	AdoptionPolicyNever AdoptionPolicy = "Never"
	// Adopt the object if it has no owner set (default).
	AdoptionPolicyIfUnowned AdoptionPolicy = "IfUnowned"
	// Adopt the object even if it has a conflicting owner.
	AdoptionPolicyAlways AdoptionPolicy = "Always"
)
```

- `Never` — never take over an object that is not already owned by us; a pre-existing
  object causes an error.
- `IfUnowned` *(default)* — adopt an object that has **no** owner; fail if it is owned
  by someone else (do not steal).
- `Always` — adopt regardless, even overwriting a conflicting owner. Use with care.

Per-object override:

```yaml
metadata:
  annotations:
    mycomponent-operator.mydomain.io/adoption-policy: "never"   # never | if-unowned | always
```

## Reconcile policy

Determines *when* an object is reconciled — see [Drift Detection](../drift-detection/)
for how this influences the digest.

```go
type ReconcilePolicy string

const (
	ReconcilePolicyOnObjectChange            ReconcilePolicy = "OnObjectChange"
	ReconcilePolicyOnObjectOrComponentChange ReconcilePolicy = "OnObjectOrComponentChange"
	ReconcilePolicyOnce                      ReconcilePolicy = "Once"
)
```

- `OnObjectChange` *(default)* — reconcile when the object's own manifest changes.
- `OnObjectOrComponentChange` — reconcile when the manifest **or** the owning component
  changes (the `componentDigest` is folded into the object digest).
- `Once` — apply the object exactly once, then never touch it again.

Per-object override:

```yaml
metadata:
  annotations:
    mycomponent-operator.mydomain.io/reconcile-policy: "once"   # on-object-change | on-object-or-component-change | once
```

> Note: the reconcile policy is not a field of `ReconcilerOptions`; the reconciler-wide
> value is fixed to `OnObjectChange`, and you customize behavior per object via the
> annotation (or, in the higher-level component reconciler, per component).

## Update policy

Determines *how* an existing object is updated:

```go
type UpdatePolicy string

const (
	// Delete and re-create the object on update.
	UpdatePolicyRecreate UpdatePolicy = "Recreate"
	// Replace the object with a full PUT request.
	UpdatePolicyReplace UpdatePolicy = "Replace"
	// Server-side apply, leaving foreign non-conflicting fields untouched.
	UpdatePolicySsaMerge UpdatePolicy = "SsaMerge"
	// Server-side apply, additionally reclaiming fields owned by e.g. kubectl
	UpdatePolicySsaOverride UpdatePolicy = "SsaOverride"
)
```

- `Replace` *(default in `pkg/reconciler`)* — a regular `PUT` to the Kubernetes API.
- `SsaMerge` — [server-side apply](https://kubernetes.io/docs/reference/using-api/server-side-apply/),
  respecting other field managers' non-conflicting fields.
- `SsaOverride` — server-side apply that additionally reclaims (and may therefore drop)
  fields owned by field managers such as `kubectl`. This matches the behavior of the
  FluxCD kustomize-controller.
- `Recreate` — delete and re-create the object instead of updating it in place.

Per-object override:

```yaml
metadata:
  annotations:
    mycomponent-operator.mydomain.io/update-policy: "ssa-merge"   # replace | recreate | ssa-merge | ssa-override
```

> The higher-level component reconciler may present a different effective default (
> [component-operator](https://github.com/sap/component-operator), for instance, defaults to `SsaOverride`). At the
> `pkg/reconciler` layer described here, the default is `Replace`.

## Delete policy

Determines what happens to an object when it should be removed. Removal happens in two
situations: the object became **redundant** during an apply (no longer in the manifest
set), or the whole set is being **deleted**.

```go
type DeletePolicy string

const (
	// Delete the object (default).
	DeletePolicyDelete DeletePolicy = "Delete"
	// Orphan in both cases (redundant during apply, and on delete).
	DeletePolicyOrphan DeletePolicy = "Orphan"
	// Orphan only when it becomes redundant during apply.
	DeletePolicyOrphanOnApply DeletePolicy = "OrphanOnApply"
	// Orphan only when the whole set is deleted.
	DeletePolicyOrphanOnDelete DeletePolicy = "OrphanOnDelete"
)
```

*Orphaning* means the object is left in the cluster but is no longer tracked in the
inventory.

Per-object override:

```yaml
metadata:
  annotations:
    mycomponent-operator.mydomain.io/delete-policy: "orphan"   # delete | orphan | orphan-on-apply | orphan-on-delete
```

> The effective delete policy of a redundant object is the **last known** one — the one
> recorded in the inventory when the object was last part of the manifest set. Changing
> the reconciler default afterwards does not retroactively affect objects that are
> already being removed.

## Missing-namespaces policy

Determines what happens when a namespaced object refers to a namespace that does not exist:

```go
type MissingNamespacesPolicy string

const (
	// Do not create missing namespaces.
	MissingNamespacesPolicyDoNotCreate MissingNamespacesPolicy = "DoNotCreate"
	// Create missing namespaces (default).
	MissingNamespacesPolicyCreate MissingNamespacesPolicy = "Create"
)
```

- `Create` *(default)* — auto-create the missing namespace.
- `DoNotCreate` — do not create it (the apply will fail if the namespace is required and absent).

This policy is reconciler-wide only; there is no per-object annotation for it.

## Setting reconciler defaults

```go
rec := reconciler.NewReconciler(name, clnt, reconciler.ReconcilerOptions{
	AdoptionPolicy:          new(reconciler.AdoptionPolicyIfUnowned),
	UpdatePolicy:            new(reconciler.UpdatePolicySsaOverride),
	DeletePolicy:            new(reconciler.DeletePolicyDelete),
	MissingNamespacesPolicy: new(reconciler.MissingNamespacesPolicyCreate),
})
```

See [Dependent Objects](../dependents/) for the complete annotation reference.
