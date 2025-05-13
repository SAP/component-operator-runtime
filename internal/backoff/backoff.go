/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package backoff

import (
	"sync"
	"time"

	"k8s.io/client-go/util/workqueue"
)

type Backoff struct {
	lock       sync.Mutex
	activities map[any]any
	limiter    workqueue.RateLimiter
}

/*
Returned backoff does
- 5 quick roundtrips (exponential, below 1s)
- then 15 roundtrips at 1s
- then 30 roundtrips at 2s
- then rounttrips at 10s
*/

func NewBackoff(maxDelay time.Duration) *Backoff {
	return &Backoff{
		activities: make(map[any]any),
		limiter: workqueue.NewMaxOfRateLimiter(
			workqueue.NewItemExponentialFailureRateLimiter(50*time.Millisecond, 1*time.Second),
			workqueue.NewItemFastSlowRateLimiter(0, 2*time.Second, 20),
			workqueue.NewItemFastSlowRateLimiter(0, maxDelay, 20+30),
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
