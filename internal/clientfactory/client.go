/*
SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package clientfactory

import (
	"context"
	"fmt"
	"sync"
	"time"

	eventsv1 "k8s.io/api/events/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/recorder"

	"github.com/sap/component-operator-runtime/internal/util"
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
	eventBroadcaster := events.NewBroadcaster(tracker.NewSinkFor(&events.EventSinkImpl{Interface: clientset.EventsV1()}))
	// TODO: should we pass through a context instead of using context.TODO(); such that the recording could
	// be stopped properly?
	if eventBroadcaster.StartRecordingToSinkWithContext(context.TODO()); err != nil {
		return nil, err
	}
	// note: NewRecorder() currently returns an events.EventRecorderLogger which includes events.EventRecorder,
	// but not events.AnnotatedEventRecorder; however looking at the implementation reveals that it actually
	// returns an object including both (at least as of now); so the below cast is safe

	recorder := tracker.NewRecorderFor(eventBroadcaster.NewRecorder(scheme, name).(recorder.EventRecorder))
	clnt := &Client{
		Client: cluster.NewClient(
			ctrlClient,
			clientset,
			recorder,
			config,
			httpClient,
		),
		eventBroadcaster: eventBroadcaster,
	}
	return clnt, nil
}

var tracker *Tracker

func init() {
	tracker = &Tracker{
		inflight: make(map[string]TrackedEvent),
	}
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		for range ticker.C {
			tracker.mu.Lock()
			for key, event := range tracker.inflight {
				if time.Since(event.registeredAt) > 1*time.Minute {
					fmt.Printf("[DEBUG] Removing stale event: %s\n", event)
					tracker.numStale++
					delete(tracker.inflight, key)
				}
			}
			fmt.Printf("[DEBUG] Current inflight events: %d, max inflight: %d, stale events: %d\n", len(tracker.inflight), tracker.maxInflight, tracker.numStale)
			tracker.mu.Unlock()
		}
	}()
}

type TrackedEvent struct {
	registeredAt time.Time
	regarding    string
	reason       string
	note         string
}

func (e TrackedEvent) String() string {
	return fmt.Sprintf("%s %s %s", e.regarding, e.reason, e.note)
}

type Tracker struct {
	inflight    map[string]TrackedEvent
	maxInflight int
	numStale    int
	mu          sync.Mutex
}

func (t *Tracker) register(regarding runtime.Object, related runtime.Object, annotations map[string]string, eventtype, reason, action, note string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	regardingUid := string(regarding.(client.Object).GetUID())
	relatedUid := ""
	if related != nil {
		relatedUid = string(related.(client.Object).GetUID())
	}
	event := TrackedEvent{
		registeredAt: time.Now(),
		regarding:    fmt.Sprintf("%s %s/%s", regarding.GetObjectKind().GroupVersionKind(), regarding.(client.Object).GetNamespace(), regarding.(client.Object).GetName()),
		reason:       reason,
		note:         note,
	}
	t.inflight[util.CalculateDigest(regardingUid, relatedUid, annotations, eventtype, reason, action, note)] = event
	if len(t.inflight) > t.maxInflight {
		t.maxInflight = len(t.inflight)
	}
}

func (t *Tracker) acknowledge(event *eventsv1.Event) {
	t.mu.Lock()
	defer t.mu.Unlock()
	regardingUid := string(event.Regarding.UID)
	relatedUid := ""
	if event.Related != nil {
		relatedUid = string(event.Related.UID)
	}
	delete(t.inflight, util.CalculateDigest(regardingUid, relatedUid, event.Annotations, event.Type, event.Reason, event.Action, event.Note))
}

type Sink struct {
	tracker *Tracker
	sink    events.EventSink
}

func (t *Tracker) NewSinkFor(sink events.EventSink) *Sink {
	return &Sink{
		tracker: t,
		sink:    sink,
	}
}

func (s *Sink) Create(ctx context.Context, event *eventsv1.Event) (*eventsv1.Event, error) {
	event, err := s.sink.Create(ctx, event)
	if err == nil {
		s.tracker.acknowledge(event)
	}
	return event, err
}

func (s *Sink) Update(ctx context.Context, event *eventsv1.Event) (*eventsv1.Event, error) {
	event, err := s.sink.Update(ctx, event)
	if err == nil {
		s.tracker.acknowledge(event)
	}
	return event, err
}

func (s *Sink) Patch(ctx context.Context, oldEvent *eventsv1.Event, data []byte) (*eventsv1.Event, error) {
	event, err := s.sink.Patch(ctx, oldEvent, data)
	if err == nil {
		s.tracker.acknowledge(event)
	}
	return event, err
}

type Recorder struct {
	tracker  *Tracker
	recorder recorder.EventRecorder
}

func (t *Tracker) NewRecorderFor(recorder recorder.EventRecorder) *Recorder {
	return &Recorder{
		tracker:  t,
		recorder: recorder,
	}
}

func (r *Recorder) Eventf(regarding runtime.Object, related runtime.Object, eventtype, reason, action, note string, args ...any) {
	r.tracker.register(regarding, related, nil, eventtype, reason, action, fmt.Sprintf(note, args...))
	r.recorder.Eventf(regarding, related, eventtype, reason, action, note, args...)
}

func (r *Recorder) AnnotatedEventf(regarding runtime.Object, related runtime.Object, annotations map[string]string, eventtype, reason, action, note string, args ...any) {
	r.tracker.register(regarding, related, annotations, eventtype, reason, action, fmt.Sprintf(note, args...))
	r.recorder.AnnotatedEventf(regarding, related, annotations, eventtype, reason, action, note, args...)
}
