/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package types

import "time"

type RetriableError struct {
	err        error
	retryAfter *time.Duration
}

func NewRetriableError(err error, retryAfter *time.Duration) RetriableError {
	return RetriableError{err: err, retryAfter: retryAfter}
}

func (e RetriableError) Error() string {
	return e.err.Error()
}

func (e RetriableError) Unwrap() error {
	return e.err
}

func (e RetriableError) Cause() error {
	return e.err
}

func (e RetriableError) RetryAfter() *time.Duration {
	return e.retryAfter
}
