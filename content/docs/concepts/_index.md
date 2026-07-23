---
title: "Concepts"
linkTitle: "Concepts"
weight: 10
type: "docs"
description: >
  Core concepts and architecture of component-operator-runtime
---

This section explains the core concepts behind component-operator-runtime and how
its pieces fit together.

At its heart, the framework is built in two layers:

- The **[Object Reconciler](reconciler/)** (package `pkg/reconciler`) is the low-level
  engine. It takes a set of resource manifests and maintains them in a target
  Kubernetes cluster — creating, updating, ordering, and deleting objects, detecting
  drift, tracking an inventory, and safeguarding extension types such as CRDs.

- The **[controller-runtime integration](controller-runtime/)** (package `pkg/component`)
  builds on top of the Object Reconciler. It wires the engine into the
  controller-runtime world: a generic `Reconciler[T]` drives a custom resource type
  (the *component*), rendering its manifests through a *generator* and delegating the
  actual cluster synchronization to the Object Reconciler.

If you are new to the framework, start with **[Getting Started](../getting-started/)**.
