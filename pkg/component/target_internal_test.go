/*
SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package component

import (
	"context"
	"fmt"
	"math"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiregistrationv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
	fieldvalue "sigs.k8s.io/structured-merge-diff/v4/value"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/sap/component-operator-runtime/internal/cluster"
	. "github.com/sap/component-operator-runtime/internal/testing"
	barv1alpha1 "github.com/sap/component-operator-runtime/internal/testing/bar/api/v1alpha1"
	foov1alpha1 "github.com/sap/component-operator-runtime/internal/testing/foo/api/v1alpha1"
	"github.com/sap/component-operator-runtime/pkg/types"
)

var _ = Describe("testing: target.go", func() {
	var reconcilerName string
	var reconcilerId string
	var ctx context.Context
	var cli cluster.Client

	BeforeEach(func() {
		reconcilerName = "reconciler.testing.local"
		reconcilerId = "1"
		ctx = Ctx()
		cli = Client()
	})

	Context("testing: ReadObject()", func() {
		var target *reconcileTarget[Component]
		var namespace string
		var foo *foov1alpha1.Foo
		var bar *barv1alpha1.Bar

		BeforeEach(func() {
			target = newReconcileTarget[Component](reconcilerName, reconcilerId, cli, nil, false, "", "")
			namespace = CreateNamespace()

			foo = &foov1alpha1.Foo{
				ObjectMeta: metav1.ObjectMeta{Namespace: namespace, GenerateName: "test-"},
			}
			CreateObject(foo, "")

			bar = &barv1alpha1.Bar{
				ObjectMeta: metav1.ObjectMeta{Namespace: namespace, GenerateName: "test-"},
			}
			CreateObject(bar, "")
		})

		AfterEach(func() {
			CleanupNamespace(namespace, &foov1alpha1.Foo{}, &barv1alpha1.Bar{})
		})

		It("should successfully read an object whose type is known to the scheme", func() {
			_foo, err := target.readObject(ctx, foo)
			Expect(err).NotTo(HaveOccurred())
			Expect(_foo.GetUID()).To(Equal(foo.UID))
		})

		It("should successfully read an object whose type is not known to the scheme", func() {
			_bar, err := target.readObject(ctx, bar)
			Expect(err).NotTo(HaveOccurred())
			Expect(_bar.GetUID()).To(Equal(bar.UID))
		})
	})

	Context("testing: CreateObject()", func() {
		var target *reconcileTarget[Component]
		var namespace string

		BeforeEach(func() {
			target = newReconcileTarget[Component](reconcilerName, reconcilerId, cli, nil, false, "", "")
			namespace = CreateNamespace()
		})

		AfterEach(func() {
			CleanupNamespace(namespace, &foov1alpha1.Foo{}, &barv1alpha1.Bar{})
		})

		It("should successfully create an object whose type is known to the scheme", func() {
			foo := &foov1alpha1.Foo{
				TypeMeta:   metav1.TypeMeta{APIVersion: foov1alpha1.GroupVersion.String(), Kind: "Foo"},
				ObjectMeta: metav1.ObjectMeta{Namespace: namespace, GenerateName: "test-"},
			}
			err := target.createObject(ctx, foo, foo)
			Expect(err).NotTo(HaveOccurred())
			_foo := ReadObject(foo)
			Expect(_foo.GetUID()).To(Equal(foo.UID))
		})

		It("should successfully create an object whose type is not known to the scheme", func() {
			bar := &barv1alpha1.Bar{
				TypeMeta:   metav1.TypeMeta{APIVersion: barv1alpha1.GroupVersion.String(), Kind: "Bar"},
				ObjectMeta: metav1.ObjectMeta{Namespace: namespace, GenerateName: "test-"},
			}
			err := target.createObject(ctx, bar, bar)
			Expect(err).NotTo(HaveOccurred())
			_bar := ReadObject(bar)
			Expect(_bar.GetUID()).To(Equal(bar.UID))
		})

		It("should successfully create an unstructured object", func() {
			bar := AsUnstructured(&barv1alpha1.Bar{
				TypeMeta:   metav1.TypeMeta{APIVersion: barv1alpha1.GroupVersion.String(), Kind: "Bar"},
				ObjectMeta: metav1.ObjectMeta{Namespace: namespace, GenerateName: "test-"},
			})
			err := target.createObject(ctx, bar, bar)
			Expect(err).NotTo(HaveOccurred())
			_bar := ReadObject(bar)
			Expect(_bar.GetUID()).To(Equal(bar.GetUID()))
		})

		// TODO: add a case validating that the finalizer is added for crds and api services
	})

	Context("testing: updateObject()", func() {
		var target *reconcileTarget[Component]
		var namespace string
		var foo *foov1alpha1.Foo
		var bar *barv1alpha1.Bar
		var fooComplex *foov1alpha1.Foo

		BeforeEach(func() {
			target = newReconcileTarget[Component](reconcilerName, reconcilerId, cli, nil, false, "", "")
			namespace = CreateNamespace()

			foo = &foov1alpha1.Foo{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:    namespace,
					GenerateName: "test-",
					Annotations:  map[string]string{"testing.cs.sap.com/key": "a"},
				},
			}
			CreateObject(foo, "")

			bar = &barv1alpha1.Bar{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:    namespace,
					GenerateName: "test-",
					Annotations:  map[string]string{"testing.cs.sap.com/key": "a"},
				},
			}
			CreateObject(bar, "")

			fooComplex = &foov1alpha1.Foo{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:   namespace,
					Name:        "test-complex",
					Annotations: map[string]string{"testing.cs.sap.com/key": "a"},
					Finalizers:  []string{"testing.cs.sap.com/finalizer"},
				},
			}
			ApplyObject(fooComplex.DeepCopy(), "creator")
			fooComplex.Annotations = map[string]string{"testing.cs.sap.com/key-from-reconciler": "a"}
			fooComplex.Finalizers = []string{"testing.cs.sap.com/finalizer-from-reconciler"}
			ApplyObject(fooComplex.DeepCopy(), reconcilerName)
			fooComplex.Annotations = map[string]string{"testing.cs.sap.com/key-from-kubectl": "a"}
			fooComplex.Finalizers = []string{"testing.cs.sap.com/finalizer-from-kubectl"}
			ApplyObject(fooComplex.DeepCopy(), "kubectl-manager")
			fooComplex.Annotations = map[string]string{"testing.cs.sap.com/key-from-helm": "a"}
			fooComplex.Finalizers = []string{"testing.cs.sap.com/finalizer-from-helm"}
			ApplyObject(fooComplex, "helm-manager")
		})

		AfterEach(func() {
			CleanupNamespace(namespace, &foov1alpha1.Foo{}, &barv1alpha1.Bar{})
		})

		It("should successfully update an object whose type is known to the scheme", func() {
			newFoo := &foov1alpha1.Foo{
				TypeMeta: metav1.TypeMeta{APIVersion: foov1alpha1.GroupVersion.String(), Kind: "Foo"},
				ObjectMeta: metav1.ObjectMeta{
					Namespace:   namespace,
					Name:        foo.Name,
					Annotations: map[string]string{"testing.cs.sap.com/key": "b"},
				},
			}
			err := target.updateObject(ctx, newFoo, AsUnstructured(foo), foo, UpdatePolicyReplace)
			Expect(err).NotTo(HaveOccurred())
			_foo := ReadObject(foo)
			Expect(_foo.GetUID()).To(Equal(foo.UID))
			Expect(_foo.GetAnnotations()).To(HaveKeyWithValue("testing.cs.sap.com/key", "b"))
		})

		It("should successfully update an object whose type is not known to the scheme", func() {
			newBar := &barv1alpha1.Bar{
				TypeMeta: metav1.TypeMeta{APIVersion: barv1alpha1.GroupVersion.String(), Kind: "Bar"},
				ObjectMeta: metav1.ObjectMeta{
					Namespace:   namespace,
					Name:        bar.Name,
					Annotations: map[string]string{"testing.cs.sap.com/key": "b"},
				},
			}
			err := target.updateObject(ctx, newBar, AsUnstructured(bar), bar, UpdatePolicyReplace)
			Expect(err).NotTo(HaveOccurred())
			_bar := ReadObject(bar)
			Expect(_bar.GetUID()).To(Equal(bar.UID))
			Expect(_bar.GetAnnotations()).To(HaveKeyWithValue("testing.cs.sap.com/key", "b"))
		})

		It("should successfully update an unstructured object", func() {
			newBar := AsUnstructured(&barv1alpha1.Bar{
				TypeMeta: metav1.TypeMeta{APIVersion: barv1alpha1.GroupVersion.String(), Kind: "Bar"},
				ObjectMeta: metav1.ObjectMeta{
					Namespace:   namespace,
					Name:        bar.Name,
					Annotations: map[string]string{"testing.cs.sap.com/key": "b"},
				},
			})
			err := target.updateObject(ctx, newBar, AsUnstructured(bar), bar, UpdatePolicyReplace)
			Expect(err).NotTo(HaveOccurred())
			_bar := ReadObject(bar)
			Expect(_bar.GetUID()).To(Equal(bar.UID))
			Expect(_bar.GetAnnotations()).To(HaveKeyWithValue("testing.cs.sap.com/key", "b"))
		})

		It("should reject an update with a wrong resourceVersion", func() {
			newBar := AsUnstructured(&barv1alpha1.Bar{
				TypeMeta: metav1.TypeMeta{APIVersion: barv1alpha1.GroupVersion.String(), Kind: "Bar"},
				ObjectMeta: metav1.ObjectMeta{
					Namespace:       namespace,
					Name:            bar.Name,
					Annotations:     map[string]string{"testing.cs.sap.com/key": "b"},
					ResourceVersion: "1",
				},
			})
			err := target.updateObject(ctx, newBar, AsUnstructured(bar), bar, UpdatePolicyReplace)
			if err == nil {
				Expect(err).To(HaveOccurred())
			}
			if !apierrors.IsConflict(err) {
				Expect(err).NotTo(HaveOccurred())
			}
			err = target.updateObject(ctx, newBar, AsUnstructured(bar), bar, UpdatePolicySsaMerge)
			if err == nil {
				Expect(err).To(HaveOccurred())
			}
			if !apierrors.IsConflict(err) {
				Expect(err).NotTo(HaveOccurred())
			}
			err = target.updateObject(ctx, newBar, AsUnstructured(bar), bar, UpdatePolicySsaOverride)
			if err == nil {
				Expect(err).To(HaveOccurred())
			}
			if !apierrors.IsConflict(err) {
				Expect(err).NotTo(HaveOccurred())
			}
		})

		It("should handle complex updates correctly (using replace)", func() {
			newFooComplex := &foov1alpha1.Foo{
				TypeMeta: metav1.TypeMeta{APIVersion: foov1alpha1.GroupVersion.String(), Kind: "Foo"},
				ObjectMeta: metav1.ObjectMeta{
					Namespace:   namespace,
					Name:        fooComplex.Name,
					Annotations: map[string]string{"testing.cs.sap.com/key-from-reconciler": "b"},
					Finalizers:  []string{"testing.cs.sap.com/finalizer-from-reconciler"},
				},
			}
			err := target.updateObject(ctx, newFooComplex, AsUnstructured(fooComplex), fooComplex, UpdatePolicyReplace)
			Expect(err).NotTo(HaveOccurred())
			_fooComplex := ReadObject(fooComplex)
			Expect(_fooComplex.GetUID()).To(Equal(fooComplex.UID))
			Expect(_fooComplex.GetAnnotations()).To(Equal(map[string]string{"testing.cs.sap.com/key-from-reconciler": "b"}))
			Expect(FieldManagersFor(_fooComplex, "metadata", "annotations", "testing.cs.sap.com/key-from-reconciler")).To(ConsistOf(reconcilerName))
			Expect(_fooComplex.GetFinalizers()).To(ConsistOf(
				"testing.cs.sap.com/finalizer",
				"testing.cs.sap.com/finalizer-from-reconciler",
				"testing.cs.sap.com/finalizer-from-kubectl",
				"testing.cs.sap.com/finalizer-from-helm",
			))
			Expect(FieldManagersFor(_fooComplex, "metadata", "finalizers", fieldvalue.NewValueInterface("testing.cs.sap.com/finalizer"))).To(ConsistOf("creator"))
			Expect(FieldManagersFor(_fooComplex, "metadata", "finalizers", fieldvalue.NewValueInterface("testing.cs.sap.com/finalizer-from-reconciler"))).To(ConsistOf(reconcilerName))
			Expect(FieldManagersFor(_fooComplex, "metadata", "finalizers", fieldvalue.NewValueInterface("testing.cs.sap.com/finalizer-from-kubectl"))).To(ConsistOf("kubectl-manager"))
			Expect(FieldManagersFor(_fooComplex, "metadata", "finalizers", fieldvalue.NewValueInterface("testing.cs.sap.com/finalizer-from-helm"))).To(ConsistOf("helm-manager"))
		})

		It("should handle complex updates correctly (using ssa-merge)", func() {
			newFooComplex := &foov1alpha1.Foo{
				TypeMeta: metav1.TypeMeta{APIVersion: foov1alpha1.GroupVersion.String(), Kind: "Foo"},
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
					Name:      fooComplex.Name,
					Annotations: map[string]string{
						"testing.cs.sap.com/key":                 "b",
						"testing.cs.sap.com/key-from-reconciler": "b",
					},
				},
			}
			err := target.updateObject(ctx, newFooComplex, AsUnstructured(fooComplex), fooComplex, UpdatePolicySsaMerge)
			Expect(err).NotTo(HaveOccurred())
			_fooComplex := ReadObject(fooComplex)
			Expect(_fooComplex.GetUID()).To(Equal(fooComplex.UID))
			Expect(_fooComplex.GetAnnotations()).To(Equal(map[string]string{
				"testing.cs.sap.com/key":                 "b",
				"testing.cs.sap.com/key-from-reconciler": "b",
				"testing.cs.sap.com/key-from-kubectl":    "a",
				"testing.cs.sap.com/key-from-helm":       "a",
			}))
			Expect(FieldManagersFor(_fooComplex, "metadata", "annotations", "testing.cs.sap.com/key")).To(ConsistOf(reconcilerName))
			Expect(FieldManagersFor(_fooComplex, "metadata", "annotations", "testing.cs.sap.com/key-from-reconciler")).To(ConsistOf(reconcilerName))
			Expect(FieldManagersFor(_fooComplex, "metadata", "annotations", "testing.cs.sap.com/key-from-kubectl")).To(ConsistOf("kubectl-manager"))
			Expect(FieldManagersFor(_fooComplex, "metadata", "annotations", "testing.cs.sap.com/key-from-helm")).To(ConsistOf("helm-manager"))
			Expect(_fooComplex.GetFinalizers()).To(ConsistOf(
				"testing.cs.sap.com/finalizer",
				"testing.cs.sap.com/finalizer-from-kubectl",
				"testing.cs.sap.com/finalizer-from-helm",
			))
			Expect(FieldManagersFor(_fooComplex, "metadata", "finalizers", fieldvalue.NewValueInterface("testing.cs.sap.com/finalizer"))).To(ConsistOf("creator"))
			Expect(FieldManagersFor(_fooComplex, "metadata", "finalizers", fieldvalue.NewValueInterface("testing.cs.sap.com/finalizer-from-kubectl"))).To(ConsistOf("kubectl-manager"))
			Expect(FieldManagersFor(_fooComplex, "metadata", "finalizers", fieldvalue.NewValueInterface("testing.cs.sap.com/finalizer-from-helm"))).To(ConsistOf("helm-manager"))
		})

		It("should handle complex updates correctly (using ssa-override)", func() {
			newFooComplex := &foov1alpha1.Foo{
				TypeMeta: metav1.TypeMeta{APIVersion: foov1alpha1.GroupVersion.String(), Kind: "Foo"},
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
					Name:      fooComplex.Name,
					Annotations: map[string]string{
						"testing.cs.sap.com/key":              "b",
						"testing.cs.sap.com/key-from-kubectl": "b",
					},
					Finalizers: []string{
						"testing.cs.sap.com/finalizer-from-reconciler",
						"testing.cs.sap.com/finalizer-from-helm",
					},
				},
			}
			err := target.updateObject(ctx, newFooComplex, AsUnstructured(fooComplex), fooComplex, UpdatePolicySsaOverride)
			Expect(err).NotTo(HaveOccurred())
			_fooComplex := ReadObject(fooComplex)
			Expect(_fooComplex.GetUID()).To(Equal(fooComplex.UID))
			Expect(_fooComplex.GetAnnotations()).To(Equal(map[string]string{
				"testing.cs.sap.com/key":              "b",
				"testing.cs.sap.com/key-from-kubectl": "b",
			}))
			Expect(FieldManagersFor(_fooComplex, "metadata", "annotations", "testing.cs.sap.com/key")).To(ConsistOf(reconcilerName))
			Expect(FieldManagersFor(_fooComplex, "metadata", "annotations", "testing.cs.sap.com/key-from-kubectl")).To(ConsistOf(reconcilerName))
			Expect(_fooComplex.GetFinalizers()).To(ConsistOf(
				"testing.cs.sap.com/finalizer",
				"testing.cs.sap.com/finalizer-from-reconciler",
				"testing.cs.sap.com/finalizer-from-helm",
			))
			Expect(FieldManagersFor(_fooComplex, "metadata", "finalizers", fieldvalue.NewValueInterface("testing.cs.sap.com/finalizer"))).To(ConsistOf("creator"))
			Expect(FieldManagersFor(_fooComplex, "metadata", "finalizers", fieldvalue.NewValueInterface("testing.cs.sap.com/finalizer-from-reconciler"))).To(ConsistOf(reconcilerName))
			Expect(FieldManagersFor(_fooComplex, "metadata", "finalizers", fieldvalue.NewValueInterface("testing.cs.sap.com/finalizer-from-helm"))).To(ConsistOf(reconcilerName))
		})

		// TODO: add a case validating that the finalizer is added for crds and api services
	})

	Context("testing: deleteObject()", func() {
		var target *reconcileTarget[Component]
		var namespace string
		var foo *foov1alpha1.Foo

		BeforeEach(func() {
			target = newReconcileTarget[Component](reconcilerName, reconcilerId, cli, nil, false, "", "")
			namespace = CreateNamespace()

			foo = &foov1alpha1.Foo{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:    namespace,
					GenerateName: "test-",
				},
			}
			CreateObject(foo, "")
		})

		AfterEach(func() {
			CleanupNamespace(namespace, &foov1alpha1.Foo{})
		})

		It("should delete the object (without providing the existing object)", func() {
			err := target.deleteObject(ctx, foo, nil)
			Expect(err).NotTo(HaveOccurred())
			EnsureObjectDoesNotExist(foo)
		})

		It("should delete the object (with providing the existing object)", func() {
			err := target.deleteObject(ctx, foo, AsUnstructured(foo))
			Expect(err).NotTo(HaveOccurred())
			EnsureObjectDoesNotExist(foo)
		})

		It("should reject a deletion with a wrong resourceVersion", func() {
			foo.ResourceVersion = "1"
			err := target.deleteObject(ctx, foo, AsUnstructured(foo))
			if err == nil {
				Expect(err).To(HaveOccurred())
			}
			if !apierrors.IsConflict(err) {
				Expect(err).NotTo(HaveOccurred())
			}
		})

		// TODO: add a case validating that the deletion of crds and api services fails if they are still in use
	})

	Context("testing: get*Policy()", func() {
		DescribeTable("testing: getAdoptionPolicy()",
			func(targetPolicy AdoptionPolicy, annotatedPolicy string, expectedPolicy AdoptionPolicy) {
				target := newReconcileTarget[Component](reconcilerName, reconcilerId, nil, nil, false, targetPolicy, "")
				object := &metav1.PartialObjectMetadata{}
				if annotatedPolicy != "" {
					object.Annotations = map[string]string{reconcilerName + "/" + types.AnnotationKeySuffixAdoptionPolicy: annotatedPolicy}
				}
				resultingPolicy, err := target.getAdoptionPolicy(object)
				Expect(err).NotTo(HaveOccurred())
				Expect(resultingPolicy).To(Equal(expectedPolicy))
			},
			Entry(nil, AdoptionPolicyNever, "", AdoptionPolicyNever),
			Entry(nil, AdoptionPolicyNever, types.AdoptionPolicyNever, AdoptionPolicyNever),
			Entry(nil, AdoptionPolicyNever, types.AdoptionPolicyIfUnowned, AdoptionPolicyIfUnowned),
			Entry(nil, AdoptionPolicyNever, types.AdoptionPolicyAlways, AdoptionPolicyAlways),

			Entry(nil, AdoptionPolicyIfUnowned, "", AdoptionPolicyIfUnowned),
			Entry(nil, AdoptionPolicyIfUnowned, types.AdoptionPolicyNever, AdoptionPolicyNever),
			Entry(nil, AdoptionPolicyIfUnowned, types.AdoptionPolicyIfUnowned, AdoptionPolicyIfUnowned),
			Entry(nil, AdoptionPolicyIfUnowned, types.AdoptionPolicyAlways, AdoptionPolicyAlways),

			Entry(nil, AdoptionPolicyAlways, "", AdoptionPolicyAlways),
			Entry(nil, AdoptionPolicyAlways, types.AdoptionPolicyNever, AdoptionPolicyNever),
			Entry(nil, AdoptionPolicyAlways, types.AdoptionPolicyIfUnowned, AdoptionPolicyIfUnowned),
			Entry(nil, AdoptionPolicyAlways, types.AdoptionPolicyAlways, AdoptionPolicyAlways),
		)

		DescribeTable("testing: getReconcilePolicy()",
			func(targetPolicy ReconcilePolicy, annotatedPolicy string, expectedPolicy ReconcilePolicy) {
				target := newReconcileTarget[Component](reconcilerName, reconcilerId, nil, nil, false, "", "")
				target.reconcilePolicy = targetPolicy
				object := &metav1.PartialObjectMetadata{}
				if annotatedPolicy != "" {
					object.Annotations = map[string]string{reconcilerName + "/" + types.AnnotationKeySuffixReconcilePolicy: annotatedPolicy}
				}
				resultingPolicy, err := target.getReconcilePolicy(object)
				Expect(err).NotTo(HaveOccurred())
				Expect(resultingPolicy).To(Equal(expectedPolicy))
			},
			Entry(nil, ReconcilePolicyOnObjectChange, "", ReconcilePolicyOnObjectChange),
			Entry(nil, ReconcilePolicyOnObjectChange, types.ReconcilePolicyOnObjectChange, ReconcilePolicyOnObjectChange),
			Entry(nil, ReconcilePolicyOnObjectChange, types.ReconcilePolicyOnObjectOrComponentChange, ReconcilePolicyOnObjectOrComponentChange),
			Entry(nil, ReconcilePolicyOnObjectChange, types.ReconcilePolicyOnce, ReconcilePolicyOnce),

			Entry(nil, ReconcilePolicyOnObjectOrComponentChange, "", ReconcilePolicyOnObjectOrComponentChange),
			Entry(nil, ReconcilePolicyOnObjectOrComponentChange, types.ReconcilePolicyOnObjectChange, ReconcilePolicyOnObjectChange),
			Entry(nil, ReconcilePolicyOnObjectOrComponentChange, types.ReconcilePolicyOnObjectOrComponentChange, ReconcilePolicyOnObjectOrComponentChange),
			Entry(nil, ReconcilePolicyOnObjectOrComponentChange, types.ReconcilePolicyOnce, ReconcilePolicyOnce),

			Entry(nil, ReconcilePolicyOnce, "", ReconcilePolicyOnce),
			Entry(nil, ReconcilePolicyOnce, types.ReconcilePolicyOnObjectChange, ReconcilePolicyOnObjectChange),
			Entry(nil, ReconcilePolicyOnce, types.ReconcilePolicyOnObjectOrComponentChange, ReconcilePolicyOnObjectOrComponentChange),
			Entry(nil, ReconcilePolicyOnce, types.ReconcilePolicyOnce, ReconcilePolicyOnce),
		)

		DescribeTable("testing: getUpdatePolicy()",
			func(targetPolicy UpdatePolicy, annotatedPolicy string, expectedPolicy UpdatePolicy) {
				target := newReconcileTarget[Component](reconcilerName, reconcilerId, nil, nil, false, "", targetPolicy)
				object := &metav1.PartialObjectMetadata{}
				if annotatedPolicy != "" {
					object.Annotations = map[string]string{reconcilerName + "/" + types.AnnotationKeySuffixUpdatePolicy: annotatedPolicy}
				}
				resultingPolicy, err := target.getUpdatePolicy(object)
				Expect(err).NotTo(HaveOccurred())
				Expect(resultingPolicy).To(Equal(expectedPolicy))
			},
			Entry(nil, UpdatePolicyRecreate, "", UpdatePolicyRecreate),
			Entry(nil, UpdatePolicyRecreate, types.UpdatePolicyDefault, UpdatePolicyRecreate),
			Entry(nil, UpdatePolicyRecreate, types.UpdatePolicyRecreate, UpdatePolicyRecreate),
			Entry(nil, UpdatePolicyRecreate, types.UpdatePolicyReplace, UpdatePolicyReplace),
			Entry(nil, UpdatePolicyRecreate, types.UpdatePolicySsaMerge, UpdatePolicySsaMerge),
			Entry(nil, UpdatePolicyRecreate, types.UpdatePolicySsaOverride, UpdatePolicySsaOverride),

			Entry(nil, UpdatePolicyReplace, "", UpdatePolicyReplace),
			Entry(nil, UpdatePolicyReplace, types.UpdatePolicyDefault, UpdatePolicyReplace),
			Entry(nil, UpdatePolicyReplace, types.UpdatePolicyRecreate, UpdatePolicyRecreate),
			Entry(nil, UpdatePolicyReplace, types.UpdatePolicyReplace, UpdatePolicyReplace),
			Entry(nil, UpdatePolicyReplace, types.UpdatePolicySsaMerge, UpdatePolicySsaMerge),
			Entry(nil, UpdatePolicyReplace, types.UpdatePolicySsaOverride, UpdatePolicySsaOverride),

			Entry(nil, UpdatePolicySsaMerge, "", UpdatePolicySsaMerge),
			Entry(nil, UpdatePolicySsaMerge, types.UpdatePolicyDefault, UpdatePolicySsaMerge),
			Entry(nil, UpdatePolicySsaMerge, types.UpdatePolicyRecreate, UpdatePolicyRecreate),
			Entry(nil, UpdatePolicySsaMerge, types.UpdatePolicyReplace, UpdatePolicyReplace),
			Entry(nil, UpdatePolicySsaMerge, types.UpdatePolicySsaMerge, UpdatePolicySsaMerge),
			Entry(nil, UpdatePolicySsaMerge, types.UpdatePolicySsaOverride, UpdatePolicySsaOverride),

			Entry(nil, UpdatePolicySsaOverride, "", UpdatePolicySsaOverride),
			Entry(nil, UpdatePolicySsaOverride, types.UpdatePolicyDefault, UpdatePolicySsaOverride),
			Entry(nil, UpdatePolicySsaOverride, types.UpdatePolicyRecreate, UpdatePolicyRecreate),
			Entry(nil, UpdatePolicySsaOverride, types.UpdatePolicyReplace, UpdatePolicyReplace),
			Entry(nil, UpdatePolicySsaOverride, types.UpdatePolicySsaMerge, UpdatePolicySsaMerge),
			Entry(nil, UpdatePolicySsaOverride, types.UpdatePolicySsaOverride, UpdatePolicySsaOverride),
		)

		DescribeTable("testing: getDeletePolicy()",
			func(targetPolicy DeletePolicy, annotatedPolicy string, expectedPolicy DeletePolicy) {
				target := newReconcileTarget[Component](reconcilerName, reconcilerId, nil, nil, false, "", "")
				target.deletePolicy = targetPolicy
				object := &metav1.PartialObjectMetadata{}
				if annotatedPolicy != "" {
					object.Annotations = map[string]string{reconcilerName + "/" + types.AnnotationKeySuffixDeletePolicy: annotatedPolicy}
				}
				resultingPolicy, err := target.getDeletePolicy(object)
				Expect(err).NotTo(HaveOccurred())
				Expect(resultingPolicy).To(Equal(expectedPolicy))
			},
			Entry(nil, DeletePolicyDelete, "", DeletePolicyDelete),
			Entry(nil, DeletePolicyDelete, types.DeletePolicyDefault, DeletePolicyDelete),
			Entry(nil, DeletePolicyDelete, types.DeletePolicyDelete, DeletePolicyDelete),
			Entry(nil, DeletePolicyDelete, types.DeletePolicyOrphan, DeletePolicyOrphan),

			Entry(nil, DeletePolicyOrphan, "", DeletePolicyOrphan),
			Entry(nil, DeletePolicyOrphan, types.DeletePolicyDefault, DeletePolicyOrphan),
			Entry(nil, DeletePolicyOrphan, types.DeletePolicyDelete, DeletePolicyDelete),
			Entry(nil, DeletePolicyOrphan, types.DeletePolicyOrphan, DeletePolicyOrphan),
		)
	})

	Context("testing: get*Order()", func() {
		var target *reconcileTarget[Component]

		BeforeEach(func() {
			target = newReconcileTarget[Component](reconcilerName, reconcilerId, nil, nil, false, "", "")
		})

		DescribeTable("testing: getOrder()",
			func(annotatedOrder string, expectError bool, expectedOrder int) {
				object := &metav1.PartialObjectMetadata{}
				if annotatedOrder != "" {
					object.Annotations = map[string]string{reconcilerName + "/" + types.AnnotationKeySuffixOrder: annotatedOrder}
				}
				order, err := target.getOrder(object)
				if expectError {
					Expect(err).To(HaveOccurred())
				} else {
					Expect(err).NotTo(HaveOccurred())
					Expect(order).To(Equal(expectedOrder))
				}
			},
			Entry(nil, "", false, 0),
			Entry(nil, "0", false, 0),
			Entry(nil, "-1", false, -1),
			Entry(nil, "1", false, 1),
			Entry(nil, fmt.Sprintf("%d", math.MinInt16), false, math.MinInt16),
			Entry(nil, fmt.Sprintf("%d", math.MaxInt16), false, math.MaxInt16),
			Entry(nil, fmt.Sprintf("%d", math.MinInt16-1), true, 0),
			Entry(nil, fmt.Sprintf("%d", math.MaxInt16+1), true, 0),
			Entry(nil, "not-a-number", true, 0),
		)

		DescribeTable("testing: getPurgeOrder()",
			func(annotatedOrder string, expectError bool, expectedOrder int) {
				object := &metav1.PartialObjectMetadata{}
				if annotatedOrder != "" {
					object.Annotations = map[string]string{reconcilerName + "/" + types.AnnotationKeySuffixPurgeOrder: annotatedOrder}
				}
				order, err := target.getPurgeOrder(object)
				if expectError {
					Expect(err).To(HaveOccurred())
				} else {
					Expect(err).NotTo(HaveOccurred())
					Expect(order).To(Equal(expectedOrder))
				}
			},
			Entry(nil, "", false, math.MaxInt),
			Entry(nil, "0", false, 0),
			Entry(nil, "-1", false, -1),
			Entry(nil, "1", false, 1),
			Entry(nil, fmt.Sprintf("%d", math.MinInt16), false, math.MinInt16),
			Entry(nil, fmt.Sprintf("%d", math.MaxInt16), false, math.MaxInt16),
			Entry(nil, fmt.Sprintf("%d", math.MinInt16-1), true, 0),
			Entry(nil, fmt.Sprintf("%d", math.MaxInt16+1), true, 0),
			Entry(nil, "not-a-number", true, 0),
		)
	})

	Context("testing: is*Used()", func() {
		var target *reconcileTarget[Component]
		var namespace string

		BeforeEach(func() {
			target = newReconcileTarget[Component](reconcilerName, reconcilerId, cli, nil, false, "", "")
			namespace = CreateNamespace()
		})

		AfterEach(func() {
			CleanupNamespace(namespace, &foov1alpha1.Foo{})
		})

		DescribeTable("testing: isCrdUsed()",
			// TODO: remove the below workaround logic (needed until we are sure that all owner id labels have the new format)
			func(createUnmanagedInstance bool, createManagedInstance bool, onlyForeign bool, crdHasLegacyOwnerId bool, managedInstanceHasLegacyOwnerId bool) {
				crd := &apiextensionsv1.CustomResourceDefinition{
					Spec: apiextensionsv1.CustomResourceDefinitionSpec{
						Group:    foov1alpha1.GroupVersion.Group,
						Versions: []apiextensionsv1.CustomResourceDefinitionVersion{{Name: foov1alpha1.GroupVersion.Version}},
						Names:    apiextensionsv1.CustomResourceDefinitionNames{Kind: "Foo"},
					},
				}
				if crdHasLegacyOwnerId {
					crd.Labels = map[string]string{reconcilerName + "/" + types.LabelKeySuffixOwnerId: "x_y"}
					crd.Annotations = map[string]string{reconcilerName + "/" + types.AnnotationKeySuffixOwnerId: "x/y"}
				} else {
					crd.Labels = map[string]string{reconcilerName + "/" + types.LabelKeySuffixOwnerId: sha256base32([]byte(reconcilerId + "/x/y"))}
					crd.Annotations = map[string]string{reconcilerName + "/" + types.AnnotationKeySuffixOwnerId: reconcilerId + "/x/y"}
				}
				if createUnmanagedInstance {
					foo := &foov1alpha1.Foo{}
					foo.Namespace = namespace
					foo.GenerateName = "test-"
					CreateObject(foo, "")
				}
				if createManagedInstance {
					foo := &foov1alpha1.Foo{}
					foo.Namespace = namespace
					foo.GenerateName = "test-"
					if managedInstanceHasLegacyOwnerId {
						foo.Labels = map[string]string{reconcilerName + "/" + types.LabelKeySuffixOwnerId: "x_y"}
					} else {
						foo.Labels = map[string]string{reconcilerName + "/" + types.LabelKeySuffixOwnerId: sha256base32([]byte(reconcilerId + "/x/y"))}
					}
					CreateObject(foo, "")
				}
				used, err := target.isCrdUsed(ctx, crd, onlyForeign)
				Expect(err).NotTo(HaveOccurred())
				Expect(used).To(Equal(createUnmanagedInstance || createManagedInstance && !onlyForeign))
			},
			Entry(nil, false, false, false, false, false),
			Entry(nil, false, true, false, false, false),
			Entry(nil, true, false, false, false, false),
			Entry(nil, true, true, false, false, false),
			Entry(nil, false, false, true, false, false),
			Entry(nil, false, true, true, false, false),
			Entry(nil, true, false, true, false, false),
			Entry(nil, true, true, true, false, false),

			Entry(nil, false, false, false, true, false),
			Entry(nil, false, true, false, true, false),
			Entry(nil, true, false, false, true, false),
			Entry(nil, true, true, false, true, false),
			Entry(nil, false, false, true, true, false),
			Entry(nil, false, true, true, true, false),
			Entry(nil, true, false, true, true, false),
			Entry(nil, true, true, true, true, false),

			Entry(nil, false, false, false, false, true),
			Entry(nil, false, true, false, false, true),
			Entry(nil, true, false, false, false, true),
			Entry(nil, true, true, false, false, true),
			Entry(nil, false, false, true, false, true),
			Entry(nil, false, true, true, false, true),
			Entry(nil, true, false, true, false, true),
			Entry(nil, true, true, true, false, true),

			Entry(nil, false, false, false, true, true),
			Entry(nil, false, true, false, true, true),
			Entry(nil, true, false, false, true, true),
			Entry(nil, true, true, false, true, true),
			Entry(nil, false, false, true, true, true),
			Entry(nil, false, true, true, true, true),
			Entry(nil, true, false, true, true, true),
			Entry(nil, true, true, true, true, true),
		)

		DescribeTable("testing: isApiServiceUsed()",
			// TODO: remove the below workaround logic (needed until we are sure that all owner id labels have the new format)
			func(createUnmanagedInstance bool, createManagedInstance bool, onlyForeign bool, apiServiceHasLegacyOwnerId bool, managedInstanceHasLegacyOwnerId bool) {
				apiService := &apiregistrationv1.APIService{
					Spec: apiregistrationv1.APIServiceSpec{
						Group:   foov1alpha1.GroupVersion.Group,
						Version: foov1alpha1.GroupVersion.Version,
					},
				}
				if apiServiceHasLegacyOwnerId {
					apiService.Labels = map[string]string{reconcilerName + "/" + types.LabelKeySuffixOwnerId: "x_y"}
					apiService.Annotations = map[string]string{reconcilerName + "/" + types.AnnotationKeySuffixOwnerId: "x/y"}
				} else {
					apiService.Labels = map[string]string{reconcilerName + "/" + types.LabelKeySuffixOwnerId: sha256base32([]byte(reconcilerId + "/x/y"))}
					apiService.Annotations = map[string]string{reconcilerName + "/" + types.AnnotationKeySuffixOwnerId: reconcilerId + "/x/y"}
				}
				if createUnmanagedInstance {
					foo := &foov1alpha1.Foo{}
					foo.Namespace = namespace
					foo.GenerateName = "test-"
					CreateObject(foo, "")
				}
				if createManagedInstance {
					foo := &foov1alpha1.Foo{}
					foo.Namespace = namespace
					foo.GenerateName = "test-"
					if managedInstanceHasLegacyOwnerId {
						foo.Labels = map[string]string{reconcilerName + "/" + types.LabelKeySuffixOwnerId: "x_y"}
					} else {
						foo.Labels = map[string]string{reconcilerName + "/" + types.LabelKeySuffixOwnerId: sha256base32([]byte(reconcilerId + "/x/y"))}
					}
					CreateObject(foo, "")
				}
				used, err := target.isApiServiceUsed(ctx, apiService, onlyForeign)
				Expect(err).NotTo(HaveOccurred())
				Expect(used).To(Equal(createUnmanagedInstance || createManagedInstance && !onlyForeign))
			},
			Entry(nil, false, false, false, false, false),
			Entry(nil, false, true, false, false, false),
			Entry(nil, true, false, false, false, false),
			Entry(nil, true, true, false, false, false),
			Entry(nil, false, false, true, false, false),
			Entry(nil, false, true, true, false, false),
			Entry(nil, true, false, true, false, false),
			Entry(nil, true, true, true, false, false),

			Entry(nil, false, false, false, true, false),
			Entry(nil, false, true, false, true, false),
			Entry(nil, true, false, false, true, false),
			Entry(nil, true, true, false, true, false),
			Entry(nil, false, false, true, true, false),
			Entry(nil, false, true, true, true, false),
			Entry(nil, true, false, true, true, false),
			Entry(nil, true, true, true, true, false),

			Entry(nil, false, false, false, false, true),
			Entry(nil, false, true, false, false, true),
			Entry(nil, true, false, false, false, true),
			Entry(nil, true, true, false, false, true),
			Entry(nil, false, false, true, false, true),
			Entry(nil, false, true, true, false, true),
			Entry(nil, true, false, true, false, true),
			Entry(nil, true, true, true, false, true),

			Entry(nil, false, false, false, true, true),
			Entry(nil, false, true, false, true, true),
			Entry(nil, true, false, false, true, true),
			Entry(nil, true, true, false, true, true),
			Entry(nil, false, false, true, true, true),
			Entry(nil, false, true, true, true, true),
			Entry(nil, true, false, true, true, true),
			Entry(nil, true, true, true, true, true),
		)
	})
})
