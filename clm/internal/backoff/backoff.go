/*
SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package backoff

import "time"

const (
	minBackoff = time.Millisecond
	maxBackoff = 10 * time.Second
)

type Backoff struct {
	duration time.Duration
}

func New() *Backoff {
	return &Backoff{}
}

func (b *Backoff) Next() time.Duration {
	if b.duration < minBackoff {
		b.duration = minBackoff
	} else {
		b.duration *= 2
	}
	if b.duration > maxBackoff {
		b.duration = maxBackoff
	}
	return b.duration
}
