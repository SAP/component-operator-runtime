/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package types

import (
	"fmt"

	"k8s.io/apimachinery/pkg/runtime/schema"
	apitypes "k8s.io/apimachinery/pkg/types"
)

// Represents types which have TypeMeta/GroupVersionKind.
// All types implementing runtime.Object or even controller-runtime's client.Object obviously implement TypeKey as well.
type TypeKey interface {
	GetObjectKind() schema.ObjectKind
}

// Represents types which a namespace and a name.
// All types implementing metav1.Object or even controller-runtime's client.Object obviously implement NameKey as well.
type NameKey interface {
	GetNamespace() string
	GetName() string
}

// Represents types which have TypeMeta, and a namespace and a name.
// All types implementing controller-runtime's client.Object obviously implement ObjectKey as well.
type ObjectKey interface {
	TypeKey
	NameKey
}

// Wrap a GroupVersionKind as TypeKey.
func TypeKeyFromGroupVersionKind(gvk schema.GroupVersionKind) TypeKey {
	return &typeKey{gvk: gvk}
}

// Wrap a group, version and kind as TypeKey.
func TypeKeyFromGroupAndVersionAndKind(group string, version string, kind string) TypeKey {
	return &typeKey{gvk: schema.GroupVersionKind{Group: group, Version: version, Kind: kind}}
}

// Return a string representation of a TypeKey.
func TypeKeyToString(key TypeKey) string {
	return key.GetObjectKind().GroupVersionKind().String()
}

// Wrap a NamespacedName as NameKey.
func NameKeyFromNamespacedName(namespaceName apitypes.NamespacedName) NameKey {
	return &nameKey{namespace: namespaceName.Namespace, name: namespaceName.Name}
}

// Wrap a namespace and name as NameKey.
func NameKeyFromNamespaceAndName(namespace string, name string) NameKey {
	return &nameKey{namespace: namespace, name: name}
}

// Return a string representation of a NameKey.
func NameKeyToString(key NameKey) string {
	namespace := key.GetNamespace()
	name := key.GetName()
	if namespace == "" {
		return name
	} else {
		return fmt.Sprintf("%s/%s", namespace, name)
	}
}

// Return a string representation of an ObjectKey.
func ObjectKeyToString(key ObjectKey) string {
	return fmt.Sprintf("%s %s", TypeKeyToString(key), NameKeyToString(key))
}

type typeKey struct {
	gvk schema.GroupVersionKind
}

var _ TypeKey = &typeKey{}

func (k *typeKey) GetObjectKind() schema.ObjectKind {
	return k
}

func (k *typeKey) GroupVersionKind() schema.GroupVersionKind {
	return k.gvk
}

func (k *typeKey) SetGroupVersionKind(gvk schema.GroupVersionKind) {
	k.gvk = gvk
}

type nameKey struct {
	namespace string
	name      string
}

var _ NameKey = &nameKey{}

func (k *nameKey) GetNamespace() string {
	return k.namespace
}

func (k *nameKey) GetName() string {
	return k.name
}
