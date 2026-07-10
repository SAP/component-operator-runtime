/*
SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package types_test

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	apitypes "k8s.io/apimachinery/pkg/types"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/sap/component-operator-runtime/pkg/types"
)

var _ = Describe("testing: object_key.go", func() {

	var group string
	var version string
	var kind string

	var namespace string
	var name string

	BeforeEach(func() {
		group = "some.group.io"
		version = "v1"
		kind = "SomeThing"

		namespace = "some-namespace"
		name = "some-name"
	})

	It("should return a TypeKey from GroupVersionKind", func() {
		gvk := schema.GroupVersionKind{Group: group, Version: version, Kind: kind}
		Expect(types.TypeKeyFromGroupVersionKind(gvk).GetObjectKind().GroupVersionKind()).To(Equal(gvk))
	})

	It("should return a TypeKey from group, version and kind", func() {
		gvk := schema.GroupVersionKind{Group: group, Version: version, Kind: kind}
		Expect(types.TypeKeyFromGroupAndVersionAndKind(group, version, kind).GetObjectKind().GroupVersionKind()).To(Equal(gvk))
	})

	It("should modify the GroupVersionKind of a TypeKey", func() {
		gvk := schema.GroupVersionKind{Group: group, Version: version, Kind: kind}
		key := types.TypeKeyFromGroupVersionKind(gvk)
		newGVK := schema.GroupVersionKind{Group: "new.group.io", Version: "v2", Kind: "NewThing"}
		key.GetObjectKind().SetGroupVersionKind(newGVK)
		Expect(key.GetObjectKind().GroupVersionKind()).To(Equal(newGVK))
	})

	It("should return the string representation of a TypeKey", func() {
		gvk := schema.GroupVersionKind{Group: group, Version: version, Kind: kind}
		Expect(types.TypeKeyToString(types.TypeKeyFromGroupVersionKind(gvk))).To(Equal(gvk.String()))
	})

	It("should return a NameKey from NamespacedName", func() {
		namespacedName := apitypes.NamespacedName{Namespace: namespace, Name: name}
		Expect(types.NameKeyFromNamespacedName(namespacedName).GetNamespace()).To(Equal(namespace))
		Expect(types.NameKeyFromNamespacedName(namespacedName).GetName()).To(Equal(name))
	})

	It("should return a NameKey from namespace and name", func() {
		Expect(types.NameKeyFromNamespaceAndName(namespace, name).GetNamespace()).To(Equal(namespace))
		Expect(types.NameKeyFromNamespaceAndName(namespace, name).GetName()).To(Equal(name))
	})

	It("should return the string representation of a NameKey", func() {
		namespacedName := apitypes.NamespacedName{Namespace: namespace, Name: name}
		Expect(types.NameKeyToString(types.NameKeyFromNamespacedName(namespacedName))).To(Equal(fmt.Sprintf("%s/%s", namespace, name)))
		namespacedName = apitypes.NamespacedName{Namespace: "", Name: name}
		Expect(types.NameKeyToString(types.NameKeyFromNamespacedName(namespacedName))).To(Equal(name))
	})

	It("should return the string representation of an ObjectKey", func() {
		obj := &metav1.PartialObjectMetadata{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "mygroup/v1",
				Kind:       "MyThing",
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "my-namespace",
				Name:      "my-name",
			},
		}
		Expect(types.ObjectKeyToString(obj)).To(Equal("mygroup/v1, Kind=MyThing my-namespace/my-name"))

		obj = &metav1.PartialObjectMetadata{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "MyThing",
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "my-namespace",
				Name:      "my-name",
			},
		}
		Expect(types.ObjectKeyToString(obj)).To(Equal("/v1, Kind=MyThing my-namespace/my-name"))

		obj = &metav1.PartialObjectMetadata{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "mygroup/v1",
				Kind:       "MyThing",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-name",
			},
		}
		Expect(types.ObjectKeyToString(obj)).To(Equal("mygroup/v1, Kind=MyThing my-name"))
	})
})
