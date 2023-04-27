/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
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
