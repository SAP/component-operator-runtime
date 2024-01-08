/*
SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package component

import (
	"fmt"
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/sap/component-operator-runtime/pkg/types"
)

// Instantiate given Component type T; panics unless T is a pointer type.
func newComponent[T Component]() T {
	var component T
	v := reflect.ValueOf(&component).Elem()
	v.Set(reflect.New(v.Type().Elem()))
	return component
}

// Get a pointer to the Spec field of a component; panics unless T is a pointer type.
func getSpec[T Component](component T) any {
	spec := reflect.ValueOf(component).Elem().FieldByName("Spec")
	if spec.Kind() != reflect.Pointer {
		spec = spec.Addr()
	}
	return spec.Interface()
}

// Check if given component or its spec implements PlacementConfiguration (and return it).
func assertPlacementConfiguration[T Component](component T) (PlacementConfiguration, bool) {
	if placementConfiguration, ok := Component(component).(PlacementConfiguration); ok {
		return placementConfiguration, true
	}
	if placementConfiguration, ok := getSpec(component).(PlacementConfiguration); ok {
		return placementConfiguration, true
	}
	return nil, false
}

// Check if given component or its spec implements ClientConfiguration (and return it).
func assertClientConfiguration[T Component](component T) (ClientConfiguration, bool) {
	if clientConfiguration, ok := Component(component).(ClientConfiguration); ok {
		return clientConfiguration, true
	}
	if clientConfiguration, ok := getSpec(component).(ClientConfiguration); ok {
		return clientConfiguration, true
	}
	return nil, false
}

// Check if given component or its spec implements ImpersonationConfiguration (and return it).
func assertImpersonationConfiguration[T Component](component T) (ImpersonationConfiguration, bool) {
	if impersonationConfiguration, ok := Component(component).(ImpersonationConfiguration); ok {
		return impersonationConfiguration, true
	}
	if impersonationConfiguration, ok := getSpec(component).(ImpersonationConfiguration); ok {
		return impersonationConfiguration, true
	}
	return nil, false
}

// Implement the PlacementConfiguration interface.
func (s *PlacementSpec) GetDeploymentNamespace() string {
	return s.Namespace
}

// Implement the PlacementConfiguration interface.
func (s *PlacementSpec) GetDeploymentName() string {
	return s.Name
}

var _ PlacementConfiguration = &PlacementSpec{}

// Implement the ClientConfiguration interface.
func (s *ClientSpec) GetKubeConfig() []byte {
	if s.KubeConfig == nil {
		return nil
	}
	return s.KubeConfig.SecretRef.value
}

var _ ClientConfiguration = &ClientSpec{}

// Implement the ImpersonationConfiguration interface.
func (s *ImpersonationSpec) GetImpersonationUser() string {
	if s.ServiceAccountName == "" {
		return ""
	}
	// note: the service account's namespace is set empty here, and will be populated with the
	// actual target namespace by the framework when calling this method.
	return fmt.Sprintf("system:serviceaccount:%s:%s", "", s.ServiceAccountName)
}

// Implement the ImpersonationConfiguration interface.
func (s *ImpersonationSpec) GetImpersonationGroups() []string {
	return nil
}

var _ ImpersonationConfiguration = &ImpersonationSpec{}

// Get state (and related details).
func (s *Status) GetState() (State, string, string) {
	var cond *Condition
	for i := 0; i < len(s.Conditions); i++ {
		if s.Conditions[i].Type == ConditionTypeReady {
			cond = &s.Conditions[i]
			break
		}
	}
	if cond == nil {
		return s.State, "", ""
	}
	return s.State, cond.Reason, cond.Message
}

// Set state and ready condition in status (according to the state value provided),
func (s *Status) SetState(state State, reason string, message string) {
	var cond *Condition
	for i := 0; i < len(s.Conditions); i++ {
		if s.Conditions[i].Type == ConditionTypeReady {
			cond = &s.Conditions[i]
			break
		}
	}
	if cond == nil {
		s.Conditions = append(s.Conditions, Condition{Type: ConditionTypeReady})
		cond = &s.Conditions[len(s.Conditions)-1]
	}
	var status ConditionStatus
	switch state {
	case StateReady:
		status = ConditionTrue
	case StateError:
		status = ConditionFalse
	default:
		status = ConditionUnknown
	}
	if status != cond.Status {
		cond.Status = status
		cond.LastTransitionTime = &[]metav1.Time{metav1.Now()}[0]
	}
	cond.Reason = reason
	cond.Message = message
	s.State = state
}

// Get inventory item's ObjectKind accessor.
func (i *InventoryItem) GetObjectKind() schema.ObjectKind {
	return i
}

// Get inventory item's GroupVersionKind.
func (i InventoryItem) GroupVersionKind() schema.GroupVersionKind {
	return schema.GroupVersionKind(i.TypeInfo)
}

// Set inventory item's GroupVersionKind.
func (i *InventoryItem) SetGroupVersionKind(gvk schema.GroupVersionKind) {
	i.TypeInfo = TypeInfo(gvk)
}

// Get inventory item's  namespace.
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
