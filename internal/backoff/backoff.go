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
		// resulting per-item backoff is the maximum of a 120-times-100ms-then-maxDelay per-item limiter,
		// and an overall 1-per-second-burst-50 bucket limiter;
		// as a consequence, we have
		// - a phase of 10 retries per second for the first 5 seconds
		// - then a phase of 1 retry per second for the next 60 seconds
		// - finally slow retries at the rate given by maxDelay
		limiter: workqueue.NewMaxOfRateLimiter(
			workqueue.NewItemFastSlowRateLimiter(100*time.Millisecond, maxDelay, 120),
			&workqueue.BucketRateLimiter{Limiter: rate.NewLimiter(rate.Limit(1), 50)},
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
