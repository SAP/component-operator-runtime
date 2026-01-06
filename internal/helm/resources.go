/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package helm

import (
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	annotationKeyResourcePolicy = "helm.sh/resource-policy"
)

// parse helm resource properties from object, return nil if none is set
func ParseResourceMetadata(object client.Object) (*ResourceMetadata, error) {
	metadata := &ResourceMetadata{}
	annotations := object.GetAnnotations()

	if value, ok := annotations[annotationKeyResourcePolicy]; ok {
		metadata.Policy = value
		switch metadata.Policy {
		case ResourcePolicyKeep:
		default:
			return nil, fmt.Errorf("invalid resource policy: %s", metadata.Policy)
		}
	}

	return metadata, nil
}
