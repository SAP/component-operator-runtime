/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package cluster

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"sync"
	"time"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	apiregistrationv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"

	"github.com/sap/component-operator-runtime/internal/metrics"
	"github.com/sap/component-operator-runtime/pkg/types"
)

type ClientFactory struct {
	mutex          sync.Mutex
	name           string
	controllerName string
	config         *rest.Config
	scheme         *runtime.Scheme
	clients        map[string]*clientImpl
}

const validity = 15 * time.Minute

func NewClientFactory(name string, controllerName string, config *rest.Config, schemeBuilders []types.SchemeBuilder) (*ClientFactory, error) {
	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := apiextensionsv1.AddToScheme(scheme); err != nil {
		return nil, err
	}
	if err := apiregistrationv1.AddToScheme(scheme); err != nil {
		return nil, err
	}
	for _, schemeBuilder := range schemeBuilders {
		if err := schemeBuilder.AddToScheme(scheme); err != nil {
			return nil, err
		}
	}

	factory := &ClientFactory{
		name:           name,
		controllerName: controllerName,
		config:         config,
		scheme:         scheme,
		clients:        make(map[string]*clientImpl),
	}

	go func() {
		ticker := time.NewTicker(10 * time.Second)
		for {
			<-ticker.C
			now := time.Now()
			factory.mutex.Lock()
			for key, clnt := range factory.clients {
				if clnt.validUntil.Before(now) {
					clnt.eventBroadcaster.Shutdown()
					// TODO: add some (debug) log output when client is removed; unfortunately, we have no logger in here ...
					delete(factory.clients, key)
				}
			}
			metrics.ActiveClients.WithLabelValues(factory.controllerName).Set(float64(len(factory.clients)))
			factory.mutex.Unlock()
		}
	}()

	return factory, nil
}

func (f *ClientFactory) Get(kubeConfig []byte, impersonationUser string, impersonationGroups []string) (Client, error) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	var keyData = make(map[string]any)

	var config *rest.Config
	if len(kubeConfig) == 0 {
		config = rest.CopyConfig(f.config)
	} else {
		var err error
		config, err = clientcmd.RESTConfigFromKubeConfig(kubeConfig)
		if err != nil {
			return nil, err
		}
		keyData["kubeConfig"] = string(kubeConfig)
	}

	if impersonationUser != "" || len(impersonationGroups) > 0 {
		if !reflect.ValueOf(config.Impersonate).IsZero() {
			return nil, fmt.Errorf("cannot impersonate an already impersonated configuration")
		}
	}
	if impersonationUser != "" {
		config.Impersonate.UserName = impersonationUser
		keyData["impersonationUser"] = impersonationUser
	}
	if len(impersonationGroups) > 0 {
		config.Impersonate.Groups = impersonationGroups
		keyData["impersonationGroups"] = impersonationGroups
	}

	key := sha256sum(keyData)

	if clnt, ok := f.clients[key]; ok {
		clnt.validUntil = time.Now().Add(validity)
		return clnt, nil
	}

	config.Wrap(func(rt http.RoundTripper) http.RoundTripper {
		return roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			metrics.Requests.WithLabelValues(f.controllerName, r.Method).Inc()
			return rt.RoundTrip(r)
		})
	})
	clnt, err := newClientFor(config, f.scheme, f.name)
	if err != nil {
		return nil, err
	}
	clnt.validUntil = time.Now().Add(validity)
	f.clients[key] = clnt
	metrics.CreatedClients.WithLabelValues(f.controllerName).Inc()
	metrics.ActiveClients.WithLabelValues(f.controllerName).Set(float64(len(f.clients)))

	// TODO: add some (debug) log output when new client is created; unfortunately, we have no logger in here ...
	// maybe we could (at least in Get()) get one from the reconcile context ...
	return clnt, nil
}

func sha256sum(data any) string {
	dataAsJson, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}
	sha256sum := sha256.Sum256(dataAsJson)
	return string(sha256sum[:])
}

type roundTripperFunc func(r *http.Request) (*http.Response, error)

var _ http.RoundTripper = roundTripperFunc(nil)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}
