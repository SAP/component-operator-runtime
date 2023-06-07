/*
SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package manifests

import (
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/sap/component-operator-runtime/pkg/types"
)

// DummyGenerator is a generator that does nothing.
type DummyGenerator struct{}

var _ Generator = &DummyGenerator{}

// Create a new DummyGenerator.
func NewDummyGenerator() (*DummyGenerator, error) {
	return &DummyGenerator{}, nil
}

// Generate resource descriptors.
func (g *DummyGenerator) Generate(namespace string, name string, parameters types.Unstructurable) ([]client.Object, error) {
	return nil, nil
}
