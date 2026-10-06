/*
SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package cluster

import (
	"net/http"

	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/recorder"
)

func NewClient(clnt client.Client, discoveryClient discovery.DiscoveryInterface, eventRecorder recorder.EventRecorder, config *rest.Config, httpClient *http.Client) Client {
	return &clientImpl{
		Client:          clnt,
		discoveryClient: discoveryClient,
		eventRecorder:   eventRecorder,
		config:          config,
		httpClient:      httpClient,
	}
}

type clientImpl struct {
	client.Client
	discoveryClient discovery.DiscoveryInterface
	eventRecorder   recorder.EventRecorder
	config          *rest.Config
	httpClient      *http.Client
}

func (c *clientImpl) DiscoveryClient() discovery.DiscoveryInterface {
	return c.discoveryClient
}

func (c *clientImpl) EventRecorder() recorder.EventRecorder {
	return c.eventRecorder
}

func (c *clientImpl) Config() *rest.Config {
	return c.config
}

func (c *clientImpl) HttpClient() *http.Client {
	return c.httpClient
}
