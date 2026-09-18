/*
SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package clientfactory

import (
	"context"
	"fmt"
	"time"

	"github.com/sap/go-generics/slices"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	apitypes "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("testing: client.go", func() {

	var scheme *runtime.Scheme

	BeforeEach(func() {
		scheme = runtime.NewScheme()
		corev1.AddToScheme(scheme)
	})

	It("should create a functional client", func() {
		clnt, err := NewClientFor(env.Config(), scheme, "test-controller")
		Expect(err).NotTo(HaveOccurred())

		Expect(clnt.Config()).To(Equal(env.Config()))

		resp, err := clnt.HttpClient().Get(fmt.Sprintf("%sversion", env.Config().Host))
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(200))

		kubeSystemNamespace := &corev1.Namespace{}
		err = clnt.Get(context.Background(), apitypes.NamespacedName{Name: "kube-system"}, kubeSystemNamespace)
		Expect(err).NotTo(HaveOccurred())

		version, err := clnt.DiscoveryClient().ServerVersion()
		Expect(err).NotTo(HaveOccurred())
		Expect(version.String()).To(Equal(env.Version().String()))
		clnt.EventRecorder().Event(kubeSystemNamespace, corev1.EventTypeNormal, "TestEvent", "This is a test event")
		Eventually(func() error {
			eventList := &corev1.EventList{}
			selector, err := fields.ParseSelector("involvedObject.apiVersion=v1,involvedObject.kind=Namespace,involvedObject.name=kube-system")
			if err != nil {
				return err
			}
			err = clnt.List(context.Background(), eventList, client.InNamespace("default"), client.MatchingFieldsSelector{Selector: selector})
			if err != nil {
				return err
			}
			if slices.Any(eventList.Items, func(event corev1.Event) bool {
				return event.Source.Component == "test-controller" && event.Reason == "TestEvent" && event.Message == "This is a test event"
			}) {
				return nil
			}
			return fmt.Errorf("event not found")
		}, 10*time.Second, 1*time.Second).Should(Succeed())
	})

})
