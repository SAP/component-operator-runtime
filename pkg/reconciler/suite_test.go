/*
SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package reconciler

import (
	"os"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/sap/component-operator-runtime/testing/environment"
)

func TestPackage(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Component Suite")
}

var env *environment.Environment

var _ = BeforeSuite(func() {
	By("initializing suite")
	var err error
	env, err = environment.Run(os.Stdout, os.Stderr, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())
})

var _ = AfterSuite(func() {
	By("tearing down suite")
	err := env.Stop()
	Expect(err).NotTo(HaveOccurred())
})
