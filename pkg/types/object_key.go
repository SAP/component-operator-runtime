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
