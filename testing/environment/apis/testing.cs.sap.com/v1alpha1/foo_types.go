/*
SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type FooSpec struct {
	Value string `json:"value,omitempty"`
}

type FooStatus struct {
	ObservedGeneration int64              `json:"observedGeneration,omitempty"`
	Conditions         []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

type Foo struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec FooSpec `json:"spec,omitempty"`
	// +kubebuilder:default={"observedGeneration":-1}
	Status FooStatus `json:"status,omitempty"`
}

func (f *Foo) SetObservedGeneration(generation int64) {
	f.Status.ObservedGeneration = generation
}

func (f *Foo) SetCondition(condition metav1.Condition) {
	for i := range f.Status.Conditions {
		if f.Status.Conditions[i].Type == condition.Type {
			f.Status.Conditions[i] = condition
			return
		}
	}
	f.Status.Conditions = append(f.Status.Conditions, condition)
}

const (
	FooFinalizer = "foo.testing.cs.sap.com/finalizer"
)

// +kubebuilder:object:root=true

type FooList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Foo `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Foo{}, &FooList{})
}
