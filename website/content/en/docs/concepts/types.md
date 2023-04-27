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
implemented by component-operator-runtime. To achieve this, basically two interfaces have to be implemented by such an operator ...

# The Component Interface

In the programming model proposed by component-operator-runtime, the declared and observed state of the component is to be represented by a custom resource type. The corresponding runtime type has to fulfill the folloing interface:

```go
package component

// Component is the central interface that component operators have to implement.
// Besides being a conroller-runtime client.Object, the implmenting type has to include
// the Spec and Status structs defined in this package, and has to define according accessor methods,
// called GetComponentSpec() and GetComponentStatus(). In addition it has to expose its whole spec and status
// as Unstructurable objects, via methods GetSpec() and GetStatus().
type Component interface {
	client.Object
	// Return target namespace for the component deployment.
	// This is the value that will be passed to Generator.Generate() as namespace.
	// In addition, rendered namespaced resources without namespace will be placed in this namespace.
	GetDeploymentNamespace() string
	// Return target name for the component deployment.
	// This is the value that will be passed to Generator.Generator() as name.
	GetDeploymentName() string
	// Return a pointer accessor to the component's spec.
	// Which, as a consequence, obviously has to implement the types.Unstructurable interface.
	GetSpec() types.Unstructurable
	// Return a pointer accessor to the component's status,
	// resp. to the corresponding sub-struct if the status extends component.Status.
	GetStatus() *Status
}
```

The custom resource is supposed to be namespaced, and it might be desired to deploy the component in a namespace different from the namespace where the component resource object is residing. This explains the presence of the `GetDeploymentNamespace()` and `GetDeploymentName()` methods.
Furthermore, two accessor methods are required to be implemented. First, `GetSpec()` gives access to the spec of the custom resource type,
which can be quite arbitrary.
The only requirement on the spec type is to implement the

```go
package types

// Unstructurable represents objects which can be converted into a string-keyed map.
// All Kubernetes API types, as well as all JSON objects could be modelled as Unstructurable objects.
type Unstructurable interface {
	ToUnstructured() map[string]any
}
```

interface. Finally, `GetStatus()` allows the framework to access (a part of) the custom resource type's status, having the following type:

```go
package component

// Component Status. Types implementing the Component interface must include this into their status.
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

# The Generator interface

While the `Component` (respectively the related custom resource type) models the desired and actual state of
the managed component, the `Generator` interface provides a recipe to render the Kubernetes manifests of the
dependent objects, according to the provided spec of the component:

```go
package manifests

// Resource generator interface.
// When called from the reconciler, namespace and name will match the respective values in the
// reconciled Component's spec, and parameters will be a pointer to the whole Component spec.
// Therefore, implementations which are directly called from the reconciler,
// can safely cast parameters back to their concrete spec struct.
type Generator interface {
	Generate(namespace string, name string, parameters types.Unstructurable) ([]client.Object, error)
}
```

When called by the framework, the arguments passed to `Generate()` are the return values of the
`GetDeploymentNamespace()`, `GetDeploymentName()` and `GetSpec()` methods of the component.

Component controllers can of course implement their own generator. In some cases (for example if there exists a 
Helm chart for the component), one of the [generators bundled with this repository](../../generators) can be used. 