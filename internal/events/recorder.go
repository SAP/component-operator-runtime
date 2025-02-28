/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package events

import (
	"fmt"
	"sync"
	"time"

	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type DeduplicatingRecorder struct {
	recorder record.EventRecorder
	mutex    sync.Mutex
	events   map[string]event
}

type event struct {
	digest    string
	timestamp time.Time
}

func NewDeduplicatingRecorder(recorder record.EventRecorder) *DeduplicatingRecorder {
	return &DeduplicatingRecorder{
		recorder: recorder,
		events:   make(map[string]event),
	}
}

func (r *DeduplicatingRecorder) Event(object client.Object, eventType string, reason string, message string) {
	if r.isDuplicate(object, nil, eventType, reason, message) {
		return
	}
	r.recorder.Event(object, eventType, reason, message)
}

func (r *DeduplicatingRecorder) Eventf(object client.Object, eventType string, reason string, messageFmt string, args ...any) {
	if r.isDuplicate(object, nil, eventType, reason, fmt.Sprintf(messageFmt, args...)) {
		return
	}
	r.recorder.Eventf(object, eventType, reason, messageFmt, args...)
}

func (r *DeduplicatingRecorder) AnnotatedEventf(object client.Object, annotations map[string]string, eventType string, reason string, messageFmt string, args ...any) {
	if r.isDuplicate(object, annotations, eventType, reason, fmt.Sprintf(messageFmt, args...)) {
		return
	}
	r.recorder.AnnotatedEventf(object, annotations, eventType, reason, messageFmt, args...)
}

func (r *DeduplicatingRecorder) isDuplicate(object client.Object, annotations map[string]string, eventType, reason, message string) bool {
	uid := string(object.GetUID())
	digest := calculateDigest(annotations, eventType, reason, message)
	now := time.Now()
	exp := time.Now().Add(-5 * time.Minute)

	r.mutex.Lock()
	defer r.mutex.Unlock()
	for uid, event := range r.events {
		if event.timestamp.Before(exp) {
			delete(r.events, uid)
		}
	}
	if r.events[uid].digest == digest {
		return true
	} else {
		r.events[uid] = event{
			digest:    digest,
			timestamp: now,
		}
		return false
	}
}
