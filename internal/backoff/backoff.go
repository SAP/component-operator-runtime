/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package backoff

import (
	"sync"
	"time"

	"golang.org/x/time/rate"
	"k8s.io/client-go/util/workqueue"
)

type Backoff struct {
	lock       sync.Mutex
	activities map[any]any
	limiter    workqueue.RateLimiter
}

func NewBackoff(maxDelay time.Duration) *Backoff {
	return &Backoff{
		activities: make(map[any]any),
		// resulting per-item backoff is the maximum of a 200-times-50ms-then-maxDelay per-item limiter,
		// and an overall 5-per-second-burst-20 bucket limiter;
		// as a consequence, we have up to
		// - up to 20 almost immediate retries
		// - then then a phase of 5 guaranteed retries per seconnd (could be more if burst capacity is refilled
		//   because of the duration of the reconcile logic execution itself)
		// - finally (after 200 iterations) slow retries at the rate given by maxDelay
		limiter: workqueue.NewMaxOfRateLimiter(
			workqueue.NewItemFastSlowRateLimiter(50*time.Millisecond, maxDelay, 200),
			&workqueue.BucketRateLimiter{Limiter: rate.NewLimiter(rate.Limit(5), 20)},
		),
	}
}

func (b *Backoff) Next(item any, activity any) time.Duration {
	b.lock.Lock()
	defer b.lock.Unlock()

	if act, ok := b.activities[item]; ok && act != activity {
		b.limiter.Forget([2]any{item, act})
	}

	b.activities[item] = activity
	return b.limiter.When([2]any{item, activity})
}

func (b *Backoff) Forget(item any) {
	b.lock.Lock()
	defer b.lock.Unlock()

	if act, ok := b.activities[item]; ok {
		b.limiter.Forget([2]any{item, act})
	}

	delete(b.activities, item)
}
