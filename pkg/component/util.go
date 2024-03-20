/*
SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package component

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// TODO: consolidate all the util files into an own internal reuse packages

func ref[T any](x T) *T {
	return &x
}

func must[T any](x T, err error) T {
	if err != nil {
		panic(err)
	}
	return x
}

func sha256hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func capitalize(s string) string {
	if len(s) <= 1 {
		return s
	}
	return strings.ToUpper(s[0:1]) + s[1:]
}
