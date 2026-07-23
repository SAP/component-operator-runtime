---
title: "Managed Types"
linkTitle: "Managed Types"
weight: 60
type: "docs"
description: >
  Special handling of CRDs and APIService types, and the stuck-finalizer safeguard
---

One of the defining features of component-operator-runtime is its careful handling of
**managed types** — the API *extension* types that a component may bring along.

## What is a managed type?

A managed type is a Kubernetes API extension type, namely:

- a type defined by a **CustomResourceDefinition** contained in the object set, or
- a type served by an **APIService** (aggregated API server) contained in the object set.

Instances of such a type are classified as:

- **managed instances** — instances that are part of the same object set (tracked in
  the inventory), or
- **foreign instances** — instances that exist in the cluster but are *not* part of the
  set (created by users or other controllers).

Managed instances are treated specially during ordering: they are applied *as late as
possible* (after everything they might depend on is ready) and deleted *as early as
possible*. This is the implicit within-wave ordering described in
[Apply and Delete Waves](../waves/).

## The stuck-finalizer problem

Consider a component that ships a CRD, the operator that reconciles instances of that
CRD, and possibly some instances. Now suppose a user has *also* created their own
instances of that CRD. If the component were torn down naively:

1. the CRD gets deleted,
2. the operator gets deleted,
3. the user's instances are left behind, typically with finalizers,
4. but **no controller remains** to process those finalizers,
5. so the instances — and often their namespaces — can never be removed.

This is the classic *stuck finalizer* deadlock.

## The safeguard: `IsDeletionAllowed`

component-operator-runtime prevents this. It will **block deletion** of the object set
as long as foreign instances of a contained managed type still exist. The check is
exposed through:

```go
func (r *Reconciler) IsDeletionAllowed(
	ctx context.Context,
	inventory *[]*InventoryItem,
	ownerId string,
) (bool, string, error)
```

The method inspects every CRD and APIService in the inventory (plus any
[additional managed types](#additional-managed-types)) and returns `false`, with a
human-readable reason, if foreign instances are found. As an exception, it returns
`true` immediately if **all** inventory items have delete policy `Orphan` or
`OrphanOnDelete` — in that case nothing is actually being deleted, so there is nothing
to guard.

A caller (for example, the higher-level component reconciler) uses it roughly like this:

```go
allowed, reason, err := rec.IsDeletionAllowed(ctx, &inventory, ownerId)
if err != nil {
	return err
}
if !allowed {
	// surface `reason` in the component status and requeue; do not delete yet
	return nil // or schedule another reconcile after a little while
}
// safe to proceed
done, err := rec.Delete(ctx, &inventory, ownerId)
```

The practical effect: the operator that is part of the component stays alive as long as
foreign custom resources exist, giving their owners a chance to clean up consistently.

## Additional managed types

Sometimes a component's workloads create CRDs *implicitly* — the CRD is not part of the
rendered manifests, so the reconciler cannot discover it automatically. To extend the
same protection to such types, declare them via `ReconcilerOptions.AdditionalManagedTypes`:

```go
rec := reconciler.NewReconciler(name, clnt, reconciler.ReconcilerOptions{
	AdditionalManagedTypes: []reconciler.TypeInfo{
		{Group: "example.com", Kind: "Widget"},
		{Group: "other.io", Kind: "Gadget"},
	},
})
```

`TypeInfo` identifies a type by group and kind:

```go
type TypeInfo struct {
	Group string
	Kind  string
}
```

Wildcards can be used as follows:
- The kind can be provided as  `*`, which matches any value.
- The group can be `*` (matching any value) or have the form `*.suffix`; in this case, the asterisk matches one or multiple DNS labels.

Foreign instances of these additional types will then block deletion just like foreign
instances of an explicitly contained CRD or APIService, closing the gap for
implicitly-created extension types.

## See also

- [Apply and Delete Waves](../waves/) — the ordering rules that managed instances follow.
- [Policies](../policies/) — how `Orphan`/`OrphanOnDelete` interact with the deletion safeguard.
