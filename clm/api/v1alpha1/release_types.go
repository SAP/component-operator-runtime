/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"

	"github.com/sap/component-operator-runtime/pkg/component"
	"github.com/sap/component-operator-runtime/pkg/types"
)

// ReleaseSpec is the spec part of Release; it is an unstructured JSON/YAML.
type ReleaseSpec struct {
	apiextensionsv1.JSON `json:"-"`
}

// ReleaseStatus is the status part of Release; it augments the default component status type.
type ReleaseStatus struct {
	component.Status      `json:",inline"`
	LastAttemptedDigest   string `json:"lastAttemptedDigest,omitempty"`
	LastAttemptedRevision string `json:"lastAttemptedRevision,omitempty"`
	LastAppliedDigest     string `json:"lastAppliedDigest,omitempty"`
	LastAppliedRevision   string `json:"lastAppliedRevision,omitempty"`
}

// Implement the component-operator-runtime Component interface.
func (s *ReleaseSpec) ToUnstructured() map[string]any {
	result, err := runtime.DefaultUnstructuredConverter.ToUnstructured(s)
	if err != nil {
		panic(err)
	}
	return result
}

// Implement the component-operator-runtime Component interface.
func (r *Release) GetSpec() types.Unstructurable {
	return &r.Spec
}

// Implement the component-operator-runtime Component interface.
func (r *Release) GetStatus() *component.Status {
	return &r.Status.Status
}

// +kubebuilder:object:root=true

// Release is the Component implementation used by clm.
type Release struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ReleaseSpec `json:"spec,omitempty"`
	// kubebuilder:default={"observedGeneration":-1}
	Status    ReleaseStatus     `json:"status,omitempty"`
	configMap *corev1.ConfigMap `json:"-"`
}

var _ component.Component = &Release{}

// +kubebuilder:object:root=true

// ReleaseList contains a list of Release.
type ReleaseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Release `json:"items"`
}
