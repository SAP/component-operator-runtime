/*
SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and component-operator-runtime contributors
SPDX-License-Identifier: Apache-2.0
*/

package reconciler_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "github.com/sap/component-operator-runtime/internal/testing"

	barv1alpha1 "github.com/sap/component-operator-runtime/internal/testing/bar/api/v1alpha1"
	foov1alpha1 "github.com/sap/component-operator-runtime/internal/testing/foo/api/v1alpha1"
	"github.com/sap/component-operator-runtime/pkg/cluster"
	. "github.com/sap/component-operator-runtime/pkg/reconciler"
)

var _ = Describe("testing: reconciler.go", func() {
	var reconcilerName string
	var ctx context.Context
	var cli cluster.Client

	BeforeEach(func() {
		reconcilerName = "reconciler.testing.local"
		ctx = Ctx()
		cli = Client()
	})

	Context("testing: Apply()", func() {
		var reconciler *Reconciler
		var inventory []*InventoryItem
		var namespace string
		var ownerId string

		BeforeEach(func() {
			reconciler = NewReconciler(reconcilerName, cli, ReconcilerOptions{})
			namespace = CreateNamespace()
			ownerId = "test-owner"
		})

		AfterEach(func() {
			CleanupNamespace(namespace, &foov1alpha1.Foo{}, &barv1alpha1.Bar{})
		})

		It("should apply objects to inventory correctly", func() {
			_, err := reconciler.Apply(ctx, &inventory, nil, namespace, ownerId, 1)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
