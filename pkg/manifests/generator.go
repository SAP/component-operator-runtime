/*
SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package manifests

import (
	"context"

	legacyerrors "github.com/pkg/errors"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/sap/component-operator-runtime/pkg/types"
)

type tranformableGenerator struct {
	generator             Generator
	parameterTransformers []ParameterTransformer
	objectTransformers    []ObjectTransformer
}

// Wrap a given Generator into a TransformableGenerator, to allow to attach further parameter or object transformers to it.
func NewGenerator(generator Generator) TransformableGenerator {
	return &tranformableGenerator{generator: generator}
}

func (g *tranformableGenerator) WithParameterTransformer(transformer ParameterTransformer) TransformableGenerator {
	g.parameterTransformers = append(g.parameterTransformers, transformer)
	return g
}

func (g *tranformableGenerator) WithObjectTransformer(transformer ObjectTransformer) TransformableGenerator {
	g.objectTransformers = append(g.objectTransformers, transformer)
	return g
}

func (g *tranformableGenerator) Generate(ctx context.Context, namespace string, name string, parameters types.Unstructurable) ([]client.Object, error) {
	for i, transformer := range g.parameterTransformers {
		_parameters, err := transformer.TransformParameters(namespace, name, parameters)
		if err != nil {
			return nil, legacyerrors.Wrapf(err, "error calling parameter transformer (%d)", i)
		}
		parameters = _parameters
	}
	objects, err := g.generator.Generate(ctx, namespace, name, parameters)
	if err != nil {
		return nil, err
	}
	for i, transformer := range g.objectTransformers {
		_objects, err := transformer.TransformObjects(namespace, name, objects)
		if err != nil {
			return nil, legacyerrors.Wrapf(err, "error calling object transformer (%d)", i)
		}
		objects = _objects
	}
	return objects, nil
}
