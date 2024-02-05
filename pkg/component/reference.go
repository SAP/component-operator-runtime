/*
SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package component

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apitypes "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/sap/component-operator-runtime/internal/walk"
	"github.com/sap/component-operator-runtime/pkg/types"
)

// TODO: should below reference actions be thread safe?
// If not, it should be stated somewhere explicitly that they are not.

const (
	tagNotFoundPolicy = "notFoundPolicy"
	tagFallbackKeys   = "fallbackKeys"

	notFoundPolicyIgnoreOnDeletion = "ignoreOnDeletion"
)

// +kubebuilder:object:generate=true

// ConfigMapReference defines a loadable reference to a configmap.
type ConfigMapReference struct {
	// +required
	Name   string            `json:"name"`
	data   map[string]string `json:"-"`
	loaded bool              `json:"-"`
}

func (r *ConfigMapReference) load(ctx context.Context, client client.Client, namespace string, ignoreNotFound bool) error {
	configMap := &corev1.ConfigMap{}
	if err := client.Get(ctx, apitypes.NamespacedName{Namespace: namespace, Name: r.Name}, configMap); err != nil {
		if apierrors.IsNotFound(err) {
			if ignoreNotFound {
				return nil
			}
			return types.NewRetriableError(err, nil)
		} else {
			return err
		}
	}
	r.data = configMap.Data
	r.loaded = true
	return nil
}

func (r *ConfigMapReference) digest() string {
	if !r.loaded {
		return ""
	}
	rawData, err := json.Marshal(r.data)
	if err != nil {
		panic("this cannot happen")
	}
	return sha256hex(rawData)
}

// Return the previously loaded configmap data.
func (r *ConfigMapReference) Data() map[string]string {
	if !r.loaded {
		panic("access to unloaded reference")
	}
	return r.data
}

// +kubebuilder:object:generate=true

// ConfigMapKeyReference defines a loadable reference to a configmap key.
type ConfigMapKeyReference struct {
	// +required
	Name string `json:"name"`
	// +optional
	Key    string `json:"key,omitempty"`
	value  string `json:"-"`
	loaded bool   `json:"-"`
}

func (r *ConfigMapKeyReference) load(ctx context.Context, client client.Client, namespace string, ignoreNotFound bool, fallbackKeys ...string) error {
	configMap := &corev1.ConfigMap{}
	if err := client.Get(ctx, apitypes.NamespacedName{Namespace: namespace, Name: r.Name}, configMap); err != nil {
		if apierrors.IsNotFound(err) {
			if ignoreNotFound {
				return nil
			}
			return types.NewRetriableError(err, nil)
		} else {
			return err
		}
	}
	if r.Key != "" {
		if value, ok := configMap.Data[r.Key]; ok {
			r.value = value
			r.loaded = true
			return nil
		} else {
			return types.NewRetriableError(fmt.Errorf("key %s not found in configmap %s/%s", r.Key, namespace, r.Name), nil)
		}
	} else {
		for _, key := range fallbackKeys {
			if value, ok := configMap.Data[key]; ok {
				r.value = value
				r.loaded = true
				return nil
			}
		}
		return types.NewRetriableError(fmt.Errorf("no matching key found in configmap %s/%s", namespace, r.Name), nil)
	}
}

func (r *ConfigMapKeyReference) digest() string {
	if !r.loaded {
		return ""
	}
	return sha256hex([]byte(r.value))
}

// Return the previously loaded value of the configmap key.
func (r *ConfigMapKeyReference) Value() string {
	if !r.loaded {
		panic("access to unloaded reference")
	}
	return r.value
}

// +kubebuilder:object:generate=true

// SecretReference defines a loadable reference to a secret.
type SecretReference struct {
	// +required
	Name   string            `json:"name"`
	data   map[string][]byte `json:"-"`
	loaded bool              `json:"-"`
}

func (r *SecretReference) load(ctx context.Context, client client.Client, namespace string, ignoreNotFound bool) error {
	secret := &corev1.Secret{}
	if err := client.Get(ctx, apitypes.NamespacedName{Namespace: namespace, Name: r.Name}, secret); err != nil {
		if apierrors.IsNotFound(err) {
			if ignoreNotFound {
				return nil
			}
			return types.NewRetriableError(err, nil)
		} else {
			return err
		}
	}
	r.data = secret.Data
	r.loaded = true
	return nil
}

func (r *SecretReference) digest() string {
	if !r.loaded {
		return ""
	}
	rawData, err := json.Marshal(r.data)
	if err != nil {
		panic("this cannot happen")
	}
	return sha256hex(rawData)
}

// Return the previously loaded secret data.
func (r *SecretReference) Data() map[string][]byte {
	if !r.loaded {
		panic("access to unloaded reference")
	}
	return r.data
}

// +kubebuilder:object:generate=true

// SecretKeyReference defines a loadable reference to a secret key.
type SecretKeyReference struct {
	// +required
	Name string `json:"name"`
	// +optional
	Key    string `json:"key,omitempty"`
	value  []byte `json:"-"`
	loaded bool   `json:"-"`
}

func (r *SecretKeyReference) load(ctx context.Context, client client.Client, namespace string, ignoreNotFound bool, fallbackKeys ...string) error {
	secret := &corev1.Secret{}
	if err := client.Get(ctx, apitypes.NamespacedName{Namespace: namespace, Name: r.Name}, secret); err != nil {
		if apierrors.IsNotFound(err) {
			if ignoreNotFound {
				return nil
			}
			return types.NewRetriableError(err, nil)
		} else {
			return err
		}
	}
	if r.Key != "" {
		if value, ok := secret.Data[r.Key]; ok {
			r.value = value
			r.loaded = true
			return nil
		} else {
			return types.NewRetriableError(fmt.Errorf("key %s not found in secret %s/%s", r.Key, namespace, r.Name), nil)
		}
	} else {
		for _, key := range fallbackKeys {
			if value, ok := secret.Data[key]; ok {
				r.value = value
				r.loaded = true
				return nil
			}
		}
		return types.NewRetriableError(fmt.Errorf("no matching key found in secret %s/%s", namespace, r.Name), nil)
	}
}

func (r *SecretKeyReference) digest() string {
	if !r.loaded {
		return ""
	}
	return sha256hex(r.value)
}

// Return the previously loaded value of the secret key.
func (r *SecretKeyReference) Value() []byte {
	if !r.loaded {
		panic("access to unloaded reference")
	}
	return r.value
}

func resolveReferences[T Component](ctx context.Context, client client.Client, component T) error {
	return walk.Walk(getSpec(component), func(x any, path []string, tag reflect.StructTag) error {
		switch r := x.(type) {
		case *ConfigMapReference:
			if r == nil {
				return nil
			}
			ignoreNotFound := !component.GetDeletionTimestamp().IsZero() && tag.Get(tagNotFoundPolicy) == notFoundPolicyIgnoreOnDeletion
			return r.load(ctx, client, component.GetNamespace(), ignoreNotFound)
		case *ConfigMapKeyReference:
			if r == nil {
				return nil
			}
			ignoreNotFound := !component.GetDeletionTimestamp().IsZero() && tag.Get(tagNotFoundPolicy) == notFoundPolicyIgnoreOnDeletion
			var fallbackKeys []string
			if s := tag.Get(tagFallbackKeys); s != "" {
				fallbackKeys = strings.Split(s, ",")
			}
			return r.load(ctx, client, component.GetNamespace(), ignoreNotFound, fallbackKeys...)
		case *SecretReference:
			if r == nil {
				return nil
			}
			ignoreNotFound := !component.GetDeletionTimestamp().IsZero() && tag.Get(tagNotFoundPolicy) == notFoundPolicyIgnoreOnDeletion
			return r.load(ctx, client, component.GetNamespace(), ignoreNotFound)
		case *SecretKeyReference:
			if r == nil {
				return nil
			}
			ignoreNotFound := !component.GetDeletionTimestamp().IsZero() && tag.Get(tagNotFoundPolicy) == notFoundPolicyIgnoreOnDeletion
			var fallbackKeys []string
			if s := tag.Get(tagFallbackKeys); s != "" {
				fallbackKeys = strings.Split(s, ",")
			}
			return r.load(ctx, client, component.GetNamespace(), ignoreNotFound, fallbackKeys...)
		}
		return nil
	})
}
