/*
SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package reconciler

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	apitypes "k8s.io/apimachinery/pkg/types"
	apiregistrationsv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/sap/component-operator-runtime/internal/util"
	"github.com/sap/component-operator-runtime/pkg/types"
	cstestingv1alpha1 "github.com/sap/component-operator-runtime/testing/environment/apis/testing.cs.sap.com/v1alpha1"
)

var _ = Describe("testing: util.go", func() {

	Describe("testing: calculateObjectDigest()", func() {

		var obj client.Object
		var objDigest string
		var componentDigest string

		BeforeEach(func() {
			obj = &cstestingv1alpha1.Foo{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "test",
					Namespace:       "default",
					ResourceVersion: "666",
					Generation:      99,
					ManagedFields: []metav1.ManagedFieldsEntry{
						{
							Manager: "test-manager",
						},
					},
				},
				Spec: cstestingv1alpha1.FooSpec{
					Value: "12345",
				},
				Status: cstestingv1alpha1.FooStatus{
					ObservedGeneration: 42,
				},
			}
			objDigest = "cd11167d766bb5193295be9acbfaf252b8f10be608c1198883824d0f3a3f8161"
			componentDigest = util.Sha256hex([]byte("12345"))
		})

		It("should calculate the right object digest (with reconcile policy 'once')", func() {
			savedObj := obj.DeepCopyObject().(client.Object)
			Expect(calculateObjectDigest(obj, componentDigest, ReconcilePolicyOnce)).To(Equal("__once__"))
			Expect(obj).To(Equal(savedObj))
		})

		It("should calculate the right object digest (with reconcile policy 'on-object-change')", func() {
			savedObj := obj.DeepCopyObject().(client.Object)
			Expect(calculateObjectDigest(obj, componentDigest, ReconcilePolicyOnObjectChange)).To(Equal(objDigest))
			Expect(obj).To(Equal(savedObj))
		})

		It("should calculate the right object digest (with reconcile policy 'on-object-or-component-change')", func() {
			savedObj := obj.DeepCopyObject().(client.Object)
			Expect(calculateObjectDigest(obj, componentDigest, ReconcilePolicyOnObjectOrComponentChange)).To(Equal(fmt.Sprintf("%s@%s", objDigest, componentDigest)))
			Expect(obj).To(Equal(savedObj))
		})

	})

	Describe("testing: checkRange()", func() {

		It("should not error if the value is within the range", func() {
			Expect(checkRange(4, 4, 5)).NotTo(HaveOccurred())
			Expect(checkRange(5, 4, 5)).NotTo(HaveOccurred())
		})

		It("should error if the value is out of the range", func() {
			Expect(checkRange(3, 4, 5)).To(HaveOccurred())
			Expect(checkRange(6, 4, 5)).To(HaveOccurred())
		})

	})

	Describe("testing: isNamespace()", func() {

		It("should detect namespaces", func() {
			Expect(isNamespace(&metav1.PartialObjectMetadata{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "Namespace",
				},
			})).To(BeTrue())

			Expect(isNamespace(&metav1.PartialObjectMetadata{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v2",
					Kind:       "Namespace",
				},
			})).To(BeTrue())

			Expect(isNamespace(&metav1.PartialObjectMetadata{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "group/v1",
					Kind:       "Namespace",
				},
			})).To(BeFalse())

			Expect(isNamespace(&metav1.PartialObjectMetadata{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "Other",
				},
			})).To(BeFalse())

			Expect(isNamespace(&metav1.PartialObjectMetadata{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "group/v1",
					Kind:       "Other",
				},
			})).To(BeFalse())
		})

	})

	Describe("testing: isCrd()", func() {

		It("should detect CRDs", func() {
			Expect(isCrd(&metav1.PartialObjectMetadata{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "apiextensions.k8s.io/v1",
					Kind:       "CustomResourceDefinition",
				},
			})).To(BeTrue())

			Expect(isCrd(&metav1.PartialObjectMetadata{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "apiextensions.k8s.io/v2",
					Kind:       "CustomResourceDefinition",
				},
			})).To(BeTrue())

			Expect(isCrd(&metav1.PartialObjectMetadata{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "group/v1",
					Kind:       "CustomResourceDefinition",
				},
			})).To(BeFalse())

			Expect(isCrd(&metav1.PartialObjectMetadata{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "apiextensions.k8s.io/v1",
					Kind:       "Other",
				},
			})).To(BeFalse())

			Expect(isCrd(&metav1.PartialObjectMetadata{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "group/v1",
					Kind:       "Other",
				},
			})).To(BeFalse())
		})

	})

	Describe("testing: isApiService()", func() {

		It("should detect APIServices", func() {
			Expect(isApiService(&metav1.PartialObjectMetadata{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "apiregistration.k8s.io/v1",
					Kind:       "APIService",
				},
			})).To(BeTrue())

			Expect(isApiService(&metav1.PartialObjectMetadata{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "apiregistration.k8s.io/v2",
					Kind:       "APIService",
				},
			})).To(BeTrue())

			Expect(isApiService(&metav1.PartialObjectMetadata{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "group/v1",
					Kind:       "APIService",
				},
			})).To(BeFalse())

			Expect(isApiService(&metav1.PartialObjectMetadata{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "apiregistration.k8s.io/v1",
					Kind:       "Other",
				},
			})).To(BeFalse())

			Expect(isApiService(&metav1.PartialObjectMetadata{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "group/v1",
					Kind:       "Other",
				},
			})).To(BeFalse())
		})

	})

	Describe("testing: isSecret()", func() {

		It("should detect secrets", func() {
			Expect(isSecret(&metav1.PartialObjectMetadata{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "Secret",
				},
			})).To(BeTrue())

			Expect(isSecret(&metav1.PartialObjectMetadata{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v2",
					Kind:       "Secret",
				},
			})).To(BeTrue())

			Expect(isSecret(&metav1.PartialObjectMetadata{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "group/v1",
					Kind:       "Secret",
				},
			})).To(BeFalse())

			Expect(isSecret(&metav1.PartialObjectMetadata{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "Other",
				},
			})).To(BeFalse())

			Expect(isSecret(&metav1.PartialObjectMetadata{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "group/v1",
					Kind:       "Other",
				},
			})).To(BeFalse())
		})

	})

	Describe("testing: getCrds(), getApiServices()", func() {

		var structuredCrd1 *apiextensionsv1.CustomResourceDefinition
		var structuredCrd2 *apiextensionsv1.CustomResourceDefinition
		var structuredApiService1 *apiregistrationsv1.APIService
		var structuredApiService2 *apiregistrationsv1.APIService
		var unstructuredCrd client.Object
		var unstructuredApiService client.Object
		var otherObject client.Object

		BeforeEach(func() {
			structuredCrd1 = &apiextensionsv1.CustomResourceDefinition{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "apiextensions.k8s.io/v1",
					Kind:       "CustomResourceDefinition",
				},
			}
			structuredCrd2 = &apiextensionsv1.CustomResourceDefinition{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "apiextensions.k8s.io/v1",
					Kind:       "CustomResourceDefinition",
				},
			}
			structuredApiService1 = &apiregistrationsv1.APIService{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "apiregistration.k8s.io/v1",
					Kind:       "APIService",
				},
			}
			structuredApiService2 = &apiregistrationsv1.APIService{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "apiregistration.k8s.io/v1",
					Kind:       "APIService",
				},
			}
			unstructuredCrd = &unstructured.Unstructured{
				Object: map[string]any{
					"apiVersion": "apiextensions.k8s.io/v1",
					"kind":       "CustomResourceDefinition",
				},
			}
			unstructuredApiService = &unstructured.Unstructured{
				Object: map[string]any{
					"apiVersion": "apiregistration.k8s.io/v1",
					"kind":       "APIService",
				},
			}
			otherObject = &unstructured.Unstructured{
				Object: map[string]any{
					"apiVersion": "v1",
					"kind":       "ConfigMap",
				},
			}
		})

		It("should detect CRDs", func() {
			Expect(getCrds([]client.Object{structuredCrd1, structuredCrd2, structuredApiService1, structuredApiService2, otherObject})).To(Equal([]*apiextensionsv1.CustomResourceDefinition{structuredCrd1, structuredCrd2}))
			Expect(getCrds([]client.Object{otherObject})).To(BeEmpty())
			Expect(getCrds([]client.Object{})).To(BeEmpty())
		})

		It("should detect APIServices", func() {
			Expect(getApiServices([]client.Object{structuredCrd1, structuredCrd2, structuredApiService1, structuredApiService2, otherObject})).To(Equal([]*apiregistrationsv1.APIService{structuredApiService1, structuredApiService2}))
			Expect(getApiServices([]client.Object{otherObject})).To(BeEmpty())
			Expect(getApiServices([]client.Object{})).To(BeEmpty())
		})

		It("should panic with unstructured CRDs", func() {
			Expect(func() { getCrds([]client.Object{unstructuredCrd}) }).To(Panic())
		})

		It("should panic with unstructured APIServices", func() {
			Expect(func() { getApiServices([]client.Object{unstructuredApiService}) }).To(Panic())
		})

	})

	Describe("testing: getManagedTypes()", func() {

		var crd *apiextensionsv1.CustomResourceDefinition
		var apiService *apiregistrationsv1.APIService
		var otherObject *corev1.ConfigMap

		BeforeEach(func() {
			crd = &apiextensionsv1.CustomResourceDefinition{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "apiextensions.k8s.io/v1",
					Kind:       "CustomResourceDefinition",
				},
				Spec: apiextensionsv1.CustomResourceDefinitionSpec{
					Group: "group",
					Names: apiextensionsv1.CustomResourceDefinitionNames{
						Kind: "Kind",
					},
				},
			}
			apiService = &apiregistrationsv1.APIService{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "apiregistration.k8s.io/v1",
					Kind:       "APIService",
				},
				Spec: apiregistrationsv1.APIServiceSpec{
					Group:   "group",
					Version: "v1",
				},
			}
			otherObject = &corev1.ConfigMap{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "ConfigMap",
				},
			}
		})

		It("should detect managed types from CRDs and APIServices", func() {
			Expect(getManagedTypes(crd)).To(Equal([]TypeVersionInfo{{Group: "group", Version: "*", Kind: "Kind"}}))
			Expect(getManagedTypes(apiService)).To(Equal([]TypeVersionInfo{{Group: "group", Version: "v1", Kind: "*"}}))
			Expect(getManagedTypes(otherObject)).To(BeNil())
		})

	})

	Describe("testing: findMissingNamespaces()", func() {

		It("should find missing namespaces", func() {
			objects := []client.Object{
				&corev1.Namespace{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "v1",
						Kind:       "Namespace",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "ns1",
					},
				},
				&metav1.PartialObjectMetadata{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "group/v1",
						Kind:       "Other",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "other",
						Namespace: "ns1",
					},
				},
				&cstestingv1alpha1.Foo{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "testing.cs.sap.com/v1alpha1",
						Kind:       "Foo",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "foo",
						Namespace: "ns2",
					},
				},
			}
			Expect(findMissingNamespaces(objects)).To(Equal([]string{"ns2"}))
		})

	})

	Describe("testing: scopeFromRestMapping()", func() {

		var namespaceScopeRestMapping *apimeta.RESTMapping
		var clusterScopeRestMapping *apimeta.RESTMapping

		BeforeEach(func() {
			namespaceScopeRestMapping = &apimeta.RESTMapping{
				Scope: apimeta.RESTScopeNamespace,
			}
			clusterScopeRestMapping = &apimeta.RESTMapping{
				Scope: apimeta.RESTScopeRoot,
			}
		})

		It("should return correct scope from rest mapping", func() {
			Expect(scopeFromRestMapping(namespaceScopeRestMapping)).To(Equal(scopeNamespaced))
			Expect(scopeFromRestMapping(clusterScopeRestMapping)).To(Equal(scopeCluster))
		})
	})

	Describe("testing: scopeFromCrd()", func() {

		var namespaceScopeCrd *apiextensionsv1.CustomResourceDefinition
		var clusterScopeCrd *apiextensionsv1.CustomResourceDefinition
		var invalidScopeCrd *apiextensionsv1.CustomResourceDefinition

		BeforeEach(func() {
			namespaceScopeCrd = &apiextensionsv1.CustomResourceDefinition{
				Spec: apiextensionsv1.CustomResourceDefinitionSpec{
					Scope: apiextensionsv1.NamespaceScoped,
				},
			}
			clusterScopeCrd = &apiextensionsv1.CustomResourceDefinition{
				Spec: apiextensionsv1.CustomResourceDefinitionSpec{
					Scope: apiextensionsv1.ClusterScoped,
				},
			}
			invalidScopeCrd = &apiextensionsv1.CustomResourceDefinition{
				Spec: apiextensionsv1.CustomResourceDefinitionSpec{
					Scope: "Invalid",
				},
			}
		})

		It("should return correct scope from CRD", func() {
			Expect(scopeFromCrd(namespaceScopeCrd)).To(Equal(scopeNamespaced))
			Expect(scopeFromCrd(clusterScopeCrd)).To(Equal(scopeCluster))
		})

		It("should panic with invalid CRD scope", func() {
			Expect(func() { scopeFromCrd(invalidScopeCrd) }).To(Panic())
		})

	})

	Describe("testing: normalizeObjects()", func() {

		var unstructuredObjectWithTypeMeta client.Object
		var unstructuredObjectWithoutTypeMeta client.Object
		var structuredObjectWithCorrectTypeMeta client.Object
		var structuredObjectWithIncorrectTypeMeta client.Object
		var structuredObjectWithoutTypeMeta client.Object

		var unstructuredCrd client.Object
		var unstructuredApiService client.Object

		var unstructuredObjectWithInvalidLabel client.Object
		var unstructuredObjectWithInvalidAnnotation client.Object

		var emptyScheme *runtime.Scheme
		var corev1Scheme *runtime.Scheme

		BeforeEach(func() {
			unstructuredObjectWithTypeMeta = &unstructured.Unstructured{
				Object: map[string]any{
					"apiVersion": "v1",
					"kind":       "ConfigMap",
				},
			}
			unstructuredObjectWithoutTypeMeta = &unstructured.Unstructured{
				Object: map[string]any{},
			}
			structuredObjectWithCorrectTypeMeta = &corev1.ConfigMap{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "ConfigMap",
				},
			}
			structuredObjectWithIncorrectTypeMeta = &corev1.ConfigMap{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v2",
					Kind:       "Other",
				},
			}
			structuredObjectWithoutTypeMeta = &corev1.ConfigMap{}

			unstructuredCrd = &unstructured.Unstructured{
				Object: map[string]any{
					"apiVersion": "apiextensions.k8s.io/v1",
					"kind":       "CustomResourceDefinition",
				},
			}
			unstructuredApiService = &unstructured.Unstructured{
				Object: map[string]any{
					"apiVersion": "apiregistration.k8s.io/v1",
					"kind":       "APIService",
				},
			}

			unstructuredObjectWithInvalidLabel = &unstructured.Unstructured{
				Object: map[string]any{
					"apiVersion": "v1",
					"kind":       "ConfigMap",
					"metadata": map[string]any{
						"name": "test",
						"labels": map[string]any{
							"key": 123,
						},
					},
				},
			}
			unstructuredObjectWithInvalidAnnotation = &unstructured.Unstructured{
				Object: map[string]any{
					"apiVersion": "v1",
					"kind":       "ConfigMap",
					"metadata": map[string]any{
						"name": "test",
						"annotations": map[string]any{
							"key": true,
						},
					},
				},
			}

			emptyScheme = runtime.NewScheme()
			corev1Scheme = runtime.NewScheme()
			corev1.AddToScheme(corev1Scheme)
		})

		It("should just clone an unstructured (but unrecognized) object with type meta", func() {
			objects, err := normalizeObjects([]client.Object{unstructuredObjectWithTypeMeta}, emptyScheme)
			Expect(err).NotTo(HaveOccurred())
			Expect(objects).To(HaveLen(1))
			Expect(objects[0]).To(Equal(unstructuredObjectWithTypeMeta))
			Expect(objects[0]).NotTo(BeIdenticalTo(unstructuredObjectWithTypeMeta))
		})

		It("should convert an unstructured (but recognized) object with type meta to a structured object", func() {
			objects, err := normalizeObjects([]client.Object{unstructuredObjectWithTypeMeta}, corev1Scheme)
			Expect(err).NotTo(HaveOccurred())
			Expect(objects).To(HaveLen(1))
			Expect(objects[0]).To(Equal(structuredObjectWithCorrectTypeMeta))
			Expect(objects[0]).NotTo(BeIdenticalTo(structuredObjectWithCorrectTypeMeta))
		})

		It("should return an error for an unstructured object without type meta", func() {
			_, err := normalizeObjects([]client.Object{unstructuredObjectWithoutTypeMeta}, emptyScheme)
			Expect(err).To(HaveOccurred())
		})

		It("should clone a structured (recognized) object with correct type meta", func() {
			objects, err := normalizeObjects([]client.Object{structuredObjectWithCorrectTypeMeta}, corev1Scheme)
			Expect(err).NotTo(HaveOccurred())
			Expect(objects).To(HaveLen(1))
			Expect(objects[0]).To(Equal(structuredObjectWithCorrectTypeMeta))
			Expect(objects[0]).NotTo(BeIdenticalTo(structuredObjectWithCorrectTypeMeta))
		})

		It("should return an error for a structured (recognized) object with incorrect type meta", func() {
			_, err := normalizeObjects([]client.Object{structuredObjectWithIncorrectTypeMeta}, corev1Scheme)
			Expect(err).To(HaveOccurred())
		})

		It("should complete and clone a structured (recognized) object without type meta", func() {
			objects, err := normalizeObjects([]client.Object{structuredObjectWithoutTypeMeta}, corev1Scheme)
			Expect(err).NotTo(HaveOccurred())
			Expect(objects).To(HaveLen(1))
			Expect(objects[0]).To(Equal(structuredObjectWithCorrectTypeMeta))
			Expect(objects[0]).NotTo(BeIdenticalTo(structuredObjectWithCorrectTypeMeta))
		})

		It("should return errors for structured unrecognized objects", func() {
			_, err := normalizeObjects([]client.Object{structuredObjectWithCorrectTypeMeta}, emptyScheme)
			Expect(err).To(HaveOccurred())
			_, err = normalizeObjects([]client.Object{structuredObjectWithIncorrectTypeMeta}, emptyScheme)
			Expect(err).To(HaveOccurred())
			_, err = normalizeObjects([]client.Object{structuredObjectWithoutTypeMeta}, emptyScheme)
			Expect(err).To(HaveOccurred())
		})

		It("should return an error for unstructured unrecognized CRD or APIService objects", func() {
			_, err := normalizeObjects([]client.Object{unstructuredCrd}, emptyScheme)
			Expect(err).To(HaveOccurred())
			_, err = normalizeObjects([]client.Object{unstructuredApiService}, emptyScheme)
			Expect(err).To(HaveOccurred())
		})

		It("should return an error for unstructured objects with non-string label or annotation values", func() {
			_, err := normalizeObjects([]client.Object{unstructuredObjectWithInvalidLabel}, emptyScheme)
			Expect(err).To(HaveOccurred())
			_, err = normalizeObjects([]client.Object{unstructuredObjectWithInvalidAnnotation}, emptyScheme)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("testing: sortObjectsForApply()", func() {

		// TODO: add test

	})

	Describe("testing: sortObjectsForDelete()", func() {

		// TODO: add test

	})

	Describe("testing: getItem(), mustGetItem()", func() {

		var groupv1Kind TypeVersionInfo
		var groupv2Kind TypeVersionInfo
		var otherGroupv1Kind TypeVersionInfo
		var clusterGroupKind TypeVersionInfo

		var name11 NameInfo
		var name12 NameInfo
		var name13 NameInfo
		var name21 NameInfo
		var name22 NameInfo
		var name31 NameInfo
		var namec1 NameInfo
		var namec2 NameInfo

		var validInventory []*InventoryItem
		var invalidInventory1 []*InventoryItem
		var invalidInventory2 []*InventoryItem

		var objectKey = func(typeVersionInfo TypeVersionInfo, nameInfo NameInfo) types.ObjectKey {
			return &struct {
				types.TypeKey
				types.NameKey
			}{
				TypeKey: types.TypeKeyFromGroupVersionKind(schema.GroupVersionKind(typeVersionInfo)),
				NameKey: types.NameKeyFromNamespacedName(apitypes.NamespacedName(nameInfo)),
			}
		}

		BeforeEach(func() {
			groupv1Kind = TypeVersionInfo{
				Group:   "group",
				Version: "v1",
				Kind:    "Kind",
			}
			groupv2Kind = TypeVersionInfo{
				Group:   "group",
				Version: "v2",
				Kind:    "Kind",
			}
			otherGroupv1Kind = TypeVersionInfo{
				Group:   "othergroup",
				Version: "v1",
				Kind:    "Kind",
			}
			clusterGroupKind = TypeVersionInfo{
				Group:   "clustergroup",
				Version: "v2",
				Kind:    "Kind",
			}

			name11 = NameInfo{
				Namespace: "namespace-1",
				Name:      "name-1",
			}
			name12 = NameInfo{
				Namespace: "namespace-1",
				Name:      "name-2",
			}
			name13 = NameInfo{
				Namespace: "namespace-1",
				Name:      "name-3",
			}
			name21 = NameInfo{
				Namespace: "namespace-2",
				Name:      "name-1",
			}
			name22 = NameInfo{
				Namespace: "namespace-2",
				Name:      "name-2",
			}
			name31 = NameInfo{
				Namespace: "namespace-3",
				Name:      "name-1",
			}

			namec1 = NameInfo{
				Name: "name-c1",
			}
			namec2 = NameInfo{
				Name: "name-c2",
			}

			validInventory = []*InventoryItem{
				{
					TypeVersionInfo: groupv1Kind,
					NameInfo:        name11,
				},
				{
					TypeVersionInfo: groupv1Kind,
					NameInfo:        name12,
				},
				{
					TypeVersionInfo: groupv1Kind,
					NameInfo:        name21,
				},
				{
					TypeVersionInfo: groupv1Kind,
					NameInfo:        name22,
				},
				{
					TypeVersionInfo: groupv2Kind,
					NameInfo:        name13,
				},
				{
					TypeVersionInfo: otherGroupv1Kind,
					NameInfo:        name11,
				},
				{
					TypeVersionInfo: clusterGroupKind,
					NameInfo:        namec1,
				},
				{
					TypeVersionInfo: clusterGroupKind,
					NameInfo:        namec2,
				},
			}

			invalidInventory1 = []*InventoryItem{
				{
					TypeVersionInfo: groupv1Kind,
					NameInfo:        name11,
				},
				{
					TypeVersionInfo: groupv1Kind,
					NameInfo:        name11,
				},
			}
			invalidInventory2 = []*InventoryItem{
				{
					TypeVersionInfo: groupv1Kind,
					NameInfo:        name11,
				},
				{
					TypeVersionInfo: groupv2Kind,
					NameInfo:        name11,
				},
			}
		})

		It("should return inventory items correctly", func() {
			Expect(getItem(validInventory, objectKey(groupv1Kind, name11))).To(Equal(validInventory[0]))
			Expect(getItem(validInventory, objectKey(groupv2Kind, name11))).To(Equal(validInventory[0]))
			Expect(getItem(validInventory, objectKey(groupv1Kind, name13))).To(Equal(validInventory[4]))
			Expect(getItem(validInventory, objectKey(clusterGroupKind, namec2))).To(Equal(validInventory[7]))
			Expect(getItem(validInventory, objectKey(groupv1Kind, name31))).To(BeNil())

			Expect(mustGetItem(validInventory, objectKey(groupv1Kind, name11))).To(Equal(validInventory[0]))
			Expect(mustGetItem(validInventory, objectKey(groupv2Kind, name11))).To(Equal(validInventory[0]))
			Expect(mustGetItem(validInventory, objectKey(groupv1Kind, name13))).To(Equal(validInventory[4]))
			Expect(mustGetItem(validInventory, objectKey(clusterGroupKind, namec2))).To(Equal(validInventory[7]))
			Expect(func() { mustGetItem(validInventory, objectKey(groupv1Kind, name31)) }).To(Panic())
		})

		It("should panic with an invalid inventory (having duplicate items)", func() {
			Expect(func() { getItem(invalidInventory1, objectKey(groupv1Kind, name11)) }).To(Panic())
			Expect(func() { getItem(invalidInventory2, objectKey(groupv1Kind, name11)) }).To(Panic())

			Expect(func() { mustGetItem(invalidInventory1, objectKey(groupv1Kind, name11)) }).To(Panic())
			Expect(func() { mustGetItem(invalidInventory2, objectKey(groupv1Kind, name11)) }).To(Panic())
		})

	})

	Describe("testing: isNamespaceUsed()", func() {

		var inventory []*InventoryItem

		BeforeEach(func() {
			inventory = []*InventoryItem{
				{
					NameInfo: NameInfo{
						Namespace: "namespace-1",
						Name:      "name-1",
					},
				},
				{
					NameInfo: NameInfo{
						Namespace: "namespace-2",
						Name:      "name-2",
					},
				},
				{
					NameInfo: NameInfo{
						Name: "name-3",
					},
				},
			}
		})

		It("should detect used namespaces correctly", func() {
			Expect(isNamespaceUsed(inventory, "namespace-1")).To(BeTrue())
			Expect(isNamespaceUsed(inventory, "namespace-2")).To(BeTrue())
			Expect(isNamespaceUsed(inventory, "namespace-3")).To(BeFalse())
			Expect(isNamespaceUsed(inventory, "")).To(BeTrue())
		})

	})

	Describe("testing: isManagedInstance()", func() {

		// no tests needed, essentially covered by matches() tests

	})

	Describe("testing: isManagedByTypeVersions()", func() {

		// no tests needed, essentially covered by matches() tests

	})

	Describe("testing: isManagedByTypes()", func() {

		// no tests needed, essentially covered by matches() tests

	})

	Describe("testing: matches()", func() {

		It("should match", func() {
			Expect(matches("foo", "foo")).To(BeTrue())
			Expect(matches("bar.foo", "bar.foo")).To(BeTrue())
			Expect(matches("foo", "*")).To(BeTrue())
			Expect(matches("bar.foo", "*.foo")).To(BeTrue())
			Expect(matches("baz.bar.foo", "*.foo")).To(BeTrue())
		})

		It("should not match", func() {
			Expect(matches("foo", "baz")).To(BeFalse())
			Expect(matches("foo", "f*o")).To(BeFalse())
			Expect(matches("foo.bar", "foo.*")).To(BeFalse())
			Expect(matches("bar.foo", "*r.foo")).To(BeFalse())
			Expect(matches("bar.foo", "b*.foo")).To(BeFalse())
			Expect(matches("baz.bar.foo", "*.*.foo")).To(BeFalse())
		})

	})

})
