---
title: "Documentation"
linkTitle: "Documentation"
weight: 5
description: >
  Everything you need to build Kubernetes component operators with component-operator-runtime
---

Welcome to the documentation of **component-operator-runtime**, a Golang framework
for building Kubernetes operators that manage complex, multi-resource components in
a consistent and reliable way.

More precisely, a component is a coherent set of Kubernetes resources that are applied
to a cluster as a unit. These resources are commonly called dependent objects (or
dependent resources, or simply dependents), and are described as a list of Kubernetes
manifests, usually in YAML, sometimes in JSON. Applying a component means to reconcile
the cluster with that manifest list by:
- creating objects that do not yet exist in the cluster
- updating objects whose current state differs from the target state
- deleting redundant objects, that is, objects that were part of a previous version of
  the manifest list but are no longer contained in it.

Browse the sections below to learn more.