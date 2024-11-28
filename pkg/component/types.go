/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package component

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/sap/component-operator-runtime/pkg/reconciler"
	"github.com/sap/component-operator-runtime/pkg/types"
)

// TODO: rename GetSpec() to GetParameters()

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

// The RequeueConfiguration interface is meant to be implemented by components (or their spec) which offer
// tweaking the requeue interval (by default, it would be 10 minutes).
type RequeueConfiguration interface {
	// Get requeue interval. Should be greater than 1 minute.
	GetRequeueInterval() time.Duration
}

// The RetryConfiguration interface is meant to be implemented by components (or their spec) which offer
// tweaking the retry interval (by default, it would be the value of the requeue interval).
type RetryConfiguration interface {
	// Get retry interval. Should be greater than 1 minute.
	GetRetryInterval() time.Duration
}

// The TimeoutConfiguration interface is meant to be implemented by components (or their spec) which offer
// tweaking the processing timeout (by default, it would be the value of the requeue interval).
type TimeoutConfiguration interface {
	// Get timeout. Should be greater than 1 minute.
	GetTimeout() time.Duration
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

var _ PlacementConfiguration = &PlacementSpec{}

// +kubebuilder:object:generate=true

// ClientSpec defines a reference to another cluster by kubeconfig. Components providing ClientConfiguration may include this into their spec.
type ClientSpec struct {
	KubeConfig *KubeConfigSpec `json:"kubeConfig,omitempty"`
}

var _ ClientConfiguration = &ClientSpec{}

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

var _ ImpersonationConfiguration = &ImpersonationSpec{}

// +kubebuilder:object:generate=true

// RequeueSpec defines the requeue interval, that is, the interval after which components will be re-reconciled after a successful reconciliation.
// Components providing RequeueConfiguration may include this into their spec.
type RequeueSpec struct {
	// +kubebuilder:validation:Type:=string
	// +kubebuilder:validation:Pattern="^([0-9]+(\\.[0-9]+)?(ns|us|µs|ms|s|m|h))+$"
	RequeueInterval *metav1.Duration `json:"requeueInterval,omitempty"`
}

var _ RequeueConfiguration = &RequeueSpec{}

// +kubebuilder:object:generate=true

// RetrySpec defines the retry interval, that is, the interval after which components will be re-reconciled after a successful reconciliation.
// Components providing RetryConfiguration may include this into their spec.
type RetrySpec struct {
	// +kubebuilder:validation:Type:=string
	// +kubebuilder:validation:Pattern="^([0-9]+(\\.[0-9]+)?(ns|us|µs|ms|s|m|h))+$"
	RetryInterval *metav1.Duration `json:"retryInterval,omitempty"`
}

var _ RetryConfiguration = &RetrySpec{}

// +kubebuilder:object:generate=true

// TimeoutSpec defines the processing timeout, that is, the duration after which all dependent objects of the component
// must have reached a ready state, or the component status will change to error.
// Components providing TimeoutConfiguration may include this into their spec.
type TimeoutSpec struct {
	// +kubebuilder:validation:Type:=string
	// +kubebuilder:validation:Pattern="^([0-9]+(\\.[0-9]+)?(ns|us|µs|ms|s|m|h))+$"
	Timeout *metav1.Duration `json:"timeout,omitempty"`
}

var _ TimeoutConfiguration = &TimeoutSpec{}

// +kubebuilder:object:generate=true

// Component Status. Components must include this into their status.
type Status struct {
	ObservedGeneration int64        `json:"observedGeneration"`
	AppliedGeneration  int64        `json:"appliedGeneration,omitempty"`
	LastObservedAt     *metav1.Time `json:"lastObservedAt,omitempty"`
	LastAppliedAt      *metav1.Time `json:"lastAppliedAt,omitempty"`
	ProcessingDigest   string       `json:"processingDigest,omitempty"`
	ProcessingSince    *metav1.Time `json:"processingSince,omitempty"`
	Conditions         []Condition  `json:"conditions,omitempty"`
	// +kubebuilder:validation:Enum=Ready;Pending;Processing;DeletionPending;Deleting;Error
	State     State                       `json:"state,omitempty"`
	Inventory []*reconciler.InventoryItem `json:"inventory,omitempty"`
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

// Component state. Can be one of 'Ready', 'Pending', 'Processing', 'DeletionPending', 'Deleting', 'Error'.
type State string

const (
	// Component state 'Ready'.
	StateReady State = "Ready"
	// Component state 'Pending'.
	StatePending State = "Pending"
	// Component state 'Processing'.
	StateProcessing State = "Processing"
	// Component state 'DeletionPending'.
	StateDeletionPending State = "DeletionPending"
	// Component state 'Deleting'.
	StateDeleting State = "Deleting"
	// Component state 'Error'.
	StateError State = "Error"
)
