---
title: "component-operator-runtime"
linkTitle: "component-operator-runtime"
weight: 40
type: "docs"
---

This repository provides a framework supporting the development of opinionated Kubernetes operators
managing the lifecycle of arbitrary deployment components of Kubernetes clusters, with a special focus
on such components that are or contain Kubernetes operators themselves.

It can therefore serve as a starting point to develop [SAP Kyma module operators](https://github.com/kyma-project/template-operator), 
but can also be used independently of Kyma.

Regarding its mission statement, this project could be compared with the [Operator Lifecycle Manager (OLM)](https://olm.operatorframework.io/).
However, other than OLM, which follows a generic modelling approach, component-operator-runtime encourages the development of opinionated,
concretely modeled, component-specific operators. This makes the resulting logic much more explicit, and also allows to react better on
specific lifecycle needs of the managed component.

Of course, components might equally be managed by using generic Kustomization or Helm chart deployers (such as provided by [ArgoCD](https://argoproj.github.io/) or [FluxCD](https://fluxcd.io/flux/)).
However, these tools have certain weaknesses when it is about to deploy other operators, i.e. components which extend the Kubernetes API,
e.g. by adding custom resource definitions, aggregated API servers, according controllers, and admission webhooks.
For example these generic controllers tend to produce race conditions or dead locks upon first installation or deletion of the managed components.
This is where component-operator-runtime tries to act in a smarter and more robust way.

This website provides the full technical documentation for the project, and can be
used as a reference; if you feel that there's anything missing, please let us know
or [raise a PR](https://github.com/sap/component-operator-runtime/pulls) to add it.