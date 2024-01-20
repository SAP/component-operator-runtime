/*
SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package component

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	apitypes "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/sap/component-operator-runtime/internal/walk"
	"github.com/sap/component-operator-runtime/pkg/types"
)

// +kubebuilder:object:generate=true

// ConfigMapReference defines a loadable reference to a configmap.
type ConfigMapReference struct {
	// +required
	Name string            `json:"name"`
	data map[string]string `json:"-"`
}

func (r *ConfigMapReference) load(ctx context.Context, client client.Client, namespace string) error {
	configMap := &corev1.ConfigMap{}
	if err := client.Get(ctx, apitypes.NamespacedName{Namespace: namespace, Name: r.Name}, configMap); err != nil {
		if errors.IsNotFound(err) {
			return types.NewRetriableError(err, nil)
		} else {
			return err
		}
	}
	r.data = configMap.Data
	return nil
}

// Return the previously loaded configmap data.
func (r *ConfigMapReference) Data() map[string]string {
	return r.data
}

// +kubebuilder:object:generate=true

// ConfigMapKeyReference defines a loadable reference to a configmap key.
type ConfigMapKeyReference struct {
	// +required
	Name string `json:"name"`
	// +optional
	Key   string `json:"key,omitempty"`
	value string `json:"-"`
}

func (r *ConfigMapKeyReference) load(ctx context.Context, client client.Client, namespace string, fallbackKeys ...string) error {
	configMap := &corev1.ConfigMap{}
	if err := client.Get(ctx, apitypes.NamespacedName{Namespace: namespace, Name: r.Name}, configMap); err != nil {
		if errors.IsNotFound(err) {
			return types.NewRetriableError(err, nil)
		} else {
			return err
		}
	}
	if r.Key != "" {
		if value, ok := configMap.Data[r.Key]; ok {
			r.value = value
			return nil
		} else {
			return types.NewRetriableError(fmt.Errorf("key %s not found in configmap %s/%s", r.Key, namespace, r.Name), nil)
		}
	} else {
		for _, key := range fallbackKeys {
			if value, ok := configMap.Data[key]; ok {
				r.value = value
				return nil
			}
		}
		return types.NewRetriableError(fmt.Errorf("no matching key found in configmap %s/%s", namespace, r.Name), nil)
	}
}

// Return the previously loaded value of the configmap key.
func (r *ConfigMapKeyReference) Value() string {
	return r.value
}

// +kubebuilder:object:generate=true

// SecretReference defines a loadable reference to a secret.
type SecretReference struct {
	// +required
	Name string            `json:"name"`
	data map[string][]byte `json:"-"`
}

func (r *SecretReference) load(ctx context.Context, client client.Client, namespace string) error {
	secret := &corev1.Secret{}
	if err := client.Get(ctx, apitypes.NamespacedName{Namespace: namespace, Name: r.Name}, secret); err != nil {
		if errors.IsNotFound(err) {
			return types.NewRetriableError(err, nil)
		} else {
			return err
		}
	}
	r.data = secret.Data
	return nil
}

// Return the previously loaded secret data.
func (r *SecretReference) Data() map[string][]byte {
	return r.data
}

// +kubebuilder:object:generate=true

// SecretKeyReference defines a loadable reference to a secret key.
type SecretKeyReference struct {
	// +required
	Name string `json:"name"`
	// +optional
	Key   string `json:"key,omitempty"`
	value []byte `json:"-"`
}

func (r *SecretKeyReference) load(ctx context.Context, client client.Client, namespace string, fallbackKeys ...string) error {
	secret := &corev1.Secret{}
	if err := client.Get(ctx, apitypes.NamespacedName{Namespace: namespace, Name: r.Name}, secret); err != nil {
		if errors.IsNotFound(err) {
			return types.NewRetriableError(err, nil)
		} else {
			return err
		}
	}
	if r.Key != "" {
		if value, ok := secret.Data[r.Key]; ok {
			r.value = value
			return nil
		} else {
			return types.NewRetriableError(fmt.Errorf("key %s not found in secret %s/%s", r.Key, namespace, r.Name), nil)
		}
	} else {
		for _, key := range fallbackKeys {
			if value, ok := secret.Data[key]; ok {
				r.value = value
				return nil
			}
		}
		return types.NewRetriableError(fmt.Errorf("no matching key found in secret %s/%s", namespace, r.Name), nil)
	}
}

// Return the previously loaded value of the secret key.
func (r *SecretKeyReference) Value() []byte {
	return r.value
}

func resolveReferences[T Component](ctx context.Context, client client.Client, component T) error {
	return walk.Walk(getSpec(component), func(x any, path []string, tag reflect.StructTag) error {
		switch r := x.(type) {
		case *ConfigMapReference:
			return r.load(ctx, client, component.GetNamespace())
		case *ConfigMapKeyReference:
			var fallbackKeys []string
			if s := tag.Get("fallbackKeys"); s != "" {
				fallbackKeys = strings.Split(s, ",")
			}
			return r.load(ctx, client, component.GetNamespace(), fallbackKeys...)
		case *SecretReference:
			return r.load(ctx, client, component.GetNamespace())
		case *SecretKeyReference:
			var fallbackKeys []string
			if s := tag.Get("fallbackKeys"); s != "" {
				fallbackKeys = strings.Split(s, ",")
			}
			return r.load(ctx, client, component.GetNamespace(), fallbackKeys...)
		}
		return nil
	})
}
