/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package component

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apitypes "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	legacyerrors "github.com/pkg/errors"
	"github.com/sap/component-operator-runtime/internal/walk"
	"github.com/sap/component-operator-runtime/pkg/types"
)

// TODO: should below reference actions be thread safe?
// If not, it should be stated somewhere explicitly that they are not.
// TODO: allow to skip loading of references upon deletion in general (e.g. by a tagLoadPolicy)
// (currently it is only possible to ignore not-found errors upon deletion)

const (
	tagNotFoundPolicy = "notFoundPolicy"
	tagFallbackKeys   = "fallbackKeys"

	notFoundPolicyIgnoreOnDeletion = "ignoreOnDeletion"

	retryAfter = 10 * time.Second
)

// +kubebuilder:object:generate=true

// ConfigMapReference defines a loadable reference to a configmap.
type ConfigMapReference struct {
	// +required
	// +kubebuilder:validation:MinLength=1
	Name   string            `json:"name"`
	data   map[string]string `json:"-"`
	loaded bool              `json:"-"`
}

func (r *ConfigMapReference) load(ctx context.Context, clnt client.Client, namespace string, ignoreNotFound bool) error {
	// TODO: shouldn't we panic if already loaded?
	configMap := &corev1.ConfigMap{}
	if err := clnt.Get(ctx, apitypes.NamespacedName{Namespace: namespace, Name: r.Name}, configMap); err != nil {
		if apierrors.IsNotFound(err) {
			if ignoreNotFound {
				return nil
			}
			return types.NewRetriableError(legacyerrors.Wrapf(err, "error loading configmap %s/%s", namespace, r.Name), ref(retryAfter))
		} else {
			return legacyerrors.Wrapf(err, "error loading configmap %s/%s", namespace, r.Name)
		}
	}
	r.data = configMap.Data
	r.loaded = true
	return nil
}

func (r *ConfigMapReference) digest() string {
	if !r.loaded {
		// note: we can't panic here because this might be called in case of not-found situations
		return ""
	}
	return calculateDigest(r.data)
}

// Return the previously loaded configmap data.
func (r *ConfigMapReference) Data() map[string]string {
	if !r.loaded {
		// note: this panic indicates a programmatic error on the consumer side
		panic("access to unloaded reference")
	}
	return r.data
}

// +kubebuilder:object:generate=true

// ConfigMapKeyReference defines a loadable reference to a configmap key.
type ConfigMapKeyReference struct {
	// +required
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
	// +optional
	// +kubebuilder:validation:MinLength=1
	Key    string `json:"key,omitempty"`
	value  string `json:"-"`
	loaded bool   `json:"-"`
}

func (r *ConfigMapKeyReference) load(ctx context.Context, clnt client.Client, namespace string, ignoreNotFound bool, fallbackKeys ...string) error {
	// TODO: shouldn't we panic if already loaded?
	configMap := &corev1.ConfigMap{}
	if err := clnt.Get(ctx, apitypes.NamespacedName{Namespace: namespace, Name: r.Name}, configMap); err != nil {
		if apierrors.IsNotFound(err) {
			if ignoreNotFound {
				return nil
			}
			return types.NewRetriableError(legacyerrors.Wrapf(err, "error loading configmap %s/%s", namespace, r.Name), ref(retryAfter))
		} else {
			return legacyerrors.Wrapf(err, "error loading configmap %s/%s", namespace, r.Name)
		}
	}
	if r.Key != "" {
		if value, ok := configMap.Data[r.Key]; ok {
			r.value = value
			r.loaded = true
			return nil
		} else {
			return types.NewRetriableError(fmt.Errorf("key %s not found in configmap %s/%s", r.Key, namespace, r.Name), ref(retryAfter))
		}
	} else {
		for _, key := range fallbackKeys {
			if value, ok := configMap.Data[key]; ok {
				r.value = value
				r.loaded = true
				return nil
			}
		}
		return types.NewRetriableError(fmt.Errorf("no matching key found in configmap %s/%s", namespace, r.Name), ref(retryAfter))
	}
}

