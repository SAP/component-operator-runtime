/*
SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package events

import (
	"fmt"
	"sync"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/recorder"

	"github.com/sap/component-operator-runtime/internal/util"
)

type DeduplicatingRecorder struct {
	recorder   recorder.EventRecorder
	mutex      sync.Mutex
	events     map[string]event
	expiration time.Duration
}

type event struct {
	digest    string
	timestamp time.Time
}

func NewDeduplicatingRecorder(recorder recorder.EventRecorder, expiration time.Duration) *DeduplicatingRecorder {
	return &DeduplicatingRecorder{
		recorder:   recorder,
		events:     make(map[string]event),
		expiration: expiration,
	}
}

func (r *DeduplicatingRecorder) Eventf(regarding client.Object, related client.Object, eventType string, reason string, action string, note string, args ...any) {
	if r.isDuplicate(regarding, related, nil, eventType, reason, action, fmt.Sprintf(note, args...)) {
		return
	}
	r.recorder.Eventf(regarding, related, eventType, reason, action, note, args...)
}

func (r *DeduplicatingRecorder) AnnotatedEventf(regarding client.Object, related client.Object, annotations map[string]string, eventType string, reason string, action string, note string, args ...any) {
	if r.isDuplicate(regarding, related, annotations, eventType, reason, action, fmt.Sprintf(note, args...)) {
		return
	}
	r.recorder.AnnotatedEventf(regarding, related, annotations, eventType, reason, action, note, args...)
}

func (r *DeduplicatingRecorder) isDuplicate(regarding client.Object, related client.Object, annotations map[string]string, eventType string, reason string, action string, message string) bool {
	regardingUid := string(regarding.GetUID())
	relatedUid := ""
	if related != nil {
		relatedUid = string(related.GetUID())
	}
	digest := util.CalculateDigest(relatedUid, annotations, eventType, reason, action, message)
	now := time.Now()
	exp := now.Add(-r.expiration)

	r.mutex.Lock()
	defer r.mutex.Unlock()
	for uid, event := range r.events {
		if event.timestamp.Before(exp) {
			delete(r.events, uid)
		}
	}
	if r.events[regardingUid].digest == digest {
		return true
	} else {
		r.events[regardingUid] = event{
			digest:    digest,
			timestamp: now,
		}
		return false
	}
}
