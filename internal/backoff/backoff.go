/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
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
