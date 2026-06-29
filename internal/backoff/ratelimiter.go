/*
SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package backoff

import (
	"time"

	"k8s.io/client-go/util/workqueue"
)

/*
The default rate limiter does
- 5 quick roundtrips (exponential, below 1s)
- then 15 roundtrips at 1s
- then 30 roundtrips at 2s
- then roundtrips at maxDelay
*/

func NewDefaultRateLimiter(maxDelay time.Duration) workqueue.RateLimiter {
	return workqueue.NewMaxOfRateLimiter(
		workqueue.NewItemExponentialFailureRateLimiter(50*time.Millisecond, 1*time.Second),
		workqueue.NewItemFastSlowRateLimiter(0, 2*time.Second, 20),
		workqueue.NewItemFastSlowRateLimiter(0, maxDelay, 20+30),
	)
}
