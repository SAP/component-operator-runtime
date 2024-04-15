/*
SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package testing

import (
	"bytes"
	"context"
	"sync"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	apitypes "k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/structured-merge-diff/v4/fieldpath"

	g "github.com/onsi/gomega"

	"github.com/sap/component-operator-runtime/internal/cluster"
	"github.com/sap/component-operator-runtime/pkg/types"
)

var (
	_cfg            *rest.Config
	_scheme         *runtime.Scheme
	_enhancedScheme *runtime.Scheme
	_ctx            context.Context
	_client         cluster.Client
	_mu             sync.Mutex
)

func SetupClient(cfg *rest.Config, schemeBuilder runtime.SchemeBuilder, extraSchemeBuilder runtime.SchemeBuilder, ctx context.Context) {
	_mu.Lock()
	_cfg = cfg
	_scheme = runtime.NewScheme()
	_enhancedScheme = runtime.NewScheme()
	_ctx = ctx
	utilruntime.Must(schemeBuilder.AddToScheme(_scheme))
	utilruntime.Must(schemeBuilder.AddToScheme(_enhancedScheme))
	utilruntime.Must(extraSchemeBuilder.AddToScheme(_enhancedScheme))
	httpClient, err := rest.HTTPClientFor(cfg)
	g.Expect(err).NotTo(g.HaveOccurred())
	ctrlClient, err := client.New(cfg, client.Options{HTTPClient: httpClient, Scheme: _scheme})
	g.Expect(err).NotTo(g.HaveOccurred())
	discoveryClient, err := discovery.NewDiscoveryClientForConfigAndClient(cfg, httpClient)
	g.Expect(err).NotTo(g.HaveOccurred())
	eventRecorder := &record.FakeRecorder{}
	_client = cluster.NewClient(ctrlClient, discoveryClient, eventRecorder)
}

func TeardownClient() {
	_cfg = nil
	_scheme = nil
	_enhancedScheme = nil
	_ctx = nil
	_client = nil
	_mu.Unlock()
}

func Client() cluster.Client {
	return _client
}

func Ctx() context.Context {
	return _ctx
}

func ReadObject(key types.ObjectKey) client.Object {
	object := &unstructured.Unstructured{}
	object.GetObjectKind().SetGroupVersionKind(key.GetObjectKind().GroupVersionKind())
	err := _client.Get(_ctx, apitypes.NamespacedName{Namespace: key.GetNamespace(), Name: key.GetName()}, object)
	g.Expect(err).NotTo(g.HaveOccurred())
	return object
}

func EnsureObjectExists(key types.ObjectKey, args ...any) {
	ctx, cancel := context.WithCancelCause(context.Background())
	args = append(args, ctx)
	g.Eventually(func() bool {
		gvk := key.GetObjectKind().GroupVersionKind()
		typeMeta := metav1.TypeMeta{APIVersion: gvk.GroupVersion().String(), Kind: gvk.Kind}
		object := &metav1.PartialObjectMetadata{TypeMeta: typeMeta}
		err := _client.Get(_ctx, apitypes.NamespacedName{Namespace: key.GetNamespace(), Name: key.GetName()}, object)
		if err == nil {
			return true
		} else if apierrors.IsNotFound(err) {
			return false
		} else {
			cancel(err)
			return false
		}
	}, args...).Should(g.BeTrue())
}

func EnsureObjectDoesNotExist(key types.ObjectKey, args ...any) {
	ctx, cancel := context.WithCancelCause(context.Background())
	args = append(args, ctx)
	g.Eventually(func() bool {
		gvk := key.GetObjectKind().GroupVersionKind()
		typeMeta := metav1.TypeMeta{APIVersion: gvk.GroupVersion().String(), Kind: gvk.Kind}
		object := &metav1.PartialObjectMetadata{TypeMeta: typeMeta}
		err := _client.Get(_ctx, apitypes.NamespacedName{Namespace: key.GetNamespace(), Name: key.GetName()}, object)
		if err == nil {
			return false
		} else if apierrors.IsNotFound(err) {
			return true
		} else {
			cancel(err)
			return false
		}
	}, args...).Should(g.BeTrue())
}

func CreateObject(object client.Object, fieldManager string) {
	gvk := gvkForObject(object)
	if _scheme.Recognizes(gvk) {
		// object.GetObjectKind().SetGroupVersionKind(gvk)
		err := _client.Create(_ctx, object, client.FieldOwner(fieldManager))
		g.Expect(err).NotTo(g.HaveOccurred())
		object.GetObjectKind().SetGroupVersionKind(gvk)
	} else {
		obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(object)
		g.Expect(err).NotTo(g.HaveOccurred())
		unstructured := &unstructured.Unstructured{Object: obj}
		unstructured.GetObjectKind().SetGroupVersionKind(gvk)
		err = _client.Create(_ctx, unstructured, client.FieldOwner(fieldManager))
		g.Expect(err).NotTo(g.HaveOccurred())
		err = runtime.DefaultUnstructuredConverter.FromUnstructured(unstructured.Object, object)
		g.Expect(err).NotTo(g.HaveOccurred())
	}
}

func ApplyObject(object client.Object, fieldManager string) {
	gvk := gvkForObject(object)
	if _scheme.Recognizes(gvk) {
		object.GetObjectKind().SetGroupVersionKind(gvk)
		err := _client.Patch(_ctx, object, client.Apply, client.FieldOwner(fieldManager))
		g.Expect(err).NotTo(g.HaveOccurred())
		object.GetObjectKind().SetGroupVersionKind(gvk)
	} else {
		obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(object)
		g.Expect(err).NotTo(g.HaveOccurred())
		unstructured := &unstructured.Unstructured{Object: obj}
		unstructured.GetObjectKind().SetGroupVersionKind(gvk)
		err = _client.Patch(_ctx, unstructured, client.Apply, client.FieldOwner(fieldManager))
		g.Expect(err).NotTo(g.HaveOccurred())
		err = runtime.DefaultUnstructuredConverter.FromUnstructured(unstructured.Object, object)
		g.Expect(err).NotTo(g.HaveOccurred())
	}
}

func CreateNamespace() string {
	namespace := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{GenerateName: "ns-"}}
	err := _client.Create(_ctx, namespace)
	g.Expect(err).NotTo(g.HaveOccurred())
	return namespace.Name
}

func CleanupNamespace(namespace string, objects ...client.Object) {
	for _, object := range objects {
		gvk := gvkForObject(object)
		typeMeta := metav1.TypeMeta{APIVersion: gvk.GroupVersion().String(), Kind: gvk.Kind}
		object = &metav1.PartialObjectMetadata{TypeMeta: typeMeta}
		err := _client.DeleteAllOf(_ctx, object, client.InNamespace(namespace), client.PropagationPolicy(metav1.DeletePropagationBackground))
		g.Expect(err).NotTo(g.HaveOccurred())
		for i := 1; i <= 2; i++ {
			// confirm that they all are deleted (actually they should be due to background deletion, but just to be completely sure...)
			objectList := &unstructured.UnstructuredList{}
			objectList.SetGroupVersionKind(gvk)
			err = _client.List(_ctx, objectList, client.InNamespace(namespace))
			g.Expect(err).NotTo(g.HaveOccurred())
			if len(objectList.Items) > 0 && i == 1 {
				for _, object := range objectList.Items {
					object.SetFinalizers(nil)
					err := _client.Update(_ctx, &object)
					g.Expect(err).NotTo(g.HaveOccurred())
				}
				continue
			}
			g.Expect(objectList.Items).To(g.BeEmpty())
			break
		}
	}
}

/*
// deleting namespaces anyway does not work in envtest (unless we forcefully clear the 'kubernetes' finalizer)
func deleteNamespace(namespace string) {
	err := cli.Delete(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}})
	g.Expect(err).NotTo(g.HaveOccurred())
}
*/

