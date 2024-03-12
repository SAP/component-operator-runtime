/*
SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package kstatus

import (
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

func NewStatusAnalyzer(reconcilerName string) StatusAnalyzer {
	return &statusAnalyzer{
		reconcilerName: reconcilerName,
	}
}

func (s *statusAnalyzer) ComputeStatus(object *unstructured.Unstructured) (Status, error) {
	if hint, ok := object.GetAnnotations()[s.reconcilerName+"/"+types.AnnotationKeySuffixStatusHint]; ok {
		object = object.DeepCopy()

		for _, hint := range strings.Split(hint, ",") {
			switch strcase.ToKebab(hint) {
			case types.StatusHintHasObservedGeneration:
				_, found, err := unstructured.NestedInt64(object.Object, "status", "observedGeneration")
				if err != nil {
					return UnknownStatus, err
				}
				if !found {
					if err := unstructured.SetNestedField(object.Object, -1, "status", "observedGeneration"); err != nil {
						return UnknownStatus, err
					}
				}
			case types.StatusHintHasReadyCondition:
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
			}
		}
	}

	res, err := kstatus.Compute(object)
	if err != nil {
		return UnknownStatus, err
	}

	switch object.GroupVersionKind() {
	case schema.GroupVersionKind{Group: "batch", Version: "v1", Kind: "Job"}:
		// other than kstatus we want to consider jobs as InProgress if its pods are still running, resp. did not (yet) finish successfully
		if res.Status == kstatus.CurrentStatus {
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
				res.Status = kstatus.InProgressStatus
			}
		}
	}

	return Status(res.Status), nil
}
