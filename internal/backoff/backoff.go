/*
SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and redis-operator contributors
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

func NewBackoff(maxDelay time.Duration) *Backoff {
	return &Backoff{
		activities: make(map[any]any),
		limiter:    workqueue.NewItemExponentialFailureRateLimiter(20*time.Millisecond, maxDelay),
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
