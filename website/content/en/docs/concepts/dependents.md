---
title: "Dependent Objects"
linkTitle: "Dependent Objects"
weight: 30
type: "docs"
description: >
  Lifecycle of dependent objects
---

Dependent objects are - by definition - the resources returned by the `Generate()` method of the used resource generator.
Under normal circumstances, the framework manages the lifecycle of the dependent objects in a sufficient way.
For example, it will create and delete objects in a meaningful order, trying to avoid corresponding transient or permanent errors.

A more remarkable feature of component-operator-runtime is that it will block deletion of dependent objects
as long as non-managed instances of managed extension types (such as custom resource definitions) exist.
To be more precise, assume for example, that the managed component contains some custom resource definition, plus the according operator.
Then, if the component resource would be deleted, none of the component's dependent objects would be touched as long as there exist foreign
instances of the managed custom resource definition in the cluster.

In some special situations however, it is desirable to have more control on the lifecycle of the dependent objects.
To support such cases, the `Generator` implementation can set the following annotations in the manifests of the dependents:
- `mycomponent-operator.mydomain.io/reconcile-policy`: defines how the object is reconciled; can be one of:
  - `on-object-change` (which is the default): the object will be reconciled whenever its generated manifest changes
  - `on-object-or-component-change`: the object will be reconciled whenever its generated manifest changes, or whenever the responsible component object changes by generation
  - `once`: the object will be reconciled once, but never be touched again
- `mycomponent-operator.mydomain.io/update-policy`: defines how the object (if existing) is updated; can be one of:
  - `default` (which is the default): a regular update (i.e. PUT) call will be made to the Kubernetes API server
  - `recreate`: if the object would be updated, it will be deleted and recreated instead
- `mycomponent-operator.mydomain.io/order`: the order at which this object will be reconciled; dependents will be reconciled order by order; that is, objects of the same order will be deployed in the canonical order, and the controller will only proceed to the next order if all objects of previous orders are ready; specified orders can be negative or positive numbers between -32768 and 32767, objects with no explicit order set are treated as order 0.
- `mycomponent-operator.mydomain.io/purge-order`: (optional) the order after which this object will be purged

Note that, in the above paragraph, `mycomponent-operator.mydomain.io` has to be replaced with whatever was passed as `name` when calling `NewReconciler()`.

