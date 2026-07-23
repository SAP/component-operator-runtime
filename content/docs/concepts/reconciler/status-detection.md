---
title: "Enhanced Status Detection"
linkTitle: "Enhanced Status Detection"
weight: 80
type: "docs"
description: >
  Why vanilla kstatus is not enough, and how the framework improves on it
---

Before proceeding to the next [apply wave](../waves/) — or declaring the whole set
reconciled — the reconciler must decide whether each dependent object has reached a
**ready** state. The check is performed by a `status.StatusAnalyzer`, which
you can override via `ReconcilerOptions.StatusAnalyzer`; when left unset, an optimized
kstatus-based analyzer is used. It is built on the
[kstatus](https://pkg.go.dev/sigs.k8s.io/cli-utils/pkg/kstatus/status) library, but
with important enhancements.

## How vanilla kstatus works

For most resources (some built-in types have special logic), kstatus applies roughly
the following algorithm.

**Step 1 — `observedGeneration` check:**

```
Does the object have status.observedGeneration?
  → Yes: Does status.observedGeneration equal metadata.generation?
           → Yes: proceed to the ready-condition check
           → No:  object is NOT ready (generation mismatch)
  → No:  proceed to the ready-condition check
```

**Step 2 — ready-condition check:**

```
Does the object have status.conditions[type == "Ready"]?
  → Yes: Is condition.status == "True"?
           → Yes: object is READY
           → No (False or Unknown): object is NOT ready
  → No:  object is READY (absence of a Ready condition means "implicitly ready")
```

This works well for controllers that set `status.observedGeneration` and
`status.conditions` **reliably and eagerly** — ideally the object is born with an
impossible `observedGeneration` (e.g. `-1`), which the controller then updates together
with its conditions on every reconcile.

## Why that is not sufficient

Real-world controllers frequently violate those assumptions, and vanilla kstatus then
produces **false positives** — reporting an object ready when it is not:

- A controller may not set `status.observedGeneration` immediately after the object is
  created, leaving it absent for a short window. kstatus then *skips* the generation
  check and may observe a previous, outdated, state of the `Ready` condition.
- A controller may set a `Ready` condition lazily. kstatus interprets the absent
  `Ready` condition as "implicitly ready".
- A controller may never set a `Ready` condition at all, using other condition types to signal
  readiness instead. kstatus interprets the absent `Ready` condition as "implicitly
  ready".

For a framework that gates whole waves on readiness, such false positives are dangerous:
subsequent waves could start against prerequisites that are not actually up.

## How the framework enhances it

The framework lets you supply per-object **status hints** through the
`…/status-hint` annotation (prefix = reconciler `name`), which tighten the analysis:

### `has-observed-generation`

Treat the object as having a `status.observedGeneration` field *even if it is not yet
set*. This forces the generation check for controllers that set the field lazily,
closing the window in which kstatus would otherwise skip it.

```yaml
metadata:
  annotations:
    mycomponent-operator.mydomain.io/status-hint: has-observed-generation
```

### `has-ready-condition`

Require a `Ready` condition. If it is absent, the object is treated as `Unknown` (not
ready) instead of "implicitly ready".

```yaml
metadata:
  annotations:
    mycomponent-operator.mydomain.io/status-hint: has-ready-condition
```

### `conditions=<list>`

A semicolon-separated list of additional condition types that must all be present and
have status `True` for the object to count as ready.

```yaml
metadata:
  annotations:
    mycomponent-operator.mydomain.io/status-hint: "conditions=Synced;Healthy"
```

### Combining hints

Hints are combined as a comma-separated list:

```yaml
metadata:
  annotations:
    mycomponent-operator.mydomain.io/status-hint: "has-observed-generation,has-ready-condition,conditions=Synced;Healthy"
```

## See also

- [Apply and Delete Waves](../waves/) — readiness is what gates progression between waves.
- [Dependent Objects](../dependents/) — the annotation reference, including `status-hint`.
