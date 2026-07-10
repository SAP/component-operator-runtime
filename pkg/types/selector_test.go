/*
SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package types_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/sap/component-operator-runtime/pkg/types"
)

var _ = Describe("testing: selector.go", func() {

	It("should be a working selector", func() {
		f := types.SelectorFunc[int](func(x int) bool {
			return x%2 == 0
		})
		Expect(f.Matches(4)).To(BeTrue())
		Expect(f.Matches(5)).To(BeFalse())
	})

})
