---
title: "Apply and Delete Waves"
linkTitle: "Apply and Delete Waves"
weight: 30
type: "docs"
description: >
  Ordered reconciliation of dependent objects using waves
---

Dependent objects are reconciled in **waves**. Waves let you express dependencies
between objects so that prerequisites are fully in place before the objects that rely
on them are applied — and, on the way out, so that objects are removed in a safe order.

Apply order and delete order are **independent** of each other and are configured with
separate annotations. Order values are integers in the range **-32768 to 32767**;
objects without an explicit order are treated as order **0**.

## Apply waves

During `Apply()`, objects are reconciled wave by wave in **ascending** apply order. The
next wave is only started once **all** objects of the previous wave have reached a
**ready** state — not merely created or updated, but actually ready according to their
[status](../status-detection/). This guarantees that, for example, a CRD and its
operator are fully running before instances of that CRD are applied.

Set the apply order with the annotation:

```yaml
metadata:
  annotations:
    mycomponent-operator.mydomain.io/apply-order: "-10"   # early wave
```

```yaml
metadata:
  annotations:
    mycomponent-operator.mydomain.io/apply-order: "10"    # late wave
```

(The annotation prefix is the reconciler `name`.)

## Delete waves

During deletion — whether because an object became redundant or because the whole set
is being deleted — objects are removed wave by wave, again in **ascending** order. The
next delete wave begins only once all objects of the previous wave are **fully gone**
(the API returns `404 Not Found`); merely having issued the delete call is not enough.

The delete order is set separately and is completely independent of the apply order:

```yaml
metadata:
  annotations:
    mycomponent-operator.mydomain.io/delete-order: "5"
```

A common pattern is to mirror the apply order so that teardown happens in the reverse
sequence of setup:

```yaml
# CRD: applied first, deleted last
metadata:
  annotations:
    mycomponent-operator.mydomain.io/apply-order: "-10"
    mycomponent-operator.mydomain.io/delete-order: "10"
```

## Implicit ordering within a wave

Within a single wave, the reconciler applies a built-in ordering to avoid obvious
problems. In particular:

- **Namespaces** are created before namespaced objects that live in them.
- **RBAC** objects are reconciled early.
- If a wave contains **instances of managed types** (for example, custom resources
  whose CRD is part of the same set), those instances are processed only after all
  other objects in the wave are ready. Concretely, within each apply order objects are
  handled in three sub-stages: regular objects first, then *late* objects (currently
  APIServices), then instances of managed types.
- A symmetric logic applies during deletion: instances of managed types are deleted
  before the remaining objects of the wave, and a namespace is only deleted once it is
  no longer used by any object in the inventory.

You do not need to account for this implicit ordering in your annotations — it happens
automatically on top of the wave numbers you assign.

### Consistency rules

Because managed instances and namespaces are constrained relative to their managing
type or namespace, the reconciler validates the object set and rejects inconsistent
orders. Specifically:

- a managed instance must not have an apply order **lower** than its managing type, nor
  a delete order **higher** than its managing type;
- a namespaced object must not have an apply order lower than its namespace, nor a
  delete order higher than its namespace.

## See also

- [Completion](../completion/) — the related *purge order*, which removes an object at
  the end of an apply wave.
- [Managed Types](../managed-types/) — why instances of extension types are ordered specially.
