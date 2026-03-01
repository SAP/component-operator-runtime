/*
SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package util

// TODO: consolidate all the util files into an internal reuse package

func must[T any](x T, err error) T {
	if err != nil {
		panic(err)
	}
	return x
}
