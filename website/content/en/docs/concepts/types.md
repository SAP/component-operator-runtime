---
title: "Components and Generators"
linkTitle: "Components and Generators"
weight: 10
type: "docs"
description: >
  Interfaces to be implemented by component operators
---

In the terminology of this project, a Kubernetes cluster component (sometimes called module) consists of a set of dependent objects that are to be
deployed consistently into the Kubernetes cluster. The continuous reconciliation of the declared state of these dependent objects is the task of an operator
implemented by component-operator-runtime. To achieve this, basically two interfaces have to be implemented by such an operator...

## The Component Interface

In the programming model proposed by component-operator-runtime, the declared and observed state of the component is represented by a dedicated custom resource type. The corresponding runtime type has to fulfill the following interface:

```go
package component

// Component is the central interface that component operators have to implement.
// Besides being a conroller-runtime client.Object, the implementing type has to expose accessor
// methods for the components's spec and status, GetSpec() and GetStatus().
type Component interface {
  client.Object
  // Return a read-only accessor to the component's spec.
  // The returned value has to implement the types.Unstructurable interface.
  GetSpec() types.Unstructurable
  // Return a read-write (usually a pointer) accessor to the component's status,
  // resp. to the corresponding substruct if the status extends component.Status.
  GetStatus() *Status
}
```

Basically, two accessor methods have to be implemented here. First, `GetSpec()` exposes the parameterization of the component.
The only requirement on the returned type is to implement the

```go
package types

// Unstructurable represents objects which can be converted into a string-keyed map.
// All Kubernetes API types, as well as all JSON objects could be modelled as Unstructurable objects.
type Unstructurable interface {
  ToUnstructured() map[string]any
}
```

interface. In most cases, the returned `Unstructurable` object is the spec itself, or a deep copy of the spec. In general, the implementation is allowed to return arbitrary content, as long as the receiving generator is able to process it. In particular, it is not expected by the framework that changes applied to the returned `Unstructurable` reflect in any way in the component; indeed, the framework will never modify the returned `Unstructurable`.

Finally, `GetStatus()` allows the framework to access (a part of) the custom resource type's status, having the following type:

```go
package component

// Component Status. Components must include this into their status.
type Status struct {
  ObservedGeneration int64            `json:"observedGeneration"`
  AppliedGeneration  int64            `json:"appliedGeneration,omitempty"`
  LastObservedAt     *metav1.Time     `json:"lastObservedAt,omitempty"`
  LastAppliedAt      *metav1.Time     `json:"lastAppliedAt,omitempty"`
  Conditions         []Condition      `json:"conditions,omitempty"`
  State              State            `json:"state,omitempty"`
  Inventory          []*InventoryItem `json:"inventory,omitempty"`
}
```

Note that, other than with the `GetSpec()` accessor, the framework will make changes to the returned `Status` structure.
Thus, in almost all cases, the returned pointer should just reference the status of the component's API type (or an according substructure of that status).

The component's custom resource type is supposed to be namespaced, and by default, dependent objects will be created in that same namespace. To be more precise, the `namespace` and `name` parameters of the used generator's `Generate()` method will be set to the component's `metadata.namespace` and `metadata.name`, respectively. Sometimes it might be desired to override these default, and to render the dependent objects with a different namespace or name. To allow this, the component (or its spec) can implement

```go
package component

// The PlacementConfiguration interface is meant to be implemented by components (or their spec) which allow
// to explicitly specify target namespace and name of the deployment (otherwise this will be defaulted as
// the namespace and name of the component object itself).
type PlacementConfiguration interface {
  // Return target namespace for the component deployment.
  // If the returned value is not the empty string, then this is the value that will be passed
  // to Generator.Generate() as namespace and, in addition, rendered namespaced resources with
  // unspecified namespace will be placed in this namespace.
  GetDeploymentNamespace() string
  // Return target name for the component deployment.
  // If the returned value is not the empty string, then this is the value that will be passed
  // to Generator.Generator() as name.
  GetDeploymentName() string
}
```

In addition, the component (or its spec) may implement

```go
package component

// The ClientConfiguration interface is meant to be implemented by components (or their spec) which offer
// remote deployments.
type ClientConfiguration interface {
  // Get kubeconfig content. Should return nil if default local client shall be used.
  GetKubeConfig() []byte
}
```

in order to support remote deployments (that is, to make the deployment of the dependent objects use the specified kubeconfig), and

```go
package component

// The ImpersonationConfiguration interface is meant to be implemented by components (or their spec) which offer
// impersonated deployments.
type ImpersonationConfiguration interface {
  // Return impersonation user. Should return system:serviceaccount:<namespace>:<serviceaccount>
  // if a service account is used for impersonation. Should return an empty string
  // if user shall not be impersonated.
  GetImpersonationUser() string
  // Return impersonation groups. Should return nil if groups shall not be impersonated.
  GetImpersonationGroups() []string
}
```

to use different user/groups for the deployment of dependent objects. Implementing both `ClientConfiguration` and `ImpersonationConfiguration` means that
the provided kubeconfig will be impersonated as specified.

Note that, as mentioned above, the interfaces `PlacementConfiguration`, `ClientConfiguration` and `ImpersonationConfiguration` can be implemented by the component
itself as well as by its spec type. In the theoretical case that both is the case, the implementation on the component level takes higher precedence.

## The Generator interface

While `Component` (respectively the related custom resource type) models the desired and actual state of
the managed component, the `Generator` interface is about implementing a recipe to render the Kubernetes manifests of the
dependent objects, according to the provided parameterization (spec) of the component:

```go
package manifests

// Resource generator interface.
// When called from the reconciler, the arguments namespace and name will match the
// component's namespace and name or, if the component or its spec implement the
// PlacementConfiguration interface, the return values of the GetDeploymentNamespace(), GetDeploymentName()
// methods (if non-empty). The parameters argument will be assigned the return value
// of the component's GetSpec() method.
type Generator interface {
  Generate(ctx context.Context, namespace string, name string, parameters types.Unstructurable) ([]client.Object, error)
}
```

In addition to `namespace`, `name`, `parameters`, generators can retrieve additional contextual information, such as a
client for the deployment target by calling `utils.ClientFromContext()`, and related functions.

Component controllers can of course implement their own generator. In many cases (for example if there exists a
Helm chart or kustomization for the component), one of the [generators bundled with this repository](../../generators) can be used.

Generators may optionally implement

```go
package types

// SchemeBuilder interface.
type SchemeBuilder interface {
  AddToScheme(scheme *runtime.Scheme) error
}

```

in order to enhance the scheme used by the dependent objects deployer.