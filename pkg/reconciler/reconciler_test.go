/*
SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package reconciler

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sap/go-generics/slices"

	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	apiregistrationsv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gcustom"

	"github.com/sap/component-operator-runtime/internal/clientfactory"
	"github.com/sap/component-operator-runtime/pkg/cluster"
	"github.com/sap/component-operator-runtime/pkg/status"
	"github.com/sap/component-operator-runtime/pkg/types"
	"github.com/sap/component-operator-runtime/testing/environment"
	cstestingv1alpha1 "github.com/sap/component-operator-runtime/testing/environment/apis/testing.cs.sap.com/v1alpha1"
)

const (
	reconcilerName = "reconciler.testing.cs.sap.com"
	fieldOwner     = "reconciler.testing.cs.sap.com"
	finalizer      = "reconciler.testing.cs.sap.com/finalizer"
)

var _ = Describe("testing: reconciler.go", func() {

	var scheme *runtime.Scheme
	var clnt cluster.Client
	var reconciler *Reconciler
	var namespace string
	var ownerId string
	var componentDigest string
	var objectsToCleanup []client.Object

	var getInventoryItemForObject = func(inventory []*InventoryItem, obj client.Object) *InventoryItem {
		obj, err := WithTypeInfo(obj, scheme)
		Expect(err).NotTo(HaveOccurred())
		item := getItem(inventory, obj)
		Expect(item).NotTo(BeNil())
		return item
	}
	var createInventoryItemForObject = func(inventory *[]*InventoryItem, obj client.Object, phase Phase, status status.Status) *InventoryItem {
		obj, err := WithTypeInfo(obj, scheme)
		Expect(err).NotTo(HaveOccurred())
		item := getItem(*inventory, obj)
		Expect(item).To(BeNil())
		item, err = CreateOrUpdateInventoryItemForObject(nil, reconciler, scheme, obj, componentDigest, phase, status)
		Expect(err).NotTo(HaveOccurred())
		*inventory = append(*inventory, item)
		return item
	}
	var updateInventoryItemForObject = func(inventory *[]*InventoryItem, obj client.Object, phase Phase, status status.Status) *InventoryItem {
		obj, err := WithTypeInfo(obj, scheme)
		Expect(err).NotTo(HaveOccurred())
		item := getItem(*inventory, obj)
		Expect(item).NotTo(BeNil())
		_, err = CreateOrUpdateInventoryItemForObject(item, reconciler, scheme, obj, componentDigest, phase, status)
		Expect(err).NotTo(HaveOccurred())
		return item
	}
	var removeInventoryItemForObject = func(inventory *[]*InventoryItem, obj client.Object) {
		obj, err := WithTypeInfo(obj, scheme)
		Expect(err).NotTo(HaveOccurred())
		item := getItem(*inventory, obj)
		Expect(item).NotTo(BeNil())
		*inventory = slices.Select(*inventory, func(i *InventoryItem) bool {
			return i != item
		})
	}

	BeforeEach(func() {
		var err error

		scheme = runtime.NewScheme()
		corev1.AddToScheme(scheme)
		apiextensionsv1.AddToScheme(scheme)
		apiregistrationsv1.AddToScheme(scheme)
		cstestingv1alpha1.AddToScheme(scheme)

		clnt, err = clientfactory.NewClientFor(env.Config(), scheme, reconcilerName)
		Expect(err).NotTo(HaveOccurred())

		reconciler = NewReconciler(reconcilerName, clnt, ReconcilerOptions{
			FieldOwner:              new(fieldOwner),
			Finalizer:               new(finalizer),
			AdoptionPolicy:          new(AdoptionPolicyIfUnowned),
			UpdatePolicy:            new(UpdatePolicySsaOverride),
			DeletePolicy:            new(DeletePolicyDelete),
			MissingNamespacesPolicy: new(MissingNamespacesPolicyCreate),
			AdditionalManagedTypes:  []TypeInfo{{Group: "testing.cs.sap.com", Kind: "Bar"}},
			ReapplyInterval:         new(9 * time.Minute),
			StatusAnalyzer:          nil,
			Metrics:                 ReconcilerMetrics{},
			EnableEvents:            new(false),
		})

		namespace, err = env.CreateNamespace()
		Expect(err).NotTo(HaveOccurred())

		ownerId = uuid.NewString()
		componentDigest = uuid.NewString()

		objectsToCleanup = nil
	})

	AfterEach(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		err := env.CleanupObjects(ctx, objectsToCleanup...)
		if err != nil {
			AbortSuite(fmt.Sprintf("failed to cleanup objects: %v", err))
		}
	})

	Describe("testing: Apply()", func() {

		It("should apply one object without status", func() {
			configMap := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "c",
					Namespace: namespace,
				},
			}

			objects := []client.Object{configMap}
			objectsToCleanup = objects

			actualInventory := make([]*InventoryItem, 0)
			expectedInventory := make([]*InventoryItem, 0)
			for _, obj := range objects {
				createInventoryItemForObject(&expectedInventory, obj, PhaseScheduledForApplication, status.InProgressStatus)
			}

			ok, err := reconciler.Apply(context.Background(), &actualInventory, objects, namespace, ownerId, componentDigest)
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeFalse())
			Expect(actualInventory).To(MatchInventory(expectedInventory))

			updateInventoryItemForObject(&expectedInventory, configMap, PhaseCreating, status.InProgressStatus)
			ok, err = reconciler.Apply(context.Background(), &actualInventory, objects, namespace, ownerId, componentDigest)
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeFalse())
			Expect(actualInventory).To(MatchInventory(expectedInventory))

			updateInventoryItemForObject(&expectedInventory, configMap, PhaseReady, status.CurrentStatus)
			ok, err = reconciler.Apply(context.Background(), &actualInventory, objects, namespace, ownerId, componentDigest)
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeTrue())
			Expect(actualInventory).To(MatchInventory(expectedInventory))

			_, err = env.EnsureObjectExists(configMap, reconcilerName, ownerId, getInventoryItemForObject(expectedInventory, configMap).Digest)
			Expect(err).NotTo(HaveOccurred())

			ok, err = reconciler.Apply(context.Background(), &actualInventory, objects, namespace, ownerId, componentDigest)
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeTrue())
			Expect(actualInventory).To(MatchInventory(expectedInventory))
		})

		It("should apply one object with status", func() {
			foo := &cstestingv1alpha1.Foo{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: namespace,
				},
			}

			objects := []client.Object{foo}
			objectsToCleanup = objects

			actualInventory := make([]*InventoryItem, 0)
			expectedInventory := make([]*InventoryItem, 0)
			for _, obj := range objects {
				createInventoryItemForObject(&expectedInventory, obj, PhaseScheduledForApplication, status.InProgressStatus)
			}

			ok, err := reconciler.Apply(context.Background(), &actualInventory, objects, namespace, ownerId, componentDigest)
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeFalse())
			Expect(actualInventory).To(MatchInventory(expectedInventory))

			updateInventoryItemForObject(&expectedInventory, foo, PhaseCreating, status.InProgressStatus)
			ok, err = reconciler.Apply(context.Background(), &actualInventory, objects, namespace, ownerId, componentDigest)
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeFalse())
			Expect(actualInventory).To(MatchInventory(expectedInventory))

			ok, err = reconciler.Apply(context.Background(), &actualInventory, objects, namespace, ownerId, componentDigest)
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeFalse())
			Expect(actualInventory).To(MatchInventory(expectedInventory))

			err = env.Observe(foo, metav1.ConditionTrue)
			Expect(err).NotTo(HaveOccurred())

			updateInventoryItemForObject(&expectedInventory, foo, PhaseReady, status.CurrentStatus)
			ok, err = reconciler.Apply(context.Background(), &actualInventory, objects, namespace, ownerId, componentDigest)
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeTrue())
			Expect(actualInventory).To(MatchInventory(expectedInventory))

			_, err = env.EnsureObjectExists(foo, reconcilerName, ownerId, getInventoryItemForObject(expectedInventory, foo).Digest)
			Expect(err).NotTo(HaveOccurred())

			ok, err = reconciler.Apply(context.Background(), &actualInventory, objects, namespace, ownerId, componentDigest)
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeTrue())
			Expect(actualInventory).To(MatchInventory(expectedInventory))
		})

		It("should apply multiple objects with apply orders and purge order", func() {
			configMap := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "c",
					Namespace: namespace,
					Annotations: map[string]string{
						fmt.Sprintf("%s/%s", reconcilerName, types.AnnotationKeySuffixApplyOrder): "0",
						fmt.Sprintf("%s/%s", reconcilerName, types.AnnotationKeySuffixPurgeOrder): "1",
					},
				},
			}
			foo := &cstestingv1alpha1.Foo{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: namespace,
					Annotations: map[string]string{
						fmt.Sprintf("%s/%s", reconcilerName, types.AnnotationKeySuffixApplyOrder): "1",
						fmt.Sprintf("%s/%s", reconcilerName, types.AnnotationKeySuffixPurgeOrder): "1",
					},
					Finalizers: []string{cstestingv1alpha1.FooFinalizer},
				},
			}
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "s",
					Namespace: namespace,
					Annotations: map[string]string{
						fmt.Sprintf("%s/%s", reconcilerName, types.AnnotationKeySuffixApplyOrder): "2",
					},
				},
			}

			objects := []client.Object{configMap, foo, secret}
			objectsToCleanup = objects

			actualInventory := make([]*InventoryItem, 0)
			expectedInventory := make([]*InventoryItem, 0)
			for _, obj := range objects {
				createInventoryItemForObject(&expectedInventory, obj, PhaseScheduledForApplication, status.InProgressStatus)
			}

			ok, err := reconciler.Apply(context.Background(), &actualInventory, objects, namespace, ownerId, componentDigest)
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeFalse())
			Expect(actualInventory).To(MatchInventory(expectedInventory))

			updateInventoryItemForObject(&expectedInventory, configMap, PhaseCreating, status.InProgressStatus)
			ok, err = reconciler.Apply(context.Background(), &actualInventory, objects, namespace, ownerId, componentDigest)
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeFalse())
			Expect(actualInventory).To(MatchInventory(expectedInventory))

			updateInventoryItemForObject(&expectedInventory, configMap, PhaseReady, status.CurrentStatus)
			updateInventoryItemForObject(&expectedInventory, foo, PhaseCreating, status.InProgressStatus)
			ok, err = reconciler.Apply(context.Background(), &actualInventory, objects, namespace, ownerId, componentDigest)
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeFalse())
			Expect(actualInventory).To(MatchInventory(expectedInventory))

			_, err = env.EnsureObjectExists(configMap, reconcilerName, ownerId, getInventoryItemForObject(expectedInventory, configMap).Digest)
			Expect(err).NotTo(HaveOccurred())

			ok, err = reconciler.Apply(context.Background(), &actualInventory, objects, namespace, ownerId, componentDigest)
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeFalse())
			Expect(actualInventory).To(MatchInventory(expectedInventory))

			err = env.Observe(foo, metav1.ConditionTrue)
			Expect(err).NotTo(HaveOccurred())

			updateInventoryItemForObject(&expectedInventory, configMap, PhaseScheduledForCompletion, status.CurrentStatus)
			updateInventoryItemForObject(&expectedInventory, foo, PhaseScheduledForCompletion, status.CurrentStatus)
			ok, err = reconciler.Apply(context.Background(), &actualInventory, objects, namespace, ownerId, componentDigest)
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeFalse())
			Expect(actualInventory).To(MatchInventory(expectedInventory))

			_, err = env.EnsureObjectExists(foo, reconcilerName, ownerId, getInventoryItemForObject(expectedInventory, foo).Digest)
			Expect(err).NotTo(HaveOccurred())

			updateInventoryItemForObject(&expectedInventory, configMap, PhaseCompleting, status.TerminatingStatus)
			updateInventoryItemForObject(&expectedInventory, foo, PhaseCompleting, status.TerminatingStatus)
			ok, err = reconciler.Apply(context.Background(), &actualInventory, objects, namespace, ownerId, componentDigest)
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeFalse())
			Expect(actualInventory).To(MatchInventory(expectedInventory))

			updateInventoryItemForObject(&expectedInventory, configMap, PhaseCompleted, "")
			ok, err = reconciler.Apply(context.Background(), &actualInventory, objects, namespace, ownerId, componentDigest)
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeFalse())
			Expect(actualInventory).To(MatchInventory(expectedInventory))

			err = env.EnsureObjectDoesNotExist(configMap)
			Expect(err).NotTo(HaveOccurred())

			err = env.Finalize(foo, cstestingv1alpha1.FooFinalizer)
			Expect(err).NotTo(HaveOccurred())

			updateInventoryItemForObject(&expectedInventory, foo, PhaseCompleted, "")
			updateInventoryItemForObject(&expectedInventory, secret, PhaseCreating, status.InProgressStatus)
			ok, err = reconciler.Apply(context.Background(), &actualInventory, objects, namespace, ownerId, componentDigest)
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeFalse())
			Expect(actualInventory).To(MatchInventory(expectedInventory))

			err = env.EnsureObjectDoesNotExist(foo)
			Expect(err).NotTo(HaveOccurred())

			updateInventoryItemForObject(&expectedInventory, secret, PhaseReady, status.CurrentStatus)
			ok, err = reconciler.Apply(context.Background(), &actualInventory, objects, namespace, ownerId, componentDigest)
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeTrue())
			Expect(actualInventory).To(MatchInventory(expectedInventory))

			_, err = env.EnsureObjectExists(secret, reconcilerName, ownerId, getInventoryItemForObject(expectedInventory, secret).Digest)
			Expect(err).NotTo(HaveOccurred())

			ok, err = reconciler.Apply(context.Background(), &actualInventory, objects, namespace, ownerId, componentDigest)
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeTrue())
			Expect(actualInventory).To(MatchInventory(expectedInventory))
		})

		It("should postpone the deployment of managed instances", func() {
			foo := &cstestingv1alpha1.Foo{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: namespace,
				},
			}
			bar := &cstestingv1alpha1.Bar{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "bar",
					Namespace: namespace,
				},
			}
			baz := &cstestingv1alpha1.Baz{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "baz",
					Namespace: namespace,
				},
			}
			crd := environment.CRDS["bazs.testing.cs.sap.com"]

			objects := []client.Object{foo, bar, baz, crd}
			objectsToCleanup = objects

			actualInventory := make([]*InventoryItem, 0)
			expectedInventory := make([]*InventoryItem, 0)

			for _, obj := range objects {
				createInventoryItemForObject(&expectedInventory, obj, PhaseScheduledForApplication, status.InProgressStatus)
			}

			ok, err := reconciler.Apply(context.Background(), &actualInventory, objects, namespace, ownerId, componentDigest)
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeFalse())
			Expect(actualInventory).To(MatchInventory(expectedInventory))

			updateInventoryItemForObject(&expectedInventory, foo, PhaseCreating, status.InProgressStatus)
			updateInventoryItemForObject(&expectedInventory, crd, PhaseCreating, status.InProgressStatus)
			ok, err = reconciler.Apply(context.Background(), &actualInventory, objects, namespace, ownerId, componentDigest)
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeFalse())
			Expect(actualInventory).To(MatchInventory(expectedInventory))

			updateInventoryItemForObject(&expectedInventory, crd, PhaseReady, status.CurrentStatus)
			ok, err = reconciler.Apply(context.Background(), &actualInventory, objects, namespace, ownerId, componentDigest)
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeFalse())
			Expect(actualInventory).To(MatchInventory(expectedInventory))

			err = env.Observe(foo, metav1.ConditionTrue)
			Expect(err).NotTo(HaveOccurred())

			updateInventoryItemForObject(&expectedInventory, foo, PhaseReady, status.CurrentStatus)
			ok, err = reconciler.Apply(context.Background(), &actualInventory, objects, namespace, ownerId, componentDigest)
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeFalse())
			Expect(actualInventory).To(MatchInventory(expectedInventory))

			_, err = env.EnsureObjectExists(foo, reconcilerName, ownerId, getInventoryItemForObject(expectedInventory, foo).Digest)
			Expect(err).NotTo(HaveOccurred())

			updateInventoryItemForObject(&expectedInventory, bar, PhaseCreating, status.InProgressStatus)
			updateInventoryItemForObject(&expectedInventory, baz, PhaseCreating, status.InProgressStatus)
			ok, err = reconciler.Apply(context.Background(), &actualInventory, objects, namespace, ownerId, componentDigest)
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeFalse())
			Expect(actualInventory).To(MatchInventory(expectedInventory))

			err = env.Observe(bar, metav1.ConditionTrue)
			Expect(err).NotTo(HaveOccurred())
			err = env.Observe(baz, metav1.ConditionTrue)
			Expect(err).NotTo(HaveOccurred())

			updateInventoryItemForObject(&expectedInventory, bar, PhaseReady, status.CurrentStatus)
			updateInventoryItemForObject(&expectedInventory, baz, PhaseReady, status.CurrentStatus)
			ok, err = reconciler.Apply(context.Background(), &actualInventory, objects, namespace, ownerId, componentDigest)
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeTrue())
			Expect(actualInventory).To(MatchInventory(expectedInventory))

			_, err = env.EnsureObjectExists(bar, reconcilerName, ownerId, getInventoryItemForObject(expectedInventory, bar).Digest)
			Expect(err).NotTo(HaveOccurred())
			_, err = env.EnsureObjectExists(baz, reconcilerName, ownerId, getInventoryItemForObject(expectedInventory, baz).Digest)
			Expect(err).NotTo(HaveOccurred())

			ok, err = reconciler.Apply(context.Background(), &actualInventory, objects, namespace, ownerId, componentDigest)
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeTrue())
			Expect(actualInventory).To(MatchInventory(expectedInventory))
		})

		It("should update objects", func() {
			foo1 := &cstestingv1alpha1.Foo{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo1",
					Namespace: namespace,
				},
				Spec: cstestingv1alpha1.FooSpec{
					Value: "bar1",
				},
			}
			foo2 := &cstestingv1alpha1.Foo{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo2",
					Namespace: namespace,
				},
				Spec: cstestingv1alpha1.FooSpec{
					Value: "bar2",
				},
			}
			foo3 := &cstestingv1alpha1.Foo{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo3",
					Namespace: namespace,
				},
				Spec: cstestingv1alpha1.FooSpec{
					Value: "bar3",
				},
			}

			objects := []client.Object{foo1, foo2, foo3}
			objectsToCleanup = objects

			actualInventory := make([]*InventoryItem, 0)
			expectedInventory := make([]*InventoryItem, 0)
			for _, obj := range objects {
				createInventoryItemForObject(&expectedInventory, obj, PhaseScheduledForApplication, status.InProgressStatus)
			}

			ok, err := reconciler.Apply(context.Background(), &actualInventory, objects, namespace, ownerId, componentDigest)
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeFalse())
			Expect(actualInventory).To(MatchInventory(expectedInventory))

			updateInventoryItemForObject(&expectedInventory, foo1, PhaseCreating, status.InProgressStatus)
			updateInventoryItemForObject(&expectedInventory, foo2, PhaseCreating, status.InProgressStatus)
			updateInventoryItemForObject(&expectedInventory, foo3, PhaseCreating, status.InProgressStatus)
			ok, err = reconciler.Apply(context.Background(), &actualInventory, objects, namespace, ownerId, componentDigest)
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeFalse())
			Expect(actualInventory).To(MatchInventory(expectedInventory))

			err = env.Observe(foo1, metav1.ConditionTrue)
			Expect(err).NotTo(HaveOccurred())
			err = env.Observe(foo2, metav1.ConditionTrue)
			Expect(err).NotTo(HaveOccurred())

			updateInventoryItemForObject(&expectedInventory, foo1, PhaseReady, status.CurrentStatus)
			updateInventoryItemForObject(&expectedInventory, foo2, PhaseReady, status.CurrentStatus)
			ok, err = reconciler.Apply(context.Background(), &actualInventory, objects, namespace, ownerId, componentDigest)
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeFalse())
			Expect(actualInventory).To(MatchInventory(expectedInventory))

			obj, err := env.EnsureObjectExists(foo1, reconcilerName, ownerId, getInventoryItemForObject(expectedInventory, foo1).Digest)
			Expect(err).NotTo(HaveOccurred())
			Expect(obj.(*cstestingv1alpha1.Foo).Spec.Value).To(Equal("bar1"))
			obj, err = env.EnsureObjectExists(foo2, reconcilerName, ownerId, getInventoryItemForObject(expectedInventory, foo2).Digest)
			Expect(err).NotTo(HaveOccurred())
			Expect(obj.(*cstestingv1alpha1.Foo).Spec.Value).To(Equal("bar2"))
			obj, err = env.EnsureObjectExists(foo3, reconcilerName, ownerId, getInventoryItemForObject(expectedInventory, foo3).Digest)
			Expect(err).NotTo(HaveOccurred())
			Expect(obj.(*cstestingv1alpha1.Foo).Spec.Value).To(Equal("bar3"))

			foo2.Spec.Value = "baz2"
			foo3.Spec.Value = "baz3"

			updateInventoryItemForObject(&expectedInventory, foo2, PhaseUpdating, status.InProgressStatus)
			updateInventoryItemForObject(&expectedInventory, foo3, PhaseUpdating, status.InProgressStatus)
			ok, err = reconciler.Apply(context.Background(), &actualInventory, objects, namespace, ownerId, componentDigest)
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeFalse())
			Expect(actualInventory).To(MatchInventory(expectedInventory))

			err = env.Observe(foo2, metav1.ConditionTrue)
			Expect(err).NotTo(HaveOccurred())
			err = env.Observe(foo3, metav1.ConditionTrue)
			Expect(err).NotTo(HaveOccurred())

			updateInventoryItemForObject(&expectedInventory, foo2, PhaseReady, status.CurrentStatus)
			updateInventoryItemForObject(&expectedInventory, foo3, PhaseReady, status.CurrentStatus)
			ok, err = reconciler.Apply(context.Background(), &actualInventory, objects, namespace, ownerId, componentDigest)
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeTrue())
			Expect(actualInventory).To(MatchInventory(expectedInventory))

			obj, err = env.EnsureObjectExists(foo1, reconcilerName, ownerId, getInventoryItemForObject(expectedInventory, foo1).Digest)
			Expect(err).NotTo(HaveOccurred())
			Expect(obj.(*cstestingv1alpha1.Foo).Spec.Value).To(Equal("bar1"))
			obj, err = env.EnsureObjectExists(foo2, reconcilerName, ownerId, getInventoryItemForObject(expectedInventory, foo2).Digest)
			Expect(err).NotTo(HaveOccurred())
			Expect(obj.(*cstestingv1alpha1.Foo).Spec.Value).To(Equal("baz2"))
			obj, err = env.EnsureObjectExists(foo3, reconcilerName, ownerId, getInventoryItemForObject(expectedInventory, foo3).Digest)
			Expect(err).NotTo(HaveOccurred())
			Expect(obj.(*cstestingv1alpha1.Foo).Spec.Value).To(Equal("baz3"))

			ok, err = reconciler.Apply(context.Background(), &actualInventory, objects, namespace, ownerId, componentDigest)
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeTrue())
			Expect(actualInventory).To(MatchInventory(expectedInventory))
		})

		It("should recreate objects when update policy is: recreate", func() {
			configMap := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "c",
					Namespace: namespace,
					Annotations: map[string]string{
						fmt.Sprintf("%s/%s", reconcilerName, types.AnnotationKeySuffixUpdatePolicy): types.UpdatePolicyRecreate,
					},
				},
				Data: map[string]string{
					"foo": "bar",
				},
			}

			objects := []client.Object{configMap}
			objectsToCleanup = objects

			actualInventory := make([]*InventoryItem, 0)
			for i := range 100 {
				ok, err := reconciler.Apply(context.Background(), &actualInventory, objects, namespace, ownerId, componentDigest)
				Expect(err).NotTo(HaveOccurred())
				if ok {
					break
				}
				if i == 99 {
					Fail("object reconciliation did not complete after 100 iterations")
				}
			}

			obj, err := env.EnsureObjectExists(configMap, reconcilerName, ownerId, getInventoryItemForObject(actualInventory, configMap).Digest)
			Expect(err).NotTo(HaveOccurred())
			Expect(obj.(*corev1.ConfigMap).Data["foo"]).To(Equal("bar"))

			previousObj := obj.DeepCopyObject().(client.Object)
			for i := range 100 {
				ok, err := reconciler.Apply(context.Background(), &actualInventory, objects, namespace, ownerId, componentDigest)
				Expect(err).NotTo(HaveOccurred())
				if ok {
					break
				}
				if i == 99 {
					Fail("object reconciliation did not complete after 100 iterations")
				}
			}
			obj, err = env.EnsureObjectExists(configMap, reconcilerName, ownerId, getInventoryItemForObject(actualInventory, configMap).Digest)
			Expect(err).NotTo(HaveOccurred())
			Expect(obj.(*corev1.ConfigMap).Data["foo"]).To(Equal("bar"))
			Expect(obj.GetUID()).To(Equal(previousObj.GetUID()))

			configMap.Data["foo"] = "baz"

			previousObj = obj.DeepCopyObject().(client.Object)
			for i := range 100 {
				ok, err := reconciler.Apply(context.Background(), &actualInventory, objects, namespace, ownerId, componentDigest)
				Expect(err).NotTo(HaveOccurred())
				if ok {
					break
				}
				if i == 99 {
					Fail("object reconciliation did not complete after 100 iterations")
				}
			}
			obj, err = env.EnsureObjectExists(configMap, reconcilerName, ownerId, getInventoryItemForObject(actualInventory, configMap).Digest)
			Expect(err).NotTo(HaveOccurred())
			Expect(obj.(*corev1.ConfigMap).Data["foo"]).To(Equal("baz"))
			Expect(obj.GetUID()).NotTo(Equal(previousObj.GetUID()))
		})

		It("should prune fields from kubctl'ish field managers when update policy is: ssa-override", func() {
			configMap := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "c",
					Namespace: namespace,
				},
				Data: map[string]string{
					"foo": "bar",
				},
			}

			objects := []client.Object{configMap}
			objectsToCleanup = objects

			actualInventory := make([]*InventoryItem, 0)
			for i := range 100 {
				ok, err := reconciler.Apply(context.Background(), &actualInventory, objects, namespace, ownerId, componentDigest)
				Expect(err).NotTo(HaveOccurred())
				if ok {
					break
				}
				if i == 99 {
					Fail("object reconciliation did not complete after 100 iterations")
				}
			}

			obj, err := env.EnsureObjectExists(configMap, reconcilerName, ownerId, getInventoryItemForObject(actualInventory, configMap).Digest)
			Expect(err).NotTo(HaveOccurred())
			Expect(obj.(*corev1.ConfigMap).Data["foo"]).To(Equal("bar"))

			c := configMap.DeepCopy()
			c.APIVersion = "v1"
			c.Kind = "ConfigMap"
			c.Annotations = map[string]string{}
			c.Annotations["from-kubectl"] = "value"
			c.Data["from-kubectl"] = "value"
			err = env.ApplyObject(c, "kubectl")
			Expect(err).NotTo(HaveOccurred())

			c = configMap.DeepCopy()
			c.APIVersion = "v1"
			c.Kind = "ConfigMap"
			c.Annotations = map[string]string{}
			c.Annotations["from-kubectl-actor"] = "value"
			c.Data["from-kubectl-actor"] = "value"
			err = env.ApplyObject(c, "kubectl-actor")
			Expect(err).NotTo(HaveOccurred())

			c = configMap.DeepCopy()
			c.APIVersion = "v1"
			c.Kind = "ConfigMap"
			c.Annotations = map[string]string{}
			c.Annotations["from-other-actor"] = "value"
			c.Data["from-other-actor"] = "value"
			err = env.ApplyObject(c, "other-actor")
			Expect(err).NotTo(HaveOccurred())

			obj, err = env.EnsureObjectExists(configMap, reconcilerName, ownerId, getInventoryItemForObject(actualInventory, configMap).Digest)
			Expect(err).NotTo(HaveOccurred())
			Expect(obj.(*corev1.ConfigMap).Annotations).To(HaveKeyWithValue("from-kubectl", "value"))
			Expect(obj.(*corev1.ConfigMap).Annotations).To(HaveKeyWithValue("from-kubectl-actor", "value"))
			Expect(obj.(*corev1.ConfigMap).Annotations).To(HaveKeyWithValue("from-other-actor", "value"))
			Expect(obj.(*corev1.ConfigMap).Data).To(Equal(map[string]string{
				"foo":                "bar",
				"from-kubectl":       "value",
				"from-kubectl-actor": "value",
				"from-other-actor":   "value",
			}))

			getInventoryItemForObject(actualInventory, configMap).LastAppliedAt = nil
			for i := range 100 {
				ok, err := reconciler.Apply(context.Background(), &actualInventory, objects, namespace, ownerId, componentDigest)
				Expect(err).NotTo(HaveOccurred())
				if ok {
					break
				}
				if i == 99 {
					Fail("object reconciliation did not complete after 100 iterations")
				}
			}

			obj, err = env.EnsureObjectExists(configMap, reconcilerName, ownerId, getInventoryItemForObject(actualInventory, configMap).Digest)
			Expect(err).NotTo(HaveOccurred())
			Expect(obj.(*corev1.ConfigMap).Annotations).NotTo(HaveKey("from-kubectl"))
			Expect(obj.(*corev1.ConfigMap).Annotations).NotTo(HaveKey("from-kubectl-actor"))
			Expect(obj.(*corev1.ConfigMap).Annotations).To(HaveKeyWithValue("from-other-actor", "value"))
			Expect(obj.(*corev1.ConfigMap).Data).To(Equal(map[string]string{
				"foo":              "bar",
				"from-other-actor": "value",
			}))
		})

		It("should not update objects with reconcile policy: once", func() {
			configMap := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "c",
					Namespace: namespace,
					Annotations: map[string]string{
						fmt.Sprintf("%s/%s", reconcilerName, types.AnnotationKeySuffixReconcilePolicy): types.ReconcilePolicyOnce,
					},
				},
				Data: map[string]string{
					"foo": "bar",
				},
			}

			objects := []client.Object{configMap}
			objectsToCleanup = objects

			actualInventory := make([]*InventoryItem, 0)
			for i := range 100 {
				ok, err := reconciler.Apply(context.Background(), &actualInventory, objects, namespace, ownerId, componentDigest)
				Expect(err).NotTo(HaveOccurred())
				if ok {
					break
				}
				if i == 99 {
					Fail("object reconciliation did not complete after 100 iterations")
				}
			}

			obj, err := env.EnsureObjectExists(configMap, reconcilerName, ownerId, getInventoryItemForObject(actualInventory, configMap).Digest)
			Expect(err).NotTo(HaveOccurred())
			Expect(obj.(*corev1.ConfigMap).Data["foo"]).To(Equal("bar"))

			configMap.Data["foo"] = "baz"
			for i := range 100 {
				ok, err := reconciler.Apply(context.Background(), &actualInventory, objects, namespace, ownerId, componentDigest)
				Expect(err).NotTo(HaveOccurred())
				if ok {
					break
				}
				if i == 99 {
					Fail("object reconciliation did not complete after 100 iterations")
				}
			}

			obj, err = env.EnsureObjectExists(configMap, reconcilerName, ownerId, getInventoryItemForObject(actualInventory, configMap).Digest)
			Expect(err).NotTo(HaveOccurred())
			Expect(obj.(*corev1.ConfigMap).Data["foo"]).To(Equal("bar"))
		})

		It("should delete redundant objects with delete orders, some orphaned", func() {
			configMap1 := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "c1",
					Namespace: namespace,
					Annotations: map[string]string{
						fmt.Sprintf("%s/%s", reconcilerName, types.AnnotationKeySuffixDeletePolicy): types.DeletePolicyOrphan,
						fmt.Sprintf("%s/%s", reconcilerName, types.AnnotationKeySuffixDeleteOrder):  "2",
					},
				},
			}
			configMap2 := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "c2",
					Namespace: namespace,
					Annotations: map[string]string{
						fmt.Sprintf("%s/%s", reconcilerName, types.AnnotationKeySuffixDeletePolicy): types.DeletePolicyOrphanOnApply,
						fmt.Sprintf("%s/%s", reconcilerName, types.AnnotationKeySuffixDeleteOrder):  "2",
					},
				},
			}
			configMap3 := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "c3",
					Namespace: namespace,
					Annotations: map[string]string{
						fmt.Sprintf("%s/%s", reconcilerName, types.AnnotationKeySuffixDeletePolicy): types.DeletePolicyOrphanOnDelete,
						fmt.Sprintf("%s/%s", reconcilerName, types.AnnotationKeySuffixDeleteOrder):  "2",
					},
				},
			}
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "s",
					Namespace: namespace,
					Annotations: map[string]string{
						fmt.Sprintf("%s/%s", reconcilerName, types.AnnotationKeySuffixDeleteOrder): "1",
					},
				},
			}
			foo := &cstestingv1alpha1.Foo{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: namespace,
					Annotations: map[string]string{
						fmt.Sprintf("%s/%s", reconcilerName, types.AnnotationKeySuffixDeleteOrder): "0",
					},
					Finalizers: []string{cstestingv1alpha1.FooFinalizer},
				},
			}

			objects := []client.Object{configMap1, configMap2, configMap3, foo, secret}
			objectsToCleanup = objects

			actualInventory := make([]*InventoryItem, 0)
			for i := range 100 {
				ok, err := reconciler.Apply(context.Background(), &actualInventory, objects, namespace, ownerId, componentDigest)
				Expect(err).NotTo(HaveOccurred())
				if ok {
					break
				}
				if i == 10 {
					err := env.Observe(foo, metav1.ConditionTrue)
					Expect(err).NotTo(HaveOccurred())
				}
				if i == 99 {
					Fail("object reconciliation did not complete after 100 iterations")
				}
			}

			objects = []client.Object{secret}

			expectedInventory := slices.Collect(actualInventory, func(item *InventoryItem) *InventoryItem {
				return item.DeepCopy()
			})

			updateInventoryItemForObject(&expectedInventory, configMap1, PhaseScheduledForDeletion, status.TerminatingStatus).Digest = ""
			updateInventoryItemForObject(&expectedInventory, configMap2, PhaseScheduledForDeletion, status.TerminatingStatus).Digest = ""
			updateInventoryItemForObject(&expectedInventory, configMap3, PhaseScheduledForDeletion, status.TerminatingStatus).Digest = ""
			updateInventoryItemForObject(&expectedInventory, foo, PhaseDeleting, status.TerminatingStatus).Digest = ""
			ok, err := reconciler.Apply(context.Background(), &actualInventory, objects, namespace, ownerId, componentDigest)
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeFalse())
			Expect(actualInventory).To(MatchInventory(expectedInventory))

			ok, err = reconciler.Apply(context.Background(), &actualInventory, objects, namespace, ownerId, componentDigest)
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeFalse())
			Expect(actualInventory).To(MatchInventory(expectedInventory))

			err = env.Finalize(foo, cstestingv1alpha1.FooFinalizer)
			Expect(err).NotTo(HaveOccurred())

			removeInventoryItemForObject(&expectedInventory, configMap1)
			removeInventoryItemForObject(&expectedInventory, configMap2)
			removeInventoryItemForObject(&expectedInventory, foo)
			updateInventoryItemForObject(&expectedInventory, configMap3, PhaseDeleting, status.TerminatingStatus).Digest = ""
			ok, err = reconciler.Apply(context.Background(), &actualInventory, objects, namespace, ownerId, componentDigest)
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeFalse())
			Expect(actualInventory).To(MatchInventory(expectedInventory))

			err = env.EnsureObjectDoesNotExist(foo)
			Expect(err).NotTo(HaveOccurred())

			removeInventoryItemForObject(&expectedInventory, configMap3)
			ok, err = reconciler.Apply(context.Background(), &actualInventory, objects, namespace, ownerId, componentDigest)
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeTrue())
			Expect(actualInventory).To(MatchInventory(expectedInventory))

			err = env.EnsureObjectDoesNotExist(configMap3)
			Expect(err).NotTo(HaveOccurred())

			_, err = env.EnsureObjectExists(configMap1, reconcilerName, ownerId, "")
			Expect(err).NotTo(HaveOccurred())
			_, err = env.EnsureObjectExists(configMap2, reconcilerName, ownerId, "")
			Expect(err).NotTo(HaveOccurred())

			ok, err = reconciler.Apply(context.Background(), &actualInventory, objects, namespace, ownerId, componentDigest)
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeTrue())
			Expect(actualInventory).To(MatchInventory(expectedInventory))
		})

		It("should prepone the deployment of managed instances", func() {
			foo := &cstestingv1alpha1.Foo{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "foo",
					Namespace:  namespace,
					Finalizers: []string{cstestingv1alpha1.FooFinalizer},
				},
			}
			bar := &cstestingv1alpha1.Bar{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "bar",
					Namespace:  namespace,
					Finalizers: []string{cstestingv1alpha1.BarFinalizer},
				},
			}
			baz := &cstestingv1alpha1.Baz{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "baz",
					Namespace:  namespace,
					Finalizers: []string{cstestingv1alpha1.BazFinalizer},
				},
			}
			crd := environment.CRDS["bazs.testing.cs.sap.com"]

			objects := []client.Object{foo, bar, baz, crd}
			objectsToCleanup = objects

			actualInventory := make([]*InventoryItem, 0)
			for i := range 100 {
				ok, err := reconciler.Apply(context.Background(), &actualInventory, objects, namespace, ownerId, componentDigest)
				Expect(err).NotTo(HaveOccurred())
				if ok {
					break
				}
				if i == 10 {
					err := env.Observe(foo, metav1.ConditionTrue)
					Expect(err).NotTo(HaveOccurred())
				}
				if i == 20 {
					err = env.Observe(bar, metav1.ConditionTrue)
					Expect(err).NotTo(HaveOccurred())
					err = env.Observe(baz, metav1.ConditionTrue)
					Expect(err).NotTo(HaveOccurred())
				}
				if i == 99 {
					Fail("object reconciliation did not complete after 100 iterations")
				}
			}

			objects = []client.Object{}

			expectedInventory := slices.Collect(actualInventory, func(item *InventoryItem) *InventoryItem {
				return item.DeepCopy()
			})

			updateInventoryItemForObject(&expectedInventory, foo, PhaseScheduledForDeletion, status.TerminatingStatus).Digest = ""
			updateInventoryItemForObject(&expectedInventory, bar, PhaseDeleting, status.TerminatingStatus).Digest = ""
			updateInventoryItemForObject(&expectedInventory, baz, PhaseDeleting, status.TerminatingStatus).Digest = ""
			updateInventoryItemForObject(&expectedInventory, crd, PhaseScheduledForDeletion, status.TerminatingStatus).Digest = ""
			ok, err := reconciler.Apply(context.Background(), &actualInventory, objects, namespace, ownerId, componentDigest)
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeFalse())
			Expect(actualInventory).To(MatchInventory(expectedInventory))

			ok, err = reconciler.Apply(context.Background(), &actualInventory, objects, namespace, ownerId, componentDigest)
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeFalse())
			Expect(actualInventory).To(MatchInventory(expectedInventory))

			err = env.Finalize(bar, cstestingv1alpha1.BarFinalizer)
			Expect(err).NotTo(HaveOccurred())
			err = env.Finalize(baz, cstestingv1alpha1.BazFinalizer)
			Expect(err).NotTo(HaveOccurred())

			removeInventoryItemForObject(&expectedInventory, bar)
			removeInventoryItemForObject(&expectedInventory, baz)
			ok, err = reconciler.Apply(context.Background(), &actualInventory, objects, namespace, ownerId, componentDigest)
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeFalse())
			Expect(actualInventory).To(MatchInventory(expectedInventory))

			err = env.EnsureObjectDoesNotExist(bar)
			Expect(err).NotTo(HaveOccurred())
			err = env.EnsureObjectDoesNotExist(baz)
			Expect(err).NotTo(HaveOccurred())

			updateInventoryItemForObject(&expectedInventory, foo, PhaseDeleting, status.TerminatingStatus).Digest = ""
			updateInventoryItemForObject(&expectedInventory, crd, PhaseDeleting, status.TerminatingStatus).Digest = ""
			ok, err = reconciler.Apply(context.Background(), &actualInventory, objects, namespace, ownerId, componentDigest)
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeFalse())
			Expect(actualInventory).To(MatchInventory(expectedInventory))

			err = env.Finalize(foo, cstestingv1alpha1.FooFinalizer)
			Expect(err).NotTo(HaveOccurred())

			removeInventoryItemForObject(&expectedInventory, foo)
			removeInventoryItemForObject(&expectedInventory, crd)
			ok, err = reconciler.Apply(context.Background(), &actualInventory, objects, namespace, ownerId, componentDigest)
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeTrue())
			Expect(actualInventory).To(MatchInventory(expectedInventory))

			err = env.EnsureObjectDoesNotExist(foo)
			Expect(err).NotTo(HaveOccurred())
			err = env.EnsureObjectDoesNotExist(crd)
			Expect(err).NotTo(HaveOccurred())
		})

		Describe("should handle adoption policies correctly", func() {

			var configMap *corev1.ConfigMap
			var secret *corev1.Secret

			BeforeEach(func() {
				configMap = &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "c",
						Namespace: namespace,
					},
				}
				secret = &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "s",
						Namespace: namespace,
						Annotations: map[string]string{
							fmt.Sprintf("%s/%s", reconcilerName, types.AnnotationKeySuffixOwnerId): "other-owner",
						},
						Labels: map[string]string{
							fmt.Sprintf("%s/%s", reconcilerName, types.LabelKeySuffixOwnerId): "other-owner-digest",
						},
					},
				}

				objectsToCleanup = []client.Object{configMap, secret}

				err := env.CreateObject(configMap)
				Expect(err).NotTo(HaveOccurred())
				err = env.CreateObject(secret)
				Expect(err).NotTo(HaveOccurred())
			})

			It("should handle AdoptionPolicyIfUnowned correctly", func() {
				actualInventory := make([]*InventoryItem, 0)

				c := configMap.DeepCopy()
				objects := []client.Object{c}
				for i := range 100 {
					ok, err := reconciler.Apply(context.Background(), &actualInventory, objects, namespace, ownerId, componentDigest)
					Expect(err).NotTo(HaveOccurred())
					if ok {
						break
					}
					if i == 99 {
						Fail("object reconciliation did not complete after 100 iterations")
					}
				}

				s := secret.DeepCopy()
				objects = []client.Object{s}
				_, err := reconciler.Apply(context.Background(), &actualInventory, objects, namespace, ownerId, componentDigest)
				Expect(err).To(HaveOccurred())
			})

			It("should handle AdoptionPolicyNever correctly", func() {

				c := configMap.DeepCopy()
				c.Annotations = map[string]string{}
				c.Annotations[fmt.Sprintf("%s/%s", reconcilerName, types.AnnotationKeySuffixAdoptionPolicy)] = types.AdoptionPolicyNever
				objects := []client.Object{c}
				actualInventory := make([]*InventoryItem, 0)
				_, err := reconciler.Apply(context.Background(), &actualInventory, objects, namespace, ownerId, componentDigest)
				Expect(err).To(HaveOccurred())

				s := secret.DeepCopy()
				s.Annotations[fmt.Sprintf("%s/%s", reconcilerName, types.AnnotationKeySuffixAdoptionPolicy)] = types.AdoptionPolicyNever
				objects = []client.Object{s}
				actualInventory = make([]*InventoryItem, 0)
				_, err = reconciler.Apply(context.Background(), &actualInventory, objects, namespace, ownerId, componentDigest)
				Expect(err).To(HaveOccurred())
			})

			It("should handle AdoptionPolicyAlways correctly", func() {
				c := configMap.DeepCopy()
				c.Annotations = map[string]string{}
				c.Annotations[fmt.Sprintf("%s/%s", reconcilerName, types.AnnotationKeySuffixAdoptionPolicy)] = types.AdoptionPolicyAlways
				objects := []client.Object{c}
				actualInventory := make([]*InventoryItem, 0)
				for i := range 100 {
					ok, err := reconciler.Apply(context.Background(), &actualInventory, objects, namespace, ownerId, componentDigest)
					Expect(err).NotTo(HaveOccurred())
					if ok {
						break
					}
					if i == 99 {
						Fail("object reconciliation did not complete after 100 iterations")
					}
				}

				s := secret.DeepCopy()
				s.Annotations[fmt.Sprintf("%s/%s", reconcilerName, types.AnnotationKeySuffixAdoptionPolicy)] = types.AdoptionPolicyAlways
				objects = []client.Object{s}
				actualInventory = make([]*InventoryItem, 0)
				for i := range 100 {
					ok, err := reconciler.Apply(context.Background(), &actualInventory, objects, namespace, ownerId, componentDigest)
					Expect(err).NotTo(HaveOccurred())
					if ok {
						break
					}
					if i == 99 {
						Fail("object reconciliation did not complete after 100 iterations")
					}
				}
			})
		})

	})

	Describe("testing: Delete()", func() {

		It("should delete objects with delete orders, some orphaned", func() {
			configMap1 := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "c1",
					Namespace: namespace,
					Annotations: map[string]string{
						fmt.Sprintf("%s/%s", reconcilerName, types.AnnotationKeySuffixDeletePolicy): types.DeletePolicyOrphan,
						fmt.Sprintf("%s/%s", reconcilerName, types.AnnotationKeySuffixDeleteOrder):  "2",
					},
				},
			}
			configMap2 := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "c2",
					Namespace: namespace,
					Annotations: map[string]string{
						fmt.Sprintf("%s/%s", reconcilerName, types.AnnotationKeySuffixDeletePolicy): types.DeletePolicyOrphanOnApply,
						fmt.Sprintf("%s/%s", reconcilerName, types.AnnotationKeySuffixDeleteOrder):  "2",
					},
				},
			}
			configMap3 := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "c3",
					Namespace: namespace,
					Annotations: map[string]string{
						fmt.Sprintf("%s/%s", reconcilerName, types.AnnotationKeySuffixDeletePolicy): types.DeletePolicyOrphanOnDelete,
						fmt.Sprintf("%s/%s", reconcilerName, types.AnnotationKeySuffixDeleteOrder):  "2",
					},
				},
			}
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "s",
					Namespace: namespace,
					Annotations: map[string]string{
						fmt.Sprintf("%s/%s", reconcilerName, types.AnnotationKeySuffixDeleteOrder): "1",
					},
				},
			}
			foo := &cstestingv1alpha1.Foo{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: namespace,
					Annotations: map[string]string{
						fmt.Sprintf("%s/%s", reconcilerName, types.AnnotationKeySuffixDeleteOrder): "0",
					},
					Finalizers: []string{cstestingv1alpha1.FooFinalizer},
				},
			}

			objects := []client.Object{configMap1, configMap2, configMap3, foo, secret}
			objectsToCleanup = objects

			actualInventory := make([]*InventoryItem, 0)
			for i := range 100 {
				ok, err := reconciler.Apply(context.Background(), &actualInventory, objects, namespace, ownerId, componentDigest)
				Expect(err).NotTo(HaveOccurred())
				if ok {
					break
				}
				if i == 10 {
					err := env.Observe(foo, metav1.ConditionTrue)
					Expect(err).NotTo(HaveOccurred())
				}
				if i == 99 {
					Fail("object reconciliation did not complete after 100 iterations")
				}
			}

			expectedInventory := slices.Collect(actualInventory, func(item *InventoryItem) *InventoryItem {
				return item.DeepCopy()
			})

			updateInventoryItemForObject(&expectedInventory, foo, PhaseDeleting, status.TerminatingStatus)
			ok, err := reconciler.Delete(context.Background(), &actualInventory, ownerId)
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeFalse())
			Expect(actualInventory).To(MatchInventory(expectedInventory))

			ok, err = reconciler.Delete(context.Background(), &actualInventory, ownerId)
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeFalse())
			Expect(actualInventory).To(MatchInventory(expectedInventory))

			err = env.Finalize(foo, cstestingv1alpha1.FooFinalizer)
			Expect(err).NotTo(HaveOccurred())

			removeInventoryItemForObject(&expectedInventory, foo)
			updateInventoryItemForObject(&expectedInventory, secret, PhaseDeleting, status.TerminatingStatus)
			ok, err = reconciler.Delete(context.Background(), &actualInventory, ownerId)
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeFalse())
			Expect(actualInventory).To(MatchInventory(expectedInventory))

			err = env.EnsureObjectDoesNotExist(foo)
			Expect(err).NotTo(HaveOccurred())

			removeInventoryItemForObject(&expectedInventory, configMap1)
			removeInventoryItemForObject(&expectedInventory, configMap3)
			removeInventoryItemForObject(&expectedInventory, secret)
			updateInventoryItemForObject(&expectedInventory, configMap2, PhaseDeleting, status.TerminatingStatus)
			ok, err = reconciler.Delete(context.Background(), &actualInventory, ownerId)
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeFalse())
			Expect(actualInventory).To(MatchInventory(expectedInventory))

			err = env.EnsureObjectDoesNotExist(secret)
			Expect(err).NotTo(HaveOccurred())

			removeInventoryItemForObject(&expectedInventory, configMap2)
			ok, err = reconciler.Delete(context.Background(), &actualInventory, ownerId)
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeTrue())
			Expect(actualInventory).To(MatchInventory(expectedInventory))

			err = env.EnsureObjectDoesNotExist(configMap2)
			Expect(err).NotTo(HaveOccurred())

			_, err = env.EnsureObjectExists(configMap1, reconcilerName, ownerId, "")
			Expect(err).NotTo(HaveOccurred())
			_, err = env.EnsureObjectExists(configMap3, reconcilerName, ownerId, "")
			Expect(err).NotTo(HaveOccurred())
		})

		It("should prepone the deployment of managed instances", func() {
			foo := &cstestingv1alpha1.Foo{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "foo",
					Namespace:  namespace,
					Finalizers: []string{cstestingv1alpha1.FooFinalizer},
				},
			}
			bar := &cstestingv1alpha1.Bar{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "bar",
					Namespace:  namespace,
					Finalizers: []string{cstestingv1alpha1.BarFinalizer},
				},
			}
			baz := &cstestingv1alpha1.Baz{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "baz",
					Namespace:  namespace,
					Finalizers: []string{cstestingv1alpha1.BazFinalizer},
				},
			}
			crd := environment.CRDS["bazs.testing.cs.sap.com"]

			objects := []client.Object{foo, bar, baz, crd}
			objectsToCleanup = objects

			actualInventory := make([]*InventoryItem, 0)
			for i := range 100 {
				ok, err := reconciler.Apply(context.Background(), &actualInventory, objects, namespace, ownerId, componentDigest)
				Expect(err).NotTo(HaveOccurred())
				if ok {
					break
				}
				if i == 10 {
					err := env.Observe(foo, metav1.ConditionTrue)
					Expect(err).NotTo(HaveOccurred())
				}
				if i == 20 {
					err = env.Observe(bar, metav1.ConditionTrue)
					Expect(err).NotTo(HaveOccurred())
					err = env.Observe(baz, metav1.ConditionTrue)
					Expect(err).NotTo(HaveOccurred())
				}
				if i == 99 {
					Fail("object reconciliation did not complete after 100 iterations")
				}
			}

			expectedInventory := slices.Collect(actualInventory, func(item *InventoryItem) *InventoryItem {
				return item.DeepCopy()
			})

			updateInventoryItemForObject(&expectedInventory, bar, PhaseDeleting, status.TerminatingStatus)
			updateInventoryItemForObject(&expectedInventory, baz, PhaseDeleting, status.TerminatingStatus)
			ok, err := reconciler.Delete(context.Background(), &actualInventory, ownerId)
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeFalse())
			Expect(actualInventory).To(MatchInventory(expectedInventory))

			ok, err = reconciler.Delete(context.Background(), &actualInventory, ownerId)
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeFalse())
			Expect(actualInventory).To(MatchInventory(expectedInventory))

			err = env.Finalize(bar, cstestingv1alpha1.BarFinalizer)
			Expect(err).NotTo(HaveOccurred())
			err = env.Finalize(baz, cstestingv1alpha1.BazFinalizer)
			Expect(err).NotTo(HaveOccurred())

			removeInventoryItemForObject(&expectedInventory, bar)
			removeInventoryItemForObject(&expectedInventory, baz)
			ok, err = reconciler.Delete(context.Background(), &actualInventory, ownerId)
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeFalse())
			Expect(actualInventory).To(MatchInventory(expectedInventory))

			err = env.EnsureObjectDoesNotExist(bar)
			Expect(err).NotTo(HaveOccurred())
			err = env.EnsureObjectDoesNotExist(baz)
			Expect(err).NotTo(HaveOccurred())

			updateInventoryItemForObject(&expectedInventory, foo, PhaseDeleting, status.TerminatingStatus)
			updateInventoryItemForObject(&expectedInventory, crd, PhaseDeleting, status.TerminatingStatus)
			ok, err = reconciler.Delete(context.Background(), &actualInventory, ownerId)
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeFalse())
			Expect(actualInventory).To(MatchInventory(expectedInventory))

			err = env.Finalize(foo, cstestingv1alpha1.FooFinalizer)
			Expect(err).NotTo(HaveOccurred())

			removeInventoryItemForObject(&expectedInventory, foo)
			removeInventoryItemForObject(&expectedInventory, crd)
			ok, err = reconciler.Delete(context.Background(), &actualInventory, ownerId)
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeTrue())
			Expect(actualInventory).To(MatchInventory(expectedInventory))

			err = env.EnsureObjectDoesNotExist(foo)
			Expect(err).NotTo(HaveOccurred())
			err = env.EnsureObjectDoesNotExist(crd)
			Expect(err).NotTo(HaveOccurred())
		})

	})

	Describe("testing: IsDeletionAllowed()", func() {

		It("should allow deletion if there are no foreign instances of managed types", func() {
			baz := &cstestingv1alpha1.Baz{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "baz",
					Namespace: namespace,
				},
			}
			crd := environment.CRDS["bazs.testing.cs.sap.com"]

			objects := []client.Object{baz, crd}
			objectsToCleanup = objects

			actualInventory := make([]*InventoryItem, 0)
			for i := range 100 {
				ok, err := reconciler.Apply(context.Background(), &actualInventory, objects, namespace, ownerId, componentDigest)
				Expect(err).NotTo(HaveOccurred())
				if ok {
					break
				}
				if i == 10 {
					err = env.Observe(baz, metav1.ConditionTrue)
					Expect(err).NotTo(HaveOccurred())
				}
				if i == 99 {
					Fail("object reconciliation did not complete after 100 iterations")
				}
			}

			ok, _, err := reconciler.IsDeletionAllowed(context.Background(), &actualInventory, ownerId)
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeTrue())
		})

		It("should not allow deletion if there are foreign instances of managed types (1)", func() {
			actualInventory := make([]*InventoryItem, 0)

			bar := &cstestingv1alpha1.Bar{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "bar",
					Namespace: namespace,
				},
			}
			objectsToCleanup = []client.Object{bar}

			err := env.CreateObject(bar)
			Expect(err).NotTo(HaveOccurred())

			ok, _, err := reconciler.IsDeletionAllowed(context.Background(), &actualInventory, ownerId)
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeFalse())
		})

		It("should not allow deletion if there are foreign instances of managed types (2)", func() {
			crd := environment.CRDS["bazs.testing.cs.sap.com"]

			objects := []client.Object{crd}
			objectsToCleanup = objects

			actualInventory := make([]*InventoryItem, 0)
			for i := range 100 {
				ok, err := reconciler.Apply(context.Background(), &actualInventory, objects, namespace, ownerId, componentDigest)
				Expect(err).NotTo(HaveOccurred())
				if ok {
					break
				}
				if i == 99 {
					Fail("object reconciliation did not complete after 100 iterations")
				}
			}

			baz := &cstestingv1alpha1.Baz{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "baz",
					Namespace: namespace,
				},
			}
			objectsToCleanup = append(objectsToCleanup, baz)

			err := env.CreateObject(baz)
			Expect(err).NotTo(HaveOccurred())

			ok, _, err := reconciler.IsDeletionAllowed(context.Background(), &actualInventory, ownerId)
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeFalse())
		})

	})

	Describe("testing: getAdoptionPolicy()", func() {

		var obj *corev1.ConfigMap

		BeforeEach(func() {
			obj = &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "cm",
					Namespace:   namespace,
					Annotations: map[string]string{},
				},
			}
		})

		It("if the annotation is not present, it should return the default adoption policy defined at the reconciler", func() {
			p, err := reconciler.getAdoptionPolicy(obj)
			Expect(err).NotTo(HaveOccurred())
			Expect(p).To(Equal(AdoptionPolicyIfUnowned))
		})

		It("if the annotation is present and valid, it should return the adoption policy specified in the annotation", func() {
			for _, policy := range []string{types.AdoptionPolicyNever, types.AdoptionPolicyIfUnowned, types.AdoptionPolicyAlways} {
				obj.Annotations[fmt.Sprintf("%s/%s", reconcilerName, types.AnnotationKeySuffixAdoptionPolicy)] = policy
				p, err := reconciler.getAdoptionPolicy(obj)
				Expect(err).NotTo(HaveOccurred())
				Expect(p).To(Equal(adoptionPolicyByAnnotation[policy]))
			}

			// we intentionally use the code values (not the kebap case annotation values defined in package types), in order to
			// validate the conversion logic as well
			for _, policy := range []AdoptionPolicy{AdoptionPolicyNever, AdoptionPolicyIfUnowned, AdoptionPolicyAlways} {
				obj.Annotations[fmt.Sprintf("%s/%s", reconcilerName, types.AnnotationKeySuffixAdoptionPolicy)] = string(policy)
				p, err := reconciler.getAdoptionPolicy(obj)
				Expect(err).NotTo(HaveOccurred())
				Expect(p).To(Equal(policy))
			}
		})

		It("if the annotation is present but invalid, it should return an error", func() {
			obj.Annotations[fmt.Sprintf("%s/%s", reconcilerName, types.AnnotationKeySuffixAdoptionPolicy)] = "invalid"
			_, err := reconciler.getAdoptionPolicy(obj)
			Expect(err).To(HaveOccurred())
		})

	})

	Describe("testing: getReconcilePolicy()", func() {

		var obj *corev1.ConfigMap

		BeforeEach(func() {
			obj = &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "cm",
					Namespace:   namespace,
					Annotations: map[string]string{},
				},
			}
		})

		It("if the annotation is not present, it should return the default reconcile policy defined at the reconciler", func() {
			p, err := reconciler.getReconcilePolicy(obj)
			Expect(err).NotTo(HaveOccurred())
			Expect(p).To(Equal(ReconcilePolicyOnObjectChange))
		})

		It("if the annotation is present and valid, it should return the reconcile policy specified in the annotation", func() {
			for _, policy := range []string{types.ReconcilePolicyOnObjectChange, types.ReconcilePolicyOnObjectOrComponentChange, types.ReconcilePolicyOnce} {
				obj.Annotations[fmt.Sprintf("%s/%s", reconcilerName, types.AnnotationKeySuffixReconcilePolicy)] = policy
				p, err := reconciler.getReconcilePolicy(obj)
				Expect(err).NotTo(HaveOccurred())
				Expect(p).To(Equal(reconcilePolicyByAnnotation[policy]))
			}

			// we intentionally use the code values (not the kebap case annotation values defined in package types), in order to
			// validate the conversion logic as well
			for _, policy := range []ReconcilePolicy{ReconcilePolicyOnObjectChange, ReconcilePolicyOnObjectOrComponentChange, ReconcilePolicyOnce} {
				obj.Annotations[fmt.Sprintf("%s/%s", reconcilerName, types.AnnotationKeySuffixReconcilePolicy)] = string(policy)
				p, err := reconciler.getReconcilePolicy(obj)
				Expect(err).NotTo(HaveOccurred())
				Expect(p).To(Equal(policy))
			}
		})

		It("if the annotation is present but invalid, it should return an error", func() {
			obj.Annotations[fmt.Sprintf("%s/%s", reconcilerName, types.AnnotationKeySuffixReconcilePolicy)] = "invalid"
			_, err := reconciler.getReconcilePolicy(obj)
			Expect(err).To(HaveOccurred())
		})

	})

	Describe("testing: getUpdatePolicy()", func() {

		var obj *corev1.ConfigMap

		BeforeEach(func() {
			obj = &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "cm",
					Namespace:   namespace,
					Annotations: map[string]string{},
				},
			}
		})

		It("if the annotation is not present, it should return the default update policy defined at the reconciler", func() {
			p, err := reconciler.getUpdatePolicy(obj)
			Expect(err).NotTo(HaveOccurred())
			Expect(p).To(Equal(UpdatePolicySsaOverride))
		})

		It("if the annotation is present and valid, it should return the update policy specified in the annotation", func() {
			for _, policy := range []string{types.UpdatePolicyRecreate, types.UpdatePolicyReplace, types.UpdatePolicySsaMerge, types.UpdatePolicySsaOverride} {
				obj.Annotations[fmt.Sprintf("%s/%s", reconcilerName, types.AnnotationKeySuffixUpdatePolicy)] = policy
				p, err := reconciler.getUpdatePolicy(obj)
				Expect(err).NotTo(HaveOccurred())
				Expect(p).To(Equal(updatePolicyByAnnotation[policy]))
			}

			// we intentionally use the code values (not the kebap case annotation values defined in package types), in order to
			// validate the conversion logic as well
			for _, policy := range []UpdatePolicy{UpdatePolicyRecreate, UpdatePolicyReplace, UpdatePolicySsaMerge, UpdatePolicySsaOverride} {
				obj.Annotations[fmt.Sprintf("%s/%s", reconcilerName, types.AnnotationKeySuffixUpdatePolicy)] = string(policy)
				p, err := reconciler.getUpdatePolicy(obj)
				Expect(err).NotTo(HaveOccurred())
				Expect(p).To(Equal(policy))
			}
		})

		It("if the annotation is present but invalid, it should return an error", func() {
			obj.Annotations[fmt.Sprintf("%s/%s", reconcilerName, types.AnnotationKeySuffixUpdatePolicy)] = "invalid"
			_, err := reconciler.getUpdatePolicy(obj)
			Expect(err).To(HaveOccurred())
		})

	})

	Describe("testing: getDeletePolicy()", func() {

		var obj *corev1.ConfigMap

		BeforeEach(func() {
			obj = &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "cm",
					Namespace:   namespace,
					Annotations: map[string]string{},
				},
			}
		})

		It("if the annotation is not present, it should return the default delete policy defined at the reconciler", func() {
			p, err := reconciler.getDeletePolicy(obj)
			Expect(err).NotTo(HaveOccurred())
			Expect(p).To(Equal(DeletePolicyDelete))
		})

		It("if the annotation is present and valid, it should return the delete policy specified in the annotation", func() {
			for _, policy := range []string{types.DeletePolicyDelete, types.DeletePolicyOrphan, types.DeletePolicyOrphanOnApply, types.DeletePolicyOrphanOnDelete} {
				obj.Annotations[fmt.Sprintf("%s/%s", reconcilerName, types.AnnotationKeySuffixDeletePolicy)] = policy
				p, err := reconciler.getDeletePolicy(obj)
				Expect(err).NotTo(HaveOccurred())
				Expect(p).To(Equal(deletePolicyByAnnotation[policy]))
			}

			// we intentionally use the code values (not the kebap case annotation values defined in package types), in order to
			// validate the conversion logic as well
			for _, policy := range []DeletePolicy{DeletePolicyDelete, DeletePolicyOrphan, DeletePolicyOrphanOnApply, DeletePolicyOrphanOnDelete} {
				obj.Annotations[fmt.Sprintf("%s/%s", reconcilerName, types.AnnotationKeySuffixDeletePolicy)] = string(policy)
				p, err := reconciler.getDeletePolicy(obj)
				Expect(err).NotTo(HaveOccurred())
				Expect(p).To(Equal(policy))
			}
		})

		It("if the annotation is present but invalid, it should return an error", func() {
			obj.Annotations[fmt.Sprintf("%s/%s", reconcilerName, types.AnnotationKeySuffixDeletePolicy)] = "invalid"
			_, err := reconciler.getDeletePolicy(obj)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("testing: getReapplyInterval()", func() {

		var obj *corev1.ConfigMap

		BeforeEach(func() {
			obj = &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "cm",
					Namespace:   namespace,
					Annotations: map[string]string{},
				},
			}
		})

		It("if the annotation is not present, it should return the default reapply interval defined at the reconciler", func() {
			p, err := reconciler.getReapplyInterval(obj)
			Expect(err).NotTo(HaveOccurred())
			Expect(p).To(Equal(9 * time.Minute))
		})

		It("if the annotation is present and valid, it should return the reapply interval specified in the annotation", func() {
			obj.Annotations[fmt.Sprintf("%s/%s", reconcilerName, types.AnnotationKeySuffixReapplyInterval)] = "3m33s"
			i, err := reconciler.getReapplyInterval(obj)
			Expect(err).NotTo(HaveOccurred())
			Expect(i).To(Equal(213 * time.Second))

			// we intentionally use the code values (not the kebap case annotation values defined in package types), in order to
			// validate the conversion logic as well
			obj.Annotations[fmt.Sprintf("%s/%s", reconcilerName, types.AnnotationKeySuffixReapplyInterval)] = "3m33s"
			i, err = reconciler.getReapplyInterval(obj)
			Expect(err).NotTo(HaveOccurred())
			Expect(i).To(Equal(213 * time.Second))
		})

		It("if the annotation is present but invalid, it should return an error", func() {
			obj.Annotations[fmt.Sprintf("%s/%s", reconcilerName, types.AnnotationKeySuffixReapplyInterval)] = "invalid"
			_, err := reconciler.getReapplyInterval(obj)
			Expect(err).To(HaveOccurred())
		})

	})

	Describe("testing: getApplyOrder()", func() {

		var obj *corev1.ConfigMap

		BeforeEach(func() {
			obj = &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "cm",
					Namespace:   namespace,
					Annotations: map[string]string{},
				},
			}
		})

		It("if the annotation is not present, it should return zero", func() {
			o, err := reconciler.getApplyOrder(obj)
			Expect(err).NotTo(HaveOccurred())
			Expect(o).To(Equal(0))
		})

		It("if the annotation is present and valid, it should return the apply order specified in the annotation", func() {
			obj.Annotations[fmt.Sprintf("%s/%s", reconcilerName, types.AnnotationKeySuffixApplyOrder)] = "5"
			o, err := reconciler.getApplyOrder(obj)
			Expect(err).NotTo(HaveOccurred())
			Expect(o).To(Equal(5))

			obj.Annotations[fmt.Sprintf("%s/%s", reconcilerName, types.AnnotationKeySuffixApplyOrder)] = "-32768"
			o, err = reconciler.getApplyOrder(obj)
			Expect(err).NotTo(HaveOccurred())
			Expect(o).To(Equal(-32768))
		})

		It("if the annotation is present but invalid, it should return an error", func() {
			obj.Annotations[fmt.Sprintf("%s/%s", reconcilerName, types.AnnotationKeySuffixApplyOrder)] = "invalid"
			_, err := reconciler.getApplyOrder(obj)
			Expect(err).To(HaveOccurred())

			obj.Annotations[fmt.Sprintf("%s/%s", reconcilerName, types.AnnotationKeySuffixApplyOrder)] = "32768"
			_, err = reconciler.getApplyOrder(obj)
			Expect(err).To(HaveOccurred())

			obj.Annotations[fmt.Sprintf("%s/%s", reconcilerName, types.AnnotationKeySuffixApplyOrder)] = "-32769"
			_, err = reconciler.getApplyOrder(obj)
			Expect(err).To(HaveOccurred())
		})

	})

	Describe("testing: getPurgeOrder()", func() {

		var obj *corev1.ConfigMap

		BeforeEach(func() {
			obj = &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "cm",
					Namespace:   namespace,
					Annotations: map[string]string{},
				},
			}
		})

		It("if the annotation is not present, it should return 32768", func() {
			o, err := reconciler.getPurgeOrder(obj)
			Expect(err).NotTo(HaveOccurred())
			Expect(o).To(Equal(32768))
		})

		It("if the annotation is present and valid, it should return the purge order specified in the annotation", func() {
			obj.Annotations[fmt.Sprintf("%s/%s", reconcilerName, types.AnnotationKeySuffixPurgeOrder)] = "5"
			o, err := reconciler.getPurgeOrder(obj)
			Expect(err).NotTo(HaveOccurred())
			Expect(o).To(Equal(5))

			obj.Annotations[fmt.Sprintf("%s/%s", reconcilerName, types.AnnotationKeySuffixPurgeOrder)] = "-32768"
			o, err = reconciler.getPurgeOrder(obj)
			Expect(err).NotTo(HaveOccurred())
			Expect(o).To(Equal(-32768))
		})

		It("if the annotation is present but invalid, it should return an error", func() {
			obj.Annotations[fmt.Sprintf("%s/%s", reconcilerName, types.AnnotationKeySuffixPurgeOrder)] = "invalid"
			_, err := reconciler.getPurgeOrder(obj)
			Expect(err).To(HaveOccurred())

			obj.Annotations[fmt.Sprintf("%s/%s", reconcilerName, types.AnnotationKeySuffixPurgeOrder)] = "32768"
			_, err = reconciler.getPurgeOrder(obj)
			Expect(err).To(HaveOccurred())

			obj.Annotations[fmt.Sprintf("%s/%s", reconcilerName, types.AnnotationKeySuffixPurgeOrder)] = "-32769"
			_, err = reconciler.getPurgeOrder(obj)
			Expect(err).To(HaveOccurred())
		})

	})

	Describe("testing: getDeleteOrder()", func() {

		var obj *corev1.ConfigMap

		BeforeEach(func() {
			obj = &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "cm",
					Namespace:   namespace,
					Annotations: map[string]string{},
				},
			}
		})

		It("if the annotation is not present, it should return zero", func() {
			o, err := reconciler.getDeleteOrder(obj)
			Expect(err).NotTo(HaveOccurred())
			Expect(o).To(Equal(0))
		})

		It("if the annotation is present and valid, it should return the delete order specified in the annotation", func() {
			obj.Annotations[fmt.Sprintf("%s/%s", reconcilerName, types.AnnotationKeySuffixDeleteOrder)] = "5"
			o, err := reconciler.getDeleteOrder(obj)
			Expect(err).NotTo(HaveOccurred())
			Expect(o).To(Equal(5))

			obj.Annotations[fmt.Sprintf("%s/%s", reconcilerName, types.AnnotationKeySuffixDeleteOrder)] = "-32768"
			o, err = reconciler.getDeleteOrder(obj)
			Expect(err).NotTo(HaveOccurred())
			Expect(o).To(Equal(-32768))
		})

		It("if the annotation is present but invalid, it should return an error", func() {
			obj.Annotations[fmt.Sprintf("%s/%s", reconcilerName, types.AnnotationKeySuffixDeleteOrder)] = "invalid"
			_, err := reconciler.getDeleteOrder(obj)
			Expect(err).To(HaveOccurred())

			obj.Annotations[fmt.Sprintf("%s/%s", reconcilerName, types.AnnotationKeySuffixDeleteOrder)] = "32768"
			_, err = reconciler.getDeleteOrder(obj)
			Expect(err).To(HaveOccurred())

			obj.Annotations[fmt.Sprintf("%s/%s", reconcilerName, types.AnnotationKeySuffixDeleteOrder)] = "-32769"
			_, err = reconciler.getDeleteOrder(obj)
			Expect(err).To(HaveOccurred())
		})

	})

})

func WithTypeInfo(obj client.Object, scheme *runtime.Scheme) (client.Object, error) {
	gvk, err := apiutil.GVKForObject(obj, scheme)
	if err != nil {
		return nil, err
	}
	obj = obj.DeepCopyObject().(client.Object)
	obj.GetObjectKind().SetGroupVersionKind(gvk)
	return obj, nil
}

func CreateOrUpdateInventoryItemForObject(item *InventoryItem, reconciler *Reconciler, scheme *runtime.Scheme, obj client.Object, componentDigest string, phase Phase, status status.Status) (*InventoryItem, error) {
	obj, err := WithTypeInfo(obj, scheme)
	if err != nil {
		return nil, err
	}

	adoptionPolicy, err := reconciler.getAdoptionPolicy(obj)
	if err != nil {
		return nil, err
	}
	reconcilePolicy, err := reconciler.getReconcilePolicy(obj)
	if err != nil {
		return nil, err
	}
	updatePolicy, err := reconciler.getUpdatePolicy(obj)
	if err != nil {
		return nil, err
	}
	deletePolicy, err := reconciler.getDeletePolicy(obj)
	if err != nil {
		return nil, err
	}
	applyOrder, err := reconciler.getApplyOrder(obj)
	if err != nil {
		return nil, err
	}
	deleteOrder, err := reconciler.getDeleteOrder(obj)
	if err != nil {
		return nil, err
	}

	managedTypes := getManagedTypes(obj)

	digest, err := calculateObjectDigest(obj, componentDigest, reconcilePolicy)
	if err != nil {
		return nil, err
	}

	if item == nil {
		item = &InventoryItem{}
	}

	item.TypeVersionInfo = TypeVersionInfo(obj.GetObjectKind().GroupVersionKind())
	item.NameInfo = NameInfo{
		Namespace: obj.GetNamespace(),
		Name:      obj.GetName(),
	}
	item.AdoptionPolicy = adoptionPolicy
	item.ReconcilePolicy = reconcilePolicy
	item.UpdatePolicy = updatePolicy
	item.DeletePolicy = deletePolicy
	item.ApplyOrder = applyOrder
	item.DeleteOrder = deleteOrder
	item.ManagedTypes = managedTypes
	item.Digest = digest
	item.Phase = phase
	item.Status = status

	return item, nil
}

func MatchInventory(expected []*InventoryItem) OmegaMatcher {
	return MakeMatcher(func(actual []*InventoryItem) (bool, error) {
		if len(actual) != len(expected) {
			return false, nil
		}
		actual = slices.Collect(actual, func(item *InventoryItem) *InventoryItem {
			item = item.DeepCopy()
			item.LastAppliedAt = nil
			return item
		})
		expected = slices.Collect(expected, func(item *InventoryItem) *InventoryItem {
			item = item.DeepCopy()
			item.LastAppliedAt = nil
			return item
		})
		return ConsistOf(expected).Match(actual)
	}).WithTemplate("Expected inventory:\n{{.FormattedActual}}\n{{.To}} to match inventory:\n{{format .Data 1}}", expected)
}
