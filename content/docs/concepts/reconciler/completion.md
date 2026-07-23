---
title: "Completion"
linkTitle: "Completion"
weight: 40
type: "docs"
description: >
  Ephemeral objects and the purge order
---

Sometimes an object needs to exist only *during* the reconciliation, not permanently —
think of a one-off migration job, a seeding task, or a certificate generator. The
Object Reconciler supports this through the **purge order**, which is comparable to a
Helm hook.

## Purge order

An object that carries a purge-order annotation is applied normally, waited for until
it is ready, and then **deleted from the cluster at the end of the specified apply
wave**. Its inventory record is not removed; instead it is set to the phase
`Completed`. This distinguishes purged objects from redundant objects (which are
removed from the inventory entirely).

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: db-migration
  annotations:
    mycomponent-operator.mydomain.io/apply-order: "0"
    mycomponent-operator.mydomain.io/purge-order: "0"
spec:
  template:
    spec:
      restartPolicy: OnFailure
      containers:
        - name: migrate
          image: my-app:migrate
          command: ["./run-migrations"]
```

(The annotation prefix is the reconciler `name`.)

## How it fits into a wave

Within an apply wave, once all objects of the wave are ready, any objects whose purge
order matches that wave are scheduled for completion: the reconciler deletes them and
marks their inventory records `Completed`. Only then does reconciliation proceed.

Because a `Completed` record stays in the inventory, the reconciler remembers that the
object was already handled and will not blindly recreate it. However, if the object's
manifest changes (its digest changes), or the object otherwise runs out of sync, it may
be **re-applied and re-purged** on a later reconciliation. Ephemeral objects should
therefore be **idempotent** — safe to run more than once.

## Restrictions

Some objects must **not** define a purge order, and the reconciler will reject the
object set if they do:

- **Namespaces**
- **CustomResourceDefinitions**
- **APIServices**

These are structural or extension resources whose removal mid-reconciliation would be
unsafe.

## Purge order vs. delete order

Do not confuse the two:

| Annotation | When it acts | Effect |
|------------|--------------|--------|
| `purge-order` | during **apply** | delete the object at the end of the given apply wave; keep a `Completed` inventory record |
| `delete-order` | during **delete** | control the wave in which the object is removed when it becomes redundant or the set is deleted |

See [Apply and Delete Waves](../waves/) for the delete order.
