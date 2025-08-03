/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package status_test

import (
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

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
		func(generation int, observedGeneration int, conditions []metav1.Condition, hintObservedGeneration bool, hintReadyCondition bool, hintConditions []string, expectedStatus status.Status) {
			type ObjectStatus struct {
				ObservedGeneration int64              `json:"observedGeneration,omitempty"`
				Conditions         []metav1.Condition `json:"conditions,omitempty"`
			}
			type Object struct {
				metav1.ObjectMeta `json:"metadata,omitempty"`
				Status            ObjectStatus `json:"status"`
			}

			obj := Object{
				ObjectMeta: metav1.ObjectMeta{
					Generation: int64(generation),
				},
				Status: ObjectStatus{
					ObservedGeneration: int64(observedGeneration),
				},
			}
			obj.Status.Conditions = append(obj.Status.Conditions, conditions...)
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

		// no conditions, has-observed-generation:true|false, has-ready-condition:false
		Entry(nil, 3, 0, nil, false, false, nil, status.CurrentStatus),
		Entry(nil, 3, 1, nil, false, false, nil, status.InProgressStatus),
		Entry(nil, 3, 3, nil, false, false, nil, status.CurrentStatus),
		Entry(nil, 3, 0, nil, true, false, nil, status.InProgressStatus),
		Entry(nil, 3, 1, nil, true, false, nil, status.InProgressStatus),
		Entry(nil, 3, 3, nil, true, false, nil, status.CurrentStatus),

		// ready condition:unknown, has-observed-generation:true|false, has-ready-condition:false
		Entry(nil, 3, 0, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionUnknown}}, false, false, nil, status.InProgressStatus),
		Entry(nil, 3, 1, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionUnknown}}, false, false, nil, status.InProgressStatus),
		Entry(nil, 3, 3, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionUnknown}}, false, false, nil, status.InProgressStatus),
		Entry(nil, 3, 0, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionUnknown}}, true, false, nil, status.InProgressStatus),
		Entry(nil, 3, 1, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionUnknown}}, true, false, nil, status.InProgressStatus),
		Entry(nil, 3, 3, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionUnknown}}, true, false, nil, status.InProgressStatus),

		// ready condition:false, has-observed-generation:true|false, has-ready-condition:false
		Entry(nil, 3, 0, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionFalse}}, false, false, nil, status.InProgressStatus),
		Entry(nil, 3, 1, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionFalse}}, false, false, nil, status.InProgressStatus),
		Entry(nil, 3, 3, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionFalse}}, false, false, nil, status.InProgressStatus),
		Entry(nil, 3, 0, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionFalse}}, true, false, nil, status.InProgressStatus),
		Entry(nil, 3, 1, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionFalse}}, true, false, nil, status.InProgressStatus),
		Entry(nil, 3, 3, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionFalse}}, true, false, nil, status.InProgressStatus),

		// ready condition:true, has-observed-generation:true|false, has-ready-condition:false
		Entry(nil, 3, 0, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionTrue}}, false, false, nil, status.CurrentStatus),
		Entry(nil, 3, 1, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionTrue}}, false, false, nil, status.InProgressStatus),
		Entry(nil, 3, 3, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionTrue}}, false, false, nil, status.CurrentStatus),
		Entry(nil, 3, 0, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionTrue}}, true, false, nil, status.InProgressStatus),
		Entry(nil, 3, 1, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionTrue}}, true, false, nil, status.InProgressStatus),
		Entry(nil, 3, 3, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionTrue}}, true, false, nil, status.CurrentStatus),

		// no conditions, has-observed-generation:true|false, has-ready-condition:true
		Entry(nil, 3, 0, nil, false, true, nil, status.InProgressStatus),
		Entry(nil, 3, 1, nil, false, true, nil, status.InProgressStatus),
		Entry(nil, 3, 3, nil, false, true, nil, status.InProgressStatus),
		Entry(nil, 3, 0, nil, true, true, nil, status.InProgressStatus),
		Entry(nil, 3, 1, nil, true, true, nil, status.InProgressStatus),
		Entry(nil, 3, 3, nil, true, true, nil, status.InProgressStatus),

		// no conditions, has-observed-generation:true|false, has-ready-condition:false, conditions:test
		Entry(nil, 3, 0, nil, false, false, []string{"Test"}, status.InProgressStatus),
		Entry(nil, 3, 1, nil, false, false, []string{"Test"}, status.InProgressStatus),
		Entry(nil, 3, 3, nil, false, false, []string{"Test"}, status.InProgressStatus),
		Entry(nil, 3, 0, nil, true, false, []string{"Test"}, status.InProgressStatus),
		Entry(nil, 3, 1, nil, true, false, []string{"Test"}, status.InProgressStatus),
		Entry(nil, 3, 3, nil, true, false, []string{"Test"}, status.InProgressStatus),

		// test condition:unknown, has-observed-generation:true|false, has-ready-condition:false, conditions:test
		Entry(nil, 3, 0, []metav1.Condition{{Type: "Test", Status: metav1.ConditionUnknown}}, false, false, []string{"Test"}, status.InProgressStatus),
		Entry(nil, 3, 1, []metav1.Condition{{Type: "Test", Status: metav1.ConditionUnknown}}, false, false, []string{"Test"}, status.InProgressStatus),
		Entry(nil, 3, 3, []metav1.Condition{{Type: "Test", Status: metav1.ConditionUnknown}}, false, false, []string{"Test"}, status.InProgressStatus),
		Entry(nil, 3, 0, []metav1.Condition{{Type: "Test", Status: metav1.ConditionUnknown}}, true, false, []string{"Test"}, status.InProgressStatus),
		Entry(nil, 3, 1, []metav1.Condition{{Type: "Test", Status: metav1.ConditionUnknown}}, true, false, []string{"Test"}, status.InProgressStatus),
		Entry(nil, 3, 3, []metav1.Condition{{Type: "Test", Status: metav1.ConditionUnknown}}, true, false, []string{"Test"}, status.InProgressStatus),

		// test condition:false, has-observed-generation:true|false, has-ready-condition:false, conditions:test
		Entry(nil, 3, 0, []metav1.Condition{{Type: "Test", Status: metav1.ConditionFalse}}, false, false, []string{"Test"}, status.InProgressStatus),
		Entry(nil, 3, 1, []metav1.Condition{{Type: "Test", Status: metav1.ConditionFalse}}, false, false, []string{"Test"}, status.InProgressStatus),
		Entry(nil, 3, 3, []metav1.Condition{{Type: "Test", Status: metav1.ConditionFalse}}, false, false, []string{"Test"}, status.InProgressStatus),
		Entry(nil, 3, 0, []metav1.Condition{{Type: "Test", Status: metav1.ConditionFalse}}, true, false, []string{"Test"}, status.InProgressStatus),
		Entry(nil, 3, 1, []metav1.Condition{{Type: "Test", Status: metav1.ConditionFalse}}, true, false, []string{"Test"}, status.InProgressStatus),
		Entry(nil, 3, 3, []metav1.Condition{{Type: "Test", Status: metav1.ConditionFalse}}, true, false, []string{"Test"}, status.InProgressStatus),

		// test condition:true, has-observed-generation:true|false, has-ready-condition:false, conditions:test
		Entry(nil, 3, 0, []metav1.Condition{{Type: "Test", Status: metav1.ConditionTrue}}, false, false, []string{"Test"}, status.CurrentStatus),
		Entry(nil, 3, 1, []metav1.Condition{{Type: "Test", Status: metav1.ConditionTrue}}, false, false, []string{"Test"}, status.InProgressStatus),
		Entry(nil, 3, 3, []metav1.Condition{{Type: "Test", Status: metav1.ConditionTrue}}, false, false, []string{"Test"}, status.CurrentStatus),
		Entry(nil, 3, 0, []metav1.Condition{{Type: "Test", Status: metav1.ConditionTrue}}, true, false, []string{"Test"}, status.InProgressStatus),
		Entry(nil, 3, 1, []metav1.Condition{{Type: "Test", Status: metav1.ConditionTrue}}, true, false, []string{"Test"}, status.InProgressStatus),
		Entry(nil, 3, 3, []metav1.Condition{{Type: "Test", Status: metav1.ConditionTrue}}, true, false, []string{"Test"}, status.CurrentStatus),

		// ready condition:unknown/0, has-observed-generation:true|false, has-ready-condition:false
		Entry(nil, 3, 0, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionUnknown, ObservedGeneration: 0}}, false, false, nil, status.InProgressStatus),
		Entry(nil, 3, 1, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionUnknown, ObservedGeneration: 0}}, false, false, nil, status.InProgressStatus),
		Entry(nil, 3, 3, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionUnknown, ObservedGeneration: 0}}, false, false, nil, status.InProgressStatus),
		Entry(nil, 3, 0, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionUnknown, ObservedGeneration: 0}}, true, false, nil, status.InProgressStatus),
		Entry(nil, 3, 1, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionUnknown, ObservedGeneration: 0}}, true, false, nil, status.InProgressStatus),
		Entry(nil, 3, 3, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionUnknown, ObservedGeneration: 0}}, true, false, nil, status.InProgressStatus),

		// ready condition:false/0, has-observed-generation:true|false, has-ready-condition:false
		Entry(nil, 3, 0, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionFalse, ObservedGeneration: 0}}, false, false, nil, status.InProgressStatus),
		Entry(nil, 3, 1, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionFalse, ObservedGeneration: 0}}, false, false, nil, status.InProgressStatus),
		Entry(nil, 3, 3, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionFalse, ObservedGeneration: 0}}, false, false, nil, status.InProgressStatus),
		Entry(nil, 3, 0, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionFalse, ObservedGeneration: 0}}, true, false, nil, status.InProgressStatus),
		Entry(nil, 3, 1, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionFalse, ObservedGeneration: 0}}, true, false, nil, status.InProgressStatus),
		Entry(nil, 3, 3, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionFalse, ObservedGeneration: 0}}, true, false, nil, status.InProgressStatus),

		// ready condition:true/0, has-observed-generation:true|false, has-ready-condition:false
		Entry(nil, 3, 0, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionTrue, ObservedGeneration: 0}}, false, false, nil, status.CurrentStatus),
		Entry(nil, 3, 1, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionTrue, ObservedGeneration: 0}}, false, false, nil, status.InProgressStatus),
		Entry(nil, 3, 3, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionTrue, ObservedGeneration: 0}}, false, false, nil, status.CurrentStatus),
		Entry(nil, 3, 0, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionTrue, ObservedGeneration: 0}}, true, false, nil, status.InProgressStatus),
		Entry(nil, 3, 1, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionTrue, ObservedGeneration: 0}}, true, false, nil, status.InProgressStatus),
		Entry(nil, 3, 3, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionTrue, ObservedGeneration: 0}}, true, false, nil, status.CurrentStatus),

		// ready condition:unknown/1, has-observed-generation:true|false, has-ready-condition:false
		Entry(nil, 3, 0, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionUnknown, ObservedGeneration: 1}}, false, false, nil, status.InProgressStatus),
		Entry(nil, 3, 1, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionUnknown, ObservedGeneration: 1}}, false, false, nil, status.InProgressStatus),
		Entry(nil, 3, 3, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionUnknown, ObservedGeneration: 1}}, false, false, nil, status.InProgressStatus),
		Entry(nil, 3, 0, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionUnknown, ObservedGeneration: 1}}, true, false, nil, status.InProgressStatus),
		Entry(nil, 3, 1, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionUnknown, ObservedGeneration: 1}}, true, false, nil, status.InProgressStatus),
		Entry(nil, 3, 3, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionUnknown, ObservedGeneration: 1}}, true, false, nil, status.InProgressStatus),

		// ready condition:false/1, has-observed-generation:true|false, has-ready-condition:false
		Entry(nil, 3, 0, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionFalse, ObservedGeneration: 1}}, false, false, nil, status.InProgressStatus),
		Entry(nil, 3, 1, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionFalse, ObservedGeneration: 1}}, false, false, nil, status.InProgressStatus),
		Entry(nil, 3, 3, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionFalse, ObservedGeneration: 1}}, false, false, nil, status.InProgressStatus),
		Entry(nil, 3, 0, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionFalse, ObservedGeneration: 1}}, true, false, nil, status.InProgressStatus),
		Entry(nil, 3, 1, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionFalse, ObservedGeneration: 1}}, true, false, nil, status.InProgressStatus),
		Entry(nil, 3, 3, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionFalse, ObservedGeneration: 1}}, true, false, nil, status.InProgressStatus),

		// ready condition:true/1, has-observed-generation:true|false, has-ready-condition:false
		Entry(nil, 3, 0, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionTrue, ObservedGeneration: 1}}, false, false, nil, status.CurrentStatus),
		Entry(nil, 3, 1, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionTrue, ObservedGeneration: 1}}, false, false, nil, status.InProgressStatus),
		Entry(nil, 3, 3, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionTrue, ObservedGeneration: 1}}, false, false, nil, status.CurrentStatus),
		Entry(nil, 3, 0, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionTrue, ObservedGeneration: 1}}, true, false, nil, status.InProgressStatus),
		Entry(nil, 3, 1, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionTrue, ObservedGeneration: 1}}, true, false, nil, status.InProgressStatus),
		Entry(nil, 3, 3, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionTrue, ObservedGeneration: 1}}, true, false, nil, status.CurrentStatus),

		// ready condition:unknown/3, has-observed-generation:true|false, has-ready-condition:false
		Entry(nil, 3, 0, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionUnknown, ObservedGeneration: 3}}, false, false, nil, status.InProgressStatus),
		Entry(nil, 3, 1, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionUnknown, ObservedGeneration: 3}}, false, false, nil, status.InProgressStatus),
		Entry(nil, 3, 3, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionUnknown, ObservedGeneration: 3}}, false, false, nil, status.InProgressStatus),
		Entry(nil, 3, 0, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionUnknown, ObservedGeneration: 3}}, true, false, nil, status.InProgressStatus),
		Entry(nil, 3, 1, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionUnknown, ObservedGeneration: 3}}, true, false, nil, status.InProgressStatus),
		Entry(nil, 3, 3, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionUnknown, ObservedGeneration: 3}}, true, false, nil, status.InProgressStatus),

		// ready condition:false/3, has-observed-generation:true|false, has-ready-condition:false
		Entry(nil, 3, 0, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionFalse, ObservedGeneration: 3}}, false, false, nil, status.InProgressStatus),
		Entry(nil, 3, 1, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionFalse, ObservedGeneration: 3}}, false, false, nil, status.InProgressStatus),
		Entry(nil, 3, 3, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionFalse, ObservedGeneration: 3}}, false, false, nil, status.InProgressStatus),
		Entry(nil, 3, 0, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionFalse, ObservedGeneration: 3}}, true, false, nil, status.InProgressStatus),
		Entry(nil, 3, 1, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionFalse, ObservedGeneration: 3}}, true, false, nil, status.InProgressStatus),
		Entry(nil, 3, 3, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionFalse, ObservedGeneration: 3}}, true, false, nil, status.InProgressStatus),

		// ready condition:true/3, has-observed-generation:true|false, has-ready-condition:false
		Entry(nil, 3, 0, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionTrue, ObservedGeneration: 3}}, false, false, nil, status.CurrentStatus),
		Entry(nil, 3, 1, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionTrue, ObservedGeneration: 3}}, false, false, nil, status.InProgressStatus),
		Entry(nil, 3, 3, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionTrue, ObservedGeneration: 3}}, false, false, nil, status.CurrentStatus),
		Entry(nil, 3, 0, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionTrue, ObservedGeneration: 3}}, true, false, nil, status.CurrentStatus),
		Entry(nil, 3, 1, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionTrue, ObservedGeneration: 3}}, true, false, nil, status.InProgressStatus),
		Entry(nil, 3, 3, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionTrue, ObservedGeneration: 3}}, true, false, nil, status.CurrentStatus),
	)
})
