/*
SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and redis-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

package component

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/sap/go-generics/slices"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	apiregistrationv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/sap/component-operator-runtime/pkg/types"
)

func sha256hash(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func setLabel(obj client.Object, key string, value string) {
	labels := obj.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}
	labels[key] = value
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
			panic("this cannot happen")
		}
	case isApiService(object):
		switch apiService := object.(type) {
		case *apiregistrationv1.APIService:
			return []TypeInfo{{Group: apiService.Spec.Group, Version: apiService.Spec.Version, Kind: "*"}}
		default:
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

func scopeFromRestMapping(mapping *meta.RESTMapping) int {
	switch mapping.Scope.Name() {
	case meta.RESTScopeNameNamespace:
		return scopeNamespaced
	case meta.RESTScopeNameRoot:
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

func sortObjectsForApply[T client.Object](s []T, orderFunc func(client.Object) int) []T {
	priority := map[string]int{
		"Namespace": -4,
		"ValidatingWebhookConfiguration.admissionregistration.k8s.io": -3,
		"MutatingWebhookConfiguration.admissionregistration.k8s.io":   -3,
		"CustomResourceDefinition.apiextensions.k8s.io":               -2,
		"ConfigMap":                             -1,
		"Secret":                                -1,
		"ClusterRole.rbac.authorization.k8s.io": -1,
		"Role.rbac.authorization.k8s.io":        -1,
		"ClusterRoleBinding.rbac.authorization.k8s.io": -1,
		"RoleBinding.rbac.authorization.k8s.io":        -1,
		"APIService.apiregistration.k8s.io":            1,
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

func sortObjectsForDelete[T types.ObjectKey](s []T) []T {
	priority := map[string]int{
		"CustomResourceDefinition.apiextensions.k8s.io":               -1,
		"APIService.apiregistration.k8s.io":                           -1,
		"ValidatingWebhookConfiguration.admissionregistration.k8s.io": 1,
		"MutatingWebhookConfiguration.admissionregistration.k8s.io":   1,
		"Service":   2,
		"ConfigMap": 2,
		"Secret":    2,
		"Namespace": 3,
	}
	f := func(x T, y T) bool {
		gvx := x.GetObjectKind().GroupVersionKind().GroupKind().String()
		gvy := y.GetObjectKind().GroupVersionKind().GroupKind().String()
		return priority[gvx] > priority[gvy]
	}
	return slices.SortBy(s, f)
}

func getItem(inventory []*InventoryItem, key types.ObjectKey) *InventoryItem {
	var item *InventoryItem
	for _, _item := range inventory {
		if _item.Matches(key) {
			if item != nil {
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

func mustParseLabelSelector(s string) labels.Selector {
	selector, err := labels.Parse(s)
	if err != nil {
		panic("this cannot happen")
	}
	return selector
}
