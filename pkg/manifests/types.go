/*
SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package manifests

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/sap/component-operator-runtime/pkg/types"
)

// Resource generator interface.
// When called from the reconciler, the arguments namespace, name and parameters will match the return values
// of the component's GetDeploymentNamespace(), GetDeploymentName() and GetSpec() methods, respectively.
type Generator interface {
	Generate(ctx context.Context, namespace string, name string, parameters types.Unstructurable) ([]client.Object, error)
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

// SchemeBuilder interface.
type SchemeBuilder interface {
	AddToScheme(scheme *runtime.Scheme) error
}
