/*
SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package component_test

import (
	// "context"

	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	apiregistrationv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	// "github.com/sap/component-operator-runtime/internal/cluster"

	. "github.com/sap/component-operator-runtime/internal/testing"
	mycomponentv1alpha1 "github.com/sap/component-operator-runtime/internal/testing/mycomponent/api/v1alpha1"
	"github.com/sap/component-operator-runtime/pkg/component"
)

var _ = Describe("testing: target.go", func() {
	var reconcilerName string
	// var ctx context.Context
	// var cli cluster.Client

	BeforeEach(func() {
		reconcilerName = "reconciler.testing.local"
		// ctx = Ctx()
		// cli = Client()
	})

	Context("testing: Reconcile()", func() {
		var reconciler *component.Reconciler[*mycomponentv1alpha1.MyComponent]
		var namespace string
		var ctx context.Context
		var cancel context.CancelFunc

		BeforeEach(func() {
			ctx, cancel = context.WithCancel(Ctx())
			scheme := runtime.NewScheme()
			utilruntime.Must(clientgoscheme.AddToScheme(scheme))
			utilruntime.Must(apiextensionsv1.AddToScheme(scheme))
			utilruntime.Must(apiregistrationv1.AddToScheme(scheme))
			utilruntime.Must(mycomponentv1alpha1.AddToScheme(scheme))

			mgr, err := ctrl.NewManager(cfg, ctrl.Options{
				Scheme: scheme,
				Client: client.Options{
					Cache: &client.CacheOptions{
						DisableFor: []client.Object{&mycomponentv1alpha1.MyComponent{}, &apiextensionsv1.CustomResourceDefinition{}, &apiregistrationv1.APIService{}},
					},
				},
				Metrics: metricsserver.Options{
					BindAddress: "0",
				},
				HealthProbeBindAddress: "0",
			})
			Expect(err).NotTo(HaveOccurred())
			reconciler = component.NewReconciler[*mycomponentv1alpha1.MyComponent](
				reconcilerName,
				nil,
				component.ReconcilerOptions{},
			)
			err = reconciler.SetupWithManager(mgr)
			Expect(err).NotTo(HaveOccurred())
			By("starting manager")
			threads.Add(1)
			go func() {
				defer threads.Done()
				defer GinkgoRecover()
				err := mgr.Start(ctx)
				Expect(err).NotTo(HaveOccurred())
			}()
			namespace = CreateNamespace()
		})

		AfterEach(func() {
			CleanupNamespace(namespace)
			cancel()
		})

		It("xxx", func() {
		})
	})
})
