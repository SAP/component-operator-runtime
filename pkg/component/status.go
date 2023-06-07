/*
SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and redis-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

package component

import (
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	kstatus "sigs.k8s.io/cli-utils/pkg/kstatus/status"
)

// wrapper around kstatus.Compute, allowing us to modify kstatus's view for certain objects
func computeStatus(obj *unstructured.Unstructured) (*kstatus.Result, error) {
	res, err := kstatus.Compute(obj)
	if err != nil {
		return nil, err
	}
	switch obj.GroupVersionKind() {
	case schema.GroupVersionKind{Group: "batch", Version: "v1", Kind: "Job"}:
		// other than kstatus we want to consider jobs as InProgress if its pods are still running, resp. did not (yet) finish successfully
		if res.Status == kstatus.CurrentStatus {
			done := false
			objc, err := kstatus.GetObjectWithConditions(obj.UnstructuredContent())
			if err != nil {
				return nil, err
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
	return res, nil
}
