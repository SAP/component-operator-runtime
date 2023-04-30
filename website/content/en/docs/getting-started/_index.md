---
title: "Getting Started"
linkTitle: "Getting Started"
weight: 10
type: "docs"
description: >
  How to get started
---

In this short tutorial you will learn how to scaffold a Kubernetes component operator using component-operator-runtime,
and how to start with the implementation of the operator.

First of all, you have to download the component-operator-runtime scaffolding tool from the [releases page](https://github.com/sap/component-operator-runtime/releases/). In the following we assume that the downloaded `scaffold` executable
is somehwere in your path (for example, as /usr/local/bin/scaffold-component-operator).

Then, you need a git repository for the operator code; in this example, we call it github.com/myorg/mycomponent-operator.
We assume that you have cloned the empty repository to your local desktop, and have changed the current directory
to the checked out repository.

We assume here that you are implementing a [Kyma module operator](https://github.com/kyma-project/template-operator), and that
the managed component shall be represented by a Kubernetes type called `MyComponent`. Then run:

```bash
scaffold-component-operator \
  --group-name operator.kyma-project.io \
  --group-version v1alpha1 \
  --kind MyComponent \
  --operator-name mycomponent-operator.kyma-project.io \
  --go-module github.com/myorg/mycomponent-operator \
  --image mycomponent-operator:latest \
  .
```

This will give you a syntactically correct Go module. In order to start the operator, you first have to apply the
custom resource definition into your development (e.g. kind) cluster:

```bash
kubectl apply -f crds/operator.kyma-project.io_mycomponents.yaml
```

Then, after copying or linking the cluster's kubeconfig to `./tmp/kubeconfig` (no worries, it will not submitted to git because `./tmp` is excluded by `.gitignore`), you can use the generated `./vscode/launch.json` to start the
operator against your cluster with your Visual Studio Code. Now you are ready to instantiate your component:

```bash
kubectl apply -f - <<END
apiVersion: operator.kyma-project.io/v1alpha1
kind: MyComponent
metadata:
  namespace: default
  name: test
END
```

Now, after having the skeleton generated, we have to breathe life into the controller.
The first step is to enhance the spec of the generated custom resource type `MyComponentSpec` in `api/v1alpha1/types.go`.
In principle, all the attributes parameterizing the deployment of the managed commponent should be modeled there.

Whenever you change the runtime type, you should invoke `make generate` and `make manifests` in order to 
update the generated code artifacts and the custom resource definition; afterwards you should re-apply the
custom resource definition to the cluster.

The next step is to implement a meaningful resource generator (the scaffolding just put a dummy implementation called `DummyGenerator` into `main.go`). Writing such a resource generator means to implement the interface

```go
type Generator interface {
	Generate(namespace string, name string, parameters types.Unstructurable) ([]client.Object, error)
}
```

When called by the framework, the `namespace`, `name`, `parameters` arguments of the `Generate()` method will be assigned the respective return values
of the `GetDeploymentNamespace()`, `GetDeploymentName()`, `GetSpec()` methods of the component's runtime custom resource type.
In other words, the spec of the component resource will be fed into the resource generator, which will return the
concrete manifests of the dependent objects, which will then be applied to the cluster.

In some cases, the best option is to implement your own resource generator from scratch. When doing so, the returned resources `[]client.Object` either have to be of type `*unstructured.Unstructured`, or the according type must be known to the scheme supplied to the component-operator-runtime reconciler.

In many other cases however, it makes more sense to just reuse one of the [generic generators shipped with this 
  repository](../generators).



