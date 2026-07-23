---
title: "Object Reconciler"
linkTitle: "Object Reconciler"
weight: 20
type: "docs"
description: >
  The low-level engine that maintains resource manifests in a Kubernetes cluster
---

The **Object Reconciler** (package `pkg/reconciler`) is the workhorse — the core, the
low-level engine — of component-operator-runtime. Its main type, `Reconciler`, takes a
set of provided resource manifests and maintains them in a target Kubernetes cluster.

Everything the higher-level [controller-runtime integration](../controller-runtime/)
does eventually funnels down into this engine. You can also use it directly, without
the component abstraction, whenever you need programmatic, ordered, drift-aware
management of a set of Kubernetes objects.

## What it does

Given a list of manifests and a persisted *inventory*, the reconciler will:

- **create** objects that do not exist yet,
- **update** objects that have drifted from their desired manifest,
- **remove** objects that are no longer part of the desired set,
- **order** all of this into *apply* and *delete* waves,
- **track** everything it manages in the inventory,
- **protect** extension types (CRDs, aggregated APIs) from unsafe deletion.

## How it is driven

The reconciler is **level-based** and **idempotent**. Its `Apply()` and `Delete()`
methods are designed to be called repeatedly: each call moves the cluster a step
closer to the desired state and reports — via a boolean return value — whether the
target state has been fully reached. The caller persists the inventory between
invocations and re-invokes until the operation is complete.

## Topics

- **[Overview](overview/)** — the `Reconciler` type, its constructor, options, and the `Apply` / `Delete` / `IsDeletionAllowed` methods.
- **[Drift Detection](drift-detection/)** — how object digests and the reapply interval keep the cluster in sync.
- **[Apply and Delete Waves](waves/)** — ordering the reconciliation of dependent objects.
- **[Completion](completion/)** — ephemeral objects and the purge order.
- **[Policies](policies/)** — adoption (ownership), reconcile, update, delete, and missing-namespaces policies.
- **[Managed Types](managed-types/)** — special handling of CRDs and APIService types, and the stuck-finalizer safeguard.
- **[Dependent Objects](dependents/)** — the full list of annotations supported on dependent objects.
- **[Enhanced Status Detection](status-detection/)** — why vanilla kstatus is not enough, and how it is improved.
- **[The Inventory](inventory/)** — anatomy and function of the inventory.
- **[Client and Scheme](client/)** — requirements on the client and scheme passed to the reconciler.
