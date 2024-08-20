/*
SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package status_test

import (
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	kstatus "sigs.k8s.io/cli-utils/pkg/kstatus/status"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sap/component-operator-runtime/pkg/status"
)

var _ = Describe("testing: analyzer.go", func() {
	var analyzer status.StatusAnalyzer

	BeforeEach(func() {
		analyzer = status.NewStatusAnalyzer("test")
	})

	DescribeTable("testing: ComputeStatus()",
		func(generation int, observedGeneration int, conditions map[kstatus.ConditionType]corev1.ConditionStatus, hintObservedGeneration bool, hintReadyCondition bool, hintConditions []string, expectedStatus status.Status) {
			obj := Object{
				ObjectMeta: metav1.ObjectMeta{
					Generation: int64(generation),
				},
				Status: ObjectStatus{
					ObservedGeneration: int64(observedGeneration),
				},
			}
			for name, status := range conditions {
				obj.Status.Conditions = append(obj.Status.Conditions, kstatus.Condition{
					Type:   name,
					Status: status,
				})
			}
			var hints []string
			if hintObservedGeneration {
				hints = append(hints, "has-observed-generation")
			}
			if hintReadyCondition {
				hints = append(hints, "has-ready-condition")
			}
			if len(hintConditions) > 0 {
				hints = append(hints, "conditions="+strings.Join(hintConditions, ";"))
			}
			if len(hints) > 0 {
				obj.Annotations = map[string]string{
					"test/status-hint": strings.Join(hints, ","),
				}
			}
			unstructuredContent, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&obj)
			Expect(err).NotTo(HaveOccurred())
			unstructuredObj := &unstructured.Unstructured{Object: unstructuredContent}

			computedStatus, err := analyzer.ComputeStatus(unstructuredObj)
			Expect(err).NotTo(HaveOccurred())

			Expect(computedStatus).To(Equal(expectedStatus))
		},

		Entry(nil, 3, 0, nil, false, false, nil, status.CurrentStatus),
		Entry(nil, 3, 1, nil, false, false, nil, status.InProgressStatus),
		Entry(nil, 3, 3, nil, false, false, nil, status.CurrentStatus),
		Entry(nil, 3, 0, nil, true, false, nil, status.InProgressStatus),
		Entry(nil, 3, 1, nil, true, false, nil, status.InProgressStatus),
		Entry(nil, 3, 3, nil, true, false, nil, status.CurrentStatus),

		Entry(nil, 3, 0, map[kstatus.ConditionType]corev1.ConditionStatus{"Ready": corev1.ConditionUnknown}, false, false, nil, status.InProgressStatus),
		Entry(nil, 3, 1, map[kstatus.ConditionType]corev1.ConditionStatus{"Ready": corev1.ConditionUnknown}, false, false, nil, status.InProgressStatus),
		Entry(nil, 3, 3, map[kstatus.ConditionType]corev1.ConditionStatus{"Ready": corev1.ConditionUnknown}, false, false, nil, status.InProgressStatus),
		Entry(nil, 3, 0, map[kstatus.ConditionType]corev1.ConditionStatus{"Ready": corev1.ConditionUnknown}, true, false, nil, status.InProgressStatus),
		Entry(nil, 3, 1, map[kstatus.ConditionType]corev1.ConditionStatus{"Ready": corev1.ConditionUnknown}, true, false, nil, status.InProgressStatus),
		Entry(nil, 3, 3, map[kstatus.ConditionType]corev1.ConditionStatus{"Ready": corev1.ConditionUnknown}, true, false, nil, status.InProgressStatus),

		Entry(nil, 3, 0, map[kstatus.ConditionType]corev1.ConditionStatus{"Ready": corev1.ConditionFalse}, false, false, nil, status.InProgressStatus),
		Entry(nil, 3, 1, map[kstatus.ConditionType]corev1.ConditionStatus{"Ready": corev1.ConditionFalse}, false, false, nil, status.InProgressStatus),
		Entry(nil, 3, 3, map[kstatus.ConditionType]corev1.ConditionStatus{"Ready": corev1.ConditionFalse}, false, false, nil, status.InProgressStatus),
		Entry(nil, 3, 0, map[kstatus.ConditionType]corev1.ConditionStatus{"Ready": corev1.ConditionFalse}, true, false, nil, status.InProgressStatus),
		Entry(nil, 3, 1, map[kstatus.ConditionType]corev1.ConditionStatus{"Ready": corev1.ConditionFalse}, true, false, nil, status.InProgressStatus),
		Entry(nil, 3, 3, map[kstatus.ConditionType]corev1.ConditionStatus{"Ready": corev1.ConditionFalse}, true, false, nil, status.InProgressStatus),

		Entry(nil, 3, 0, map[kstatus.ConditionType]corev1.ConditionStatus{"Ready": corev1.ConditionTrue}, false, false, nil, status.CurrentStatus),
		Entry(nil, 3, 1, map[kstatus.ConditionType]corev1.ConditionStatus{"Ready": corev1.ConditionTrue}, false, false, nil, status.InProgressStatus),
		Entry(nil, 3, 3, map[kstatus.ConditionType]corev1.ConditionStatus{"Ready": corev1.ConditionTrue}, false, false, nil, status.CurrentStatus),
		Entry(nil, 3, 0, map[kstatus.ConditionType]corev1.ConditionStatus{"Ready": corev1.ConditionTrue}, true, false, nil, status.InProgressStatus),
		Entry(nil, 3, 1, map[kstatus.ConditionType]corev1.ConditionStatus{"Ready": corev1.ConditionTrue}, true, false, nil, status.InProgressStatus),
		Entry(nil, 3, 3, map[kstatus.ConditionType]corev1.ConditionStatus{"Ready": corev1.ConditionTrue}, true, false, nil, status.CurrentStatus),

		Entry(nil, 3, 0, nil, false, true, nil, status.InProgressStatus),
		Entry(nil, 3, 1, nil, false, true, nil, status.InProgressStatus),
		Entry(nil, 3, 3, nil, false, true, nil, status.InProgressStatus),
		Entry(nil, 3, 0, nil, true, true, nil, status.InProgressStatus),
		Entry(nil, 3, 1, nil, true, true, nil, status.InProgressStatus),
		Entry(nil, 3, 3, nil, true, true, nil, status.InProgressStatus),

		Entry(nil, 3, 0, nil, false, false, []string{"Test"}, status.InProgressStatus),
		Entry(nil, 3, 1, nil, false, false, []string{"Test"}, status.InProgressStatus),
		Entry(nil, 3, 3, nil, false, false, []string{"Test"}, status.InProgressStatus),
		Entry(nil, 3, 0, nil, true, false, []string{"Test"}, status.InProgressStatus),
		Entry(nil, 3, 1, nil, true, false, []string{"Test"}, status.InProgressStatus),
		Entry(nil, 3, 3, nil, true, false, []string{"Test"}, status.InProgressStatus),

		Entry(nil, 3, 0, map[kstatus.ConditionType]corev1.ConditionStatus{"Test": corev1.ConditionFalse}, false, false, []string{"Test"}, status.InProgressStatus),
		Entry(nil, 3, 1, map[kstatus.ConditionType]corev1.ConditionStatus{"Test": corev1.ConditionFalse}, false, false, []string{"Test"}, status.InProgressStatus),
		Entry(nil, 3, 3, map[kstatus.ConditionType]corev1.ConditionStatus{"Test": corev1.ConditionFalse}, false, false, []string{"Test"}, status.InProgressStatus),
		Entry(nil, 3, 0, map[kstatus.ConditionType]corev1.ConditionStatus{"Test": corev1.ConditionFalse}, true, false, []string{"Test"}, status.InProgressStatus),
		Entry(nil, 3, 1, map[kstatus.ConditionType]corev1.ConditionStatus{"Test": corev1.ConditionFalse}, true, false, []string{"Test"}, status.InProgressStatus),
		Entry(nil, 3, 3, map[kstatus.ConditionType]corev1.ConditionStatus{"Test": corev1.ConditionFalse}, true, false, []string{"Test"}, status.InProgressStatus),

		Entry(nil, 3, 0, map[kstatus.ConditionType]corev1.ConditionStatus{"Test": corev1.ConditionFalse}, false, false, []string{"Test"}, status.InProgressStatus),
		Entry(nil, 3, 1, map[kstatus.ConditionType]corev1.ConditionStatus{"Test": corev1.ConditionFalse}, false, false, []string{"Test"}, status.InProgressStatus),
		Entry(nil, 3, 3, map[kstatus.ConditionType]corev1.ConditionStatus{"Test": corev1.ConditionFalse}, false, false, []string{"Test"}, status.InProgressStatus),
		Entry(nil, 3, 0, map[kstatus.ConditionType]corev1.ConditionStatus{"Test": corev1.ConditionFalse}, true, false, []string{"Test"}, status.InProgressStatus),
		Entry(nil, 3, 1, map[kstatus.ConditionType]corev1.ConditionStatus{"Test": corev1.ConditionFalse}, true, false, []string{"Test"}, status.InProgressStatus),
		Entry(nil, 3, 3, map[kstatus.ConditionType]corev1.ConditionStatus{"Test": corev1.ConditionFalse}, true, false, []string{"Test"}, status.InProgressStatus),

		Entry(nil, 3, 0, map[kstatus.ConditionType]corev1.ConditionStatus{"Test": corev1.ConditionTrue}, false, false, []string{"Test"}, status.CurrentStatus),
		Entry(nil, 3, 1, map[kstatus.ConditionType]corev1.ConditionStatus{"Test": corev1.ConditionTrue}, false, false, []string{"Test"}, status.InProgressStatus),
		Entry(nil, 3, 3, map[kstatus.ConditionType]corev1.ConditionStatus{"Test": corev1.ConditionTrue}, false, false, []string{"Test"}, status.CurrentStatus),
		Entry(nil, 3, 0, map[kstatus.ConditionType]corev1.ConditionStatus{"Test": corev1.ConditionTrue}, true, false, []string{"Test"}, status.InProgressStatus),
		Entry(nil, 3, 1, map[kstatus.ConditionType]corev1.ConditionStatus{"Test": corev1.ConditionTrue}, true, false, []string{"Test"}, status.InProgressStatus),
		Entry(nil, 3, 3, map[kstatus.ConditionType]corev1.ConditionStatus{"Test": corev1.ConditionTrue}, true, false, []string{"Test"}, status.CurrentStatus),
	)
})

type Object struct {
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Status            ObjectStatus `json:"status"`
}

type ObjectStatus struct {
	ObservedGeneration int64               `json:"observedGeneration,omitempty"`
	Conditions         []kstatus.Condition `json:"conditions,omitempty"`
}
