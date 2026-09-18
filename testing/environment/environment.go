/*
SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package environment

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/sap/go-generics/slices"

	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	apitypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/discovery"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/retry"
	apiregistrationv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/sap/component-operator-runtime/internal/util"
	"github.com/sap/component-operator-runtime/pkg/types"
	cstestingv1alpha1 "github.com/sap/component-operator-runtime/testing/environment/apis/testing.cs.sap.com/v1alpha1"
)

type Environment struct {
	testenv         *envtest.Environment
	config          *rest.Config
	kubeConfig      string
	scheme          *runtime.Scheme
	client          client.Client
	discoveryClient discovery.DiscoveryInterface
	version         *version.Info
	id              string
	tmpdir          string
}

func Run(stdout io.Writer, stderr io.Writer, logger io.Writer) (_ *Environment, err error) {
	log.SetLogger(zap.New(zap.WriteTo(logger), zap.UseDevMode(true)))

	var tmpdir string
	var testenv *envtest.Environment

	defer func() {
		if err == nil {
			return
		}
		if testenv != nil {
			if stopErr := testenv.Stop(); stopErr != nil {
				err = errors.Join(err, stopErr)
			}
		}
		if tmpdir != "" {
			if tmpdirErr := os.RemoveAll(tmpdir); tmpdirErr != nil {
				err = errors.Join(err, tmpdirErr)
			}
		}
	}()

	tmpdir, err = os.MkdirTemp("", "")
	if err != nil {
		return nil, err
	}

	testenv = &envtest.Environment{
		CRDs: []*apiextensionsv1.CustomResourceDefinition{
			CRDS["foos.testing.cs.sap.com"],
			CRDS["bars.testing.cs.sap.com"],
		},
	}

	cfg, err := testenv.Start()
	if err != nil {
		return nil, err
	}

	kubeConfig := buildKubeConfig(cfg)
	kubeConfigRaw, err := clientcmd.Write(*kubeConfig)
	if err != nil {
		return nil, err
	}

	scheme := runtime.NewScheme()
	clientgoscheme.AddToScheme(scheme)
	apiextensionsv1.AddToScheme(scheme)
	apiregistrationv1.AddToScheme(scheme)
	cstestingv1alpha1.AddToScheme(scheme)

	httpClient, err := rest.HTTPClientFor(cfg)
	if err != nil {
		return nil, err
	}

	clnt, err := client.New(cfg, client.Options{
		HTTPClient: httpClient,
		Scheme:     scheme,
	})
	if err != nil {
		return nil, err
	}

	discoveryClient, err := discovery.NewDiscoveryClientForConfigAndClient(cfg, httpClient)
	if err != nil {
		return nil, err
	}

	version, err := discoveryClient.ServerVersion()
	if err != nil {
		return nil, err
	}

	fmt.Fprintf(stdout, "Kubernetes version: %s\n", version.String())

	kubeSystemNamespace := &corev1.Namespace{}
	if err := clnt.Get(context.Background(), apitypes.NamespacedName{Name: "kube-system"}, kubeSystemNamespace); err != nil {
		return nil, err
	}
	id := string(kubeSystemNamespace.GetUID())

	if err := clientcmd.WriteToFile(*kubeConfig, fmt.Sprintf("%s/kubeconfig", tmpdir)); err != nil {
		return nil, err
	}
	fmt.Fprintf(stdout, "A temporary kubeconfig for the envtest environment can be found here: %s/kubeconfig\n", tmpdir)

	return &Environment{
		testenv:         testenv,
		config:          cfg,
		kubeConfig:      string(kubeConfigRaw),
		scheme:          scheme,
		client:          clnt,
		discoveryClient: discoveryClient,
		version:         version,
		id:              id,
		tmpdir:          tmpdir,
	}, nil
}

func (e *Environment) Stop() error {
	stopErr := e.testenv.Stop()
	tmpdirErr := os.RemoveAll(e.tmpdir)
	return errors.Join(stopErr, tmpdirErr)
}

func (e *Environment) Config() *rest.Config {
	return e.config
}

func (e *Environment) KubeConfig() string {
	return e.kubeConfig
}

func (e *Environment) Scheme() *runtime.Scheme {
	return e.scheme
}

func (e *Environment) Client() client.Client {
	return e.client
}

func (e *Environment) DiscoveryClient() discovery.DiscoveryInterface {
	return e.discoveryClient
}

func (e *Environment) Version() *version.Info {
	return e.version
}

func (e *Environment) Id() string {
	return e.id
}

func (e *Environment) EnsureObjectExists(obj client.Object, reconcilerName, ownerId string, digest string) (client.Object, error) {
	obj = obj.DeepCopyObject().(client.Object)
	if err := e.client.Get(context.Background(), client.ObjectKeyFromObject(obj), obj); err != nil {
		return nil, err
	}
	annotations := obj.GetAnnotations()
	labels := obj.GetLabels()

	if ownerId != "" && annotations[fmt.Sprintf("%s/%s", reconcilerName, types.AnnotationKeySuffixOwnerId)] != ownerId {
		return nil, fmt.Errorf("object %s/%s is not owned by %s", obj.GetNamespace(), obj.GetName(), ownerId)
	}
	if digest != "" && annotations[fmt.Sprintf("%s/%s", reconcilerName, types.AnnotationKeySuffixDigest)] != digest {
		return nil, fmt.Errorf("object %s/%s does not have the expected component digest", obj.GetNamespace(), obj.GetName())
	}
	if ownerId != "" && labels[fmt.Sprintf("%s/%s", reconcilerName, types.LabelKeySuffixOwnerId)] != util.Sha256base32([]byte(ownerId)) {
		return nil, fmt.Errorf("object %s/%s does not have the expected owner label", obj.GetNamespace(), obj.GetName())
	}
	return obj, nil
}

func (e *Environment) EnsureObjectDoesNotExist(obj client.Object) error {
	obj = obj.DeepCopyObject().(client.Object)
	if err := e.client.Get(context.Background(), client.ObjectKeyFromObject(obj), obj); err == nil {
		return fmt.Errorf("object %s/%s still exists", obj.GetNamespace(), obj.GetName())
	} else if !apierrors.IsNotFound(err) {
		return err
	}
	return nil
}

func (e *Environment) CreateObject(obj client.Object) error {
	return e.client.Create(context.Background(), obj)
}

func (e *Environment) ApplyObject(obj client.Object, fieldOwner string) error {
	return e.client.Patch(context.Background(), obj, client.Apply, client.FieldOwner(fieldOwner), client.ForceOwnership)
}

func (e *Environment) CreateNamespace() (string, error) {
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-",
		},
	}
	if err := e.client.Create(context.Background(), namespace); err != nil {
		return "", err
	}
	return namespace.Name, nil
}

type ObserveableObject interface {
	client.Object
	SetObservedGeneration(generation int64)
	SetCondition(condition metav1.Condition)
}

func (e *Environment) Observe(obj ObserveableObject, status metav1.ConditionStatus) error {
	obj = obj.DeepCopyObject().(ObserveableObject)
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		if err := e.client.Get(context.Background(), client.ObjectKeyFromObject(obj), obj); err != nil {
			return err
		}
		obj.SetObservedGeneration(obj.GetGeneration())
		obj.SetCondition(metav1.Condition{
			Type:               "Ready",
			Status:             status,
			ObservedGeneration: obj.GetGeneration(),
			LastTransitionTime: metav1.Now(),
			Reason:             "Observed",
			Message:            fmt.Sprintf("Object is observed with status %s", status),
		})
		if err := e.client.Status().Update(context.Background(), obj); err != nil {
			return err
		}
		return nil
	})
}

func (e *Environment) Finalize(obj client.Object, finalizer string) error {
	obj = obj.DeepCopyObject().(client.Object)
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		if err := e.client.Get(context.Background(), client.ObjectKeyFromObject(obj), obj); err != nil {
			return err
		}
		if controllerutil.RemoveFinalizer(obj, finalizer) {
			if err := e.client.Update(context.Background(), obj); err != nil {
				return err
			}
		}
		return nil
	})
}

func (e *Environment) CleanupObject(ctx context.Context, obj client.Object) error {
	obj = obj.DeepCopyObject().(client.Object)

	if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		if err := e.client.Get(context.Background(), client.ObjectKeyFromObject(obj), obj); err != nil {
			if apierrors.IsNotFound(err) {
				return nil
			}
			return err
		}

		if len(obj.GetFinalizers()) > 0 {
			obj.SetFinalizers(nil)
			if err := e.client.Update(context.Background(), obj); err != nil {
				return err
			}
		}

		if err := e.client.Delete(context.Background(), obj, client.Preconditions{ResourceVersion: new(obj.GetResourceVersion())}); err != nil {
			if apierrors.IsNotFound(err) {
				return nil
			}
			return err
		}

		return nil
	}); err != nil {
		return err
	}

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		if err := e.client.Get(context.Background(), client.ObjectKeyFromObject(obj), obj); err != nil {
			if apierrors.IsNotFound(err) {
				return nil
			}
			return err
		}
		select {
		case <-ticker.C:
			continue
		case <-ctx.Done():
			return fmt.Errorf("context cancelled while waiting for object: %w", ctx.Err())
		}
	}
}

func (e *Environment) CleanupObjects(ctx context.Context, objs ...client.Object) error {
	var merr error
	objs = slices.Collect(objs, func(obj client.Object) client.Object {
		obj = obj.DeepCopyObject().(client.Object)
		gvk, err := apiutil.GVKForObject(obj, e.scheme)
		if err != nil {
			merr = errors.Join(merr, err)
		}
		obj.GetObjectKind().SetGroupVersionKind(gvk)
		return obj
	})
	if merr != nil {
		return merr
	}

	objs = slices.SortBy(objs, func(obj1, obj2 client.Object) bool {
		order := func(gvk schema.GroupVersionKind) int {
			switch gvk {
			case schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Namespace"}:
				return 2
			case schema.GroupVersionKind{Group: "apiextensions.k8s.io", Version: "v1", Kind: "CustomResourceDefinition"}:
				return 1
			default:
				return 0
			}
		}
		return order(obj1.GetObjectKind().GroupVersionKind()) > order(obj2.GetObjectKind().GroupVersionKind())
	})

	for _, obj := range objs {
		if err := e.CleanupObject(ctx, obj); err != nil {
			return err
		}
	}

	return nil
}
