/*
SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package backoff_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/sap/component-operator-runtime/internal/backoff"
)

var _ = Describe("testing: ratelimiter.go", func() {

	It("should produce correct intervals", func() {
		ratelimiter := backoff.NewDefaultRateLimiter(5 * time.Second)
		for i := range 100 {
			switch {
			case i < 5:
				Expect(ratelimiter.When("test")).To(Equal(50 * (1 << i) * time.Millisecond))
			case i < 20:
				Expect(ratelimiter.When("test")).To(Equal(1 * time.Second))
			case i < 50:
				Expect(ratelimiter.When("test")).To(Equal(2 * time.Second))
			default:
				Expect(ratelimiter.When("test")).To(Equal(5 * time.Second))
			}
		}
	})

})
