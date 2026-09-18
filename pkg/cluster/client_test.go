/*
SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package cluster_test

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/sap/component-operator-runtime/pkg/cluster"
)

var _ = Describe("testing: cluster.go", func() {

	It("should return a valid client", func() {
		cfg := &rest.Config{}
		scheme := runtime.NewScheme()
		httpClient, err := rest.HTTPClientFor(cfg)
		Expect(err).ToNot(HaveOccurred())
		ctrlClient, err := client.New(cfg, client.Options{
			HTTPClient: httpClient,
			Scheme:     scheme,
		})
		Expect(err).ToNot(HaveOccurred())
		discoveryClient, err := discovery.NewDiscoveryClientForConfigAndClient(cfg, httpClient)
		Expect(err).ToNot(HaveOccurred())
		eventRecorder := record.NewBroadcaster().NewRecorder(nil, corev1.EventSource{})

		clnt := cluster.NewClient(ctrlClient, discoveryClient, eventRecorder, cfg, httpClient)
		Expect(clnt.Scheme()).To(BeIdenticalTo(scheme))
		Expect(clnt.DiscoveryClient()).To(BeIdenticalTo(discoveryClient))
		Expect(clnt.EventRecorder()).To(BeIdenticalTo(eventRecorder))
		Expect(clnt.Config()).To(BeIdenticalTo(cfg))
		Expect(clnt.HttpClient()).To(BeIdenticalTo(httpClient))
	})

})
