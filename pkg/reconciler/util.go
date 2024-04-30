/*
SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package reconciler

import (
	"crypto/sha256"
	"encoding/base32"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/sap/go-generics/slices"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	apiregistrationv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"

	"github.com/sap/component-operator-runtime/pkg/types"
)

// TODO: consolidate all the util files into an internal reuse package

func ref[T any](x T) *T {
	return &x
}

func must[T any](x T, err error) T {
	if err != nil {
		panic(err)
	}
	return x
}

func sha256hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func sha256base32(data []byte) string {
	sum := sha256.Sum256(data)
	return strings.ToLower(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(sum[:]))
}

func checkRange(x int, min int, max int) error {
	if x < min || x > max {
		return fmt.Errorf("value %d not in allowed range [%d,%d]", x, min, max)
	}
	return nil
}

func calculateObjectDigest(obj client.Object, item *InventoryItem, revision int64, reconcilePolicy ReconcilePolicy) (string, error) {
	if reconcilePolicy == ReconcilePolicyOnce {
		return "__once__", nil
	}

	resourceVersion := obj.GetResourceVersion()
	defer obj.SetResourceVersion(resourceVersion)
	obj.SetResourceVersion("")
	generation := obj.GetGeneration()
	defer obj.SetGeneration(generation)
	obj.SetGeneration(0)
	managedFields := obj.GetManagedFields()
	defer obj.SetManagedFields(managedFields)
	obj.SetManagedFields(nil)
	raw, err := json.Marshal(obj)
	if err != nil {
		return "", errors.Wrapf(err, "error serializing object %s", types.ObjectKeyToString(obj))
	}
	digest := sha256hex(raw)

	if reconcilePolicy == ReconcilePolicyOnObjectOrComponentChange {
		digest = fmt.Sprintf("%s@%d", digest, revision)
	}

	previousDigest := ""
	previousTimestamp := int64(0)
	if item != nil {
		if m := regexp.MustCompile(`^([0-9a-f@]+):(\d{10})$`).FindStringSubmatch(item.Digest); m != nil {
			previousDigest = m[1]
			previousTimestamp = must(strconv.ParseInt(m[2], 10, 64))
		}
	}
	now := time.Now().Unix()
	timestamp := int64(0)
	if previousDigest == digest && now-previousTimestamp <= 3600 {
		timestamp = previousTimestamp
	} else {
		timestamp = now
	}

	return fmt.Sprintf("%s:%d", digest, timestamp), nil
}

func setLabel(obj client.Object, key string, value string) {
	labels := obj.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}
	labels[key] = value
	obj.SetLabels(labels)
}

func removeLabel(obj client.Object, key string) {
	labels := obj.GetLabels()
	delete(labels, key)
	obj.SetLabels(labels)
}

func setAnnotation(obj client.Object, key string, value string) {
	annotations := obj.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}
	annotations[key] = value
	obj.SetAnnotations(annotations)
}

func removeAnnotation(obj client.Object, key string) {
	annotations := obj.GetAnnotations()
	delete(annotations, key)
	obj.SetAnnotations(annotations)
}

func isNamespace(key types.ObjectKey) bool {
	return key.GetObjectKind().GroupVersionKind().GroupKind() == schema.GroupKind{Group: "", Kind: "Namespace"}
}

func isCrd(key types.ObjectKey) bool {
	return key.GetObjectKind().GroupVersionKind().GroupKind() == schema.GroupKind{Group: "apiextensions.k8s.io", Kind: "CustomResourceDefinition"}
}

func isApiService(key types.ObjectKey) bool {
	return key.GetObjectKind().GroupVersionKind().GroupKind() == schema.GroupKind{Group: "apiregistration.k8s.io", Kind: "APIService"}
}

func getCrds(objects []client.Object) []*apiextensionsv1.CustomResourceDefinition {
	var crds []*apiextensionsv1.CustomResourceDefinition
	for _, object := range objects {
		if !isCrd(object) {
			continue
		}
		if crd, ok := object.(*apiextensionsv1.CustomResourceDefinition); ok {
			crds = append(crds, crd)
		} else {
			// note: this panic relies on v1 being the only version in which apiextensions.k8s.io/CustomResourceDefinition is available in the cluster
			panic("this cannot happen")
		}
	}
	return crds
}

func getApiServices(objects []client.Object) []*apiregistrationv1.APIService {
	var apiServices []*apiregistrationv1.APIService
	for _, object := range objects {
		if !isApiService(object) {
			continue
		}
		if apiService, ok := object.(*apiregistrationv1.APIService); ok {
			apiServices = append(apiServices, apiService)
		} else {
			// note: this panic relies on v1 being the only version in which apiregistration.k8s.io/APIService is available in the cluster
			panic("this cannot happen")
		}
	}
	return apiServices
}

func getManagedTypes(object client.Object) []TypeInfo {
	switch {
	case isCrd(object):
		switch crd := object.(type) {
		case *apiextensionsv1.CustomResourceDefinition:
			return []TypeInfo{{Group: crd.Spec.Group, Version: "*", Kind: crd.Spec.Names.Kind}}
		default:
			// note: this panic relies on v1 being the only version in which apiextensions.k8s.io/CustomResourceDefinition is available in the cluster
			panic("this cannot happen")
		}
	case isApiService(object):
		switch apiService := object.(type) {
		case *apiregistrationv1.APIService:
			return []TypeInfo{{Group: apiService.Spec.Group, Version: apiService.Spec.Version, Kind: "*"}}
		default:
			// note: this panic relies on v1 being the only version in which apiregistration.k8s.io/APIService is available in the cluster
			panic("this cannot happen")
		}
	default:
		return nil
	}
}

func findMissingNamespaces(objects []client.Object) []string {
	var namespaces []string
	for _, object := range objects {
		namespace := object.GetNamespace()
		if namespace != "" && !slices.Contains(namespaces, namespace) {
			found := false
			for _, obj := range objects {
				if isNamespace(obj) && obj.GetName() == namespace {
					found = true
					break
				}
			}
			if !found {
				namespaces = append(namespaces, namespace)
			}
		}
	}
	return namespaces
}

func scopeFromRestMapping(mapping *apimeta.RESTMapping) int {
	switch mapping.Scope.Name() {
	case apimeta.RESTScopeNameNamespace:
		return scopeNamespaced
	case apimeta.RESTScopeNameRoot:
		return scopeCluster
	default:
		panic("this cannot happen")
	}
}

func scopeFromCrd(crd *apiextensionsv1.CustomResourceDefinition) int {
	switch crd.Spec.Scope {
	case apiextensionsv1.NamespaceScoped:
		return scopeNamespaced
	case apiextensionsv1.ClusterScoped:
		return scopeCluster
	default:
		panic("this cannot happen")
	}
}

func normalizeObjects(objects []client.Object, scheme *runtime.Scheme) ([]client.Object, error) {
	normalizedObjects := make([]client.Object, len(objects))
	for i, object := range objects {
		gvk := object.GetObjectKind().GroupVersionKind()
		if unstructuredObject, ok := object.(*unstructured.Unstructured); ok {
			if gvk.Version == "" || gvk.Kind == "" {
				return nil, fmt.Errorf("unstructured object %s is missing type information", types.ObjectKeyToString(object))
			}
			if scheme.Recognizes(gvk) {
				typedObject, err := scheme.New(gvk)
				if err != nil {
					return nil, errors.Wrapf(err, "error instantiating type for object %s", types.ObjectKeyToString(object))
				}
				if typedObject, ok := typedObject.(client.Object); ok {
					if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObject.Object, typedObject); err != nil {
						return nil, errors.Wrapf(err, "error converting object %s", types.ObjectKeyToString(object))
					}
					normalizedObjects[i] = typedObject
				} else {
					return nil, errors.Wrapf(err, "error instantiating type for object %s", types.ObjectKeyToString(object))
				}
			} else if isCrd(object) || isApiService(object) {
				return nil, fmt.Errorf("scheme does not recognize type of object %s", types.ObjectKeyToString(object))
			} else {
				normalizedObjects[i] = unstructuredObject.DeepCopy()
			}
		} else {
			_gvk, err := apiutil.GVKForObject(object, scheme)
			if err != nil {
				return nil, errors.Wrapf(err, "error retrieving scheme type information for object %s", types.ObjectKeyToString(object))
			}
			object = object.DeepCopyObject().(client.Object)
			if gvk.Version == "" || gvk.Kind == "" {
				object.GetObjectKind().SetGroupVersionKind(_gvk)
			} else if gvk != _gvk {
				return nil, fmt.Errorf("object %s specifies inconsistent type information (expected: %s)", types.ObjectKeyToString(object), _gvk)
			}
			normalizedObjects[i] = object
		}
	}
	return normalizedObjects, nil
}

func sortObjectsForApply[T client.Object](s []T, orderFunc func(client.Object) int) []T {
	priority := map[string]int{
		"Namespace": -4,
		"ValidatingWebhookConfiguration.admissionregistration.k8s.io": -3,
		"MutatingWebhookConfiguration.admissionregistration.k8s.io":   -3,
		"CustomResourceDefinition.apiextensions.k8s.io":               -2,
		"IngressClass.networking.k8s.io":                              -2,
		"RuntimeClass.node.k8s.io":                                    -2,
		"PriorityClass.scheduling.k8s.io":                             -2,
		"StorageClass.storage.k8s.io":                                 -2,
		"ConfigMap":                                                   -1,
		"Secret":                                                      -1,
		"ClusterRole.rbac.authorization.k8s.io":                       -1,
		"Role.rbac.authorization.k8s.io":                              -1,
		"ClusterRoleBinding.rbac.authorization.k8s.io":                -1,
		"RoleBinding.rbac.authorization.k8s.io":                       -1,
		"APIService.apiregistration.k8s.io":                           1,
	}
	f := func(x T, y T) bool {
		orderx := orderFunc(x)
		ordery := orderFunc(y)
		gvx := x.GetObjectKind().GroupVersionKind().GroupKind().String()
		gvy := y.GetObjectKind().GroupVersionKind().GroupKind().String()
		return orderx > ordery || orderx == ordery && priority[gvx] > priority[gvy]
	}
	return slices.SortBy(s, f)
}

func sortObjectsForDelete(inventory []*InventoryItem) []*InventoryItem {
	priority := map[string]int{
		"CustomResourceDefinition.apiextensions.k8s.io": -1,
		"APIService.apiregistration.k8s.io":             -1,
		// TODO: should webhook configurations be deleted before order-zero objects?
		"ValidatingWebhookConfiguration.admissionregistration.k8s.io": 1,
		"MutatingWebhookConfiguration.admissionregistration.k8s.io":   1,
		"Service":                         2,
		"ConfigMap":                       2,
		"Secret":                          2,
		"Namespace":                       3,
		"IngressClass.networking.k8s.io":  4,
		"RuntimeClass.node.k8s.io":        4,
		"PriorityClass.scheduling.k8s.io": 4,
		"StorageClass.storage.k8s.io":     4,
	}
	f := func(x *InventoryItem, y *InventoryItem) bool {
		orderx := x.DeleteOrder
		ordery := y.DeleteOrder
		gvx := x.GroupVersionKind().GroupKind().String()
		gvy := y.GroupVersionKind().GroupKind().String()
		return orderx > ordery || orderx == ordery && priority[gvx] > priority[gvy]
	}
	return slices.SortBy(inventory, f)
}

func getItem(inventory []*InventoryItem, key types.ObjectKey) *InventoryItem {
	var item *InventoryItem
	for _, _item := range inventory {
		if _item.Matches(key) {
			if item != nil {
				// panic if there is more than one matching item in the inventory;
				// although this is technically possible, it would indicate a severe issue elsewhere
				panic("this cannot happen")
			}
			item = _item
		}
	}
	return item
}

func mustGetItem(inventory []*InventoryItem, key types.ObjectKey) *InventoryItem {
	item := getItem(inventory, key)
	if item == nil {
		panic("this cannot happen")
	}
	return item
}

func isNamespaceUsed(inventory []*InventoryItem, namespace string) bool {
	// TODO: do not consider inventory items with certain Phases (e.g. Completed)?
	for _, item := range inventory {
		if item.Namespace == namespace {
			return true
		}
	}
	return false
}

func isInstanceOfManagedType(inventory []*InventoryItem, key types.TypeKey) bool {
	// TODO: do not consider inventory items with certain Phases (e.g. Completed)?
	for _, item := range inventory {
		if isManaged := isManagedBy(item, key); isManaged {
			return true
		}
	}
	return false
}

func isManagedBy(item *InventoryItem, key types.TypeKey) bool {
	gvk := key.GetObjectKind().GroupVersionKind()
	for _, t := range item.ManagedTypes {
		if (t.Group == "*" || t.Group == gvk.Group) && (t.Version == "*" || t.Version == gvk.Version) && (t.Kind == "*" || t.Kind == gvk.Kind) {
			return true
		}
	}
	return false
}
