/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package component

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"math/rand"
	"strings"
	"time"
)

// TODO: consolidate all the util files into an internal reuse package

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

func calculateDigest(values ...any) string {
	// note: this must() is ok because the input values are expected to be JSON values
	return sha256hex(must(json.Marshal(values)))
}

func capitalize(s string) string {
	if len(s) <= 1 {
		return s
	}
	return strings.ToUpper(s[0:1]) + s[1:]
}

func addJitter(d *time.Duration, minPercent int, maxPercent int) {
	if minPercent > maxPercent {
		return
	}
	min := int64(minPercent)
	max := int64(maxPercent)
	*d = *d + time.Duration((rand.Int63n(max-min+1)+min)*int64(*d)/100)
}
