/*
SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package events_test

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/sap/component-operator-runtime/internal/events"
)

var _ = Describe("testing: recorder.go", func() {

	var object1 client.Object
	var object2 client.Object
	var object3 client.Object

	var event11 *Event
	var event12 *Event
	var event13 *Event
	var event14 *Event
	var event15 *Event
	var event16 *Event
	var event17 *Event
	var event18 *Event
	var event19 *Event

	var event21 *Event

	var capture *Capture
	var recorder *events.DeduplicatingRecorder

	var eventf func(event *Event)
	var annotatedEventf func(event *Event)

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
		object3 = &metav1.PartialObjectMetadata{
			ObjectMeta: metav1.ObjectMeta{
				UID: "3",
			},
		}

		event11 = &Event{
			Regarding: object1,
		}
		event12 = &Event{
			Regarding: object1,
			Related:   object3,
		}
		event13 = &Event{
			Regarding: object1,
			Related:   object3,
			Type:      "type-a",
		}
		event14 = &Event{
			Regarding: object1,
			Related:   object3,
			Type:      "type-a",
			Reason:    "reason-a",
		}
		event15 = &Event{
			Regarding: object1,
			Related:   object3,
			Type:      "type-a",
			Reason:    "reason-a",
			Action:    "action-a",
		}
		event16 = &Event{
			Regarding: object1,
			Related:   object3,
			Type:      "type-a",
			Reason:    "reason-a",
			Action:    "action-a",
			Note:      "note-a",
		}
		event17 = &Event{
			Regarding: object1,
			Related:   object3,
			Type:      "type-a",
			Reason:    "reason-a",
			Action:    "action-a",
			Note:      "note-a",
			Args:      []any{"arg-a1"},
		}
		event18 = &Event{
			Regarding:   object1,
			Related:     object3,
			Annotations: map[string]string{"foo": "bar"},
			Type:        "type-a",
			Reason:      "reason-a",
			Action:      "action-a",
			Note:        "note-a",
			Args:        []any{"arg-a1"},
		}
		event19 = &Event{
			Regarding:   object1,
			Related:     object3,
			Annotations: map[string]string{"foo": "baz"},
			Type:        "type-a",
			Reason:      "reason-a",
			Action:      "action-a",
			Note:        "note-a",
			Args:        []any{"arg-a1"},
		}

		event21 = &Event{
			Regarding: object2,
		}

		capture = &Capture{}
		recorder = events.NewDeduplicatingRecorder(capture, 2000*time.Millisecond)

		eventf = func(event *Event) {
			recorder.Eventf(event.Regarding, event.Related, event.Type, event.Reason, event.Action, event.Note, event.Args...)
		}
		annotatedEventf = func(event *Event) {
			recorder.AnnotatedEventf(event.Regarding, event.Related, event.Annotations, event.Type, event.Reason, event.Action, event.Note, event.Args...)
		}
	})

	It("should deduplicate events for one object", func() {
		capture.Start()
		eventf(event11)
		Expect(capture.Stop()).To(Equal(captured(event11)))

		capture.Start()
		eventf(event11)
		Expect(capture.Stop()).To(BeNil())

		capture.Start()
		eventf(event12)
		Expect(capture.Stop()).To(Equal(captured(event12)))

		capture.Start()
		eventf(event12)
		Expect(capture.Stop()).To(BeNil())

		capture.Start()
		eventf(event13)
		Expect(capture.Stop()).To(Equal(captured(event13)))

		capture.Start()
		eventf(event13)
		Expect(capture.Stop()).To(BeNil())

		capture.Start()
		eventf(event14)
		Expect(capture.Stop()).To(Equal(captured(event14)))

		capture.Start()
		eventf(event14)
		Expect(capture.Stop()).To(BeNil())

		capture.Start()
		eventf(event15)
		Expect(capture.Stop()).To(Equal(captured(event15)))

		capture.Start()
		eventf(event15)
		Expect(capture.Stop()).To(BeNil())

		capture.Start()
		eventf(event16)
		Expect(capture.Stop()).To(Equal(captured(event16)))

		capture.Start()
		eventf(event16)
		Expect(capture.Stop()).To(BeNil())

		capture.Start()
		eventf(event17)
		Expect(capture.Stop()).To(Equal(captured(event17)))

		capture.Start()
		eventf(event17)
		Expect(capture.Stop()).To(BeNil())

		capture.Start()
		annotatedEventf(event18)
		Expect(capture.Stop()).To(Equal(captured(event18)))

		capture.Start()
		annotatedEventf(event18)
		Expect(capture.Stop()).To(BeNil())

		capture.Start()
		annotatedEventf(event19)
		Expect(capture.Stop()).To(Equal(captured(event19)))

		capture.Start()
		annotatedEventf(event19)
		Expect(capture.Stop()).To(BeNil())
	})

	It("should not deduplicate events for different objects", func() {
		capture.Start()
		eventf(event11)
		Expect(capture.Stop()).To(Equal(captured(event11)))

		capture.Start()
		eventf(event21)
		Expect(capture.Stop()).To(Equal(captured(event21)))
	})

	It("should produce the same results, regardless of using Eventf() or AnnotatedEventf()", func() {
		capture.Start()
		eventf(event17)
		Expect(capture.Stop()).To(Equal(captured(event17)))

		capture.Start()
		eventf(event17)
		Expect(capture.Stop()).To(BeNil())

		capture.Start()
		annotatedEventf(event17)
		Expect(capture.Stop()).To(BeNil())
	})

	It("should forget stored events after epxiration", func() {
		capture.Start()
		eventf(event15)
		Expect(capture.Stop()).To(Equal(captured(event15)))

		time.Sleep(1500 * time.Millisecond)

		capture.Start()
		eventf(event15)
		Expect(capture.Stop()).To(BeNil())

		time.Sleep(600 * time.Millisecond)

		capture.Start()
		eventf(event15)
		Expect(capture.Stop()).To(Equal(captured(event15)))
	})

})

type Event struct {
	Regarding   client.Object
	Related     client.Object
	Annotations map[string]string
	Type        string
	Reason      string
	Action      string
	Note        string
	Args        []any
}

type CapturedEvent struct {
	Regarding   runtime.Object
	Related     runtime.Object
	Annotations map[string]string
	Type        string
	Reason      string
	Action      string
	Note        string
	Args        []any
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

func (c *Capture) Eventf(regarding runtime.Object, related runtime.Object, eventtype string, reason string, action string, note string, args ...any) {
	if !c.active {
		panic("Capture not started")
	}
	c.event = &CapturedEvent{
		Regarding:   regarding,
		Related:     related,
		Annotations: nil,
		Type:        eventtype,
		Reason:      reason,
		Action:      action,
		Note:        note,
		Args:        args,
	}
}

func (c *Capture) AnnotatedEventf(regarding runtime.Object, related runtime.Object, annotations map[string]string, eventtype string, reason string, action string, note string, args ...any) {
	if !c.active {
		panic("Capture not started")
	}
	c.event = &CapturedEvent{
		Regarding:   regarding,
		Related:     related,
		Annotations: annotations,
		Type:        eventtype,
		Reason:      reason,
		Action:      action,
		Note:        note,
		Args:        args,
	}
}

func captured(event *Event) *CapturedEvent {
	return &CapturedEvent{
		Regarding:   event.Regarding,
		Related:     event.Related,
		Annotations: event.Annotations,
		Type:        event.Type,
		Reason:      event.Reason,
		Action:      event.Action,
		Note:        event.Note,
		Args:        event.Args,
	}
}
