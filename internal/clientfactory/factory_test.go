/*
SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package clientfactory

import (
	"context"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	apitypes "k8s.io/apimachinery/pkg/types"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/sap/component-operator-runtime/pkg/types"
	cstestingv1alpha1 "github.com/sap/component-operator-runtime/testing/environment/apis/testing.cs.sap.com/v1alpha1"
)

var _ = Describe("testing: factory.go", func() {

	var factory *ClientFactory
	var namespace string

	BeforeEach(func() {
		var err error

		schemeBuilder := runtime.NewSchemeBuilder(cstestingv1alpha1.AddToScheme)

		factory, err = NewClientFactory("test-controller", "testing", env.Config(), []types.SchemeBuilder{&schemeBuilder})
		Expect(err).NotTo(HaveOccurred())

		namespace, err = env.CreateNamespace()
		Expect(err).NotTo(HaveOccurred())

		err = env.CreateObject(
			&corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: namespace,
				},
			})
		Expect(err).NotTo(HaveOccurred())
	})

	It("should create a functional client without impersonation", func() {
		clnt, err := factory.Get(nil, "", nil)
		Expect(err).NotTo(HaveOccurred())

		Expect(clnt.Scheme().Recognizes(schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Namespace"})).To(BeTrue())
		Expect(clnt.Scheme().Recognizes(schema.GroupVersionKind{Group: "apiextensions.k8s.io", Version: "v1", Kind: "CustomResourceDefinition"})).To(BeTrue())
		Expect(clnt.Scheme().Recognizes(schema.GroupVersionKind{Group: "apiregistration.k8s.io", Version: "v1", Kind: "APIService"})).To(BeTrue())
		Expect(clnt.Scheme().Recognizes(schema.GroupVersionKind{Group: "testing.cs.sap.com", Version: "v1alpha1", Kind: "Foo"})).To(BeTrue())

		err = clnt.Get(context.Background(), apitypes.NamespacedName{Name: "kube-system"}, &corev1.Namespace{})
		Expect(err).NotTo(HaveOccurred())

		clnt, err = factory.Get([]byte{}, "", nil)
		Expect(err).NotTo(HaveOccurred())

		Expect(clnt.Scheme().Recognizes(schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Namespace"})).To(BeTrue())
		Expect(clnt.Scheme().Recognizes(schema.GroupVersionKind{Group: "apiextensions.k8s.io", Version: "v1", Kind: "CustomResourceDefinition"})).To(BeTrue())
		Expect(clnt.Scheme().Recognizes(schema.GroupVersionKind{Group: "apiregistration.k8s.io", Version: "v1", Kind: "APIService"})).To(BeTrue())
		Expect(clnt.Scheme().Recognizes(schema.GroupVersionKind{Group: "testing.cs.sap.com", Version: "v1alpha1", Kind: "Foo"})).To(BeTrue())

		err = clnt.Get(context.Background(), apitypes.NamespacedName{Name: "kube-system"}, &corev1.Namespace{})
		Expect(err).NotTo(HaveOccurred())

		clnt, err = factory.Get([]byte(env.KubeConfig()), "", nil)
		Expect(err).NotTo(HaveOccurred())

		Expect(clnt.Scheme().Recognizes(schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Namespace"})).To(BeTrue())
		Expect(clnt.Scheme().Recognizes(schema.GroupVersionKind{Group: "apiextensions.k8s.io", Version: "v1", Kind: "CustomResourceDefinition"})).To(BeTrue())
		Expect(clnt.Scheme().Recognizes(schema.GroupVersionKind{Group: "apiregistration.k8s.io", Version: "v1", Kind: "APIService"})).To(BeTrue())
		Expect(clnt.Scheme().Recognizes(schema.GroupVersionKind{Group: "testing.cs.sap.com", Version: "v1alpha1", Kind: "Foo"})).To(BeTrue())

		err = clnt.Get(context.Background(), apitypes.NamespacedName{Name: "kube-system"}, &corev1.Namespace{})
		Expect(err).NotTo(HaveOccurred())
	})

	It("should create a functional client with impersonation (service account)", func() {
		clnt, err := factory.Get(nil, "system:serviceaccount:default:configmap-reader", nil)
		Expect(err).NotTo(HaveOccurred())

		Expect(clnt.Scheme().Recognizes(schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Namespace"})).To(BeTrue())
		Expect(clnt.Scheme().Recognizes(schema.GroupVersionKind{Group: "apiextensions.k8s.io", Version: "v1", Kind: "CustomResourceDefinition"})).To(BeTrue())
		Expect(clnt.Scheme().Recognizes(schema.GroupVersionKind{Group: "apiregistration.k8s.io", Version: "v1", Kind: "APIService"})).To(BeTrue())
		Expect(clnt.Scheme().Recognizes(schema.GroupVersionKind{Group: "testing.cs.sap.com", Version: "v1alpha1", Kind: "Foo"})).To(BeTrue())

		err = clnt.Get(context.Background(), apitypes.NamespacedName{Name: "kube-system"}, &corev1.Namespace{})
		Expect(err).To(HaveOccurred())
		Expect(apierrors.IsForbidden(err)).To(BeTrue())

		err = clnt.Get(context.Background(), apitypes.NamespacedName{Namespace: namespace, Name: "test"}, &corev1.ConfigMap{})
		Expect(err).NotTo(HaveOccurred())
	})

	It("should create a functional client with impersonation (user)", func() {
		clnt, err := factory.Get(nil, "configmap-reader", nil)
		Expect(err).NotTo(HaveOccurred())

		Expect(clnt.Scheme().Recognizes(schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Namespace"})).To(BeTrue())
		Expect(clnt.Scheme().Recognizes(schema.GroupVersionKind{Group: "apiextensions.k8s.io", Version: "v1", Kind: "CustomResourceDefinition"})).To(BeTrue())
		Expect(clnt.Scheme().Recognizes(schema.GroupVersionKind{Group: "apiregistration.k8s.io", Version: "v1", Kind: "APIService"})).To(BeTrue())
		Expect(clnt.Scheme().Recognizes(schema.GroupVersionKind{Group: "testing.cs.sap.com", Version: "v1alpha1", Kind: "Foo"})).To(BeTrue())

		err = clnt.Get(context.Background(), apitypes.NamespacedName{Name: "kube-system"}, &corev1.Namespace{})
		Expect(err).To(HaveOccurred())
		Expect(apierrors.IsForbidden(err)).To(BeTrue())

		err = clnt.Get(context.Background(), apitypes.NamespacedName{Namespace: namespace, Name: "test"}, &corev1.ConfigMap{})
		Expect(err).NotTo(HaveOccurred())
	})

	It("should create a functional client with impersonation (group)", func() {
		clnt, err := factory.Get(nil, "nobody", []string{"configmap-readers"})
		Expect(err).NotTo(HaveOccurred())

		Expect(clnt.Scheme().Recognizes(schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Namespace"})).To(BeTrue())
		Expect(clnt.Scheme().Recognizes(schema.GroupVersionKind{Group: "apiextensions.k8s.io", Version: "v1", Kind: "CustomResourceDefinition"})).To(BeTrue())
		Expect(clnt.Scheme().Recognizes(schema.GroupVersionKind{Group: "apiregistration.k8s.io", Version: "v1", Kind: "APIService"})).To(BeTrue())
		Expect(clnt.Scheme().Recognizes(schema.GroupVersionKind{Group: "testing.cs.sap.com", Version: "v1alpha1", Kind: "Foo"})).To(BeTrue())

		err = clnt.Get(context.Background(), apitypes.NamespacedName{Name: "kube-system"}, &corev1.Namespace{})
		Expect(err).To(HaveOccurred())
		Expect(apierrors.IsForbidden(err)).To(BeTrue())

		err = clnt.Get(context.Background(), apitypes.NamespacedName{Namespace: namespace, Name: "test"}, &corev1.ConfigMap{})
		Expect(err).NotTo(HaveOccurred())
	})

	It("should return cached clients and evict expired clients", func() {
		factory.validity = 5 * time.Second

		clnt, err := factory.Get(nil, "", nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(factory.clients).To(HaveLen(1))

		clnt2, err := factory.Get(nil, "", nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(factory.clients).To(HaveLen(1))
		Expect(clnt2).To(BeIdenticalTo(clnt))

		clnt, err = factory.Get([]byte(env.KubeConfig()), "", nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(factory.clients).To(HaveLen(2))

		clnt2, err = factory.Get([]byte(env.KubeConfig()), "", nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(factory.clients).To(HaveLen(2))
		Expect(clnt2).To(BeIdenticalTo(clnt))

		clnt, err = factory.Get([]byte(env.KubeConfig()), "user", nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(factory.clients).To(HaveLen(3))

		clnt2, err = factory.Get([]byte(env.KubeConfig()), "user", nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(factory.clients).To(HaveLen(3))
		Expect(clnt2).To(BeIdenticalTo(clnt))

		clnt, err = factory.Get([]byte(env.KubeConfig()), "user", []string{"group"})
		Expect(err).NotTo(HaveOccurred())
		Expect(factory.clients).To(HaveLen(4))

		clnt2, err = factory.Get([]byte(env.KubeConfig()), "user", []string{"group"})
		Expect(err).NotTo(HaveOccurred())
		Expect(factory.clients).To(HaveLen(4))
		Expect(clnt2).To(BeIdenticalTo(clnt))

		Eventually(func() int {
			return len(factory.clients)
		}, 15*time.Second, 1*time.Second).Should(Equal(0))
	})

})
