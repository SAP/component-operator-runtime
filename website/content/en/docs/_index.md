---
title: "component-operator-runtime"
linkTitle: "component-operator-runtime"
weight: 40
type: "docs"
---

This repository provides a framework supporting the development of Kubernetes operators
managing the lifecycle of arbitrary deployment components of Kubernetes clusters, with a special focus
on such components that are or contain Kubernetes operators themselves.

It can therefore serve as a starting point to develop [SAP Kyma module operators](https://github.com/kyma-project/template-operator),
but can also be used independently of Kyma. While being perfectly suited to develop opiniated operators like Kyma module operators, it can be
equally used to cover more generic use cases. A prominent example for such a generic operator is the [SAP component operator](https://github.com/sap/component-operator) which can be compared to flux's [kustomize controller](https://github.com/fluxcd/kustomize-controller) and [helm controller](https://github.com/fluxcd/helm-controller).

If you want to report bugs, or request new features or enhancements, please [open an issue](https://github.com/sap/component-operator-runtime/issues)
or [raise a pull request](https://github.com/sap/component-operator-runtime/pulls).