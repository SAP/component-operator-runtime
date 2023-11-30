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

Other than existing tools addressing this case, such as the [Operator Lifecycle Manager (OLM)](https://olm.operatorframework.io/),
this project proposes a more opinionated programming model. That is, the idea is to represent the managed component by an own custom resource type,
which (usually) will be instantiated only once in the cluster. We feel encouraged to go this way, as many community projects are following the pattern of providing dedicated lifecycle operators.

The component-operator-runtime framework plugs into the [controller-runtime](https://github.com/kubernetes-sigs/controller-runtime) SDK by implementing the controller-runtime `Reconciler` interface.