func FieldManagersFor(object client.Object, parts ...any) []string {
	var managers []string
	managedFields := object.GetManagedFields()
	for _, entry := range managedFields {
		set := fieldpath.Set{}
		err := set.FromJSON(bytes.NewReader(entry.FieldsV1.Raw))
		g.Expect(err).NotTo(g.HaveOccurred())
		set.Iterate(func(path fieldpath.Path) {
			if path.Equals(fieldpath.MakePathOrDie(parts...)) {
				managers = append(managers, entry.Manager)
			}
		})
	}
	return managers
}

func AsUnstructured(object client.Object) *unstructured.Unstructured {
	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(object)
	g.Expect(err).NotTo(g.HaveOccurred())
	return &unstructured.Unstructured{Object: obj}
}

func gvkForObject(object client.Object) schema.GroupVersionKind {
	gvkFromObject := object.GetObjectKind().GroupVersionKind()
	gvkFromScheme, err := apiutil.GVKForObject(object, _enhancedScheme)
	if gvkFromObject.Empty() {
		g.Expect(err).NotTo(g.HaveOccurred())
		return gvkFromScheme
	} else {
		if err == nil {
			g.Expect(gvkFromObject).To(g.Equal(gvkFromScheme))
		}
		return gvkFromObject
	}
}

func KubeConfig() *clientcmdapi.Config {
	apiConfig := clientcmdapi.NewConfig()

	apiConfig.Clusters["envtest"] = clientcmdapi.NewCluster()
	cluster := apiConfig.Clusters["envtest"]
	cluster.Server = _cfg.Host
	cluster.CertificateAuthorityData = _cfg.CAData

	apiConfig.AuthInfos["envtest"] = clientcmdapi.NewAuthInfo()
	authInfo := apiConfig.AuthInfos["envtest"]
	authInfo.ClientKeyData = _cfg.KeyData
	authInfo.ClientCertificateData = _cfg.CertData

	apiConfig.Contexts["envtest"] = clientcmdapi.NewContext()
	context := apiConfig.Contexts["envtest"]
	context.Cluster = "envtest"
	context.AuthInfo = "envtest"

	apiConfig.CurrentContext = "envtest"

	return apiConfig
}