func (r *ConfigMapKeyReference) digest() string {
	if !r.loaded {
		// note: we can't panic here because this might be called in case of not-found situations
		return ""
	}
	return sha256hex([]byte(r.value))
}

// Return the previously loaded value of the configmap key.
func (r *ConfigMapKeyReference) Value() string {
	if !r.loaded {
		// note: this panic indicates a programmatic error on the consumer side
		panic("access to unloaded reference")
	}
	return r.value
}

// +kubebuilder:object:generate=true

// SecretReference defines a loadable reference to a secret.
type SecretReference struct {
	// +required
	// +kubebuilder:validation:MinLength=1
	Name   string            `json:"name"`
	data   map[string][]byte `json:"-"`
	loaded bool              `json:"-"`
}

func (r *SecretReference) load(ctx context.Context, clnt client.Client, namespace string, ignoreNotFound bool) error {
	// TODO: shouldn't we panic if already loaded?
	secret := &corev1.Secret{}
	if err := clnt.Get(ctx, apitypes.NamespacedName{Namespace: namespace, Name: r.Name}, secret); err != nil {
		if apierrors.IsNotFound(err) {
			if ignoreNotFound {
				return nil
			}
			return types.NewRetriableError(legacyerrors.Wrapf(err, "error loading secret %s/%s", namespace, r.Name), ref(retryAfter))
		} else {
			return legacyerrors.Wrapf(err, "error loading secret %s/%s", namespace, r.Name)
		}
	}
	r.data = secret.Data
	r.loaded = true
	return nil
}

func (r *SecretReference) digest() string {
	if !r.loaded {
		// note: we can't panic here because this might be called in case of not-found situations
		return ""
	}
	return calculateDigest(r.data)
}

// Return the previously loaded secret data.
func (r *SecretReference) Data() map[string][]byte {
	if !r.loaded {
		// note: this panic indicates a programmatic error on the consumer side
		panic("access to unloaded reference")
	}
	return r.data
}

// +kubebuilder:object:generate=true

// SecretKeyReference defines a loadable reference to a secret key.
type SecretKeyReference struct {
	// +required
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
	// +optional
	// +kubebuilder:validation:MinLength=1
	Key    string `json:"key,omitempty"`
	value  []byte `json:"-"`
	loaded bool   `json:"-"`
}

func (r *SecretKeyReference) load(ctx context.Context, clnt client.Client, namespace string, ignoreNotFound bool, fallbackKeys ...string) error {
	// TODO: shouldn't we panic if already loaded?
	secret := &corev1.Secret{}
	if err := clnt.Get(ctx, apitypes.NamespacedName{Namespace: namespace, Name: r.Name}, secret); err != nil {
		if apierrors.IsNotFound(err) {
			if ignoreNotFound {
				return nil
			}
			return types.NewRetriableError(legacyerrors.Wrapf(err, "error loading secret %s/%s", namespace, r.Name), ref(retryAfter))
		} else {
			return legacyerrors.Wrapf(err, "error loading secret %s/%s", namespace, r.Name)
		}
	}
	if r.Key != "" {
		if value, ok := secret.Data[r.Key]; ok {
			r.value = value
			r.loaded = true
			return nil
		} else {
			return types.NewRetriableError(fmt.Errorf("key %s not found in secret %s/%s", r.Key, namespace, r.Name), ref(retryAfter))
		}
	} else {
		for _, key := range fallbackKeys {
			if value, ok := secret.Data[key]; ok {
				r.value = value
				r.loaded = true
				return nil
			}
		}
		return types.NewRetriableError(fmt.Errorf("no matching key found in secret %s/%s", namespace, r.Name), ref(retryAfter))
	}
}

func (r *SecretKeyReference) digest() string {
	if !r.loaded {
		// note: we can't panic here because this might be called in case of not-found situations
		return ""
	}
	return sha256hex(r.value)
}

// Return the previously loaded value of the secret key.
func (r *SecretKeyReference) Value() []byte {
	if !r.loaded {
		// note: this panic indicates a programmatic error on the consumer side
		panic("access to unloaded reference")
	}
	return r.value
}

