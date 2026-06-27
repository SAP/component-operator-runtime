/*
SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package util

import (
	"crypto/sha256"
	"encoding/base32"
	"encoding/hex"
	"encoding/json"
	"strings"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

func Must[T any](x T, err error) T {
	if err != nil {
		panic(err)
	}
	return x
}

func Sha256hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func Sha256base32(data []byte) string {
	sum := sha256.Sum256(data)
	return strings.ToLower(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(sum[:]))
}

func CalculateDigest(values ...any) string {
	// note: this Must() is ok because the input values are expected to be JSON values
	return Sha256hex(Must(json.Marshal(values)))
}

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
