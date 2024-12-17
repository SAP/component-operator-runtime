/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package cluster

import (
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewClient(clnt client.Client, discoveryClient discovery.DiscoveryInterface, eventRecorder record.EventRecorder) Client {
	return &clientImpl{
		Client:          clnt,
		discoveryClient: discoveryClient,
		eventRecorder:   eventRecorder,
	}
}

func NewClientFor(config *rest.Config, scheme *runtime.Scheme, name string) (Client, error) {
	return newClientFor(config, scheme, name)
}

func newClientFor(config *rest.Config, scheme *runtime.Scheme, name string) (*clientImpl, error) {
	httpClient, err := rest.HTTPClientFor(config)
	if err != nil {
		return nil, err
	}
	ctrlClient, err := client.New(config, client.Options{HTTPClient: httpClient, Scheme: scheme})
	if err != nil {
		return nil, err
	}
	clientset, err := kubernetes.NewForConfigAndClient(config, httpClient)
	if err != nil {
		return nil, err
	}
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: clientset.CoreV1().Events("")})
	eventRecorder := eventBroadcaster.NewRecorder(scheme, corev1.EventSource{Component: name})
	clnt := &clientImpl{
		Client:           ctrlClient,
		discoveryClient:  clientset,
		eventBroadcaster: eventBroadcaster,
		eventRecorder:    eventRecorder,
	}
	return clnt, nil
}

type clientImpl struct {
	client.Client
	discoveryClient  discovery.DiscoveryInterface
	eventBroadcaster record.EventBroadcaster
	eventRecorder    record.EventRecorder
	validUntil       time.Time
}

func (c *clientImpl) DiscoveryClient() discovery.DiscoveryInterface {
	return c.discoveryClient
}

func (c *clientImpl) EventRecorder() record.EventRecorder {
	return c.eventRecorder
}
