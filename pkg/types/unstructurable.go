/*
SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and redis-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

package types

// Unstructurable represents objects which can be converted into a string-keyed map.
// All Kubernetes API types, as well as all JSON objects could be modelled as Unstructurable objects.
type Unstructurable interface {
	ToUnstructured() map[string]any
}

type UnstructurableMap map[string]any

func (m UnstructurableMap) ToUnstructured() map[string]any {
	return m
}
