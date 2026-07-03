/*
SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package util_test

import (
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/sap/component-operator-runtime/internal/util"
)

var _ = Describe("testing: util.go", func() {

	Describe("testing: Must()", func() {

		It("should not panic if no error occurs", func() {
			Expect(func() { util.Must(5, nil) }).NotTo(Panic())
		})

		It("should panic if an error occurs", func() {
			Expect(func() { util.Must(5, errors.New("an error occurred")) }).To(Panic())
		})

	})

	Describe("testing: Sha256hex()", func() {

		It("should produce a valid SHA-256 hex string", func() {
			Expect(util.Sha256hex([]byte("test"))).To(Equal("9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08"))
		})

	})

	Describe("testing: Sha256base32()", func() {

		It("should produce a valid SHA-256 base32 string", func() {
			Expect(util.Sha256base32([]byte("test"))).To(Equal("t6dnbamijr6wlgrp5kqmkwwqcwr36ty3fmfyelgrlvwblmhqbiea"))
		})

	})

	Describe("testing: CalculateDigest()", func() {

		It("should produce a valid digest", func() {
			Expect(util.CalculateDigest("foo", 42)).To(Equal("d264cec0c2b8e8a9ea304071dc7351f75b6b4f0de4567bd33a443fc6590d561b"))
		})

	})

})
