/*
SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and redis-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

package manifests

import (
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/sap/component-operator-runtime/pkg/types"
)

// Resource generator interface.
// When called from the reconciler, namespace and name will match the respective values in the
// reconciled Component's spec, and parameters will be a pointer to the whole Component spec.
// Therefore, implementations which are directly called from the reconciler,
// can safely cast parameters back to their concrete spec struct.
type Generator interface {
	Generate(namespace string, name string, parameters types.Unstructurable) ([]client.Object, error)
}

// Interface for generators that can be enhanced with parameter/object transformers.
type TransformableGenerator interface {
	Generator
	WithParameterTransformer(transformer ParameterTransformer) TransformableGenerator
	WithObjectTransformer(transformer ObjectTransformer) TransformableGenerator
}

// Parameter transformer interface.
// Allows to manipulate the parameters passed to an existing generator.
type ParameterTransformer interface {
	TransformParameters(namespace string, name string, parameters types.Unstructurable) (types.Unstructurable, error)
}

// Object transformer interface.
// Allows to manipulate the parameters returned by an existing generator.
type ObjectTransformer interface {
	TransformObjects(namespace string, name string, objects []client.Object) ([]client.Object, error)
}
