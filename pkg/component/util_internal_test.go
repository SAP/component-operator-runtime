/*
SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package component

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sap/component-operator-runtime/pkg/types"
)

var _ = Describe("testing: util.go", func() {
	Context("testing: sha256hex()", func() {
		It("should generate correct sha256 digest (as hex)", func() {
			Expect(sha256hex([]byte("BzmXTiWs0KGdcDapQYmfVmzfDvEIXrCz"))).To(Equal("720350190661930afb966b47a42b98ca078f3394243f078442f2938ee3031980"))
		})
	})

	Context("testing: sha256base32()", func() {
		It("should generate correct sha256 digest (as base32)", func() {
			Expect(sha256base32([]byte("BzmXTiWs0KGdcDapQYmfVmzfDvEIXrCz"))).To(Equal("oibvagigmgjqv64wnnd2ik4yzidy6m4ueq7qpbcc6kjy5yyddgaa"))
		})
	})

	Context("testing: capitalize()", func() {
		It("should convert the first letter of the string to upper case", func() {
			Expect(capitalize("this is a test")).To(Equal("This is a test"))
		})
	})

	Context("testing: isManaged()", func() {
		var inventory []*InventoryItem

		BeforeEach(func() {
			inventory = []*InventoryItem{
				{
					ManagedTypes: []TypeInfo{
						{Group: "g1", Version: "v", Kind: "k"},
						{Group: "g2", Version: "v", Kind: "*"},
					},
				},
				{
					ManagedTypes: []TypeInfo{
						{Group: "g3", Version: "*", Kind: "k"},
					},
				},
			}
		})

		DescribeTable("testing: isManaged()",
			func(group string, version string, kind string, expectedIsManaged bool) {
				isManaged := isManaged(inventory, types.TypeKeyFromGroupAndVersionAndKind(group, version, kind))
				Expect(isManaged).To(Equal(expectedIsManaged))
			},
			Entry(nil, "g1", "v", "k", true),
			Entry(nil, "g1", "w", "k", false),
			Entry(nil, "g1", "v", "l", false),
			Entry(nil, "g2", "v", "k", true),
			Entry(nil, "g2", "w", "k", false),
			Entry(nil, "g2", "v", "l", true),
			Entry(nil, "g3", "v", "k", true),
			Entry(nil, "g3", "w", "k", true),
			Entry(nil, "g3", "v", "l", false),
			Entry(nil, "g4", "v", "k", false),
		)
	})
})
