/*
SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package component

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("testing: util.go", func() {
	Context("testing: sha256hex()", func() {
		It("should generate correct sha256 digest (as hex)", func() {
			Expect(sha256hex([]byte("BzmXTiWs0KGdcDapQYmfVmzfDvEIXrCz"))).To(Equal("720350190661930afb966b47a42b98ca078f3394243f078442f2938ee3031980"))
		})
	})

	Context("testing: capitalize()", func() {
		It("should convert the first letter of the string to upper case", func() {
			Expect(capitalize("this is a test")).To(Equal("This is a test"))
		})
	})
})
