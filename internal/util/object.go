/*
SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package util

import (
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func SetLabel(obj client.Object, key string, value string) {
	labels := obj.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}
	labels[key] = value
	obj.SetLabels(labels)
}

func RemoveLabel(obj client.Object, key string) {
	labels := obj.GetLabels()
	delete(labels, key)
	obj.SetLabels(labels)
}

func SetAnnotation(obj client.Object, key string, value string) {
	annotations := obj.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}
	annotations[key] = value
	obj.SetAnnotations(annotations)
}

func RemoveAnnotation(obj client.Object, key string) {
	annotations := obj.GetAnnotations()
	delete(annotations, key)
	obj.SetAnnotations(annotations)
}
