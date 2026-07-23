---
title: "Drift Detection"
linkTitle: "Drift Detection"
weight: 20
type: "docs"
description: >
  How object digests and the reapply interval keep the cluster in sync
---

The Object Reconciler continuously keeps managed objects in sync with their desired
manifests. It detects when an object has *drifted* from its declared state and
reapplies it. This page explains the mechanism.

## Object digest

Whenever a dependent object is about to be applied, the reconciler calculates a
**digest** of the object and persists it as an annotation on the object:

```yaml
metadata:
  annotations:
    mycomponent-operator.mydomain.io/digest: "..."
```

(The annotation prefix is the reconciler `name`, so it varies per operator.)

On each reconciliation the reconciler computes the digest of the *desired* manifest and
compares it with the digest stored in the inventory:

- if the new digest **differs** from the stored one, the object is considered **out of
  sync** and is reapplied to the Kubernetes API (with the new digest);
- if the digest is **unchanged**, the object is considered **in sync** and is normally
  **not** reapplied.

There is one important exception to "not reapplied": the reapply interval, described below.

### What goes into the digest

By default only the submitted manifest is considered. This behavior depends on the
effective [reconcile policy](../policies/):

- With `OnObjectChange` (the default), the digest reflects only the object's own manifest.
- With `OnObjectOrComponentChange`, the `componentDigest` passed to `Apply()` is folded
  in as well, so that any change to the owning component triggers an immediate
  reconciliation of the object.
- With `Once`, the object is applied a single time and then never touched again.

## Reapply interval

Even when an object is in sync, the reconciler will **force-reapply** it once the
*reapply interval* has elapsed since it was last applied. This reverts manual,
out-of-band changes that would otherwise be invisible to the digest comparison (for
example, someone editing the live object directly), and it heals other transient glitches.

The default reapply interval is **60 minutes**.

It can be configured on the reconciler and per object.
The per-object setting takes precedence over the value set on the reconciler.

1. **Reconciler level** — via `ReconcilerOptions.ReapplyInterval`:

   ```go
   rec := reconciler.NewReconciler(name, clnt, reconciler.ReconcilerOptions{
       ReapplyInterval: new(30 * time.Minute),
   })
   ```

2. **Object level** — via an annotation on the individual manifest:

   ```yaml
   metadata:
     annotations:
       mycomponent-operator.mydomain.io/reapply-interval: "15m"
   ```

> Because a force-reapply can only happen during a reconciliation, the reapply interval
> should be chosen noticeably larger than the effective requeue interval; otherwise the
> requeue cadence, not the reapply interval, dominates when drift is corrected.

## See also

- [Policies](../policies/) — the reconcile, update and adoption policies referenced above.
- [The Inventory](../inventory/) — where the digest and last-applied timestamp are stored.
