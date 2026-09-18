/*
SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package util_test

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/sap/component-operator-runtime/internal/util"
	cstestingv1alpha1 "github.com/sap/component-operator-runtime/testing/environment/apis/testing.cs.sap.com/v1alpha1"
)

var _ = Describe("testing: object.go", func() {

	Describe("testing: SetLabel() and RemoveLabel()", func() {

		var objWithNilLabels client.Object
		var objWithZeroLabels client.Object
		var objWithOneLabel client.Object
		var objWithTwoLabels client.Object
		var objWithTwoLabelsModified client.Object

		BeforeEach(func() {
			objWithNilLabels = &cstestingv1alpha1.Foo{}
			objWithZeroLabels = &cstestingv1alpha1.Foo{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{},
				},
			}
			objWithOneLabel = &cstestingv1alpha1.Foo{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"key1": "value1",
					},
				},
			}
			objWithTwoLabels = &cstestingv1alpha1.Foo{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"key1": "value1",
						"key2": "value2",
					},
				},
			}
			objWithTwoLabelsModified = &cstestingv1alpha1.Foo{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"key1": "value1",
						"key2": "othervalue2",
					},
				},
			}
		})

		It("should set labels correctly", func() {
			obj := objWithNilLabels.DeepCopyObject().(client.Object)
			util.SetLabel(obj, "key1", "value1")
			Expect(obj).To(Equal(objWithOneLabel))

			obj = objWithZeroLabels.DeepCopyObject().(client.Object)
			util.SetLabel(obj, "key1", "value1")
			Expect(obj).To(Equal(objWithOneLabel))

			obj = objWithOneLabel.DeepCopyObject().(client.Object)
			util.SetLabel(obj, "key1", "value1")
			Expect(obj).To(Equal(objWithOneLabel))

			obj = objWithOneLabel.DeepCopyObject().(client.Object)
			util.SetLabel(obj, "key2", "value2")
			Expect(obj).To(Equal(objWithTwoLabels))

			obj = objWithTwoLabels.DeepCopyObject().(client.Object)
			util.SetLabel(obj, "key2", "value2")
			Expect(obj).To(Equal(objWithTwoLabels))

			obj = objWithTwoLabels.DeepCopyObject().(client.Object)
			util.SetLabel(obj, "key2", "othervalue2")
			Expect(obj).To(Equal(objWithTwoLabelsModified))
		})

		It("should remove labels correctly", func() {
			obj := objWithNilLabels.DeepCopyObject().(client.Object)
			util.RemoveLabel(obj, "key1")
			Expect(obj).To(Equal(objWithNilLabels))

			obj = objWithZeroLabels.DeepCopyObject().(client.Object)
			util.RemoveLabel(obj, "key1")
			Expect(obj).To(Equal(objWithZeroLabels))

			obj = objWithOneLabel.DeepCopyObject().(client.Object)
			util.RemoveLabel(obj, "key1")
			Expect(obj).To(Equal(objWithZeroLabels))

			obj = objWithTwoLabels.DeepCopyObject().(client.Object)
			util.RemoveLabel(obj, "key2")
			Expect(obj).To(Equal(objWithOneLabel))
		})
	})

	Describe("testing: SetAnnotation() and RemoveAnnotation()", func() {

		var objWithNilAnnotations client.Object
		var objWithZeroAnnotations client.Object
		var objWithOneAnnotation client.Object
		var objWithTwoAnnotations client.Object
		var objWithTwoAnnotationsModified client.Object

		BeforeEach(func() {
			objWithNilAnnotations = &cstestingv1alpha1.Foo{}
			objWithZeroAnnotations = &cstestingv1alpha1.Foo{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
				},
			}
			objWithOneAnnotation = &cstestingv1alpha1.Foo{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"key1": "value1",
					},
				},
			}
			objWithTwoAnnotations = &cstestingv1alpha1.Foo{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"key1": "value1",
						"key2": "value2",
					},
				},
			}
			objWithTwoAnnotationsModified = &cstestingv1alpha1.Foo{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"key1": "value1",
						"key2": "othervalue2",
					},
				},
			}
		})

		It("should set annotations correctly", func() {
			obj := objWithNilAnnotations.DeepCopyObject().(client.Object)
			util.SetAnnotation(obj, "key1", "value1")
			Expect(obj).To(Equal(objWithOneAnnotation))

			obj = objWithZeroAnnotations.DeepCopyObject().(client.Object)
			util.SetAnnotation(obj, "key1", "value1")
			Expect(obj).To(Equal(objWithOneAnnotation))

			obj = objWithOneAnnotation.DeepCopyObject().(client.Object)
			util.SetAnnotation(obj, "key1", "value1")
			Expect(obj).To(Equal(objWithOneAnnotation))

			obj = objWithOneAnnotation.DeepCopyObject().(client.Object)
			util.SetAnnotation(obj, "key2", "value2")
			Expect(obj).To(Equal(objWithTwoAnnotations))

			obj = objWithTwoAnnotations.DeepCopyObject().(client.Object)
			util.SetAnnotation(obj, "key2", "value2")
			Expect(obj).To(Equal(objWithTwoAnnotations))

			obj = objWithTwoAnnotations.DeepCopyObject().(client.Object)
			util.SetAnnotation(obj, "key2", "othervalue2")
			Expect(obj).To(Equal(objWithTwoAnnotationsModified))
		})

		It("should remove annotations correctly", func() {
			obj := objWithNilAnnotations.DeepCopyObject().(client.Object)
			util.RemoveAnnotation(obj, "key1")
			Expect(obj).To(Equal(objWithNilAnnotations))

			obj = objWithZeroAnnotations.DeepCopyObject().(client.Object)
			util.RemoveAnnotation(obj, "key1")
			Expect(obj).To(Equal(objWithZeroAnnotations))

			obj = objWithOneAnnotation.DeepCopyObject().(client.Object)
			util.RemoveAnnotation(obj, "key1")
			Expect(obj).To(Equal(objWithZeroAnnotations))

			obj = objWithTwoAnnotations.DeepCopyObject().(client.Object)
			util.RemoveAnnotation(obj, "key2")
			Expect(obj).To(Equal(objWithOneAnnotation))
		})
	})

})
