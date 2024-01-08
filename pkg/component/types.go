/*
SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package component

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/sap/component-operator-runtime/pkg/types"
)

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

// The ClientConfiguration interface is meant to be implemented by components (or their spec) which offer
// remote deployments.
type ClientConfiguration interface {
	// Get kubeconfig content. Should return nil if default local client shall be used.
	GetKubeConfig() []byte
}

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

// +kubebuilder:object:generate=true

// Legacy placement spec. Components may include this into their spec.
// Deprecation warning: use PlacementSpec instead.
type Spec struct {
	Namespace string `json:"namespace,omitempty"`
	Name      string `json:"name,omitempty"`
}

// +kubebuilder:object:generate=true

// PlacementSpec defines a namespace and name. Components providing PlacementConfiguration may include this into their spec.
type PlacementSpec struct {
	Namespace string `json:"namespace,omitempty"`
	Name      string `json:"name,omitempty"`
}

// +kubebuilder:object:generate=true

// ClientSpec defines a reference to another cluster by kubeconfig. Components providing ClientConfiguration may include this into their spec.
type ClientSpec struct {
	KubeConfig *KubeConfigSpec `json:"kubeConfig,omitempty"`
}

// +kubebuilder:object:generate=true

// KubeConfigSpec defines a reference to a kubeconfig.
type KubeConfigSpec struct {
	SecretRef SecretKeyReference `json:"secretRef" fallbackKeys:"value,value.yaml,value.yml"`
}

// +kubebuilder:object:generate=true

// ImpersonationSpec defines a service account name. Components providing ImpersonationConfiguration may include this into their spec.
type ImpersonationSpec struct {
	ServiceAccountName string `json:"serviceAccountName,omitempty"`
}

// +kubebuilder:object:generate=true

// Component Status. Components must include this into their status.
type Status struct {
	ObservedGeneration int64        `json:"observedGeneration"`
	AppliedGeneration  int64        `json:"appliedGeneration,omitempty"`
	LastObservedAt     *metav1.Time `json:"lastObservedAt,omitempty"`
	LastAppliedAt      *metav1.Time `json:"lastAppliedAt,omitempty"`
	Conditions         []Condition  `json:"conditions,omitempty"`
	// +kubebuilder:validation:Enum=Processing;Deleting;Ready;Error
	State     State            `json:"state,omitempty"`
	Inventory []*InventoryItem `json:"inventory,omitempty"`
}

// +kubebuilder:object:generate=true

// Component status Condition.
type Condition struct {
	Type ConditionType `json:"type"`
	// +kubebuilder:validation:Enum=True;False;Unknown
	Status ConditionStatus `json:"status"`
	// +optional
	LastTransitionTime *metav1.Time `json:"lastTransitionTime,omitempty"`
	// +optional
	Reason string `json:"reason,omitempty"`
	// +optional
	Message string `json:"message,omitempty"`
}

// Condition type. Currently, only the 'Ready' type is used.
type ConditionType string

const (
	// Condition type representing the 'Ready' condition.
	ConditionTypeReady ConditionType = "Ready"
)

// Condition Status. Can be one of 'True', 'False', 'Unknown'.
type ConditionStatus string

const (
	// Condition status 'True'.
	ConditionTrue ConditionStatus = "True"
	// Condition status 'False'.
	ConditionFalse ConditionStatus = "False"
	// Condition status 'Unknown'.
	ConditionUnknown ConditionStatus = "Unknown"
)

// Component state. Can be one of 'Ready', 'Processing', 'Error', 'Deleting'.
type State string

const (
	// Component state 'Ready'.
	StateReady State = "Ready"
	// Component state 'Processing'.
	StateProcessing State = "Processing"
	// Component state 'Error'.
	StateError State = "Error"
	// Component state 'Deleting'.
	StateDeleting State = "Deleting"
)

// TypeInfo represents a Kubernetes type.
type TypeInfo struct {
	// API group.
	Group string `json:"group"`
	// API group version.
	Version string `json:"version"`
	// API kind.
	Kind string `json:"kind"`
}

// NameInfo represents an object's namespace and name.
type NameInfo struct {
	// Namespace of the referenced object; empty for non-namespaced objects
	Namespace string `json:"namespace,omitempty"`
	// Name of the referenced object.
	Name string `json:"name"`
}

// +kubebuilder:object:generate=true

// InventoryItem represents a dependent object managed by this operator.
type InventoryItem struct {
	// Type of the dependent object.
	TypeInfo `json:",inline"`
	// Namespace and name of the dependent object.
	NameInfo `json:",inline"`
	// Managed types
	ManagedTypes []TypeInfo `json:"managedTypes,omitempty"`
	// Digest of the descriptor of the dependent object.
	Digest string `json:"digest"`
	// Phase of the dependent object.
	Phase string `json:"phase,omitempty"`
	// Observed status of the dependent object, as observed by kstatus.
	Status string `json:"status,omitempty"`
}

const (
	PhaseScheduledForApplication = "ScheduledForApplication"
	PhaseScheduledForDeletion    = "ScheduledForDeletion"
	PhaseScheduledForCompletion  = "ScheduledForCompletion"
	PhaseCreating                = "Creating"
	PhaseUpdating                = "Updating"
	PhaseDeleting                = "Deleting"
	PhaseCompleting              = "Completing"
	PhaseReady                   = "Ready"
	PhaseCompleted               = "Completed"
)
