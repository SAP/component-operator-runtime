---
title: "Dependent Objects"
linkTitle: "Dependent Objects"
weight: 70
type: "docs"
description: >
  The full set of annotations supported on dependent objects
---

*Dependent objects* are the resources managed by the reconciler — in the higher-level
model, the objects returned by a generator's `Generate()` method. Under normal
circumstances the framework manages their lifecycle well: it creates and deletes them
in a sensible order and avoids the obvious transient errors.

For finer control, a manifest may carry any of the annotations below. In all cases the
annotation prefix is the reconciler `name` — the value passed to `NewReconciler()`.
Throughout this page it is written as `mycomponent-operator.mydomain.io`, but it is a
variable; your operator will use its own prefix.

Annotation values are normalized before being evaluated, so `PascalCase`, `camelCase`,
and `kebab-case` representations are all accepted. For example, `IfUnowned`,
`ifUnowned`, and `if-unowned` are equivalent values for `…/adoption-policy`.

## Annotation reference

### `…/adoption-policy`

How the reconciler reacts if the object exists but has no or a different owner.

- `never` — fail if the object exists and has no or a different owner
- `if-unowned` — adopt the object if it has no owner set
- `always` — adopt the object even if it has a conflicting owner

See [Policies → Adoption policy](../policies/#adoption-policy--and-the-ownership-concept).

### `…/reconcile-policy`

When the object is reconciled.

- `on-object-change` — reconcile whenever the generated manifest changes
- `on-object-or-component-change` — reconcile whenever the manifest changes, or the owning component changes
- `once` — reconcile once, then never touch the object again

See [Policies → Reconcile policy](../policies/#reconcile-policy).

### `…/update-policy`

How an existing object is updated.

- `replace` — regular `PUT` to the Kubernetes API
- `ssa-merge` — server-side apply, leaving foreign non-conflicting fields untouched
- `ssa-override` — server-side apply, additionally reclaiming fields owned by field managers such as `kubectl`
- `recreate` — delete and re-create the object instead of updating in place

See [Policies → Update policy](../policies/#update-policy).

### `…/delete-policy`

What happens to the object when it becomes redundant or the component is deleted.

- `delete` — send a delete request to the Kubernetes API
- `orphan` — never delete; stop tracking the object in both cases (redundant during apply, and on component deletion)
- `orphan-on-apply` — orphan only when the object becomes redundant during apply
- `orphan-on-delete` — orphan only when the component itself is deleted

See [Policies → Delete policy](../policies/#delete-policy).

### `…/apply-order`

The wave in which the object is applied. Dependents are reconciled wave by wave in
ascending order; the reconciler proceeds to the next wave only once all objects of the
previous wave are ready. Value: an integer in **-32768 to 32767**; unset means **0**.

See [Apply and Delete Waves](../waves/).

### `…/purge-order`

The wave at the end of which the object is purged (deleted from the cluster while
remaining as a `Completed` record in the inventory). Useful for ephemeral, hook-like
objects. Value: an integer in **-32768 to 32767**. Namespaces, CRDs and APIServices
must **not** set a purge order.

See [Completion](../completion/).

### `…/delete-order`

The wave in which the object is deleted, when it is redundant or the component is being
deleted. Deletion proceeds wave by wave in ascending order; the next wave starts only
once all objects of previous waves are gone. Value: an integer in **-32768 to 32767**;
unset means **0**. The delete order is fully independent of the apply order.

See [Apply and Delete Waves](../waves/).

### `…/reapply-interval`

The interval after which the object is force-reapplied even when it appears to be in
sync. If unset, the reconciler (or component) default is used. Because a reapply can
only happen during a reconciliation, choose a value noticeably larger than the
effective requeue interval.

See [Drift Detection](../drift-detection/).

### `…/status-hint`

A comma-separated list of hints that help the framework determine the object's
readiness when vanilla kstatus is insufficient. Supported hints:

- `has-observed-generation` — treat the object as having a `status.observedGeneration`
  field even if it is not yet set (for controllers that set it lazily)
- `has-ready-condition` — require a `Ready` condition; if absent, treat status as `Unknown`
- `conditions=<list>` — a semicolon-separated list of additional condition types that
  must all be present with status `True`

Hints may be combined, e.g.
`has-observed-generation,has-ready-condition,conditions=Synced;Healthy`.

See [Enhanced Status Detection](../status-detection/).

### `…/disable-events`

Whether the reconciler emits Kubernetes events on the object (such as created, updated,
or deleted) while managing it. Set to `true` to suppress events for this particular
object. This is the per-object counterpart of the reconciler-wide
`ReconcilerOptions.DisableEvents`; if events are already disabled globally, this
annotation has no additional effect.

See [Reconciler Options](../overview/#reconcileroptions).

## Annotations set by the reconciler

The reconciler also *writes* some annotations/labels (all prefixed with the reconciler
`name`) that you should not set yourself:

- `…/owner-id` — identifies the managing owner (also stored as a label)
- `…/digest` — the digest used for [drift detection](../drift-detection/)
