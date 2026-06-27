/*
SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package manifests

import (
	"encoding/json"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/sap/component-operator-runtime/clm/internal/release"
	"github.com/sap/component-operator-runtime/internal/util"
	"github.com/sap/component-operator-runtime/pkg/component"
	"github.com/sap/component-operator-runtime/pkg/types"
)

type Component struct {
	metav1.PartialObjectMetadata
	Spec   ComponentSpec   `json:"spec"`
	Status ComponentStatus `json:"status"`
	values map[string]any
}

type ComponentSpec struct {
	Values *apiextensionsv1.JSON `json:"values,omitempty"`
}

type ComponentStatus struct {
	component.Status      `json:",inline"`
	LastAttemptedDigest   string `json:"lastAttemptedDigest,omitempty"`
	LastAttemptedRevision string `json:"lastAttemptedRevision,omitempty"`
}

var _ component.Component = &Component{}

func (c *Component) GetSpec() types.Unstructurable {
	return types.UnstructurableMap(c.values)
}

func (c *Component) GetStatus() *component.Status {
	return &c.Status.Status
}

func componentFromRelease(release *release.Release, values map[string]any) *Component {
	return &Component{
		PartialObjectMetadata: metav1.PartialObjectMetadata{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "clm.cs.sap.com/v1alpha1",
				Kind:       "Component",
			},
			ObjectMeta: metav1.ObjectMeta{
				// TODO: add more metadata (maybe from the configmap backing the release)
				Namespace: release.GetNamespace(),
				Name:      release.GetName(),
			},
		},
		Spec: ComponentSpec{
			Values: &apiextensionsv1.JSON{
				Raw: util.Must(json.Marshal(values)),
			},
		},
		Status: ComponentStatus{
			Status: component.Status{
				// TODO: populate missing fields
				// ObservedGeneration
				// AppliedGeneration
				// LastObservedAt
				// LastAppliedAt
				ProcessingDigest: release.GetDigest(),
				// ProcessingSince
				// LastProcessingDigest
				Revision: release.Revision,
				// Conditions
				State:     release.State,
				Inventory: release.Inventory,
			},
			LastAttemptedDigest:   release.GetDigest(),
			LastAttemptedRevision: release.GetDigest(),
		},
		values: values,
	}
}
