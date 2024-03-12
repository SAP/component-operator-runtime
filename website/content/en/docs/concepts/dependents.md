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
Then, if the component resource gets deleted, none of the component's dependent objects will be touched as long as there exist foreign
instances of the managed custom resource definition in the cluster.

In some special situations, it is desirable to have even more control on the lifecycle of the dependent objects.
To support such cases, the `Generator` implementation can set the following annotations in the manifests of the dependents:
- `mycomponent-operator.mydomain.io/adoption-policy`: defines how the reconciler reacts if the object exists but has no or a different owner; can be one of:
  - `never`: fail if the object exists and has no or a different owner
  - `if-unowned` (which is the default): adopt the object if it has no owner set
  - `always`: adopt the object, even if it has a conflicting owner
- `mycomponent-operator.mydomain.io/reconcile-policy`: defines how the object is reconciled; can be one of:
  - `on-object-change` (which is the default): the object will be reconciled whenever its generated manifest changes
  - `on-object-or-component-change`: the object will be reconciled whenever its generated manifest changes, or whenever the responsible component object changes by generation
  - `once`: the object will be reconciled once, but never be touched again
- `mycomponent-operator.mydomain.io/update-policy`: defines how the object (if existing) is updated; can be one of:
  - `default` (deprecated): equivalent to the annotation being unset (which means that the reconciler default will be used)
  - `replace` (which is the default): a regular update (i.e. PUT) call will be made to the Kubernetes API server
  - `ssa-merge`: use server side apply to update existing dependents
  - `ssa-override`: use server side apply to update existing dependents and, in addition, reclaim fields owned by certain field owners, such as kubectl or helm 
  - `recreate`: if the object would be updated, it will be deleted and recreated instead
- `mycomponent-operator.mydomain.io/delete-policy`: defines what happens if the object is deleted; can be one of:
  - `default` (deprecated): equivalent to the annotation being unset (which means that the reconciler default will be used)
  - `delete` (which is the default): a delete call will be sent to the Kubernetes API server
  - `orphan`: the object will not be deleted, and it will be no longer tracked
- `mycomponent-operator.mydomain.io/apply-order`: the wave in which this object will be reconciled; dependents will be reconciled wave by wave; that is, objects of the same wave will be deployed in a canonical order, and the reconciler will only proceed to the next wave if all objects of previous waves are ready; specified orders can be negative or positive numbers between -32768 and 32767, objects with no explicit order set are treated as if they would specify order 0
- `mycomponent-operator.mydomain.io/purge-order` (optional): the wave by which this object will be purged; here, purged means that, while applying the dependents, the object will be deleted from the cluster at the end of the specified wave; the according record in `status.Inventory` will be set to phase `Completed`; setting purge orders is useful to spawn ad-hoc objects during the reconcilation, which are not permanently needed; so it's comparable to helm hooks, in a certain sense
- `mycomponent-operator.mydomain.io/delete-order` (optional): the wave by which this object will be deleted; that is, if the dependent is no longer part of the component, or if the whole component is being deleted; dependents will be deleted wave by wave; that is, objects of the same wave will be deleted in a canonical order, and the reconciler will only proceed to the next wave if all objects of previous saves are gone; specified orders can be negative or positive numbers between -32768 and 32767, objects with no explicit order set are treated as if they would specify order 0; note that the delete order is completely independent of the apply order
- `mycomponent-operator.mydomain.io/status-hint` (optional): a comma-separated list of hints that may help the framework to properly identify the state of the annotated dependent object; currently, the following hints are possible:
  - `has-observed-generation`: tells the framework that the dependent object has a `status.observedGeneration` field, even if it is not (yet) set by the responsible controller (some controllers are known to set the observed generation lazily, with the consequence that there is a period right after creation of the dependent object, where the field is missing in the dependent's status)
  - `has-ready-condition`: tells the framework to count with a ready condition; if it is absent, the condition state will be considered as `Unknown`

Note that, in the above paragraph, `mycomponent-operator.mydomain.io` has to be replaced with whatever was passed as `name` when calling `NewReconciler()`.

