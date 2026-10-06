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

// The Client interface extends the controller-runtime client by discovery and event recording capabilities.
type Client interface {
	client.Client
	// Return a discovery client.
	DiscoveryClient() discovery.DiscoveryInterface
	// Return an event recorder.
	EventRecorder() recorder.EventRecorder
	// Return a rest config for this client.
	Config() *rest.Config
	// Return a http client for this client.
	HttpClient() *http.Client
}
