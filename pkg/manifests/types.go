/*
SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package manifests

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/sap/component-operator-runtime/pkg/types"
)

// Resource generator interface.
// When called from the reconciler, the arguments namespace and name will match the
// component's namespace and name or, if the component or its spec implement the
// PlacementConfiguration interface, the return values of the GetDeploymentNamespace(), GetDeploymentName()
// methods (if non-empty). The parameters argument will be assigned the return value
// of the component's GetSpec() method.
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

// Decryptor interface.
// Allows to decrypt content of referenced manifest sources.
type Decryptor interface {
	Decrypt(input []byte, path string) ([]byte, error)
}
