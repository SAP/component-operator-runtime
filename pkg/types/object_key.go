/*
SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and redis-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

package types

import (
	"fmt"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Represents types which have TypeMeta, and a namespace and a name.
// All types implementing controller-runtime's client.Object obviously implement ObjectKey as well.
type ObjectKey interface {
	GetObjectKind() schema.ObjectKind
	GetNamespace() string
	GetName() string
}

// Return a string representation of an ObjectKey.
func ObjectKeyToString(key ObjectKey) string {
	gvk := key.GetObjectKind().GroupVersionKind()
	namespace := key.GetNamespace()
	name := key.GetName()

	if namespace == "" {
		return fmt.Sprintf("%s %s", gvk, name)
	} else {
		return fmt.Sprintf("%s %s/%s", gvk, namespace, name)
	}
}
