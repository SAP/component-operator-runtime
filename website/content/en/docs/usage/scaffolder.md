---
title: "Scaffolder"
linkTitle: "Scaffolder"
weight: 20
type: "docs"
description: >
  Generate a component operator by the comonent-operator-runtime scaffolder
---

The recommended way to get started with the implementation of a new component operator is to use the included
scaffolding tool, which can be downloaded from the [releases page](https://github.com/sap/component-operator-runtime/releases/).

After installing the scaffolder, a new project can be created like this:

```bash
scaffold \
  --group-name operator.group.my-domain.io \
  --kind MyComponent \
  --operator-name mycomponent-operator.group.my-domain.io \
  --go-module github.com/myorg/mycomponent-operator \
  --image mycomponent-operator:latest \
  <output-directory>
```

In this example, some options were left out, using the according default values; the full option list is as follows:

```
Usage: scaffold [options] [output directory]
  [output directory]: Target directory for the generated scaffold; must exist
  [options]:
      --version                                    Show version
      --owner string                               Owner of this project, as written to the license header (default "SAP SE")
      --spdx-license-headers                       Whether to write license headers in SPDX format
      --group-name string                          API group name
      --group-version string                       API group version (default "v1alpha1")
      --kind string                                API kind for the component
      --resource string                            API resource (plural) for the component; if empty, it will be the pluralized kind
      --operator-name string                       Unique name for this operator, used e.g. for leader election and labels; should be a valid DNS hostname
      --with-validating-webhook                    Whether to scaffold validating webhook
      --with-mutating-webhook                      Whether to scaffold mutating webhook
      --go-version string                          Go version to be used (default "1.23.4")
      --go-module string                           Name of the Go module, as written to the go.mod file
      --kubernetes-version string                  Kubernetes go-client version to be used (default "v0.32.0")
      --controller-runtime-version string          Controller-runtime version to be used (default "v0.19.3")
      --controller-tools-version string            Controller-tools version to be used (default "v0.16.5")
      --code-generator-version string              Code-generator version to be used (default "v0.32.0")
      --admission-webhook-runtime-version string   Admission-webhook-runtime version to be used (default "v0.1.52")
      --envtest-kubernetes-version string          Kubernetes version to be used by envtest (default "1.30.3")
      --image string                               Name of the Docker/OCI image produced by this project (default "controller:latest")
      --skip-post-processing                       Skip post-processing
```

After generating the scaffold, the next steps are:
- Enhance the spec type of the generated custom resource type (in `api/<group-version>/types.go`) according to the needs of
  your component
- Implement a meaningful resource generator and use it in `main.go` instead of `manifests.NewDummyGenerator()`;
  to do so you can either implement your own generator, or reuse one of the [generic generators shipped with this
  repository](../../generators).