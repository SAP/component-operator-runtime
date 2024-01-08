/*
SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package cluster

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"reflect"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/record"
	apiregistrationv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/sap/component-operator-runtime/pkg/types"
)

type ClientFactory struct {
	mutex   sync.Mutex
	name    string
	config  *rest.Config
	scheme  *runtime.Scheme
	clients map[string]*clientImpl
}

func NewClientFactory(name string, config *rest.Config, schemeBuilder types.SchemeBuilder) (*ClientFactory, error) {
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
	if schemeBuilder != nil {
		if err := schemeBuilder.AddToScheme(scheme); err != nil {
			return nil, err
		}
	}

	factory := &ClientFactory{
		name:    name,
		config:  config,
		scheme:  scheme,
		clients: make(map[string]*clientImpl),
	}

	go func() {
		ticker := time.NewTicker(10 * time.Second)
		for {
			<-ticker.C
			now := time.Now()
			factory.mutex.Lock()
			for key, client := range factory.clients {
				if client.validUntil.Before(now) {
					client.eventBroadcaster.Shutdown()
					delete(factory.clients, key)
				}
			}
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

	if client, ok := f.clients[key]; ok {
		client.validUntil = time.Now().Add(15 * time.Minute)
		return client, nil
	}

	httpClient, err := rest.HTTPClientFor(config)
	if err != nil {
		return nil, err
	}
	ctrlclient, err := client.New(config, client.Options{HTTPClient: httpClient, Scheme: f.scheme})
	if err != nil {
		return nil, err
	}
	clientset, err := kubernetes.NewForConfigAndClient(config, httpClient)
	if err != nil {
		return nil, err
	}
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: clientset.CoreV1().Events("")})
	eventRecorder := eventBroadcaster.NewRecorder(f.scheme, corev1.EventSource{Component: f.name})
	client := &clientImpl{
		Client:           ctrlclient,
		discoveryClient:  clientset,
		eventBroadcaster: eventBroadcaster,
		eventRecorder:    eventRecorder,
		validUntil:       time.Now().Add(15 * time.Minute),
	}
	f.clients[key] = client

	return client, nil
}

func sha256sum(data any) string {
	dataAsJson, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}
	sha256sum := sha256.Sum256(dataAsJson)
	return string(sha256sum[:])
}
