/*
SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package reconciler_test

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	apiregistrationv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "github.com/sap/component-operator-runtime/internal/testing"

	barv1alpha1 "github.com/sap/component-operator-runtime/internal/testing/bar/api/v1alpha1"
	foov1alpha1 "github.com/sap/component-operator-runtime/internal/testing/foo/api/v1alpha1"
)

func TestComponent(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Component Suite")
}

var testEnv *envtest.Environment
var cfg *rest.Config
var ctx context.Context
var cancel context.CancelFunc
var threads sync.WaitGroup
var tmpdir string

var _ = BeforeSuite(func() {
	var err error

	By("initializing")
	log.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))
	ctx, cancel = context.WithCancel(context.TODO())
	tmpdir, err = os.MkdirTemp("", "")
	Expect(err).NotTo(HaveOccurred())

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{
			"../../internal/testing/foo/crds",
			"../../internal/testing/bar/crds",
			"../../internal/testing/mycomponent/crds",
		},
		ErrorIfCRDPathMissing: true,
		//WebhookInstallOptions: envtest.WebhookInstallOptions{
		//	ValidatingWebhooks: []*admissionv1.ValidatingWebhookConfiguration{
		//		buildValidatingWebhookConfiguration(),
		//	},
		//},
	}
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())
	// webhookInstallOptions := &testEnv.WebhookInstallOptions

	By("initializing client")
	SetupClient(
		cfg,
		runtime.NewSchemeBuilder(clientgoscheme.AddToScheme, apiextensionsv1.AddToScheme, apiregistrationv1.AddToScheme, foov1alpha1.AddToScheme),
		runtime.NewSchemeBuilder(barv1alpha1.AddToScheme),
		ctx,
	)
	err = clientcmd.WriteToFile(*KubeConfig(), fmt.Sprintf("%s/kubeconfig", tmpdir))
	Expect(err).NotTo(HaveOccurred())
	fmt.Printf("A temporary kubeconfig for the envtest environment can be found here: %s/kubeconfig\n", tmpdir)

	/*
		By("creating manager")
		mgr, err := ctrl.NewManager(cfg, ctrl.Options{
			Scheme: scheme,
			Client: client.Options{
				Cache: &client.CacheOptions{
					DisableFor: append(operator.GetUncacheableTypes(), &apiextensionsv1.CustomResourceDefinition{}, &apiregistrationv1.APIService{}),
				},
			},
			WebhookServer: webhook.NewServer(webhook.Options{
				Host:    webhookInstallOptions.LocalServingHost,
				Port:    webhookInstallOptions.LocalServingPort,
				CertDir: webhookInstallOptions.LocalServingCertDir,
			}),
			Metrics: metricsserver.Options{
				BindAddress: "0",
			},
			HealthProbeBindAddress: "0",
		})
		Expect(err).NotTo(HaveOccurred())

		discoveryClient, err := discovery.NewDiscoveryClientForConfig(mgr.GetConfig())
		Expect(err).NotTo(HaveOccurred())

		err = operator.Setup(mgr, discoveryClient)
		Expect(err).NotTo(HaveOccurred())

		By("starting dummy controller-manager")
		threads.Add(1)
		go func() {
			defer threads.Done()
			defer GinkgoRecover()
			// since there is no controller-manager in envtest, we fake all statefulsets into a ready state
			// such that kstatus recognizes them as 'Current'
			for {
				select {
				case <-ctx.Done():
					return
				case <-time.After(100 * time.Millisecond):
					statefulSetList := &appsv1.StatefulSetList{}
					err := cli.List(context.Background(), statefulSetList)
					Expect(err).NotTo(HaveOccurred())
					for _, statefulSet := range statefulSetList.Items {
						if statefulSet.DeletionTimestamp.IsZero() {
							status := &statefulSet.Status
							oldStatus := status.DeepCopy()
							status.ObservedGeneration = statefulSet.Generation
							status.Replicas = *statefulSet.Spec.Replicas
							status.ReadyReplicas = *statefulSet.Spec.Replicas
							status.AvailableReplicas = *statefulSet.Spec.Replicas
							status.CurrentReplicas = *statefulSet.Spec.Replicas
							status.UpdatedReplicas = *statefulSet.Spec.Replicas
							if reflect.DeepEqual(status, oldStatus) {
								continue
							}
							err = cli.Status().Update(context.Background(), &statefulSet)
							if apierrors.IsNotFound(err) || apierrors.IsConflict(err) {
								err = nil
							}
							Expect(err).NotTo(HaveOccurred())
						} else {
							if controllerutil.RemoveFinalizer(&statefulSet, metav1.FinalizerDeleteDependents) {
								err = cli.Update(context.Background(), &statefulSet)
								if apierrors.IsNotFound(err) || apierrors.IsConflict(err) {
									err = nil
								}
								Expect(err).NotTo(HaveOccurred())
							}
						}
					}
				}
			}
		}()

		By("starting manager")
		threads.Add(1)
		go func() {
			defer threads.Done()
			defer GinkgoRecover()
			err := mgr.Start(ctx)
			Expect(err).NotTo(HaveOccurred())
		}()

		By("waiting for operator to become ready")
		Eventually(func() error { return mgr.GetWebhookServer().StartedChecker()(nil) }, "10s", "100ms").Should(Succeed())
	*/
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	defer TeardownClient()
	cancel()
	threads.Wait()
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
	err = os.RemoveAll(tmpdir)
	Expect(err).NotTo(HaveOccurred())
})
