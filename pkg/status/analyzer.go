/*
SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package status

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/iancoleman/strcase"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	kstatus "sigs.k8s.io/cli-utils/pkg/kstatus/status"

	"github.com/sap/component-operator-runtime/pkg/types"
)

const conditionTypeReady = "Ready"

type statusAnalyzer struct {
	reconcilerName string
}

// Create a default StatusAnalyzer implementation.
// This implementation uses kstatus internally, with the following modifications:
//   - certain hints about the object's status type can be passed by setting the annotation '<reconcilerName>/status-hint' on the object, as a comma-separated list;
//     if this list contains the value 'has-observed-generation', the object's status (where appropriate) will be enhanced with an observed generation of -1 before passing it to kstatus;
//     if this list contains the value 'has-ready-condition', the object's status (where appropriate) will be enhanced by a ready condition with 'Unknown' state
//   - jobs will be treated differently as with kstatus; other than kstatus, which considers jobs as ready if they are successfully started, this implementation waits for
//     the JobComplete or JobFailed condidtion to be present on the job's status.
func NewStatusAnalyzer(reconcilerName string) StatusAnalyzer {
	return &statusAnalyzer{
		reconcilerName: reconcilerName,
	}
}

// Implement the StatusAnalyzer interface.
func (s *statusAnalyzer) ComputeStatus(object *unstructured.Unstructured) (Status, error) {
	var extraConditions []string

	if hint, ok := object.GetAnnotations()[s.reconcilerName+"/"+types.AnnotationKeySuffixStatusHint]; ok {
		object = object.DeepCopy()
		for _, hint := range strings.Split(hint, ",") {
			var key, value string
			var hasValue bool
			if match := regexp.MustCompile(`^([^=]+)=(.*)$`).FindStringSubmatch(hint); match != nil {
				key = match[1]
				value = match[2]
				hasValue = true
			} else {
				key = hint
			}
			switch strcase.ToKebab(key) {
			case types.StatusHintHasObservedGeneration:
				if hasValue {
					return UnknownStatus, fmt.Errorf("status hint %s does not take a value", types.StatusHintHasObservedGeneration)
				}
				_, found, err := unstructured.NestedInt64(object.Object, "status", "observedGeneration")
				if err != nil {
					return UnknownStatus, err
				}
				if !found {
					if err := unstructured.SetNestedField(object.Object, int64(-1), "status", "observedGeneration"); err != nil {
						return UnknownStatus, err
					}
				}
			case types.StatusHintHasReadyCondition:
				if hasValue {
					return UnknownStatus, fmt.Errorf("status hint %s does not take a value", types.StatusHintHasReadyCondition)
				}
				foundReadyCondition := false
				conditions, found, err := unstructured.NestedSlice(object.Object, "status", "conditions")
				if err != nil {
					return UnknownStatus, err
				}
				if !found {
					conditions = make([]any, 0)
				}
				for _, condition := range conditions {
					if condition, ok := condition.(map[string]any); ok {
						condType, found, err := unstructured.NestedString(condition, "type")
						if err != nil {
							return UnknownStatus, err
						}
						if found && condType == conditionTypeReady {
							foundReadyCondition = true
							break
						}
					}
				}
				if !foundReadyCondition {
					conditions = append(conditions, map[string]any{
						"type":   conditionTypeReady,
						"status": string(corev1.ConditionUnknown),
					})
					if err := unstructured.SetNestedSlice(object.Object, conditions, "status", "conditions"); err != nil {
						return UnknownStatus, err
					}
				}
			case types.StatusHintConditions:
				if !hasValue {
					return UnknownStatus, fmt.Errorf("status hint %s requires a value", types.StatusHintConditions)
				}
				extraConditions = append(extraConditions, strings.Split(value, ";")...)
			default:
				return UnknownStatus, fmt.Errorf("unknown status hint %s", key)
			}
		}
	}

	res, err := kstatus.Compute(object)
	if err != nil {
		return UnknownStatus, err
	}
	status := Status(res.Status)

	if status == CurrentStatus && len(extraConditions) > 0 {
		objc, err := kstatus.GetObjectWithConditions(object.UnstructuredContent())
		if err != nil {
			return UnknownStatus, err
		}
		for _, condition := range extraConditions {
			found := false
			for _, cond := range objc.Status.Conditions {
				if cond.Type == condition {
					found = true
					if cond.Status != corev1.ConditionTrue {
						status = InProgressStatus
					}
				}
			}
			if !found {
				status = InProgressStatus
			}
		}
	}

	switch object.GroupVersionKind() {
	case schema.GroupVersionKind{Group: "batch", Version: "v1", Kind: "Job"}:
		// other than kstatus we want to consider jobs as InProgress if its pods are still running, resp. did not (yet) finish successfully
		if status == CurrentStatus {
			done := false
			objc, err := kstatus.GetObjectWithConditions(object.UnstructuredContent())
			if err != nil {
				return UnknownStatus, err
			}
			for _, cond := range objc.Status.Conditions {
				if cond.Type == string(batchv1.JobComplete) && cond.Status == corev1.ConditionTrue {
					done = true
					break
				}
				if cond.Type == string(batchv1.JobFailed) && cond.Status == corev1.ConditionTrue {
					done = true
					break
				}
			}
			if !done {
				status = InProgressStatus
			}
		}
	}

	return status, nil
}
