/*
SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and component-operator-runtime contributors
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
	type ObjectStatus struct {
		ObservedGeneration int64              `json:"observedGeneration,omitempty"`
		Conditions         []metav1.Condition `json:"conditions,omitempty"`
	}
	type Object struct {
		metav1.ObjectMeta `json:"metadata,omitempty"`
		Status            ObjectStatus `json:"status"`
	}

	var analyzer status.StatusAnalyzer

	BeforeEach(func() {
		analyzer = status.NewStatusAnalyzer("test")
	})

	Describe("testing errors: ComputeStatus()", func() {
		It("should return an error if 'always-ready' is combined with other hints", func() {
			obj := Object{
				ObjectMeta: metav1.ObjectMeta{
					Generation: 1,
					Annotations: map[string]string{
						"test/status-hint": "always-ready,has-observed-generation",
					},
				},
				Status: ObjectStatus{
					ObservedGeneration: 0,
					Conditions: []metav1.Condition{
						metav1.Condition{Type: "Ready", Status: metav1.ConditionFalse},
					},
				},
			}
			unstructuredContent, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&obj)
			Expect(err).NotTo(HaveOccurred())
			unstructuredObj := &unstructured.Unstructured{Object: unstructuredContent}

			_, err = analyzer.ComputeStatus(unstructuredObj)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("status hint 'always-ready' cannot be combined with other hints"))
		})

		It("should return an error if 'has-observed-generation' is provided with a value", func() {
			obj := Object{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"test/status-hint": "has-observed-generation=foo",
					},
				},
			}
			unstructuredContent, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&obj)
			Expect(err).NotTo(HaveOccurred())
			unstructuredObj := &unstructured.Unstructured{Object: unstructuredContent}

			_, err = analyzer.ComputeStatus(unstructuredObj)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("status hint 'has-observed-generation' does not take a value"))
		})

		It("should return an error if 'has-ready-condition' is provided with a value", func() {
			obj := Object{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"test/status-hint": "has-ready-condition=foo",
					},
				},
			}
			unstructuredContent, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&obj)
			Expect(err).NotTo(HaveOccurred())
			unstructuredObj := &unstructured.Unstructured{Object: unstructuredContent}

			_, err = analyzer.ComputeStatus(unstructuredObj)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("status hint 'has-ready-condition' does not take a value"))
		})

		It("should return an error if 'conditions' is provided without a value", func() {
			obj := Object{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"test/status-hint": "conditions",
					},
				},
			}
			unstructuredContent, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&obj)
			Expect(err).NotTo(HaveOccurred())
			unstructuredObj := &unstructured.Unstructured{Object: unstructuredContent}

			_, err = analyzer.ComputeStatus(unstructuredObj)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("status hint 'conditions' requires a value"))
		})

		It("should return an error if an unkown hint is provided", func() {
			obj := Object{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"test/status-hint": "unknown-hint",
					},
				},
			}
			unstructuredContent, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&obj)
			Expect(err).NotTo(HaveOccurred())
			unstructuredObj := &unstructured.Unstructured{Object: unstructuredContent}

			_, err = analyzer.ComputeStatus(unstructuredObj)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("unknown status hint 'unknown-hint'"))
		})
	})

	DescribeTable("testing: ComputeStatus()",
		func(generation int, observedGeneration int, conditions []metav1.Condition, hintAlwaysReady bool, hintObservedGeneration bool, hintReadyCondition bool, hintConditions []string, expectedStatus status.Status) {
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
			if hintAlwaysReady {
				hints = append(hints, "always-ready")
			}
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

		// always-ready, no other conditions
		Entry(nil, 3, 0, nil, true, false, false, nil, status.CurrentStatus),
		Entry(nil, 3, 1, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionFalse}}, true, false, false, nil, status.CurrentStatus),
		Entry(nil, 3, 3, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionUnknown}}, true, false, false, nil, status.CurrentStatus),

		// no conditions, has-observed-generation:true|false, has-ready-condition:false
		Entry(nil, 3, 0, nil, false, false, false, nil, status.CurrentStatus),
		Entry(nil, 3, 1, nil, false, false, false, nil, status.InProgressStatus),
		Entry(nil, 3, 3, nil, false, false, false, nil, status.CurrentStatus),
		Entry(nil, 3, 0, nil, false, true, false, nil, status.InProgressStatus),
		Entry(nil, 3, 1, nil, false, true, false, nil, status.InProgressStatus),
		Entry(nil, 3, 3, nil, false, true, false, nil, status.CurrentStatus),

		// ready condition:unknown, has-observed-generation:true|false, has-ready-condition:false
		Entry(nil, 3, 0, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionUnknown}}, false, false, false, nil, status.InProgressStatus),
		Entry(nil, 3, 1, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionUnknown}}, false, false, false, nil, status.InProgressStatus),
		Entry(nil, 3, 3, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionUnknown}}, false, false, false, nil, status.InProgressStatus),
		Entry(nil, 3, 0, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionUnknown}}, false, true, false, nil, status.InProgressStatus),
		Entry(nil, 3, 1, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionUnknown}}, false, true, false, nil, status.InProgressStatus),
		Entry(nil, 3, 3, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionUnknown}}, false, true, false, nil, status.InProgressStatus),

		// ready condition:false, has-observed-generation:true|false, has-ready-condition:false
		Entry(nil, 3, 0, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionFalse}}, false, false, false, nil, status.InProgressStatus),
		Entry(nil, 3, 1, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionFalse}}, false, false, false, nil, status.InProgressStatus),
		Entry(nil, 3, 3, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionFalse}}, false, false, false, nil, status.InProgressStatus),
		Entry(nil, 3, 0, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionFalse}}, false, true, false, nil, status.InProgressStatus),
		Entry(nil, 3, 1, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionFalse}}, false, true, false, nil, status.InProgressStatus),
		Entry(nil, 3, 3, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionFalse}}, false, true, false, nil, status.InProgressStatus),

		// ready condition:true, has-observed-generation:true|false, has-ready-condition:false
		Entry(nil, 3, 0, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionTrue}}, false, false, false, nil, status.CurrentStatus),
		Entry(nil, 3, 1, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionTrue}}, false, false, false, nil, status.InProgressStatus),
		Entry(nil, 3, 3, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionTrue}}, false, false, false, nil, status.CurrentStatus),
		Entry(nil, 3, 0, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionTrue}}, false, true, false, nil, status.InProgressStatus),
		Entry(nil, 3, 1, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionTrue}}, false, true, false, nil, status.InProgressStatus),
		Entry(nil, 3, 3, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionTrue}}, false, true, false, nil, status.CurrentStatus),

		// no conditions, has-observed-generation:true|false, has-ready-condition:true
		Entry(nil, 3, 0, nil, false, false, true, nil, status.InProgressStatus),
		Entry(nil, 3, 1, nil, false, false, true, nil, status.InProgressStatus),
		Entry(nil, 3, 3, nil, false, false, true, nil, status.InProgressStatus),
		Entry(nil, 3, 0, nil, false, true, true, nil, status.InProgressStatus),
		Entry(nil, 3, 1, nil, false, true, true, nil, status.InProgressStatus),
		Entry(nil, 3, 3, nil, false, true, true, nil, status.InProgressStatus),

		// no conditions, has-observed-generation:true|false, has-ready-condition:false, conditions:test
		Entry(nil, 3, 0, nil, false, false, false, []string{"Test"}, status.InProgressStatus),
		Entry(nil, 3, 1, nil, false, false, false, []string{"Test"}, status.InProgressStatus),
		Entry(nil, 3, 3, nil, false, false, false, []string{"Test"}, status.InProgressStatus),
		Entry(nil, 3, 0, nil, false, true, false, []string{"Test"}, status.InProgressStatus),
		Entry(nil, 3, 1, nil, false, true, false, []string{"Test"}, status.InProgressStatus),
		Entry(nil, 3, 3, nil, false, true, false, []string{"Test"}, status.InProgressStatus),

		// test condition:unknown, has-observed-generation:true|false, has-ready-condition:false, conditions:test
		Entry(nil, 3, 0, []metav1.Condition{{Type: "Test", Status: metav1.ConditionUnknown}}, false, false, false, []string{"Test"}, status.InProgressStatus),
		Entry(nil, 3, 1, []metav1.Condition{{Type: "Test", Status: metav1.ConditionUnknown}}, false, false, false, []string{"Test"}, status.InProgressStatus),
		Entry(nil, 3, 3, []metav1.Condition{{Type: "Test", Status: metav1.ConditionUnknown}}, false, false, false, []string{"Test"}, status.InProgressStatus),
		Entry(nil, 3, 0, []metav1.Condition{{Type: "Test", Status: metav1.ConditionUnknown}}, false, true, false, []string{"Test"}, status.InProgressStatus),
		Entry(nil, 3, 1, []metav1.Condition{{Type: "Test", Status: metav1.ConditionUnknown}}, false, true, false, []string{"Test"}, status.InProgressStatus),
		Entry(nil, 3, 3, []metav1.Condition{{Type: "Test", Status: metav1.ConditionUnknown}}, false, true, false, []string{"Test"}, status.InProgressStatus),

		// test condition:false, has-observed-generation:true|false, has-ready-condition:false, conditions:test
		Entry(nil, 3, 0, []metav1.Condition{{Type: "Test", Status: metav1.ConditionFalse}}, false, false, false, []string{"Test"}, status.InProgressStatus),
		Entry(nil, 3, 1, []metav1.Condition{{Type: "Test", Status: metav1.ConditionFalse}}, false, false, false, []string{"Test"}, status.InProgressStatus),
		Entry(nil, 3, 3, []metav1.Condition{{Type: "Test", Status: metav1.ConditionFalse}}, false, false, false, []string{"Test"}, status.InProgressStatus),
		Entry(nil, 3, 0, []metav1.Condition{{Type: "Test", Status: metav1.ConditionFalse}}, false, true, false, []string{"Test"}, status.InProgressStatus),
		Entry(nil, 3, 1, []metav1.Condition{{Type: "Test", Status: metav1.ConditionFalse}}, false, true, false, []string{"Test"}, status.InProgressStatus),
		Entry(nil, 3, 3, []metav1.Condition{{Type: "Test", Status: metav1.ConditionFalse}}, false, true, false, []string{"Test"}, status.InProgressStatus),

		// test condition:true, has-observed-generation:true|false, has-ready-condition:false, conditions:test
		Entry(nil, 3, 0, []metav1.Condition{{Type: "Test", Status: metav1.ConditionTrue}}, false, false, false, []string{"Test"}, status.CurrentStatus),
		Entry(nil, 3, 1, []metav1.Condition{{Type: "Test", Status: metav1.ConditionTrue}}, false, false, false, []string{"Test"}, status.InProgressStatus),
		Entry(nil, 3, 3, []metav1.Condition{{Type: "Test", Status: metav1.ConditionTrue}}, false, false, false, []string{"Test"}, status.CurrentStatus),
		Entry(nil, 3, 0, []metav1.Condition{{Type: "Test", Status: metav1.ConditionTrue}}, false, true, false, []string{"Test"}, status.InProgressStatus),
		Entry(nil, 3, 1, []metav1.Condition{{Type: "Test", Status: metav1.ConditionTrue}}, false, true, false, []string{"Test"}, status.InProgressStatus),
		Entry(nil, 3, 3, []metav1.Condition{{Type: "Test", Status: metav1.ConditionTrue}}, false, true, false, []string{"Test"}, status.CurrentStatus),

		// ready condition:unknown/0, has-observed-generation:true|false, has-ready-condition:false
		Entry(nil, 3, 0, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionUnknown, ObservedGeneration: 0}}, false, false, false, nil, status.InProgressStatus),
		Entry(nil, 3, 1, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionUnknown, ObservedGeneration: 0}}, false, false, false, nil, status.InProgressStatus),
		Entry(nil, 3, 3, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionUnknown, ObservedGeneration: 0}}, false, false, false, nil, status.InProgressStatus),
		Entry(nil, 3, 0, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionUnknown, ObservedGeneration: 0}}, false, true, false, nil, status.InProgressStatus),
		Entry(nil, 3, 1, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionUnknown, ObservedGeneration: 0}}, false, true, false, nil, status.InProgressStatus),
		Entry(nil, 3, 3, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionUnknown, ObservedGeneration: 0}}, false, true, false, nil, status.InProgressStatus),

		// ready condition:false/0, has-observed-generation:true|false, has-ready-condition:false
		Entry(nil, 3, 0, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionFalse, ObservedGeneration: 0}}, false, false, false, nil, status.InProgressStatus),
		Entry(nil, 3, 1, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionFalse, ObservedGeneration: 0}}, false, false, false, nil, status.InProgressStatus),
		Entry(nil, 3, 3, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionFalse, ObservedGeneration: 0}}, false, false, false, nil, status.InProgressStatus),
		Entry(nil, 3, 0, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionFalse, ObservedGeneration: 0}}, false, true, false, nil, status.InProgressStatus),
		Entry(nil, 3, 1, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionFalse, ObservedGeneration: 0}}, false, true, false, nil, status.InProgressStatus),
		Entry(nil, 3, 3, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionFalse, ObservedGeneration: 0}}, false, true, false, nil, status.InProgressStatus),

		// ready condition:true/0, has-observed-generation:true|false, has-ready-condition:false
		Entry(nil, 3, 0, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionTrue, ObservedGeneration: 0}}, false, false, false, nil, status.CurrentStatus),
		Entry(nil, 3, 1, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionTrue, ObservedGeneration: 0}}, false, false, false, nil, status.InProgressStatus),
		Entry(nil, 3, 3, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionTrue, ObservedGeneration: 0}}, false, false, false, nil, status.CurrentStatus),
		Entry(nil, 3, 0, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionTrue, ObservedGeneration: 0}}, false, true, false, nil, status.InProgressStatus),
		Entry(nil, 3, 1, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionTrue, ObservedGeneration: 0}}, false, true, false, nil, status.InProgressStatus),
		Entry(nil, 3, 3, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionTrue, ObservedGeneration: 0}}, false, true, false, nil, status.CurrentStatus),

		// ready condition:unknown/1, has-observed-generation:true|false, has-ready-condition:false
		Entry(nil, 3, 0, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionUnknown, ObservedGeneration: 1}}, false, false, false, nil, status.InProgressStatus),
		Entry(nil, 3, 1, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionUnknown, ObservedGeneration: 1}}, false, false, false, nil, status.InProgressStatus),
		Entry(nil, 3, 3, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionUnknown, ObservedGeneration: 1}}, false, false, false, nil, status.InProgressStatus),
		Entry(nil, 3, 0, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionUnknown, ObservedGeneration: 1}}, false, true, false, nil, status.InProgressStatus),
		Entry(nil, 3, 1, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionUnknown, ObservedGeneration: 1}}, false, true, false, nil, status.InProgressStatus),
		Entry(nil, 3, 3, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionUnknown, ObservedGeneration: 1}}, false, true, false, nil, status.InProgressStatus),

		// ready condition:false/1, has-observed-generation:true|false, has-ready-condition:false
		Entry(nil, 3, 0, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionFalse, ObservedGeneration: 1}}, false, false, false, nil, status.InProgressStatus),
		Entry(nil, 3, 1, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionFalse, ObservedGeneration: 1}}, false, false, false, nil, status.InProgressStatus),
		Entry(nil, 3, 3, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionFalse, ObservedGeneration: 1}}, false, false, false, nil, status.InProgressStatus),
		Entry(nil, 3, 0, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionFalse, ObservedGeneration: 1}}, false, true, false, nil, status.InProgressStatus),
		Entry(nil, 3, 1, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionFalse, ObservedGeneration: 1}}, false, true, false, nil, status.InProgressStatus),
		Entry(nil, 3, 3, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionFalse, ObservedGeneration: 1}}, false, true, false, nil, status.InProgressStatus),

		// ready condition:true/1, has-observed-generation:true|false, has-ready-condition:false
		Entry(nil, 3, 0, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionTrue, ObservedGeneration: 1}}, false, false, false, nil, status.CurrentStatus),
		Entry(nil, 3, 1, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionTrue, ObservedGeneration: 1}}, false, false, false, nil, status.InProgressStatus),
		Entry(nil, 3, 3, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionTrue, ObservedGeneration: 1}}, false, false, false, nil, status.CurrentStatus),
		Entry(nil, 3, 0, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionTrue, ObservedGeneration: 1}}, false, true, false, nil, status.InProgressStatus),
		Entry(nil, 3, 1, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionTrue, ObservedGeneration: 1}}, false, true, false, nil, status.InProgressStatus),
		Entry(nil, 3, 3, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionTrue, ObservedGeneration: 1}}, false, true, false, nil, status.CurrentStatus),

		// ready condition:unknown/3, has-observed-generation:true|false, has-ready-condition:false
		Entry(nil, 3, 0, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionUnknown, ObservedGeneration: 3}}, false, false, false, nil, status.InProgressStatus),
		Entry(nil, 3, 1, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionUnknown, ObservedGeneration: 3}}, false, false, false, nil, status.InProgressStatus),
		Entry(nil, 3, 3, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionUnknown, ObservedGeneration: 3}}, false, false, false, nil, status.InProgressStatus),
		Entry(nil, 3, 0, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionUnknown, ObservedGeneration: 3}}, false, true, false, nil, status.InProgressStatus),
		Entry(nil, 3, 1, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionUnknown, ObservedGeneration: 3}}, false, true, false, nil, status.InProgressStatus),
		Entry(nil, 3, 3, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionUnknown, ObservedGeneration: 3}}, false, true, false, nil, status.InProgressStatus),

		// ready condition:false/3, has-observed-generation:true|false, has-ready-condition:false
		Entry(nil, 3, 0, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionFalse, ObservedGeneration: 3}}, false, false, false, nil, status.InProgressStatus),
		Entry(nil, 3, 1, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionFalse, ObservedGeneration: 3}}, false, false, false, nil, status.InProgressStatus),
		Entry(nil, 3, 3, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionFalse, ObservedGeneration: 3}}, false, false, false, nil, status.InProgressStatus),
		Entry(nil, 3, 0, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionFalse, ObservedGeneration: 3}}, false, true, false, nil, status.InProgressStatus),
		Entry(nil, 3, 1, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionFalse, ObservedGeneration: 3}}, false, true, false, nil, status.InProgressStatus),
		Entry(nil, 3, 3, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionFalse, ObservedGeneration: 3}}, false, true, false, nil, status.InProgressStatus),

		// ready condition:true/3, has-observed-generation:true|false, has-ready-condition:false
		Entry(nil, 3, 0, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionTrue, ObservedGeneration: 3}}, false, false, false, nil, status.CurrentStatus),
		Entry(nil, 3, 1, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionTrue, ObservedGeneration: 3}}, false, false, false, nil, status.InProgressStatus),
		Entry(nil, 3, 3, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionTrue, ObservedGeneration: 3}}, false, false, false, nil, status.CurrentStatus),
		Entry(nil, 3, 0, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionTrue, ObservedGeneration: 3}}, false, true, false, nil, status.CurrentStatus),
		Entry(nil, 3, 1, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionTrue, ObservedGeneration: 3}}, false, true, false, nil, status.InProgressStatus),
		Entry(nil, 3, 3, []metav1.Condition{{Type: "Ready", Status: metav1.ConditionTrue, ObservedGeneration: 3}}, false, true, false, nil, status.CurrentStatus),
	)
})
