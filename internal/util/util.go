/*
SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package util

import (
	"crypto/sha256"
	"encoding/base32"
	"encoding/hex"
	"encoding/json"
	"strings"
)

func Must[T any](x T, err error) T {
	if err != nil {
		panic(err)
	}
	return x
}

func Sha256hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func Sha256base32(data []byte) string {
	sum := sha256.Sum256(data)
	return strings.ToLower(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(sum[:]))
}

func CalculateDigest(values ...any) string {
	// note: this Must() is ok because the input values are expected to be JSON values
	return Sha256hex(Must(json.Marshal(values)))
}
