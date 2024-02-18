/*
SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package cluster

import (
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// TODO: the Client interface should be public, because it is returned by component.GetClientFromContext()
// TODO: should we add a Config() method, i.e. expose the rest.Config used by the client?

// The Client interface extends the controller-runtime client by discovery and event recording capabilities.
type Client interface {
	client.Client
	// Return a discovery client.
	DiscoveryClient() discovery.DiscoveryInterface
	// Return an event recorder.
	EventRecorder() record.EventRecorder
}
