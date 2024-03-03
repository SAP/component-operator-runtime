/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"

	"github.com/sap/component-operator-runtime/pkg/component"
	componentoperatorruntimetypes "github.com/sap/component-operator-runtime/pkg/types"
)

// MyComponentSpec defines the desired state of MyComponent.
type MyComponentSpec struct {
	// Uncomment the following if you want to implement the PlacementConfiguration interface
	// (that is, want to make deployment namespace and name configurable here in the spec, independently of
	// the component's metadata.namespace and metadata.name).
	// component.PlacementSpec                  `json:",inline"`
	// Uncomment the following if you want to implement the ClientConfiguration interface
	// (that is, want to allow remote deployments via a specified kubeconfig).
	// Note, that when implementing the ClientConfiguration interface, then also the PlacementConfiguration
	// interface should be implemented.
	// component.ClientSpec        `json:",inline"`
	// Uncomment the following if you want to implement the ImpersonationConfiguratio interface
	// (that is, want to allow use a specified service account in the target namespace for the deployment).
	// component.ImpersonationSpec `json:",inline"`
	// Uncomment the following if you want to implement the RequeueConfiguration interface
	// (that is, want to allow to override the default requeue interval of 10m).
	// component.RequeueSpec `json:",inline"`
	// Uncomment the following if you want to implement the RetryConfiguration interface
	// (that is, want to allow to override the default retry interval, which equals the effective requeue interval).
	// component.RequeueSpec `json:",inline"`

	// Add your own fields here, describing the deployment of the managed component.
}

// MyComponentStatus defines the observed state of MyComponent.
type MyComponentStatus struct {
	component.Status `json:",inline"`

	// You may add your own fields here; this is rarely needed.
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="State",type=string,JSONPath=`.status.state`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +genclient

// MyComponent is the Schema for the mycomponents API.
type MyComponent struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec MyComponentSpec `json:"spec,omitempty"`
	// +kubebuilder:default={"observedGeneration":-1}
	Status MyComponentStatus `json:"status,omitempty"`
}

var _ component.Component = &MyComponent{}

// +kubebuilder:object:root=true

// MyComponentList contains a list of MyComponent.
type MyComponentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MyComponent `json:"items"`
}

func (s *MyComponentSpec) ToUnstructured() map[string]any {
	result, err := runtime.DefaultUnstructuredConverter.ToUnstructured(s)
	if err != nil {
		panic(err)
	}
	return result
}

func (c *MyComponent) GetSpec() componentoperatorruntimetypes.Unstructurable {
	return &c.Spec
}

func (c *MyComponent) GetStatus() *component.Status {
	return &c.Status.Status
}

func init() {
	SchemeBuilder.Register(&MyComponent{}, &MyComponentList{})
}
