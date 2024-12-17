/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package types

// Unstructurable represents objects which can be converted into a string-keyed map.
// All Kubernetes API types, as well as all JSON objects could be modelled as Unstructurable objects.
type Unstructurable interface {
	ToUnstructured() map[string]any
}

// UnstructurableMap is a string-keyed map, implementing the Unstructurable interface in the natural way.
type UnstructurableMap map[string]any

var _ Unstructurable = UnstructurableMap(nil)

// ToUnstructured() just returns the map itself.
func (m UnstructurableMap) ToUnstructured() map[string]any {
	return m
}
