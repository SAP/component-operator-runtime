/*
SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package clientfactory

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/recorder"

	"github.com/sap/component-operator-runtime/pkg/cluster"
)

func NewClientFor(config *rest.Config, scheme *runtime.Scheme, name string) (*Client, error) {
	httpClient, err := rest.HTTPClientFor(config)
	if err != nil {
		return nil, err
	}
	ctrlClient, err := client.New(config, client.Options{HTTPClient: httpClient, Scheme: scheme})
	if err != nil {
		return nil, err
	}
	// TODO: a full clientset which contains all builtin groups is actually not needed here;
	// a clientset for eventsv1 events, plus discovery would be sufficient
	clientset, err := kubernetes.NewForConfigAndClient(config, httpClient)
	if err != nil {
		return nil, err
	}
	eventBroadcaster := events.NewBroadcaster(&events.EventSinkImpl{Interface: clientset.EventsV1()})
	// TODO: should we pass through a context instead of using context.TODO(); such that the recording could
	// be stopped properly?
	if eventBroadcaster.StartRecordingToSinkWithContext(context.TODO()); err != nil {
		return nil, err
	}
	// note: NewRecorder() currently returns an events.EventRecorderLogger which includes events.EventRecorder,
	// but not events.AnnotatedEventRecorder; however looking at the implementation reveals that it actually
	// returns an object including both (at least as of now); so the below cast is safe
	eventRecorder := eventBroadcaster.NewRecorder(scheme, name).(recorder.EventRecorder)
	clnt := &Client{
		Client: cluster.NewClient(
			ctrlClient,
			clientset,
			eventRecorder,
			config,
			httpClient,
		),
		eventBroadcaster: eventBroadcaster,
	}
	return clnt, nil
}
