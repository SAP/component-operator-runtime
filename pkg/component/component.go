/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package component

import (
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/sap/component-operator-runtime/internal/walk"
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

// Check if given component or its spec implements RequeueConfiguration (and return it).
func assertRequeueConfiguration[T Component](component T) (RequeueConfiguration, bool) {
	if requeueConfiguration, ok := Component(component).(RequeueConfiguration); ok {
		return requeueConfiguration, true
	}
	if requeueConfiguration, ok := getSpec(component).(RequeueConfiguration); ok {
		return requeueConfiguration, true
	}
	return nil, false
}

// Check if given component or its spec implements RetryConfiguration (and return it).
func assertRetryConfiguration[T Component](component T) (RetryConfiguration, bool) {
	if retryConfiguration, ok := Component(component).(RetryConfiguration); ok {
		return retryConfiguration, true
	}
	if retryConfiguration, ok := getSpec(component).(RetryConfiguration); ok {
		return retryConfiguration, true
	}
	return nil, false
}

// Check if given component or its spec implements TimeoutConfiguration (and return it).
func assertTimeoutConfiguration[T Component](component T) (TimeoutConfiguration, bool) {
	if timeoutConfiguration, ok := Component(component).(TimeoutConfiguration); ok {
		return timeoutConfiguration, true
	}
	if timeoutConfiguration, ok := getSpec(component).(TimeoutConfiguration); ok {
		return timeoutConfiguration, true
	}
	return nil, false
}

// Calculate digest of given component, honoring annotations, spec, and references.
func calculateComponentDigest[T Component](component T) string {
	digestData := make(map[string]any)
	spec := getSpec(component)
	digestData["annotations"] = component.GetAnnotations()
	digestData["spec"] = spec
	if err := walk.Walk(getSpec(component), func(x any, path []string, _ reflect.StructTag) error {
		// note: this must() is ok because marshalling []string should always work
		rawPath := must(json.Marshal(path))
		switch r := x.(type) {
		case *ConfigMapReference:
			if r != nil {
				digestData["refs:"+string(rawPath)] = r.digest()
			}
		case *ConfigMapKeyReference:
			if r != nil {
				digestData["refs:"+string(rawPath)] = r.digest()
			}
		case *SecretReference:
			if r != nil {
				digestData["refs:"+string(rawPath)] = r.digest()
			}
		case *SecretKeyReference:
			if r != nil {
				digestData["refs:"+string(rawPath)] = r.digest()
			}
		}
		return nil
	}); err != nil {
		// note: this panic is ok because walk.Walk() only produces errors if the given walker function raises any (which ours here does not do)
		panic("this cannot happen")
	}
	return calculateDigest(digestData)
}

// Implement the PlacementConfiguration interface.
func (s *PlacementSpec) GetDeploymentNamespace() string {
	return s.Namespace
}

// Implement the PlacementConfiguration interface.
func (s *PlacementSpec) GetDeploymentName() string {
	return s.Name
}

// Implement the ClientConfiguration interface.
func (s *ClientSpec) GetKubeConfig() []byte {
	if s.KubeConfig == nil {
		return nil
	}
	return s.KubeConfig.SecretRef.value
}

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

// Implement the RequeueConfiguration interface.
func (s *RequeueSpec) GetRequeueInterval() time.Duration {
	if s.RequeueInterval != nil {
		return s.RequeueInterval.Duration
	}
	return time.Duration(0)
}

// Implement the RetryConfiguration interface.
func (s *RetrySpec) GetRetryInterval() time.Duration {
	if s.RetryInterval != nil {
		return s.RetryInterval.Duration
	}
	return time.Duration(0)
}

// Check if state is Ready.
func (s *Status) IsReady() bool {
	// caveat: this operates only on the status, so it does not check that observedGeneration == generation
	return s.State == StateReady
}

// Implement the TimeoutConfiguration interface.
func (s *TimeoutSpec) GetTimeout() time.Duration {
	if s.Timeout != nil {
		return s.Timeout.Duration
	}
	return time.Duration(0)
}

// Get condition (and return nil if not existing).
// Caveat: the returned pointer might become invalid if further appends happen to the Conditions slice in the status object.
func (s *Status) getCondition(condType ConditionType) *Condition {
	for i := 0; i < len(s.Conditions); i++ {
		if s.Conditions[i].Type == condType {
			return &s.Conditions[i]
		}
	}
	return nil
}

// Get condition (adding it with initial values if not existing).
// Caveat: the returned pointer might become invalid if further appends happen to the Conditions slice in the status object.
func (s *Status) getOrAddCondition(condType ConditionType) *Condition {
	var cond *Condition
	for i := 0; i < len(s.Conditions); i++ {
		if s.Conditions[i].Type == condType {
			cond = &s.Conditions[i]
			break
		}
	}
	if cond == nil {
		s.Conditions = append(s.Conditions, Condition{Type: condType, Status: ConditionUnknown})
		cond = &s.Conditions[len(s.Conditions)-1]
	}
	return cond
}

// Get state (and related details).
func (s *Status) GetState() (State, string, string) {
	cond := s.getCondition(ConditionTypeReady)
	if cond == nil {
		return s.State, "", ""
	}
	return s.State, cond.Reason, cond.Message
}

// Set state and ready condition in status (according to the state value provided).
// Note: this method does not touch the condition's LastTransitionTime.
func (s *Status) SetState(state State, reason string, message string) {
	cond := s.getOrAddCondition(ConditionTypeReady)
	switch state {
	case StateReady:
		cond.Status = ConditionTrue
	case StateError:
		cond.Status = ConditionFalse
	default:
		cond.Status = ConditionUnknown
	}
	cond.Reason = reason
	cond.Message = message
	s.State = state
}
