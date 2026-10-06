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
	eventsv1 "k8s.io/api/events/v1"
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
		eventsv1.AddToScheme(scheme)
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

		defaultNamespace := &corev1.Namespace{}
		err = clnt.Get(context.Background(), apitypes.NamespacedName{Name: "default"}, defaultNamespace)
		Expect(err).NotTo(HaveOccurred())

		version, err := clnt.DiscoveryClient().ServerVersion()
		Expect(err).NotTo(HaveOccurred())
		Expect(version.String()).To(Equal(env.Version().String()))
		clnt.EventRecorder().Eventf(kubeSystemNamespace, defaultNamespace, corev1.EventTypeNormal, "TestEvent", "Testing", "This is a test event")
		Eventually(func() error {
			eventList := &eventsv1.EventList{}
			selector, err := fields.ParseSelector("regarding.apiVersion=v1,regarding.kind=Namespace,regarding.name=kube-system")
			if err != nil {
				return err
			}
			err = clnt.List(context.Background(), eventList, client.InNamespace("default"), client.MatchingFieldsSelector{Selector: selector})
			if err != nil {
				return err
			}
			if slices.Any(eventList.Items, func(event eventsv1.Event) bool {
				return event.ReportingController == "test-controller" &&
					event.Related.APIVersion == "v1" &&
					event.Related.Kind == "Namespace" &&
					event.Related.Name == "default" &&
					event.Type == corev1.EventTypeNormal &&
					event.Reason == "TestEvent" &&
					event.Action == "Testing" &&
					event.Note == "This is a test event"
			}) {
				return nil
			}
			return fmt.Errorf("event not found")
		}, 10*time.Second, 1*time.Second).Should(Succeed())
	})

})
