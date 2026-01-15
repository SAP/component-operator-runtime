/*
SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package kustomize

// TODO: consolidate all the util files into an internal reuse package

func ref[T any](x T) *T {
	return &x
}
