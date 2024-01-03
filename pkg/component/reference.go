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
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/sap/component-operator-runtime/internal/walk"
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
	if err := client.Get(ctx, types.NamespacedName{Namespace: namespace, Name: r.Name}, configMap); err != nil {
		return err
	}
	r.data = configMap.Data
	return nil
}

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
	if err := client.Get(ctx, types.NamespacedName{Namespace: namespace, Name: r.Name}, configMap); err != nil {
		return err
	}
	if r.Key != "" {
		if value, ok := configMap.Data[r.Key]; ok {
			r.value = value
			return nil
		} else {
			return fmt.Errorf("key %s not found in configmap %s/%s", r.Key, namespace, r.Name)
		}
	} else {
		for _, key := range fallbackKeys {
			if value, ok := configMap.Data[key]; ok {
				r.value = value
				return nil
			}
		}
		return fmt.Errorf("no matching key found in configmap %s/%s", namespace, r.Name)
	}
}

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
	if err := client.Get(ctx, types.NamespacedName{Namespace: namespace, Name: r.Name}, secret); err != nil {
		return err
	}
	r.data = secret.Data
	return nil
}

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
	if err := client.Get(ctx, types.NamespacedName{Namespace: namespace, Name: r.Name}, secret); err != nil {
		return err
	}
	if r.Key != "" {
		if value, ok := secret.Data[r.Key]; ok {
			r.value = value
			return nil
		} else {
			return fmt.Errorf("key %s not found in secret %s/%s", r.Key, namespace, r.Name)
		}
	} else {
		for _, key := range fallbackKeys {
			if value, ok := secret.Data[key]; ok {
				r.value = value
				return nil
			}
		}
		return fmt.Errorf("no matching key found in secret %s/%s", namespace, r.Name)
	}
}

func (r *SecretKeyReference) Value() []byte {
	return r.value
}

func resolveReferences[T Component](ctx context.Context, client client.Client, component T) error {
	// note: the following relies on T being a pointer type; but this is reasonable, the same assumption
	// is made in newComponent() ...
	spec := reflect.ValueOf(component).Elem().FieldByName("Spec")
	if spec.Kind() != reflect.Pointer {
		spec = spec.Addr()
	}
	return walk.Walk(spec, func(x any, path []string, tag reflect.StructTag) error {
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