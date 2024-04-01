/*
SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package reconciler

import (
	"context"
	"fmt"
	"math"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
	apiregistrationv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
	fieldvalue "sigs.k8s.io/structured-merge-diff/v4/value"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "github.com/sap/component-operator-runtime/internal/testing"

	"github.com/sap/component-operator-runtime/internal/cluster"
	barv1alpha1 "github.com/sap/component-operator-runtime/internal/testing/bar/api/v1alpha1"
	foov1alpha1 "github.com/sap/component-operator-runtime/internal/testing/foo/api/v1alpha1"
	"github.com/sap/component-operator-runtime/pkg/types"
)

var _ = Describe("testing: reconciler.go", func() {
	var reconcilerName string
	var ctx context.Context
	var cli cluster.Client

	BeforeEach(func() {
		reconcilerName = "reconciler.testing.local"
		ctx = Ctx()
		cli = Client()
	})

	Context("testing: readObject()", func() {
		var reconciler *Reconciler
		var namespace string
		var foo *foov1alpha1.Foo
		var bar *barv1alpha1.Bar

		BeforeEach(func() {
			reconciler = NewReconciler(reconcilerName, cli, ReconcilerOptions{})
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
			_foo, err := reconciler.readObject(ctx, foo)
			Expect(err).NotTo(HaveOccurred())
			Expect(_foo.GetUID()).To(Equal(foo.UID))
		})

		It("should successfully read an object whose type is not known to the scheme", func() {
			_bar, err := reconciler.readObject(ctx, bar)
			Expect(err).NotTo(HaveOccurred())
			Expect(_bar.GetUID()).To(Equal(bar.UID))
		})
	})

	Context("testing: createObject()", func() {
		var reconciler *Reconciler
		var namespace string
		var existingBar *barv1alpha1.Bar

		BeforeEach(func() {
			reconciler = NewReconciler(reconcilerName, cli, ReconcilerOptions{})
			namespace = CreateNamespace()

			existingBar = &barv1alpha1.Bar{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:    namespace,
					GenerateName: "test-",
				},
			}
			CreateObject(existingBar, "")
		})

		AfterEach(func() {
			CleanupNamespace(namespace, &foov1alpha1.Foo{}, &barv1alpha1.Bar{})
		})

		It("should successfully create an object whose type is known to the scheme", func() {
			foo := &foov1alpha1.Foo{
				TypeMeta:   metav1.TypeMeta{APIVersion: foov1alpha1.GroupVersion.String(), Kind: "Foo"},
				ObjectMeta: metav1.ObjectMeta{Namespace: namespace, GenerateName: "test-"},
			}
			err := reconciler.createObject(ctx, foo, foo, UpdatePolicyReplace)
			Expect(err).NotTo(HaveOccurred())
			_foo := ReadObject(foo)
			Expect(_foo.GetUID()).To(Equal(foo.UID))
		})

		It("should successfully create an object whose type is not known to the scheme", func() {
			bar := &barv1alpha1.Bar{
				TypeMeta:   metav1.TypeMeta{APIVersion: barv1alpha1.GroupVersion.String(), Kind: "Bar"},
				ObjectMeta: metav1.ObjectMeta{Namespace: namespace, GenerateName: "test-"},
			}
			err := reconciler.createObject(ctx, bar, bar, UpdatePolicyReplace)
			Expect(err).NotTo(HaveOccurred())
			_bar := ReadObject(bar)
			Expect(_bar.GetUID()).To(Equal(bar.UID))
		})

		It("should successfully create an unstructured object", func() {
			bar := AsUnstructured(&barv1alpha1.Bar{
				TypeMeta:   metav1.TypeMeta{APIVersion: barv1alpha1.GroupVersion.String(), Kind: "Bar"},
				ObjectMeta: metav1.ObjectMeta{Namespace: namespace, GenerateName: "test-"},
			})
			err := reconciler.createObject(ctx, bar, bar, UpdatePolicyReplace)
			Expect(err).NotTo(HaveOccurred())
			_bar := ReadObject(bar)
			Expect(_bar.GetUID()).To(Equal(bar.GetUID()))
		})

		It("should successfully create an object (using ssa-merge)", func() {
			bar := AsUnstructured(&barv1alpha1.Bar{
				TypeMeta:   metav1.TypeMeta{APIVersion: barv1alpha1.GroupVersion.String(), Kind: "Bar"},
				ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: fmt.Sprintf("test-%s", uuid.NewUUID())},
			})
			err := reconciler.createObject(ctx, bar, bar, UpdatePolicySsaMerge)
			Expect(err).NotTo(HaveOccurred())
			_bar := ReadObject(bar)
			Expect(_bar.GetUID()).To(Equal(bar.GetUID()))
		})

		It("should successfully create an object (using ssa-override)", func() {
			bar := AsUnstructured(&barv1alpha1.Bar{
				TypeMeta:   metav1.TypeMeta{APIVersion: barv1alpha1.GroupVersion.String(), Kind: "Bar"},
				ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: fmt.Sprintf("test-%s", uuid.NewUUID())},
			})
			err := reconciler.createObject(ctx, bar, bar, UpdatePolicySsaOverride)
			Expect(err).NotTo(HaveOccurred())
			_bar := ReadObject(bar)
			Expect(_bar.GetUID()).To(Equal(bar.GetUID()))
		})

		It("should reject the creation of an existing object", func() {
			bar := AsUnstructured(&barv1alpha1.Bar{
				TypeMeta:   metav1.TypeMeta{APIVersion: barv1alpha1.GroupVersion.String(), Kind: "Bar"},
				ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: existingBar.Name},
			})
			err := reconciler.createObject(ctx, bar, bar, UpdatePolicyReplace)
			if err == nil {
				Expect(err).To(HaveOccurred())
			}
			if !apierrors.IsAlreadyExists(err) {
				Expect(err).NotTo(HaveOccurred())
			}
			err = reconciler.createObject(ctx, bar, bar, UpdatePolicySsaMerge)
			if err == nil {
				Expect(err).To(HaveOccurred())
			}
			if !apierrors.IsConflict(err) {
				Expect(err).NotTo(HaveOccurred())
			}
			err = reconciler.createObject(ctx, bar, bar, UpdatePolicySsaOverride)
			if err == nil {
				Expect(err).To(HaveOccurred())
			}
			if !apierrors.IsConflict(err) {
				Expect(err).NotTo(HaveOccurred())
			}
		})

		// TODO: add a case validating that the finalizer is added for crds and api services
	})

	Context("testing: updateObject()", func() {
		var reconciler *Reconciler
		var namespace string
		var foo *foov1alpha1.Foo
		var bar *barv1alpha1.Bar
		var fooComplex *foov1alpha1.Foo

		BeforeEach(func() {
			reconciler = NewReconciler(reconcilerName, cli, ReconcilerOptions{})
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
			err := reconciler.updateObject(ctx, newFoo, AsUnstructured(foo), foo, UpdatePolicyReplace)
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
			err := reconciler.updateObject(ctx, newBar, AsUnstructured(bar), bar, UpdatePolicyReplace)
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
			err := reconciler.updateObject(ctx, newBar, AsUnstructured(bar), bar, UpdatePolicyReplace)
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
			err := reconciler.updateObject(ctx, newBar, AsUnstructured(bar), bar, UpdatePolicyReplace)
			if err == nil {
				Expect(err).To(HaveOccurred())
			}
			if !apierrors.IsConflict(err) {
				Expect(err).NotTo(HaveOccurred())
			}
			err = reconciler.updateObject(ctx, newBar, AsUnstructured(bar), bar, UpdatePolicySsaMerge)
			if err == nil {
				Expect(err).To(HaveOccurred())
			}
			if !apierrors.IsConflict(err) {
				Expect(err).NotTo(HaveOccurred())
			}
			err = reconciler.updateObject(ctx, newBar, AsUnstructured(bar), bar, UpdatePolicySsaOverride)
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
			err := reconciler.updateObject(ctx, newFooComplex, AsUnstructured(fooComplex), fooComplex, UpdatePolicyReplace)
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
			err := reconciler.updateObject(ctx, newFooComplex, AsUnstructured(fooComplex), fooComplex, UpdatePolicySsaMerge)
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
			err := reconciler.updateObject(ctx, newFooComplex, AsUnstructured(fooComplex), fooComplex, UpdatePolicySsaOverride)
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
		var reconciler *Reconciler
		var namespace string
		var foo *foov1alpha1.Foo

		BeforeEach(func() {
			reconciler = NewReconciler(reconcilerName, cli, ReconcilerOptions{})
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
			err := reconciler.deleteObject(ctx, foo, nil)
			Expect(err).NotTo(HaveOccurred())
			EnsureObjectDoesNotExist(foo)
		})

		It("should delete the object (with providing the existing object)", func() {
			err := reconciler.deleteObject(ctx, foo, AsUnstructured(foo))
			Expect(err).NotTo(HaveOccurred())
			EnsureObjectDoesNotExist(foo)
		})

		It("should reject a deletion with a wrong resourceVersion", func() {
			foo.ResourceVersion = "1"
			err := reconciler.deleteObject(ctx, foo, AsUnstructured(foo))
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
				reconciler := NewReconciler(reconcilerName, nil, ReconcilerOptions{AdoptionPolicy: &targetPolicy})
				object := &metav1.PartialObjectMetadata{}
				if annotatedPolicy != "" {
					object.Annotations = map[string]string{reconcilerName + "/" + types.AnnotationKeySuffixAdoptionPolicy: annotatedPolicy}
				}
				resultingPolicy, err := reconciler.getAdoptionPolicy(object)
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
				reconciler := NewReconciler(reconcilerName, nil, ReconcilerOptions{})
				reconciler.reconcilePolicy = targetPolicy
				object := &metav1.PartialObjectMetadata{}
				if annotatedPolicy != "" {
					object.Annotations = map[string]string{reconcilerName + "/" + types.AnnotationKeySuffixReconcilePolicy: annotatedPolicy}
				}
				resultingPolicy, err := reconciler.getReconcilePolicy(object)
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
				reconciler := NewReconciler(reconcilerName, nil, ReconcilerOptions{UpdatePolicy: &targetPolicy})
				object := &metav1.PartialObjectMetadata{}
				if annotatedPolicy != "" {
					object.Annotations = map[string]string{reconcilerName + "/" + types.AnnotationKeySuffixUpdatePolicy: annotatedPolicy}
				}
				resultingPolicy, err := reconciler.getUpdatePolicy(object)
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
				reconciler := NewReconciler(reconcilerName, nil, ReconcilerOptions{})
				reconciler.deletePolicy = targetPolicy
				object := &metav1.PartialObjectMetadata{}
				if annotatedPolicy != "" {
					object.Annotations = map[string]string{reconcilerName + "/" + types.AnnotationKeySuffixDeletePolicy: annotatedPolicy}
				}
				resultingPolicy, err := reconciler.getDeletePolicy(object)
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
		var reconciler *Reconciler

		BeforeEach(func() {
			reconciler = NewReconciler(reconcilerName, nil, ReconcilerOptions{})
		})

		DescribeTable("testing: getApplyOrder()",
			func(annotatedOrder string, expectError bool, expectedOrder int) {
				object := &metav1.PartialObjectMetadata{}
				if annotatedOrder != "" {
					object.Annotations = map[string]string{reconcilerName + "/" + types.AnnotationKeySuffixApplyOrder: annotatedOrder}
				}
				order, err := reconciler.getApplyOrder(object)
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
				order, err := reconciler.getPurgeOrder(object)
				if expectError {
					Expect(err).To(HaveOccurred())
				} else {
					Expect(err).NotTo(HaveOccurred())
					Expect(order).To(Equal(expectedOrder))
				}
			},
			Entry(nil, "", false, math.MaxInt16+1),
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
		var reconciler *Reconciler
		var namespace string
		var ownerId string

		BeforeEach(func() {
			reconciler = NewReconciler(reconcilerName, cli, ReconcilerOptions{})
			namespace = CreateNamespace()
			ownerId = "test-owner"
		})

		AfterEach(func() {
			CleanupNamespace(namespace, &foov1alpha1.Foo{})
		})

		DescribeTable("testing: isCrdUsed()",
			func(createUnmanagedInstance bool, createManagedInstance bool, onlyForeign bool) {
				crd := &apiextensionsv1.CustomResourceDefinition{
					Spec: apiextensionsv1.CustomResourceDefinitionSpec{
						Group:    foov1alpha1.GroupVersion.Group,
						Versions: []apiextensionsv1.CustomResourceDefinitionVersion{{Name: foov1alpha1.GroupVersion.Version}},
						Names:    apiextensionsv1.CustomResourceDefinitionNames{Kind: "Foo"},
					},
				}
				crd.Labels = map[string]string{reconcilerName + "/" + types.LabelKeySuffixOwnerId: sha256base32([]byte(ownerId))}
				crd.Annotations = map[string]string{reconcilerName + "/" + types.AnnotationKeySuffixOwnerId: ownerId}
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
					foo.Labels = map[string]string{reconcilerName + "/" + types.LabelKeySuffixOwnerId: sha256base32([]byte(ownerId))}
					CreateObject(foo, "")
				}
				used, err := reconciler.isCrdUsed(ctx, crd, onlyForeign)
				Expect(err).NotTo(HaveOccurred())
				Expect(used).To(Equal(createUnmanagedInstance || createManagedInstance && !onlyForeign))
			},
			Entry(nil, false, false, false),
			Entry(nil, false, true, false),
			Entry(nil, true, false, false),
			Entry(nil, true, true, false),
			Entry(nil, false, false, true),
			Entry(nil, false, true, true),
			Entry(nil, true, false, true),
			Entry(nil, true, true, true),
		)

		DescribeTable("testing: isApiServiceUsed()",
			func(createUnmanagedInstance bool, createManagedInstance bool, onlyForeign bool) {
				apiService := &apiregistrationv1.APIService{
					Spec: apiregistrationv1.APIServiceSpec{
						Group:   foov1alpha1.GroupVersion.Group,
						Version: foov1alpha1.GroupVersion.Version,
					},
				}
				apiService.Labels = map[string]string{reconcilerName + "/" + types.LabelKeySuffixOwnerId: sha256base32([]byte(ownerId))}
				apiService.Annotations = map[string]string{reconcilerName + "/" + types.AnnotationKeySuffixOwnerId: ownerId}
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
					foo.Labels = map[string]string{reconcilerName + "/" + types.LabelKeySuffixOwnerId: sha256base32([]byte(ownerId))}
					CreateObject(foo, "")
				}
				used, err := reconciler.isApiServiceUsed(ctx, apiService, onlyForeign)
				Expect(err).NotTo(HaveOccurred())
				Expect(used).To(Equal(createUnmanagedInstance || createManagedInstance && !onlyForeign))
			},
			Entry(nil, false, false, false),
			Entry(nil, false, true, false),
			Entry(nil, true, false, false),
			Entry(nil, true, true, false),
			Entry(nil, false, false, true),
			Entry(nil, false, true, true),
			Entry(nil, true, false, true),
			Entry(nil, true, true, true),
		)
	})
})