// Generic reference. All occurrences in the component's spec of types implementing this interface are automatically resolved
// by the framework during reconcile by calling the Load() method. The digests returned by the Digest() methods are
// incorporated into the component's digest.
type Reference[T Component] interface {
	// Load the referenced content. The framework calls this at most once. So it is ok if implementation
	// errors out or even panics if invoked more than once. The implementation may skip loading in certain cases,
	// for example if deletion is ongoing.
	Load(ctx context.Context, clnt client.Client, component T) error
	// Return a digest of the referenced content. This digest is incorporated into the component digest which
	// is passed to generators and hooks (per context) and which decides when the processing timer is reset,
	// and therefore influences the timeout behavior of the compoment. In case the reference is not loaded,
	// the implementation should return the empty string.
	Digest() string
}

func resolveReferences[T Component](ctx context.Context, clnt client.Client, hookClient client.Client, component T) (string, error) {
	digestData := make(map[string]any)
	spec := getSpec(component)
	digestData["generation"] = component.GetGeneration()
	digestData["annotations"] = component.GetAnnotations()
	// TODO: including spec into the digest is actually not required (since generation is included)
	digestData["spec"] = spec
	if err := walk.Walk(spec, func(x any, path []string, tag reflect.StructTag) error {
		// note: this must() is ok because marshalling []string should always work
		rawPath := must(json.Marshal(path))
		switch r := x.(type) {
		case *ConfigMapReference:
			if r == nil {
				return nil
			}
			ignoreNotFound := !component.GetDeletionTimestamp().IsZero() && tag.Get(tagNotFoundPolicy) == notFoundPolicyIgnoreOnDeletion
			if err := r.load(ctx, clnt, component.GetNamespace(), ignoreNotFound); err != nil {
				return err
			}
			digestData["refs:"+string(rawPath)] = r.digest()
		case *ConfigMapKeyReference:
			if r == nil {
				return nil
			}
			ignoreNotFound := !component.GetDeletionTimestamp().IsZero() && tag.Get(tagNotFoundPolicy) == notFoundPolicyIgnoreOnDeletion
			var fallbackKeys []string
			if s := tag.Get(tagFallbackKeys); s != "" {
				fallbackKeys = strings.Split(s, ",")
			}
			if err := r.load(ctx, clnt, component.GetNamespace(), ignoreNotFound, fallbackKeys...); err != nil {
				return err
			}
			digestData["refs:"+string(rawPath)] = r.digest()
		case *SecretReference:
			if r == nil {
				return nil
			}
			ignoreNotFound := !component.GetDeletionTimestamp().IsZero() && tag.Get(tagNotFoundPolicy) == notFoundPolicyIgnoreOnDeletion
			if err := r.load(ctx, clnt, component.GetNamespace(), ignoreNotFound); err != nil {
				return err
			}
			digestData["refs:"+string(rawPath)] = r.digest()
		case *SecretKeyReference:
			if r == nil {
				return nil
			}
			ignoreNotFound := !component.GetDeletionTimestamp().IsZero() && tag.Get(tagNotFoundPolicy) == notFoundPolicyIgnoreOnDeletion
			var fallbackKeys []string
			if s := tag.Get(tagFallbackKeys); s != "" {
				fallbackKeys = strings.Split(s, ",")
			}
			if err := r.load(ctx, clnt, component.GetNamespace(), ignoreNotFound, fallbackKeys...); err != nil {
				return err
			}
			digestData["refs:"+string(rawPath)] = r.digest()
		case Reference[T]:
			if v := reflect.ValueOf(r); r == nil || v.Kind() == reflect.Pointer && v.IsNil() {
				return nil
			}
			if err := r.Load(ctx, hookClient, component); err != nil {
				return err
			}
			digestData["refs:"+string(rawPath)] = r.Digest()
		}
		return nil
	}); err != nil {
		return "", err
	}
	return calculateDigest(digestData), nil
}
