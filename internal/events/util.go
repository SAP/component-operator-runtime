/*
SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package events

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

// TODO: consolidate all the util files into an internal reuse package

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

func calculateDigest(values ...any) string {
	// note: this must() is ok because the input values are expected to be JSON values
	return sha256hex(must(json.Marshal(values)))
}
