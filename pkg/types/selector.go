/*
SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package types

type Selector[T any] interface {
	Matches(object T) bool
}

type SelectorFunc[T any] func(object T) bool

func (s SelectorFunc[T]) Matches(object T) bool {
	return s(object)
}

var _ Selector[any] = SelectorFunc[any](nil)
