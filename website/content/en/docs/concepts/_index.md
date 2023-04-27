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
which (usually) would be instantiated only once in the cluster. We feel encouraged to go this way, as many community projects (e.g. Istio), are following the same pattern.
The component-operator-runtime framework now provides a generic controller, implemented as [controller-runtime](https://github.com/kubernetes-sigs/controller-runtime) reonciler, to facilitate the development of the according controller logic for the custom resource type modeling the component.