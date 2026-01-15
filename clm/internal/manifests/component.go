/*
SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package manifests

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/sap/component-operator-runtime/clm/internal/release"
	"github.com/sap/component-operator-runtime/pkg/component"
	"github.com/sap/component-operator-runtime/pkg/types"
)

type Component struct {
	metav1.PartialObjectMetadata
	release *release.Release
	values  map[string]any
}

var _ component.Component = &Component{}

func (c *Component) GetSpec() types.Unstructurable {
	return types.UnstructurableMap(c.values)
}

func (c *Component) GetStatus() *component.Status {
	return &component.Status{
		// TODO: populate missing fields
		// ObservedGeneration
		// AppliedGeneration
		// LastObservedAt
		// LastAppliedAt
		// ProcessingDigest
		// ProcessingSince
		Revision: c.release.Revision,
		// Conditions
		State:     c.release.State,
		Inventory: c.release.Inventory,
	}
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
		release: release,
		values:  values,
	}
}
