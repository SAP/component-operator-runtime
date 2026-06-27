/*
SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type BazSpec struct {
	Value string `json:"value,omitempty"`
}

type BazStatus struct {
	ObservedGeneration int64              `json:"observedGeneration,omitempty"`
	Conditions         []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

type Baz struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec BazSpec `json:"spec,omitempty"`
	// +kubebuilder:default={"observedGeneration":-1}
	Status BazStatus `json:"status,omitempty"`
}

func (f *Baz) SetObservedGeneration(generation int64) {
	f.Status.ObservedGeneration = generation
}

func (f *Baz) SetCondition(condition metav1.Condition) {
	for i := range f.Status.Conditions {
		if f.Status.Conditions[i].Type == condition.Type {
			f.Status.Conditions[i] = condition
			return
		}
	}
	f.Status.Conditions = append(f.Status.Conditions, condition)
}

const (
	BazFinalizer = "baz.testing.cs.sap.com/finalizer"
)

// +kubebuilder:object:root=true

type BazList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Baz `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Baz{}, &BazList{})
}
