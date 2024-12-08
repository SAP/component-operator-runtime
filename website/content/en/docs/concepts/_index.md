---
title: "Concepts"
linkTitle: "Concepts"
weight: 20
type: "docs"
description: >
  Concepts
---

The framework provided in this repository aims to automate the lifecycle of an arbitrary component in a Kubernetes cluster.
Usually (but not necessarily) the managed component contains one or multiple other operators, including extension types, such as custom resource definitions.

Components are described as a set of Kubernetes manifests. How these manifests are produced is up to the consumer of the framework.
It is possible to build up the manifests from scratch in code, or to reuse or enhance the included helm generator or kustomize generator generators.
The manifest list is then applied to (or removed from) the cluster by an own deployer logic, standing out with the following features:
- apply and delete waves
- configurable status handling
- apply through replace or server-side-apply patch
- smart deletion handling in case the component contains custom types which are still in use
- impersonination
- remote deployment mode via a given kubeconfig

The component-operator-runtime framework plugs into the [controller-runtime](https://github.com/kubernetes-sigs/controller-runtime) SDK by implementing controller-runtime's `Reconciler` interface.