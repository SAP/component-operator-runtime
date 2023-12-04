/*
SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package cluster

import (
	"time"

	"k8s.io/client-go/discovery"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/sap/component-operator-runtime/pkg/cluster"
)

func NewClient(client client.Client, discoveryClient discovery.DiscoveryInterface, eventRecorder record.EventRecorder) cluster.Client {
	return &Client{
		Client:          client,
		discoveryClient: discoveryClient,
		eventRecorder:   eventRecorder,
	}
}

type Client struct {
	client.Client
	discoveryClient  discovery.DiscoveryInterface
	eventBroadcaster record.EventBroadcaster
	eventRecorder    record.EventRecorder
	validUntil       time.Time
}

func (c *Client) DiscoveryClient() discovery.DiscoveryInterface {
	return c.discoveryClient
}

func (c *Client) EventRecorder() record.EventRecorder {
	return c.eventRecorder
}
