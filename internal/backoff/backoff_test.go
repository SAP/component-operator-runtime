/*
SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package backoff_test

import (
	"time"

	"k8s.io/client-go/util/workqueue"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/sap/component-operator-runtime/internal/backoff"
)

var _ = Describe("testing: backoff.go", func() {

	It("should return sequential intervals per item/activity", func() {
		backoff := backoff.NewBackoff(workqueue.NewItemExponentialFailureRateLimiter(1*time.Millisecond, 1*time.Second))

		Expect(backoff.Next("item-1", "activity-1")).To(Equal(1 * time.Millisecond))
		Expect(backoff.Next("item-1", "activity-1")).To(Equal(2 * time.Millisecond))
		Expect(backoff.Next("item-1", "activity-1")).To(Equal(4 * time.Millisecond))
		Expect(backoff.Next("item-1", "activity-1")).To(Equal(8 * time.Millisecond))

		Expect(backoff.Next("item-2", "activity-1")).To(Equal(1 * time.Millisecond))
		Expect(backoff.Next("item-2", "activity-1")).To(Equal(2 * time.Millisecond))

		Expect(backoff.Next("item-1", "activity-2")).To(Equal(1 * time.Millisecond))
		Expect(backoff.Next("item-1", "activity-2")).To(Equal(2 * time.Millisecond))

		backoff.Forget("item-2")

		Expect(backoff.Next("item-2", "activity-1")).To(Equal(1 * time.Millisecond))
	})

})
