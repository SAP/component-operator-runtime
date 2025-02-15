/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package reconciler

import (
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/sap/component-operator-runtime/pkg/types"
)

// Get inventory item's ObjectKind accessor.
func (i *InventoryItem) GetObjectKind() schema.ObjectKind {
	return i
}

// Get inventory item's GroupVersionKind.
func (i InventoryItem) GroupVersionKind() schema.GroupVersionKind {
	return schema.GroupVersionKind(i.TypeVersionInfo)
}

// Set inventory item's GroupVersionKind.
func (i *InventoryItem) SetGroupVersionKind(gvk schema.GroupVersionKind) {
	i.TypeVersionInfo = TypeVersionInfo(gvk)
}

// Get inventory item's namespace.
func (i InventoryItem) GetNamespace() string {
	return i.Namespace
}

// Get inventory item's name.
func (i InventoryItem) GetName() string {
	return i.Name
}

// Check whether inventory item matches given ObjectKey, in the sense that Group, Kind, Namespace and Name are the same.
// Note that this does not compare the group's Version.
func (i InventoryItem) Matches(key types.ObjectKey) bool {
	return i.GroupVersionKind().GroupKind() == key.GetObjectKind().GroupVersionKind().GroupKind() && i.Namespace == key.GetNamespace() && i.Name == key.GetName()
}

// Return a string representation of the inventory item; makes InventoryItem implement the Stringer interface.
func (i InventoryItem) String() string {
	return types.ObjectKeyToString(&i)
}
