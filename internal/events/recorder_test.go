/*
SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package events_test

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	apitypes "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/sap/component-operator-runtime/internal/events"
)

var _ = Describe("testing: recorder.go", func() {

	var object1 client.Object
	var object2 client.Object
	var event1 *Event
	var event2 *Event
	var event3 *Event
	var capture *Capture
	var recorder *events.DeduplicatingRecorder

	BeforeEach(func() {
		object1 = &metav1.PartialObjectMetadata{
			ObjectMeta: metav1.ObjectMeta{
				UID: "1",
			},
		}
		object2 = &metav1.PartialObjectMetadata{
			ObjectMeta: metav1.ObjectMeta{
				UID: "2",
			},
		}
		event1 = &Event{
			Type:    corev1.EventTypeNormal,
			Reason:  "reason1",
			Message: "message1",
		}
		event2 = &Event{
			Type:    corev1.EventTypeWarning,
			Reason:  "reason2",
			Message: "message2",
		}
		event3 = &Event{
			Type:    corev1.EventTypeWarning,
			Reason:  "reason2",
			Message: "message2",
			Annotations: map[string]string{
				"key": "value",
			},
		}
		capture = &Capture{}
		recorder = events.NewDeduplicatingRecorder(capture, 2000*time.Millisecond)
	})

	It("should deduplicate events for one object", func() {
		capture.Start()
		recorder.Event(object1, event1.Type, event1.Reason, event1.Message)
		Expect(capture.Stop()).To(Equal(captured(object1, event1)))

		capture.Start()
		recorder.Event(object1, event1.Type, event1.Reason, event1.Message)
		Expect(capture.Stop()).To(BeNil())

		capture.Start()
		recorder.Event(object1, event2.Type, event2.Reason, event2.Message)
		Expect(capture.Stop()).To(Equal(captured(object1, event2)))

		capture.Start()
		recorder.Event(object1, event1.Type, event1.Reason, event1.Message)
		Expect(capture.Stop()).To(Equal(captured(object1, event1)))
	})

	It("should not deduplicate events for different objects", func() {
		capture.Start()
		recorder.Event(object1, event1.Type, event1.Reason, event1.Message)
		Expect(capture.Stop()).To(Equal(captured(object1, event1)))

		capture.Start()
		recorder.Event(object2, event1.Type, event1.Reason, event1.Message)
		Expect(capture.Stop()).To(Equal(captured(object2, event1)))
	})

	It("should produce the same results, regardless of using Event(), Eventf() or AnnotatedEventf()", func() {
		capture.Start()
		recorder.Event(object1, event1.Type, event1.Reason, event1.Message)
		Expect(capture.Stop()).To(Equal(captured(object1, event1)))

		capture.Start()
		recorder.Eventf(object1, event1.Type, event1.Reason, "%s", event1.Message)
		Expect(capture.Stop()).To(BeNil())

		capture.Start()
		recorder.AnnotatedEventf(object1, nil, event1.Type, event1.Reason, "%s", event1.Message)
		Expect(capture.Stop()).To(BeNil())
	})

	It("should handle event annotations correctly", func() {
		capture.Start()
		recorder.AnnotatedEventf(object1, event3.Annotations, event3.Type, event3.Reason, "%s", event3.Message)
		Expect(capture.Stop()).To(Equal(captured(object1, event3)))
	})

	It("should forget stored events after epxiration", func() {
		capture.Start()
		recorder.Event(object1, event1.Type, event1.Reason, event1.Message)
		Expect(capture.Stop()).To(Equal(captured(object1, event1)))

		time.Sleep(1500 * time.Millisecond)

		capture.Start()
		recorder.Event(object1, event1.Type, event1.Reason, event1.Message)
		Expect(capture.Stop()).To(BeNil())

		time.Sleep(600 * time.Millisecond)

		capture.Start()
		recorder.Event(object1, event1.Type, event1.Reason, event1.Message)
		Expect(capture.Stop()).To(Equal(captured(object1, event1)))
	})

})

type Event struct {
	Type        string
	Reason      string
	Message     string
	Annotations map[string]string
}

type CapturedEvent struct {
	Event
	UID apitypes.UID
}

type Capture struct {
	active bool
	event  *CapturedEvent
}

func (c *Capture) Start() {
	if c.active {
		panic("Capture already started")
	}
	c.active = true
	c.event = nil
}

func (c *Capture) Stop() *CapturedEvent {
	if !c.active {
		panic("Capture not started")
	}
	c.active = false
	return c.event
}

func (c *Capture) Event(object runtime.Object, eventtype, reason, message string) {
	if !c.active {
		panic("Capture not started")
	}
	c.event = &CapturedEvent{
		Event: Event{
			Type:    eventtype,
			Reason:  reason,
			Message: message,
		},
		UID: object.(metav1.Object).GetUID(),
	}
}

func (c *Capture) Eventf(object runtime.Object, eventtype, reason, messageFmt string, args ...any) {
	if !c.active {
		panic("Capture not started")
	}
	c.event = &CapturedEvent{
		Event: Event{
			Type:    eventtype,
			Reason:  reason,
			Message: fmt.Sprintf(messageFmt, args...),
		},
		UID: object.(metav1.Object).GetUID(),
	}
}

func (c *Capture) AnnotatedEventf(object runtime.Object, annotations map[string]string, eventtype, reason, messageFmt string, args ...any) {
	if !c.active {
		panic("Capture not started")
	}
	c.event = &CapturedEvent{
		Event: Event{
			Type:        eventtype,
			Reason:      reason,
			Message:     fmt.Sprintf(messageFmt, args...),
			Annotations: annotations,
		},
		UID: object.(metav1.Object).GetUID(),
	}
}

func captured(object client.Object, event *Event) *CapturedEvent {
	return &CapturedEvent{
		Event: *event,
		UID:   object.GetUID(),
	}
}
