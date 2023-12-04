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

// The Client interface extends the controller-runtime client by discovery and event recording capabilities.
type Client interface {
	client.Client
	// Return a discovery client.
	DiscoveryClient() discovery.DiscoveryInterface
	// Return an event recorder.
	EventRecorder() record.EventRecorder
}

// The ClientConfiguration interface is meant to be implemented by components which offer remote deployments.
type ClientConfiguration interface {
	// Get kubeconfig content.
	GetKubeConfig() []byte
}

// The ImpersonationConfiguration interface is meant to be implemented by components which offer impersonated deployments.
type ImpersonationConfiguration interface {
	// Return impersonation user.
	// Should return system:serviceaccount:<namespace>:<serviceaccount> if a service account is used for impersonation.
	// Should return an empty string if user shall not be impersonated.
	GetImpersonationUser() string
	// Return impersonation groups.
	// Should return nil if groups shall not be impersonated.
	GetImpersonationGroups() []string
}
