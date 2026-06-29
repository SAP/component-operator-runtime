/*
SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type BarSpec struct {
	Value string `json:"value,omitempty"`
}

type BarStatus struct {
	ObservedGeneration int64              `json:"observedGeneration,omitempty"`
	Conditions         []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

type Bar struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec BarSpec `json:"spec,omitempty"`
	// +kubebuilder:default={"observedGeneration":-1}
	Status BarStatus `json:"status,omitempty"`
}

func (f *Bar) SetObservedGeneration(generation int64) {
	f.Status.ObservedGeneration = generation
}

func (f *Bar) SetCondition(condition metav1.Condition) {
	for i := range f.Status.Conditions {
		if f.Status.Conditions[i].Type == condition.Type {
			f.Status.Conditions[i] = condition
			return
		}
	}
	f.Status.Conditions = append(f.Status.Conditions, condition)
}

const (
	BarFinalizer = "bar.testing.cs.sap.com/finalizer"
)

// +kubebuilder:object:root=true

type BarList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Bar `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Bar{}, &BarList{})
}
